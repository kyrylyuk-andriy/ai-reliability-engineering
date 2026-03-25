# Lab-2: MCP Server GitOps

Deploy an MCP tool server and agent using GitOps with Flux CD syncing from a Git repository.

Based on [abox](https://github.com/den-vasyliev/abox), adapted to use **Terraform** and **GitRepository** (Git sync) instead of OpenTofu and OCI artifacts.

## Architecture

```
git push → GitHub repo → Flux GitRepository → Kustomization → Helm Releases
```

**Components:**

| Component | Version | Role |
|-----------|---------|------|
| KinD | latest | Local K8s cluster (1 control-plane + 2 workers) |
| Flux CD | 2.x | GitOps operator (syncs from Git) |
| agentgateway | v2.2.1 | AI-aware API gateway (Gateway API native) |
| kagent | 0.7.23 | K8s-native AI agent framework with MCP server |
| Gateway API CRDs | 1.5.0 | Standard K8s Gateway API (experimental channel) |
| k8s-health-checker | 0.1.0 | Custom KMCP server — K8s health check tools |

**Two-phase deployment:** CRDs install first (`releases-crds`, `wait: true`), then apps (`releases`, `dependsOn: releases-crds`).

---

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.9.0
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Helm](https://helm.sh/docs/intro/install/)
- [KinD](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [k9s](https://k9scli.io/topics/install/) (optional)

## Quick Start

### 1. Clone and navigate

```bash
git clone https://github.com/andriy-kyrylyuk/ai-reliability-engineering.git
cd ai-reliability-engineering/mcp-server-gitops
```

### 2. Set your Anthropic API key

```bash
export ANTHROPIC_API_KEY="your-api-key-here"
```

### 3. Bootstrap the cluster

```bash
make run
```

This will:
- Check prerequisites
- Create a Kind cluster with 1 control-plane + 2 workers
- Install Flux Operator and Flux Instance
- Create a GitRepository source pointing at this repo
- Create Kustomizations for CRDs and releases
- Flux reconciles: installs Gateway API CRDs, agentgateway, kagent

### 4. Check Flux reconciliation status

```bash
kubectl get kustomizations -n flux-system
```

**Expected output:**

```
NAME             AGE   READY   STATUS
releases-crds    1m    True    Applied revision: main@sha1:...
releases         1m    True    Applied revision: main@sha1:...
```

### 5. Verify all pods are running

```bash
kubectl get pods -n agentgateway-system
kubectl get pods -n kagent
```

### 6. Access the UIs

```bash
kubectl port-forward -n agentgateway-system deployment/agentgateway-external 8080:80 15000:15000
```

- **AgentGateway Admin UI:** http://localhost:15000/ui
- **kagent UI:** http://localhost:8080
- **kagent MCP API:** http://localhost:8080/api

---

## GitOps Workflow

To make changes, edit files in `releases/` and push:

```bash
# Edit a release manifest
vim releases/kagent.yaml

# Push to trigger reconciliation
git add . && git commit -m "update kagent config" && git push
```

Flux detects the change within ~1 minute and reconciles automatically.

### Key difference from abox

| | abox | mcp-server-gitops |
|---|---|---|
| IaC tool | OpenTofu | Terraform |
| Flux source | OCI artifacts (GitlessOps) | GitRepository (GitOps) |
| Trigger | `make push` → OCI tag → Flux | `git push` → Flux polls Git |

---

## Model Configuration

The Anthropic model is configured automatically during setup:

- **API key secret** — created by Terraform in the bootstrap phase (from `$ANTHROPIC_API_KEY` env var)
- **ModelConfig** — declared in `releases/kagent.yaml`, deployed by Flux

No manual steps needed. After `make run`, kagent is ready to use Anthropic (claude-haiku-4-5).

To verify, open the kagent dashboard at http://localhost:8080, select an agent, and send a test prompt.

---

## Custom KMCP Server (k8s-health-checker)

A custom MCP tool server written in Go that provides Kubernetes health check tools to kagent agents.

### Tools

| Tool | Description |
|------|-------------|
| `get_pod_status` | List pods with status, readiness, and restart counts |
| `get_node_status` | List cluster nodes with conditions and versions |
| `get_deployment_status` | List deployments with ready/desired replica counts |
| `get_events` | Get recent warning events from the cluster |
| `cluster_dashboard` | Interactive HTML dashboard (MCP App) showing cluster health |

### SDLC

The server follows a full software development lifecycle:

- **Source code** — Go with [mcp-go](https://github.com/mark3labs/mcp-go) SDK and [client-go](https://github.com/kubernetes/client-go)
- **Unit tests** — Using fake k8s clientset (`go test ./...`)
- **Container image** — Multi-stage Dockerfile with distroless base, pushed to GHCR
- **GitOps deployment** — `releases/kmcp-server.yaml` deploys MCPServer CR, Agent CR, RBAC via Flux
- **CI/CD** — GitHub Actions workflow: lint, test, build & push to `ghcr.io`

### MCP App: Cluster Dashboard

The `cluster_dashboard` tool implements [MCP Apps](https://modelcontextprotocol.io/extensions/apps/overview) — an MCP extension that returns interactive HTML UIs rendered inside the conversation as sandboxed iframes.

**How it works:**
1. Tool declares `ui://k8s-dashboard` resource containing a self-contained HTML dashboard
2. MCP Apps-compatible clients (Claude Desktop, VS Code Copilot) render the HTML in a sandboxed iframe
3. The dashboard calls existing tools (`get_pod_status`, `get_node_status`, etc.) via postMessage
4. Results display as styled tables with status badges, namespace selector, and refresh button

**Dashboard features:**
- Nodes overview with Ready/NotReady status
- Pods table with namespace selector (kagent, flux-system, kube-system, etc.)
- Deployments with ready/desired replica counts
- Warning events panel
- Summary with health percentages

**Testing with MCP Inspector:**

```bash
# Port-forward the KMCP server
kubectl port-forward -n kagent pod/<k8s-health-checker-pod> 3000:3000

# Launch Inspector, connect via Streamable HTTP to http://localhost:3000/mcp
npx @modelcontextprotocol/inspector@0.21.1
```

**Testing with Claude Desktop:**

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "k8s-health-checker": {
      "command": "<path-to-repo>/mcp-server-gitops/kmcp-server/bin/k8s-health-checker"
    }
  }
}
```

Build locally with `cd kmcp-server && go build -o bin/k8s-health-checker .`, restart Claude Desktop, and ask "Show me the cluster dashboard".

![MCP Apps Dashboard](docs/images/mcp-dashboard.png)

### How it works

1. kagent deploys the KMCP server pod with an agentgateway sidecar
2. The sidecar spawns `/k8s-health-checker` via stdio transport
3. kagent controller discovers available tools from the MCPServer CR
4. The `k8s-health-agent` Agent CR references these tools
5. Users interact with the agent through the kagent UI

### Build & push manually

```bash
cd kmcp-server
make docker-build docker-push
```

### Verify

```bash
kubectl get mcpserver -n kagent
kubectl get agent k8s-health-agent -n kagent
```

Open kagent UI → select **k8s-health-agent** → ask "What pods are running in the kagent namespace?"

---

## Project Structure

```
mcp-server-gitops/
├── README.md
├── Makefile
├── scripts/
│   └── setup.sh
├── bootstrap/               # Terraform — cluster + Flux
│   ├── providers.tf
│   ├── variables.tf
│   ├── cluster.tf
│   └── flux.tf
├── docs/                    # Research & screenshots
│   ├── mcp-research.md      # MCP Sampling/Elicitation/Apps research
│   └── images/
├── kmcp-server/             # Custom KMCP server (Go)
│   ├── main.go
│   ├── tools/
│   │   ├── k8s.go           # K8s health check tools
│   │   ├── k8s_test.go
│   │   └── dashboard.go     # MCP App: HTML dashboard
│   ├── go.mod / go.sum
│   ├── Dockerfile
│   └── Makefile
└── releases/                # Flux syncs this directory
    ├── kustomization.yaml
    ├── agentgateway.yaml    # Namespace + HelmRelease + Gateway
    ├── kagent.yaml          # Namespace + HelmRelease + HTTPRoute + ModelConfig
    ├── kmcp-server.yaml     # MCPServer + Agent + RBAC
    └── crds/
        ├── kustomization.yaml
        ├── agentgateway-crds.yaml
        └── kagent-crds.yaml
```

---

## Cleanup

```bash
make down
```

This destroys the Kind cluster and all resources.
