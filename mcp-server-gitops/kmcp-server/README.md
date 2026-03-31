# K8s Health Checker (KMCP Server)

Custom MCP tool server written in Go that provides Kubernetes health check tools to [kagent](https://kagent.dev) agents. Deployed via Flux as a `MCPServer` CRD.

## Tools

| Tool | Description |
|------|-------------|
| `get_pod_status` | List pods with status, readiness, and restart counts |
| `get_node_status` | List cluster nodes with conditions and versions |
| `get_deployment_status` | List deployments with ready/desired replica counts |
| `get_events` | Get recent warning events from the cluster |
| `cluster_dashboard` | Interactive HTML dashboard ([MCP App](https://modelcontextprotocol.io/extensions/apps/overview)) |

## MCP App: Cluster Dashboard

The `cluster_dashboard` tool returns an interactive HTML UI rendered inside MCP Apps-compatible clients (Claude Desktop, VS Code Copilot) as a sandboxed iframe.

Features: node overview, pods table with namespace selector, deployments, warning events, health percentages.

## Build & Test

```bash
# Build
go build -o bin/k8s-health-checker .

# Run tests
go test -v -race ./...

# Docker build & push
make docker-build docker-push
```

## Test with MCP Inspector

```bash
kubectl port-forward -n kagent pod/<k8s-health-checker-pod> 3000:3000
npx @modelcontextprotocol/inspector@0.21.1
# Connect via Streamable HTTP to http://localhost:3000/mcp
```

## Deployment

Deployed via Flux from `releases/kmcp-server.yaml`:
- ServiceAccount + RBAC (read-only access to pods, nodes, deployments, events)
- `MCPServer` CR with stdio transport
- `Agent` CR (`k8s-health-agent`) referencing the tools

Image: `ghcr.io/kyrylyuk-andriy/k8s-health-checker:latest`

## Tech

- **Language:** Go
- **MCP SDK:** [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- **K8s client:** client-go
- **Container:** Multi-stage Dockerfile with distroless base
- **Transport:** stdio (spawned by kagent sidecar)

---
*Back to [mcp-server-gitops](../)*
