# AI Reliability Engineering

Course labs for building AI-native infrastructure on Kubernetes — from local LLM proxies to full GitOps-managed agent platforms with observability.

## Labs

### [Basic Agentic Infrastructure](basic-agentic-infrastructure/)

Local and Kubernetes-based AI gateway and agent setup.

| Tier | What | Key Tech |
|------|------|----------|
| [Basic](basic-agentic-infrastructure/basic/) | AgentGateway locally with Anthropic + Gemini backends | AgentGateway config, curl |
| [Experienced](basic-agentic-infrastructure/experienced/) | AgentGateway + kagent in Kubernetes (Helm) | Kind, Helm, Gateway API, kagent |
| [Max](basic-agentic-infrastructure/max/) | kagent with Gateway API integration | Gateway API, AgentGateway routing |

### [MCP Server GitOps](mcp-server-gitops/)

Full GitOps AI platform with custom MCP server, A2A agents, governance, and observability.

| Tier | What | Key Tech |
|------|------|----------|
| Basic | Deploy MCP server + agent via Flux CD | Terraform, Kind, Flux, kagent |
| Experienced | Custom KMCP server (Go) with SDLC | Go, mcp-go, Dockerfile, GHCR, MCPServer CRD |
| Max | MCP Apps (HTML dashboard), MCP Sampling | MCP Apps extension, interactive UI |

### [A2A Protocol & Observability](mcp-server-gitops/docs/)

A2A agent communication, security governance, and AI observability.

| Tier | What | Key Tech |
|------|------|----------|
| Basic | A2A research, MCP Inspector, agent inventory | A2A spec, Inspector, kubectl |
| Experienced | A2A task communication, MCPG deployment | a2a-go SDK, MCP Security Governance |
| Max | A2A team (custom + kagent), tracing & evaluation | A2A team coordinator, Phoenix, OpenTelemetry |

### Prompt Enrichment & Tracing

| Tier | What | Key Tech |
|------|------|----------|
| Basic | Prompt enrichment, LangChain tracing tutorial | AgentgatewayPolicy, Phoenix |
| Experienced | MCP server tracing, Phoenix evaluation | OpenTelemetry, Pydantic Evals |
| Max | Full tracing & evaluation for A2A agent team | trace_team.py, evaluate_team.py, Phoenix |

---

## Infrastructure Stack

All components deployed via GitOps (Flux CD) on a Kind cluster:

| Component | Version | Purpose |
|-----------|---------|---------|
| [AgentGateway](https://agentgateway.dev) | v2.2.1 | AI-aware API gateway with prompt enrichment |
| [kagent](https://kagent.dev) | 0.7.23 | K8s-native AI agent framework (MCP + A2A) |
| [Phoenix](https://arize.com/docs/phoenix) | 5.0.20 | AI observability, tracing & evaluation |
| [Qdrant](https://qdrant.tech) | 1.17.1 | Vector database |
| [MCPG](https://github.com/techwithhuz/mcp-security-governance) | latest | MCP security governance & scoring |
| [Agentregistry](https://github.com/agentregistry-dev/agentregistry) | 0.3.2 | AI resource inventory & registry |
| Flux CD | 2.x | GitOps operator |
| Gateway API | 1.5.0 | K8s standard traffic routing (experimental) |

## Custom Components

| Component | Language | Protocol | Description |
|-----------|----------|----------|-------------|
| [k8s-health-checker](mcp-server-gitops/kmcp-server/) | Go | MCP | KMCP server with 5 tools + HTML dashboard |
| [a2a-agent](mcp-server-gitops/a2a-agent/) | Go | A2A | Standalone K8s health agent |
| [a2a-sre-coordinator](mcp-server-gitops/a2a-sre-coordinator/) | Go | A2A | SRE agent delegating to health agent |
| [a2a-team](mcp-server-gitops/a2a-team/) | Go | A2A | Team coordinator (custom + kagent agents) |
| [tracing](mcp-server-gitops/tracing/) | Python | OTLP | Phoenix tracing & evaluation scripts |

## Quick Start

```bash
# 1. Bootstrap cluster with all components
cd mcp-server-gitops
export ANTHROPIC_API_KEY="your-key"
make run

# 2. Access all UIs
kubectl port-forward -n agentgateway-system deployment/agentgateway-external 8080:80 15000:15000 &
kubectl port-forward svc/phoenix-svc -n phoenix 6006:6006 &
kubectl port-forward svc/qdrant -n qdrant 6333:6333 &
kubectl port-forward svc/mcp-governance-dashboard -n mcp-governance 3000:3000 &
kubectl port-forward svc/agentregistry-server -n agentregistry 12121:8080 &
```

| UI | URL |
|----|-----|
| kagent | http://localhost:8080 |
| AgentGateway Admin | http://localhost:15000/ui |
| Phoenix | http://localhost:6006 |
| Qdrant | http://localhost:6333/dashboard |
| MCPG | http://localhost:3000 |
| Agentregistry | http://localhost:12121 |

## Research Documents

| Document | Topic |
|----------|-------|
| [MCP Research](mcp-server-gitops/docs/mcp-research.md) | MCP Sampling, Elicitation, Apps |
| [A2A Research](mcp-server-gitops/docs/a2a-research.md) | A2A Protocol, Agent Cards, Tasks, Teams |

## References

- [abox](https://github.com/den-vasyliev/abox) — reference GitOps AI platform
- [AgentGateway](https://agentgateway.dev) — AI-aware gateway
- [kagent](https://kagent.dev) — K8s AI agent framework
- [A2A Protocol](https://a2a-protocol.org) — Agent-to-Agent communication
- [MCP](https://modelcontextprotocol.io) — Model Context Protocol
- [Phoenix](https://arize.com/docs/phoenix) — AI observability
- [Qdrant](https://qdrant.tech) — Vector database
