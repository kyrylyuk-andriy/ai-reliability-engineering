package main

import (
	"context"
	"fmt"
	"iter"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type k8sExecutor struct {
	client kubernetes.Interface
}

func newK8sExecutor() (*k8sExecutor, error) {
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
	return &k8sExecutor{client: client}, nil
}

func (e *k8sExecutor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		if execCtx.StoredTask == nil {
			if !yield(a2a.NewSubmittedTask(execCtx, execCtx.Message), nil) {
				return
			}
		}
		if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, nil), nil) {
			return
		}

		text := extractText(execCtx.Message)
		result := e.processRequest(ctx, text)

		event := a2a.NewArtifactEvent(execCtx, a2a.NewTextPart(result))
		if !yield(event, nil) {
			return
		}
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
	}
}

func (e *k8sExecutor) Cancel(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil), nil)
	}
}

func (e *k8sExecutor) processRequest(ctx context.Context, text string) string {
	switch {
	case containsAny(text, "pod", "pods"):
		r, _ := e.getPodStatus(ctx, extractNamespace(text))
		return r
	case containsAny(text, "node", "nodes"):
		r, _ := e.getNodeStatus(ctx)
		return r
	case containsAny(text, "deploy", "deployment"):
		r, _ := e.getDeploymentStatus(ctx, extractNamespace(text))
		return r
	case containsAny(text, "event", "warning"):
		r, _ := e.getEvents(ctx)
		return r
	default:
		r, _ := e.getClusterSummary(ctx)
		return r
	}
}

func (e *k8sExecutor) getPodStatus(ctx context.Context, ns string) (string, error) {
	pods, err := e.client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
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

func (e *k8sExecutor) getNodeStatus(ctx context.Context) (string, error) {
	nodes, err := e.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
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

func (e *k8sExecutor) getDeploymentStatus(ctx context.Context, ns string) (string, error) {
	deployments, err := e.client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
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

func (e *k8sExecutor) getEvents(ctx context.Context) (string, error) {
	events, err := e.client.CoreV1().Events("").List(ctx, metav1.ListOptions{
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
	for _, ev := range events.Items {
		sb.WriteString(fmt.Sprintf("  [%s] %s/%s: %s (x%d)\n",
			ev.LastTimestamp.Format("15:04:05"),
			ev.InvolvedObject.Kind, ev.InvolvedObject.Name,
			ev.Message, ev.Count))
	}
	return sb.String(), nil
}

func (e *k8sExecutor) getClusterSummary(ctx context.Context) (string, error) {
	nodes, _ := e.getNodeStatus(ctx)
	pods, _ := e.getPodStatus(ctx, "kagent")
	events, _ := e.getEvents(ctx)
	return fmt.Sprintf("=== Cluster Health Summary ===\n\n%s\n%s\n%s", nodes, pods, events), nil
}

func extractText(msg *a2a.Message) string {
	if msg == nil {
		return ""
	}
	for _, part := range msg.Parts {
		if t := part.Text(); t != "" {
			return strings.ToLower(t)
		}
	}
	return ""
}

func extractNamespace(text string) string {
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

func main() {
	port := ":9090"
	if p := os.Getenv("PORT"); p != "" {
		port = ":" + p
	}

	executor, err := newK8sExecutor()
	if err != nil {
		log.Fatalf("Failed to create K8s client: %v", err)
	}

	handler := a2asrv.NewHandler(executor)
	jsonrpcHandler := a2asrv.NewJSONRPCHandler(handler)

	card := &a2a.AgentCard{
		Name:        "K8s Health Agent",
		Description: "A2A-compliant Kubernetes cluster health checker. Inspects pods, nodes, deployments, and warning events.",
		Version:     "0.1.0",
		Provider: &a2a.AgentProvider{
			Org: "AI Reliability Engineering",
			URL: "https://github.com/kyrylyuk-andriy/ai-reliability-engineering",
		},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Skills: []a2a.AgentSkill{
			{ID: "pod-status", Name: "Pod Status", Description: "List pods with status in a namespace"},
			{ID: "node-status", Name: "Node Status", Description: "List cluster nodes with conditions"},
			{ID: "deployment-status", Name: "Deployment Status", Description: "List deployments with replica counts"},
			{ID: "warning-events", Name: "Warning Events", Description: "Get recent warning events"},
			{ID: "cluster-summary", Name: "Cluster Summary", Description: "Full cluster health summary"},
		},
		SupportedInterfaces: []*a2a.AgentInterface{
			a2a.NewAgentInterface(fmt.Sprintf("http://localhost%s", port), a2a.TransportProtocolJSONRPC),
		},
	}

	mux := http.NewServeMux()
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(card))
	mux.Handle("/", jsonrpcHandler)

	log.Printf("K8s Health A2A Agent started on %s (official a2a-go SDK)", port)
	log.Printf("Agent Card: http://localhost%s%s", port, a2asrv.WellKnownAgentCardPath)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
