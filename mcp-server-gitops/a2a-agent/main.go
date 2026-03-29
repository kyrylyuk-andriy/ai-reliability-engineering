package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"trpc.group/trpc-go/trpc-a2a-go/protocol"
	"trpc.group/trpc-go/trpc-a2a-go/server"
	"trpc.group/trpc-go/trpc-a2a-go/taskmanager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type k8sHealthProcessor struct {
	client kubernetes.Interface
}

func newK8sHealthProcessor() (*k8sHealthProcessor, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, fmt.Errorf("cannot create k8s config: %w", err)
		}
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create k8s client: %w", err)
	}
	return &k8sHealthProcessor{client: client}, nil
}

func (p *k8sHealthProcessor) ProcessMessage(
	ctx context.Context,
	message protocol.Message,
	options taskmanager.ProcessOptions,
	handle taskmanager.TaskHandler,
) (*taskmanager.MessageProcessingResult, error) {
	text := extractText(message)
	var result string
	var err error

	switch {
	case containsAny(text, "pod", "pods"):
		result, err = p.getPodStatus(ctx, extractNamespace(text))
	case containsAny(text, "node", "nodes"):
		result, err = p.getNodeStatus(ctx)
	case containsAny(text, "deploy", "deployment"):
		result, err = p.getDeploymentStatus(ctx, extractNamespace(text))
	case containsAny(text, "event", "warning"):
		result, err = p.getEvents(ctx)
	default:
		result, err = p.getClusterSummary(ctx)
	}

	if err != nil {
		result = fmt.Sprintf("Error: %v", err)
	}

	responseMessage := protocol.NewMessage(
		protocol.MessageRoleAgent,
		[]protocol.Part{protocol.NewTextPart(result)},
	)
	return &taskmanager.MessageProcessingResult{
		Result: &responseMessage,
	}, nil
}

func (p *k8sHealthProcessor) getPodStatus(ctx context.Context, ns string) (string, error) {
	pods, err := p.client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	if len(pods.Items) == 0 {
		return fmt.Sprintf("No pods found in namespace %q", ns), nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Pods in namespace %q:\n\n", ns))
	for _, pod := range pods.Items {
		ready := 0
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
		}
		sb.WriteString(fmt.Sprintf("  %-45s %s (%d/%d ready)\n",
			pod.Name, pod.Status.Phase, ready, len(pod.Spec.Containers)))
	}
	return sb.String(), nil
}

func (p *k8sHealthProcessor) getNodeStatus(ctx context.Context) (string, error) {
	nodes, err := p.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString("Cluster Nodes:\n\n")
	for _, node := range nodes.Items {
		status := "NotReady"
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				status = "Ready"
			}
		}
		sb.WriteString(fmt.Sprintf("  %-35s %s (%s)\n",
			node.Name, status, node.Status.NodeInfo.KubeletVersion))
	}
	return sb.String(), nil
}

func (p *k8sHealthProcessor) getDeploymentStatus(ctx context.Context, ns string) (string, error) {
	deployments, err := p.client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	if len(deployments.Items) == 0 {
		return fmt.Sprintf("No deployments found in namespace %q", ns), nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Deployments in namespace %q:\n\n", ns))
	for _, d := range deployments.Items {
		sb.WriteString(fmt.Sprintf("  %-40s %d/%d ready\n",
			d.Name, d.Status.ReadyReplicas, *d.Spec.Replicas))
	}
	return sb.String(), nil
}

func (p *k8sHealthProcessor) getEvents(ctx context.Context) (string, error) {
	events, err := p.client.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: "type=Warning",
		Limit:         10,
	})
	if err != nil {
		return "", err
	}
	if len(events.Items) == 0 {
		return "No warning events found — cluster is healthy", nil
	}
	var sb strings.Builder
	sb.WriteString("Warning Events:\n\n")
	for _, e := range events.Items {
		sb.WriteString(fmt.Sprintf("  [%s] %s/%s: %s (x%d)\n",
			e.LastTimestamp.Format("15:04:05"),
			e.InvolvedObject.Kind, e.InvolvedObject.Name,
			e.Message, e.Count))
	}
	return sb.String(), nil
}

func (p *k8sHealthProcessor) getClusterSummary(ctx context.Context) (string, error) {
	nodes, _ := p.getNodeStatus(ctx)
	pods, _ := p.getPodStatus(ctx, "kagent")
	events, _ := p.getEvents(ctx)
	return fmt.Sprintf("=== Cluster Health Summary ===\n\n%s\n%s\n%s", nodes, pods, events), nil
}

func extractText(msg protocol.Message) string {
	for _, part := range msg.Parts {
		if tp, ok := part.(protocol.TextPart); ok {
			return strings.ToLower(tp.Text)
		}
	}
	return ""
}

func extractNamespace(text string) string {
	// Simple namespace extraction from text
	for _, word := range []string{"kagent", "kube-system", "flux-system", "agentgateway-system", "default"} {
		if strings.Contains(text, word) {
			return word
		}
	}
	return "default"
}

func containsAny(text string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}

func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }

func main() {
	port := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		port = ":" + p
	}

	processor, err := newK8sHealthProcessor()
	if err != nil {
		log.Fatalf("Failed to create K8s client: %v", err)
	}

	providerURL := "https://github.com/kyrylyuk-andriy/ai-reliability-engineering"
	apiKeyName := "X-API-Key"
	apiKeyIn := server.SecuritySchemeIn("header")

	agentCard := server.AgentCard{
		Name:        "K8s Health Agent",
		Description: "A2A-compliant Kubernetes cluster health checker. Inspects pods, nodes, deployments, and warning events.",
		URL:         fmt.Sprintf("http://localhost%s/", port),
		Version:     "0.1.0",
		Provider: &server.AgentProvider{
			Organization: "AI Reliability Engineering",
			URL:          &providerURL,
		},
		Capabilities: server.AgentCapabilities{
			Streaming:         boolPtr(false),
			PushNotifications: boolPtr(false),
		},
		DefaultInputModes:  []string{protocol.KindText},
		DefaultOutputModes: []string{protocol.KindText},
		Skills: []server.AgentSkill{
			{
				ID:          "pod-status",
				Name:        "Pod Status",
				Description: stringPtr("List pods with their status in a namespace"),
				InputModes:  []string{protocol.KindText},
				OutputModes: []string{protocol.KindText},
			},
			{
				ID:          "node-status",
				Name:        "Node Status",
				Description: stringPtr("List cluster nodes with conditions"),
				InputModes:  []string{protocol.KindText},
				OutputModes: []string{protocol.KindText},
			},
			{
				ID:          "deployment-status",
				Name:        "Deployment Status",
				Description: stringPtr("List deployments with replica counts"),
				InputModes:  []string{protocol.KindText},
				OutputModes: []string{protocol.KindText},
			},
			{
				ID:          "warning-events",
				Name:        "Warning Events",
				Description: stringPtr("Get recent warning events from the cluster"),
				InputModes:  []string{protocol.KindText},
				OutputModes: []string{protocol.KindText},
			},
			{
				ID:          "cluster-summary",
				Name:        "Cluster Summary",
				Description: stringPtr("Full cluster health summary including nodes, pods, and events"),
				InputModes:  []string{protocol.KindText},
				OutputModes: []string{protocol.KindText},
			},
		},
		SecuritySchemes: map[string]server.SecurityScheme{
			"apiKey": {
				Type: server.SecuritySchemeTypeAPIKey,
				Name: &apiKeyName,
				In:   &apiKeyIn,
			},
		},
		Security: []map[string][]string{
			{"apiKey": {}},
		},
	}

	taskManager, err := taskmanager.NewMemoryTaskManager(processor)
	if err != nil {
		log.Fatalf("Failed to create task manager: %v", err)
	}

	srv, err := server.NewA2AServer(agentCard, taskManager)
	if err != nil {
		log.Fatalf("Failed to create A2A server: %v", err)
	}

	log.Printf("K8s Health A2A Agent started on %s", port)
	log.Printf("Agent Card: http://localhost%s/.well-known/agent-card.json", port)
	if err := srv.Start(port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
