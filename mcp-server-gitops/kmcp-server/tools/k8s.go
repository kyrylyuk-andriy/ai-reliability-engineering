package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sTools struct {
	client kubernetes.Interface
}

func NewK8sTools() (*K8sTools, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig for local development
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, fmt.Errorf("cannot create k8s config: %w", err)
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create k8s client: %w", err)
	}

	return &K8sTools{client: client}, nil
}

func (k *K8sTools) GetPodStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ns := req.GetArguments()["namespace"].(string)

	pods, err := k.client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list pods: %v", err)), nil
	}

	if len(pods.Items) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No pods found in namespace %q", ns)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Pods in namespace %q:\n\n", ns))
	sb.WriteString(fmt.Sprintf("%-50s %-12s %-10s %-8s\n", "NAME", "STATUS", "READY", "RESTARTS"))
	sb.WriteString(strings.Repeat("-", 82) + "\n")

	for _, pod := range pods.Items {
		ready := 0
		total := len(pod.Spec.Containers)
		restarts := int32(0)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
			restarts += cs.RestartCount
		}
		sb.WriteString(fmt.Sprintf("%-50s %-12s %d/%-8d %-8d\n",
			pod.Name, string(pod.Status.Phase), ready, total, restarts))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (k *K8sTools) GetNodeStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	nodes, err := k.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list nodes: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("Cluster Nodes:\n\n")
	sb.WriteString(fmt.Sprintf("%-40s %-10s %-20s\n", "NAME", "STATUS", "VERSION"))
	sb.WriteString(strings.Repeat("-", 72) + "\n")

	for _, node := range nodes.Items {
		status := "NotReady"
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				status = "Ready"
			}
		}
		sb.WriteString(fmt.Sprintf("%-40s %-10s %-20s\n",
			node.Name, status, node.Status.NodeInfo.KubeletVersion))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (k *K8sTools) GetDeploymentStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ns := req.GetArguments()["namespace"].(string)

	deployments, err := k.client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list deployments: %v", err)), nil
	}

	if len(deployments.Items) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No deployments found in namespace %q", ns)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Deployments in namespace %q:\n\n", ns))
	sb.WriteString(fmt.Sprintf("%-45s %-12s %-12s %-10s\n", "NAME", "READY", "UP-TO-DATE", "AVAILABLE"))
	sb.WriteString(strings.Repeat("-", 81) + "\n")

	for _, d := range deployments.Items {
		sb.WriteString(fmt.Sprintf("%-45s %d/%-10d %-12d %-10d\n",
			d.Name, d.Status.ReadyReplicas, *d.Spec.Replicas,
			d.Status.UpdatedReplicas, d.Status.AvailableReplicas))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (k *K8sTools) GetEvents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ns := ""
	if v, ok := req.GetArguments()["namespace"]; ok && v != nil {
		ns = v.(string)
	}

	limit := int64(20)
	if v, ok := req.GetArguments()["limit"]; ok && v != nil {
		limit = int64(v.(float64))
	}

	events, err := k.client.CoreV1().Events(ns).List(ctx, metav1.ListOptions{
		FieldSelector: "type=Warning",
		Limit:         limit,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list events: %v", err)), nil
	}

	if len(events.Items) == 0 {
		return mcp.NewToolResultText("No warning events found"), nil
	}

	var sb strings.Builder
	sb.WriteString("Warning Events:\n\n")

	for _, e := range events.Items {
		sb.WriteString(fmt.Sprintf("[%s] %s/%s: %s (x%d)\n",
			e.LastTimestamp.Format("15:04:05"),
			e.InvolvedObject.Kind, e.InvolvedObject.Name,
			e.Message, e.Count))
	}

	return mcp.NewToolResultText(sb.String()), nil
}
