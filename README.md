# AI Reliability Engineering

Course labs for building AI-native infrastructure on Kubernetes — from local LLM proxies to full GitOps-managed agent platforms with observability.

## Labs

### [Basic Agentic Infrastructure](basic-agentic-infrastructure/)

Local and Kubernetes-based AI gateway and agent setup.

| Tier | What |
|------|------|
| [Basic](basic-agentic-infrastructure/basic/) | AgentGateway locally with Anthropic + Gemini backends |
| [Experienced](basic-agentic-infrastructure/experienced/) | AgentGateway + kagent in Kubernetes via Helm |
| [Max](basic-agentic-infrastructure/max/) | kagent with Gateway API integration |

### [MCP Server GitOps](mcp-server-gitops/)

Full GitOps AI platform — custom MCP/A2A agents, security governance, observability, prompt enrichment. All deployed via Flux CD on a Kind cluster.

| Tier | What |
|------|------|
| Basic | Deploy MCP server + agent via Flux CD |
| Experienced | Custom KMCP server (Go) with full SDLC |
| Max | MCP Apps (HTML dashboard) |

### [A2A Protocol & Observability](mcp-server-gitops/docs/)

A2A agent communication, security governance, and AI observability.

| Tier | What |
|------|------|
| Basic | A2A research, MCP Inspector, agent inventory |
| Experienced | A2A task communication, MCPG deployment |
| Max | A2A team (custom + kagent agents), tracing & evaluation |

### [Prompt Enrichment & Tracing](mcp-server-gitops/tracing/)

| Tier | What |
|------|------|
| Basic | Prompt enrichment via AgentgatewayPolicy |
| Experienced | MCP server tracing, Phoenix evaluation |
| Max | Full tracing & evaluation for A2A agent team |

---

## Quick Start

```bash
cd mcp-server-gitops
export ANTHROPIC_API_KEY="your-key"
make run
```

See [mcp-server-gitops/](mcp-server-gitops/) for full setup guide, UI access, and component details.

## References

- [abox](https://github.com/den-vasyliev/abox) — reference GitOps AI platform
- [AgentGateway](https://agentgateway.dev) — AI-aware gateway
- [kagent](https://kagent.dev) — K8s AI agent framework
- [A2A Protocol](https://a2a-protocol.org) — Agent-to-Agent communication
- [MCP](https://modelcontextprotocol.io) — Model Context Protocol
- [Phoenix](https://arize.com/docs/phoenix) — AI observability
- [Qdrant](https://qdrant.tech) — Vector database
