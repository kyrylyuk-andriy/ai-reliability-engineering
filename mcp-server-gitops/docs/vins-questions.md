# Vin's Questions — AI Infrastructure Assessment

Answers based on our deployed infrastructure: AgentGateway v2.2.1, kagent 0.7.23, Phoenix, custom A2A agents, and Flux CD GitOps on Kubernetes.

---

## 1. How could we handle 'agent got stuck' scenarios?

Multiple layers of protection:

**Kubernetes level:**
- **Liveness/readiness probes** — K8s automatically restarts stuck agent pods. All our deployments have health checks configured.
- **Resource limits** — CPU/memory limits prevent runaway agents from consuming cluster resources. If an agent OOMs, K8s kills and restarts it.
- **Pod Disruption Budgets** — ensure availability during restarts.

**Agent framework level (kagent):**
- **Task timeouts** — kagent controller enforces execution timeouts on agent tasks. The MCPServer CRD supports a `timeout` field (default 30s) for tool server connections.
- **A2A protocol** — supports `CANCELED` task state. Clients can send `CancelTask` requests to stop stuck agents.
- **Watchdog reconciliation** — kagent controller reconciles agent state every 30s, detecting and recovering from stale states.

**Gateway level (AgentGateway):**
- **Request timeouts** — AgentGateway enforces request-level timeouts on LLM calls, preventing indefinite waits.
- **Rate limiting** — prevents agents from making infinite loops of LLM calls (see Q12).

**Observability (Phoenix):**
- Our [tracing setup](../tracing/) sends spans to Phoenix for every A2A call. Stuck agents show up as long-running or incomplete traces, enabling rapid detection.

---

## 2. Any automatic timeout/circuit breaker patterns from this framework?

**AgentGateway provides:**
- **Token bucket rate limiting** — limits requests or tokens per time window. Prevents runaway loops.
- **Health-based routing** — the P2C (Power of Two Choices) algorithm routes away from unhealthy backends based on latency and error rates.
- **429-triggered failover** — automatically switches to backup models when rate limits are hit (see Q3).

**kagent provides:**
- **MCPServer timeout** — configurable per-server connection timeout (`spec.timeout: 30s`).
- **Streaming timeouts** — configurable via `Streaming.Timeout` in the controller config (default 600s).

**What's missing:**
- No built-in circuit breaker pattern (open/half-open/closed) in kagent or AgentGateway currently. For circuit breaking, you'd layer [Istio](https://istio.io) or [Cilium](https://cilium.io) service mesh on top, or use the upcoming [AgentGateway resilience policies](https://agentgateway.dev/docs/kubernetes/latest/configuration/resiliency/rate-limits/).

---

## 3. How does kgateway handle model failover?

AgentGateway uses **priority groups** with automatic failover:

```yaml
apiVersion: agentgateway.dev/v1alpha1
kind: AgentgatewayBackend
metadata:
  name: model-failover
spec:
  ai:
    groups:
      # Tier 1: Budget models (P2C load balanced)
      - providers:
          - name: claude-haiku
            anthropic:
              model: claude-haiku-4-5-20251001
          - name: openai-mini
            openai:
              model: gpt-4.1-mini
      # Tier 2: Premium fallback
      - providers:
          - name: claude-sonnet
            anthropic:
              model: claude-sonnet-4-5-20250514
```

**How it works:**
1. Requests go to Tier 1 first, load balanced across models using P2C algorithm
2. If a model returns **429 (rate limited)**, failover triggers to the next tier
3. Within each tier, routing considers health scores, latency, and current load
4. Failed providers get lower health scores but remain in their tier for other error types

Reference: [AgentGateway Model Failover](https://agentgateway.dev/docs/kubernetes/latest/llm/failover/)

---

## 4. Can we automatically switch from OpenAI to Claude to local model?

**Yes.** Configure a three-tier failover:

```yaml
groups:
  # Tier 1: Cheapest (local vLLM)
  - providers:
      - name: local-llm
        openai:
          model: llama-3.1-8b
          baseUrl: http://vllm-service:8000/v1
  # Tier 2: Cloud budget models
  - providers:
      - name: openai-mini
        openai:
          model: gpt-4.1-mini
      - name: claude-haiku
        anthropic:
          model: claude-haiku-4-5-20251001
  # Tier 3: Cloud premium
  - providers:
      - name: claude-sonnet
        anthropic:
          model: claude-sonnet-4-5-20250514
```

AgentGateway exposes a **unified OpenAI-compatible API** regardless of the backend provider. Clients send requests to one endpoint; the gateway handles provider-specific translation (OpenAI, Anthropic, Gemini, Bedrock, local).

---

## 5. Could we seamlessly handle the response formats from these providers?

**Yes.** AgentGateway normalizes all provider responses to the **OpenAI chat completions format**. Regardless of whether the backend is Anthropic, Gemini, or a local model, the client always receives:

```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "..."
    }
  }]
}
```

This is handled transparently — no client-side changes needed when switching providers. Our [prompt enrichment setup](../releases/prompt-enrichment.yaml) demonstrates this: the same `curl` command works whether the backend is Anthropic or OpenAI.

For **streaming**, AgentGateway also normalizes SSE streams across providers.

---

## 6. Can we version the agents built from kagent?

**Yes, through GitOps:**

- Agent definitions are YAML manifests in our [`releases/`](../releases/) directory
- Every change is a git commit with full history
- Flux reconciles the desired state from git
- Helm chart versions pin kagent itself (currently `0.7.23`)

**For custom agents (our KMCP server):**

- Docker images tagged with git SHA: `ghcr.io/kyrylyuk-andriy/k8s-health-checker:8f67c11`
- Semantic versioning in code (`version: "0.1.0"`)
- A2A Agent Cards include a `version` field for runtime discovery

**What's not built-in:**
- kagent doesn't have native agent versioning (v1, v2 of the same agent). You'd manage this through separate Agent CRs (`k8s-agent-v1`, `k8s-agent-v2`) or Helm chart values.

---

## 7. Any blue/green or canary deployment patterns for agents?

**Not built into kagent directly**, but achievable through the infrastructure:

**Blue/Green via GitOps:**
- Deploy `agent-v2` alongside `agent-v1` as separate Agent CRs
- Update the A2A team coordinator or HTTPRoute to point to v2
- Roll back by reverting the git commit — Flux reconciles back to v1

**Canary via AgentGateway:**
- Use **weighted routing** in HTTPRoute to split traffic: potential issue long running connection and missed context 

```yaml
rules:
  - backendRefs:
      - name: agent-v1
        weight: 90
      - name: agent-v2
        weight: 10
```

**Canary via Argo Rollouts:**
- kagent ships with an `argo-rollouts-conversion-agent` — it's already in our cluster
- Argo Rollouts supports canary and blue/green natively for K8s deployments

**Evaluation-driven canary:**
- Use our [evaluation scripts](../tracing/evaluate_team.py) to compare v1 vs v2 scores in Phoenix before promoting

---

## 8. What's the FastMCP Python framework mentioned?

[FastMCP](https://gofastmcp.com) is the most popular Python framework for building MCP servers. It's the "FastAPI of MCP" — hides protocol complexity behind decorators.

```python
from fastmcp import FastMCP

mcp = FastMCP("My Server")

@mcp.tool()
def get_weather(city: str) -> str:
    """Get weather for a city."""
    return f"Sunny in {city}"

mcp.run()
```

Key features:
- **Decorator-based** — `@mcp.tool()`, `@mcp.resource()`, `@mcp.prompt()`
- **Auto-generated schemas** — from Python type hints
- **Multiple transports** — stdio, HTTP, WebSocket, SSE
- **1M+ daily downloads**, powers ~70% of MCP servers across all languages

Reference: [FastMCP GitHub](https://github.com/jlowin/fastmcp)

---

## 9. Is it the easiest path to MCP?

**For Python: Yes.** FastMCP is the fastest way — a working MCP server in ~10 lines of code.

**For Go (our choice):** We used [mcp-go](https://github.com/mark3labs/mcp-go) which is similarly ergonomic:

```go
s.AddTool(mcp.NewTool("get_pod_status",
    mcp.WithDescription("List pods"),
    mcp.WithString("namespace", mcp.DefaultString("default")),
), handler)
```

**For Kubernetes deployment:** kagent's MCPServer CRD is the easiest path — just provide a Docker image and kagent handles deployment, sidecar injection, service discovery, and agent integration.

**Comparison:**

| Path | Lines of code | Deploy to K8s |
|------|--------------|---------------|
| FastMCP (Python) | ~10 | Need Dockerfile + manifests |
| mcp-go (Go) | ~30 | Need Dockerfile + manifests |
| kagent MCPServer CRD | Image only | Automatic (CRD → pod → sidecar → agent) |

---

## 10. About FinOps: how much control can I have?

AgentGateway provides **gateway-level FinOps controls**:

- **Token-based rate limiting** — limit tokens per time window per route
- **Request-based rate limiting** — limit requests per second
- **Cost tracking** — automatic token counting on every request/response, exposed via Prometheus metrics and OpenTelemetry traces
- **Spend alerts** — track consumption through metrics and dashboards

**In our setup:**
- Phoenix traces capture token usage per agent call
- MCPG governance scores include rate limiting checks
- AgentGateway metrics can feed into Grafana dashboards

Reference: [AgentGateway Spend Control](https://agentgateway.dev/docs/local/main/llm/spending/)

---

## 11. Token level / per agent level

**Token level:**
- AgentGateway tracks input/output tokens on every LLM call
- Token metrics exposed via Prometheus: `agentgateway_tokens_total{direction="input|output", model="...", route="..."}`
- Per-request token counts in OpenTelemetry spans visible in Phoenix

**Per agent level:**
- Achievable by assigning **different HTTPRoutes per agent** in AgentGateway, each with its own rate limit policy
- Each route can have independent `maxTokens`, `fillInterval`, and budget limits
- Our A2A agents already use separate endpoints, making per-agent tracking natural

```yaml
# Agent-specific rate limit
apiVersion: agentgateway.dev/v1alpha1
kind: AgentgatewayPolicy
metadata:
  name: health-agent-budget
spec:
  targetRefs:
    - kind: HTTPRoute
      name: health-agent-route
  backend:
    ai:
      localRateLimit:
        - maxTokens: 10000
          tokensPerFill: 1000
          fillInterval: 3600s
          type: tokens
```

---

## 12. Can I implement custom cost controls?

**Yes, multiple approaches:**

**1. Gateway-level (AgentGateway):**
```yaml
localRateLimit:
  - maxTokens: 5000        # Budget: 5000 tokens
    tokensPerFill: 5000     # Refill every hour
    fillInterval: 3600s
    type: tokens
```
With `tokenize: true`, requests are rejected before reaching the LLM if budget is exceeded.

**2. Governance-level (MCPG):**
- Our MCPG deployment scores whether rate limiting is configured
- `maxToolsWarning: 10` / `maxToolsCritical: 15` — governance alerts on excessive tool exposure

**3. Application-level:**
- Our [evaluation scripts](../tracing/evaluate_team.py) track response times and could be extended to track costs
- A2A team coordinator could enforce per-task token budgets before delegating

**4. Observability-level (Phoenix):**
- Token usage visible per trace
- Set up alerts on token consumption anomalies

---

## 13. Per-agent budgets or depth of token limits

**Per-agent budgets:**
- Create separate `AgentgatewayPolicy` per agent route with different `maxTokens` values
- Example: health-agent gets 10K tokens/hour, SRE-coordinator gets 50K tokens/hour

**Depth limits (recursive agent calls):**
- AgentGateway's token rate limiting naturally caps recursive loops — an agent making 15 LLM calls hits the budget ceiling
- The A2A team coordinator can enforce a max number of delegated tasks per request (currently hardcoded in our `planTasks()` function)
- kagent's streaming timeout (600s default) prevents indefinitely running agent chains

**What's missing:**
- No built-in "max recursion depth" or "max tool calls per task" in kagent. This is application logic — our A2A agents handle it by design (fixed delegation plans, not open-ended loops).

---

## 14. vLLM suitable for agents with many back-and-forth tool calls, or is it better for single-shot inference?

**vLLM handles both**, but with different optimization profiles:

**Good for agents (multi-turn tool calls):**
- **Continuous batching** — doesn't wait for a batch to fill; processes requests as they arrive
- **PagedAttention** — efficient KV cache management across many concurrent sessions
- **Prefix caching** — reuses computation from shared system prompts across calls (important for agents that make 15+ calls with the same system prompt)

**Better for single-shot at scale:**
- vLLM's throughput optimizations (speculative decoding, tensor parallelism) shine most with high-throughput batch workloads
- Single-shot inference benefits most from batching optimizations

**For our setup:**
- A local vLLM instance would work as a Tier 1 fallback in AgentGateway's failover chain
- The agent's multi-turn nature means latency matters more than throughput — consider smaller, faster models (Llama 3.1 8B) for tool-call-heavy agents

---

## 15. llm-d's scheduler — helps when agent makes 15 LLM calls?

**Yes, significantly.** [llm-d](https://llm-d.ai) enhances vLLM with a cloud-native control plane:

**Prefix-cache-aware routing:**
- When an agent makes 15 calls with the same system prompt, llm-d routes them to the same vLLM instance that already has the prefix cached
- Reduces time-to-first-token (TTFT) by skipping redundant prefill computation
- Critical for agents — the system prompt is re-sent on every call but only computed once

**Disaggregated serving:**
- Splits inference into **prefill servers** (process prompt) and **decode servers** (generate tokens)
- Agent tool calls have short responses but long prompts (context accumulates) — prefill disaggregation reduces TTFT

**Utilization-based load balancing:**
- Distributes agent calls across vLLM instances based on actual GPU utilization, not just request count
- Prevents hotspots when one agent is making rapid sequential calls

**Multi-tenant fairness:**
- When multiple agents share the same vLLM cluster, llm-d ensures fair scheduling
- Prevents one chatty agent from starving others

**Our recommendation:** For production agent workloads with frequent tool calls, deploy **vLLM + llm-d** as the local inference tier behind AgentGateway's failover. AgentGateway handles provider routing; llm-d handles inference-level scheduling.

Reference: [llm-d Intelligent Scheduling](https://llm-d.ai/docs/guide/Installation/inference-scheduling)

---

## References

### AgentGateway / kgateway
- [Model Failover](https://agentgateway.dev/docs/kubernetes/latest/llm/failover/)
- [Spend Control](https://agentgateway.dev/docs/local/main/llm/spending/)
- [Rate Limiting](https://agentgateway.dev/docs/standalone/latest/configuration/resiliency/rate-limits/)
- [LLM Cost Tracking](https://agentgateway.dev/docs/kubernetes/main/llm/cost-tracking/)
- [Prompt Enrichment](https://agentgateway.dev/docs/kubernetes/latest/tutorials/prompt-enrichment/)
- [Why Traditional Gateways Failed AI Workloads (Solo.io)](https://www.solo.io/blog/why-traditional-gateways-failed-ai-workloads-and-how-kgateways-rust-powered-agentgateway-fixes-it)

### kagent
- [kagent Documentation](https://kagent.dev/docs/kagent)
- [kagent A2A Agents](https://kagent.dev/docs/kagent/examples/a2a-agents)
- [kagent Getting Started](https://kagent.dev/docs/kagent/getting-started/quickstart)
- [Building Agents on K8s with kagent (Deep Dive)](https://www.cloudnativedeepdive.com/building-agents-on-kubernetes-with-kagent/)

### MCP
- [FastMCP Framework](https://gofastmcp.com/getting-started/welcome)
- [FastMCP GitHub](https://github.com/jlowin/fastmcp)
- [mcp-go SDK](https://github.com/mark3labs/mcp-go)
- [MCP Protocol Specification](https://modelcontextprotocol.io)

### A2A Protocol
- [A2A Specification](https://a2a-protocol.org/latest/specification/)
- [Official Go SDK](https://github.com/a2aproject/a2a-go)

### FinOps
- [Agent FinOps: Token Cost Governance (Cordum)](https://cordum.io/blog/agent-finops-token-cost-governance)
- [FinOps for Agents (InfoWorld)](https://www.infoworld.com/article/4138748/finops-for-agents-loop-limits-tool-call-caps-and-the-new-unit-economics-of-agentic-saas.html)
- [FinOps for Agentic: Token Usage Cost (Deep Dive)](https://www.cloudnativedeepdive.com/finops-for-agentic-how-to-capture-token-usage-cost-across-llms/)
- [Rate Limiting LLM Tokens with AgentGateway (Deep Dive)](https://www.cloudnativedeepdive.com/rate-limiting-llm-token-usage-with-agentgateway/)

### vLLM & llm-d
- [vLLM Project](https://vllm.ai/)
- [llm-d Intelligent Scheduling](https://llm-d.ai/docs/guide/Installation/inference-scheduling)
- [Demystifying llm-d and vLLM (Red Hat)](https://www.redhat.com/en/blog/demystifying-llm-d-and-vllm-race-production)
- [Inside vLLM: Anatomy of High-Throughput Inference](https://www.aleksagordic.com/blog/vllm)

### Phoenix & Observability
- [Phoenix Self-Hosting](https://arize.com/docs/phoenix/self-hosting)
- [MCP Tracing with Phoenix](https://arize.com/docs/phoenix/integrations/python/mcp-tracing)
- [Pydantic Evals](https://arize.com/docs/phoenix/integrations/python/pydantic/pydantic-evals)
