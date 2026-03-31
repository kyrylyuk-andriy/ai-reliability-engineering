# Basic Agentic Infrastructure

Local and Kubernetes-based AI gateway and agent setup using [AgentGateway](https://agentgateway.dev) and [kagent](https://kagent.dev).

## Tiers

| Tier | What | Folder |
|------|------|--------|
| [Basic](basic/) | AgentGateway locally with Anthropic + Gemini backends | `basic/` |
| [Experienced](experienced/) | AgentGateway + kagent in Kubernetes via Helm | `experienced/` |
| [Max](max/) | kagent with Gateway API integration | `max/` |

> **Note:** The basic and experienced tiers were built with AgentGateway v1.0.0-rc.1. The [GitOps platform](../mcp-server-gitops/) uses v2.2.1 with additional components.
