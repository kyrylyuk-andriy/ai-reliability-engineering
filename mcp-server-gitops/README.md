# MCP Server GitOps

AI-native infrastructure platform deployed via GitOps with Flux CD — includes custom MCP/A2A agents, security governance, observability, and prompt enrichment.

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
| [k8s-health-checker](kmcp-server/) | 0.1.0 | Custom KMCP server — K8s health check tools |
| [MCPG](https://github.com/techwithhuz/mcp-security-governance) | latest | MCP Security Governance — scores MCP infrastructure |
| [Agentregistry](https://github.com/agentregistry-dev/agentregistry) | 0.3.2 | AI resource inventory and registry |
| [Phoenix](https://arize.com/docs/phoenix) | 5.0.20 | AI Observability & Evaluation platform |
| [Qdrant](https://qdrant.tech) | 1.17.1 | Vector database for AI/ML workloads |

**Two-phase deployment:** CRDs install first (`releases-crds`, `wait: true`), then apps (`releases`, `dependsOn: releases-crds`).

---

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.9.0
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Helm](https://helm.sh/docs/intro/install/)
- [KinD](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [k9s](https://k9scli.io/topics/install/) (optional)

## Quick Start

```bash
git clone https://github.com/kyrylyuk-andriy/ai-reliability-engineering.git
cd ai-reliability-engineering/mcp-server-gitops

export ANTHROPIC_API_KEY="your-api-key-here"
make run
```

This creates a Kind cluster, installs Flux, and reconciles all components. See [bootstrap/](bootstrap/) for details.

### Check status

```bash
kubectl get kustomizations -n flux-system
# Both releases-crds and releases should be Ready: True
```

### Access the UIs

```bash
kubectl port-forward -n agentgateway-system deployment/agentgateway-external 8080:80 15000:15000 &
kubectl port-forward svc/phoenix-svc -n phoenix 6006:6006 &
kubectl port-forward svc/qdrant -n qdrant 6333:6333 &
kubectl port-forward svc/mcp-governance-dashboard -n mcp-governance 3000:3000 &
kubectl port-forward svc/agentregistry-server -n agentregistry 12121:8080 &
```

| UI | URL | Description |
|----|-----|-------------|
| kagent | http://localhost:8080 | AI agent dashboard |
| AgentGateway Admin | http://localhost:15000/ui | Gateway backends & routes |
| Phoenix | http://localhost:6006 | AI observability & traces |
| Qdrant | http://localhost:6333/dashboard | Vector database |
| MCPG | http://localhost:3000 | MCP security governance |
| Agentregistry | http://localhost:12121 | AI resource inventory |

---

## GitOps Workflow

Edit files in [`releases/`](releases/) and push — Flux reconciles within ~1 minute.

```bash
vim releases/kagent.yaml
git add . && git commit -m "update kagent config" && git push
```

---

## Custom Components

### [KMCP Server (k8s-health-checker)](kmcp-server/)

Custom MCP tool server in Go with 5 tools (pods, nodes, deployments, events) + an interactive [MCP Apps](https://modelcontextprotocol.io/extensions/apps/overview) HTML dashboard. Deployed via Flux as a `MCPServer` CRD.

### [A2A Health Agent](a2a-agent/)

Standalone A2A-compliant K8s health checker using the official [a2a-go SDK](https://github.com/a2aproject/a2a-go). Serves Agent Card at `/.well-known/agent-card.json`.

### [A2A SRE Coordinator](a2a-sre-coordinator/)

SRE orchestrator that delegates health checks to the Health Agent via A2A and compiles reports.

### [A2A Team Coordinator](a2a-team/)

Orchestrates a team of agents — custom + [kagent](https://kagent.dev) built-in agents (Helm Agent, K8s Agent) — via A2A protocol.

```
User → A2A Team Coordinator (port 9092)
         ├── K8s Health Agent (custom, port 9090)
         ├── Helm Agent (kagent, port 8083)
         └── K8s Agent (kagent, port 8083)
```

### [Tracing & Evaluation](tracing/)

Python scripts that instrument A2A agent communication with OpenTelemetry and send traces to Phoenix. Includes evaluation test suite (6 tests, 0.94 avg score).

---

## Platform Components

### MCP Security Governance (MCPG)

Discovers all MCP resources in the cluster, scores them across 9 security categories. Deployed via [`releases/mcpg.yaml`](releases/) with images from GHCR.

```bash
kubectl port-forward svc/mcp-governance-dashboard -n mcp-governance 3000:3000
# Open http://localhost:3000
```

### Agentregistry (AI Resource Inventory)

Centralized registry for AI resources. Deployed via [`releases/agentregistry.yaml`](releases/).

```bash
kubectl port-forward svc/agentregistry-server -n agentregistry 12121:8080
# Open http://localhost:12121
```

List AI resources via kubectl:

```bash
kubectl get mcpservers,remotemcpservers,agents,modelconfigs -A
```

### Phoenix (AI Observability)

Tracing, evaluation, and debugging for LLM applications. Deployed via [`releases/phoenix.yaml`](releases/).

- [Phoenix Docs](https://arize.com/docs/phoenix/self-hosting)

### Qdrant (Vector Database)

Vector database for embeddings and similarity search. Deployed via [`releases/qdrant.yaml`](releases/).

- [Qdrant Docs](https://qdrant.tech/documentation/)

### Prompt Enrichment

[AgentgatewayPolicy](https://agentgateway.dev/docs/kubernetes/latest/tutorials/prompt-enrichment/) that injects an SRE system prompt at the gateway layer. Deployed via [`releases/prompt-enrichment.yaml`](releases/).

```bash
kubectl port-forward deployment/agentgateway-external -n agentgateway-system 8080:80 &
curl -s http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"How do I debug a CrashLoopBackOff?"}]}' | jq -r '.choices[].message.content'
```

---

## Model Configuration

Configured automatically during bootstrap:
- **API key secret** — created by Terraform (from `$ANTHROPIC_API_KEY`)
- **ModelConfig** — declared in `releases/kagent.yaml` (claude-haiku-4-5)

---

## Research Documents

| Document | Topic |
|----------|-------|
| [MCP Research](docs/mcp-research.md) | MCP Sampling, Elicitation, Apps |
| [A2A Research](docs/a2a-research.md) | A2A Protocol, Agent Cards, Tasks, Teams |

---

## Project Structure

```
mcp-server-gitops/
├── bootstrap/               # Terraform — cluster + Flux
├── docs/                    # Research documents & screenshots
├── kmcp-server/             # Custom KMCP server (Go, MCP protocol)
├── a2a-agent/               # Standalone A2A health agent (Go)
├── a2a-sre-coordinator/     # A2A SRE coordinator (Go)
├── a2a-team/                # A2A Team coordinator (Go)
├── tracing/                 # Phoenix tracing & evaluation (Python)
├── releases/                # Flux GitOps manifests
├── scripts/                 # Bootstrap helper scripts
└── Makefile
```

Each subdirectory has its own README with detailed instructions.

---

## Cleanup

```bash
make down
```
