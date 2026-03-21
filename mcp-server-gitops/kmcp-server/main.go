package main

import (
	"fmt"
	"os"

	"github.com/kyrylyuk-andriy/ai-reliability-engineering/mcp-server-gitops/kmcp-server/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer(
		"k8s-health-checker",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	k8sTools, err := tools.NewK8sTools()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create k8s client: %v\n", err)
		os.Exit(1)
	}

	s.AddTool(mcp.NewTool("get_pod_status",
		mcp.WithDescription("List pods with their status in a namespace"),
		mcp.WithString("namespace", mcp.DefaultString("default"), mcp.Description("Kubernetes namespace")),
	), k8sTools.GetPodStatus)

	s.AddTool(mcp.NewTool("get_node_status",
		mcp.WithDescription("List cluster nodes with their conditions and resource usage"),
	), k8sTools.GetNodeStatus)

	s.AddTool(mcp.NewTool("get_deployment_status",
		mcp.WithDescription("List deployments with ready/desired replica counts"),
		mcp.WithString("namespace", mcp.DefaultString("default"), mcp.Description("Kubernetes namespace")),
	), k8sTools.GetDeploymentStatus)

	s.AddTool(mcp.NewTool("get_events",
		mcp.WithDescription("Get recent warning events from the cluster"),
		mcp.WithString("namespace", mcp.DefaultString(""), mcp.Description("Kubernetes namespace (empty for all namespaces)")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of events to return")),
	), k8sTools.GetEvents)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
