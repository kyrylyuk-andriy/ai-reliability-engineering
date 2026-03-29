# A2A Protocol Research

Research document covering the Agent-to-Agent (A2A) protocol specification, architecture, and implementation patterns.

---

## 1. What is A2A?

The **Agent-to-Agent (A2A) Protocol** is an open standard created by Google that enables independent AI agents to communicate and collaborate — regardless of framework, language, or vendor.

While **MCP** (Model Context Protocol) standardizes how agents talk to **tools**, A2A standardizes how agents talk to **each other**.

```
┌──────────┐   A2A Protocol    ┌──────────┐
│  Agent A  │ ◄──────────────► │  Agent B  │
│ (Client)  │  Tasks/Messages  │ (Server)  │
└──────────┘                   └──────────┘
      │                              │
      │ MCP                          │ MCP
      ▼                              ▼
   [Tools]                        [Tools]
```

**Key principle:** Agents are opaque — they don't expose internal state, prompts, or memory. They communicate only through the A2A protocol's defined messages, tasks, and artifacts.

### References

- [A2A Specification](https://a2a-protocol.org/latest/specification/)
- [Official GitHub](https://github.com/a2aproject/A2A)
- [Google Announcement](https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/)

---

## 2. Architecture

The specification is structured in three layers:

| Layer | Purpose |
|-------|---------|
| **Data Model** | Protocol-agnostic structures: Task, Message, AgentCard, Part, Artifact |
| **Operations** | Capabilities: SendMessage, Stream, GetTask, ListTasks, CancelTask |
| **Protocol Bindings** | Concrete mappings to JSON-RPC, gRPC, HTTP/REST |

### Core Concepts

| Concept | Description |
|---------|-------------|
| **Agent Card** | JSON document describing agent identity, capabilities, skills, and endpoints |
| **Task** | Stateful work unit with lifecycle (created → working → completed) |
| **Message** | Communication unit with role (user/agent) and content parts |
| **Artifact** | Output produced by an agent (files, data, structured content) |
| **Part** | Content block within a message (text, file, structured data) |
| **Context ID** | Groups related tasks for conversational continuity |

---

## 3. Agent Card

The Agent Card is the foundation of A2A — a JSON document that serves as a digital "business card" describing what an agent can do and how to interact with it.

### Well-Known URI

Agents expose their card at a standardized URL following [RFC 8615](https://datatracker.ietf.org/doc/html/rfc8615):

```
GET https://{agent-domain}/.well-known/agent-card.json
```

For kagent agents specifically:

```
GET /api/a2a/{namespace}/{agent-name}/.well-known/agent.json
```

### Agent Card Schema

```json
{
  "id": "k8s-health-agent",
  "name": "Kubernetes Health Agent",
  "description": "Monitors Kubernetes cluster health including pods, nodes, deployments, and events",
  "provider": {
    "name": "AI Reliability Engineering",
    "url": "https://github.com/kyrylyuk-andriy/ai-reliability-engineering"
  },
  "url": "https://agent.example.com/a2a",
  "capabilities": {
    "streaming": true,
    "pushNotifications": false,
    "extendedAgentCard": false
  },
  "skills": [
    {
      "id": "cluster-health",
      "name": "Cluster Health Check",
      "description": "Check the health status of Kubernetes cluster resources",
      "inputModes": ["text"],
      "outputModes": ["text"],
      "examples": [
        "What pods are running in the kagent namespace?",
        "Show me the cluster node status",
        "Are there any warning events?"
      ]
    }
  ],
  "interfaces": [
    {
      "type": "json-rpc",
      "url": "https://agent.example.com/rpc"
    }
  ],
  "securitySchemes": {
    "apiKey": {
      "type": "apiKey",
      "name": "X-API-Key",
      "in": "header"
    }
  },
  "security": [
    { "apiKey": [] }
  ]
}
```

### Agent Card Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique agent identifier |
| `name` | string | Human-readable name |
| `description` | string | What the agent does |
| `provider` | object | Organization behind the agent (name, url) |
| `url` | string | A2A service endpoint |
| `capabilities` | object | streaming, pushNotifications, extendedAgentCard |
| `skills` | array | List of AgentSkill objects |
| `interfaces` | array | Protocol bindings (json-rpc, grpc, http) |
| `securitySchemes` | object | Authentication methods (apiKey, OAuth2, bearer) |
| `security` | array | Required security for access |

### Discovery Methods

1. **Well-Known URI** — `/.well-known/agent-card.json` (standard for public agents)
2. **Curated Registries** — central catalog with search/filter (enterprise)
3. **Direct Configuration** — hardcoded URLs, env vars (static setups)

---

## 4. Task Lifecycle

Tasks are stateful work units — the core of A2A communication.

```
CREATED → WORKING → COMPLETED
                  → FAILED
                  → INPUT_REQUIRED → WORKING → COMPLETED
                  → AUTH_REQUIRED → WORKING → COMPLETED
         CANCELED (client-initiated)
         REJECTED (agent-initiated)
```

### Task States

| State | Description |
|-------|-------------|
| `CREATED` | Newly instantiated |
| `WORKING` | Agent is processing |
| `INPUT_REQUIRED` | Agent needs more info from client |
| `AUTH_REQUIRED` | Agent needs authentication |
| `COMPLETED` | Successfully finished |
| `FAILED` | Error occurred |
| `CANCELED` | Client canceled the task |
| `REJECTED` | Agent rejected the request |

### Task Object

```json
{
  "id": "task-123",
  "contextId": "context-456",
  "status": {
    "state": "COMPLETED",
    "message": "Health check complete",
    "timestamp": "2026-03-28T10:30:00Z"
  },
  "messages": [
    {
      "role": "user",
      "parts": [{ "text": "Check pod status in kagent namespace" }]
    },
    {
      "role": "agent",
      "parts": [{ "text": "All 8 pods running healthy in kagent namespace" }]
    }
  ],
  "artifacts": [
    {
      "id": "artifact-789",
      "mimeType": "application/json",
      "parts": [{ "text": "{\"healthy\": 8, \"unhealthy\": 0}" }]
    }
  ]
}
```

---

## 5. Operations (API)

### HTTP/REST Endpoints

| Operation | Method | URL | Description |
|-----------|--------|-----|-------------|
| Send Message | POST | `/messages` | Start or continue a task |
| Stream Message | POST | `/messages:stream` | Real-time task updates |
| Get Task | GET | `/tasks/{id}` | Retrieve task state |
| List Tasks | GET | `/tasks` | Query tasks with filters |
| Cancel Task | POST | `/tasks/{id}:cancel` | Cancel a running task |
| Subscribe | GET | `/tasks/{id}:subscribe` | Monitor task in real-time |
| Get Agent Card | GET | `/.well-known/agent-card.json` | Discover agent |

### Send Message Example

```json
POST /messages
{
  "message": {
    "role": "user",
    "parts": [{ "text": "What pods are running in kagent?" }]
  },
  "configuration": {
    "returnImmediately": false,
    "acceptedOutputModes": ["text/plain", "application/json"],
    "historyLength": 10
  }
}
```

### JSON-RPC Binding

```json
{
  "jsonrpc": "2.0",
  "method": "SendMessage",
  "params": {
    "message": {
      "role": "user",
      "parts": [{ "text": "Check cluster health" }]
    }
  },
  "id": 1
}
```

### Streaming Events

**TaskStatusUpdateEvent:**
```json
{
  "taskId": "task-123",
  "newStatus": {
    "state": "COMPLETED",
    "message": "Done",
    "timestamp": "2026-03-28T10:35:00Z"
  }
}
```

**TaskArtifactUpdateEvent:**
```json
{
  "taskId": "task-123",
  "artifact": {
    "id": "artifact-789",
    "mimeType": "text/plain",
    "parts": [{ "text": "3 nodes ready, 8 pods running" }]
  }
}
```

---

## 6. A2A vs MCP

| Aspect | A2A | MCP |
|--------|-----|-----|
| **Purpose** | Agent ↔ Agent communication | Agent ↔ Tool communication |
| **Participants** | Opaque agents (can't see internals) | Client + server with defined tools |
| **Communication** | Tasks with messages and artifacts | Tool calls with parameters and results |
| **Discovery** | Agent Card at well-known URI | Tool definitions in server capabilities |
| **State** | Stateful tasks with lifecycle | Stateless tool calls |
| **Protocol** | JSON-RPC / gRPC / HTTP REST | JSON-RPC over stdio / HTTP |
| **Created by** | Google | Anthropic |

**They are complementary:** A2A handles agent-to-agent orchestration; MCP handles agent-to-tool integration. An agent can use both — MCP to access tools, A2A to collaborate with other agents.

---

## 7. kagent A2A Support

Every kagent agent natively implements the A2A protocol.

### Enable A2A on a kagent Agent

Add `a2aConfig` to the Agent spec:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: k8s-a2a-agent
  namespace: kagent
spec:
  description: "K8s health agent with A2A support"
  type: Declarative
  declarative:
    modelConfig: default-model-config
    systemMessage: |
      You are a Kubernetes health check assistant.
    tools:
      - type: McpServer
        mcpServer:
          name: k8s-health-checker
          kind: MCPServer
          apiGroup: kagent.dev
  a2aConfig:
    skills:
      - id: cluster-health
        name: Cluster Health Check
        description: Check K8s cluster health
        inputModes: [text]
        outputModes: [text]
        tags: [k8s, health, monitoring]
        examples:
          - "What pods are running in kagent?"
          - "Show cluster node status"
```

### Access the Agent Card

```bash
kubectl port-forward svc/kagent-controller 8083:8083 -n kagent

curl http://localhost:8083/api/a2a/kagent/k8s-a2a-agent/.well-known/agent.json
```

### Invoke via CLI

```bash
kagent invoke --agent k8s-a2a-agent --task "Check pod status in kagent namespace"
```

### References

- [kagent A2A Agents](https://kagent.dev/docs/kagent/examples/a2a-agents)
- [kagent Discord A2A](https://kagent.dev/docs/kagent/examples/discord-a2a)

---

## 8. Real Technical & Business Use Cases

### Agent-to-Agent Communication

| Use Case | Description |
|----------|-------------|
| **SRE Incident Response** | Monitoring agent detects anomaly → sends A2A task to diagnosis agent → diagnosis agent uses tools to analyze → returns root cause to monitoring agent |
| **CI/CD Pipeline** | Build agent completes → sends A2A task to test agent → test agent runs suites → sends results to deploy agent |
| **Customer Support Escalation** | Tier-1 agent can't resolve → sends A2A task to specialist agent with full context → specialist resolves and returns answer |
| **Multi-cloud Management** | Orchestrator agent delegates tasks to AWS agent, GCP agent, Azure agent via A2A |
| **Security Audit** | Compliance agent sends audit tasks to infrastructure agents across clusters |

### A2A Teams

| Use Case | Description |
|----------|-------------|
| **K8s Operations Team** | Health agent + deployment agent + security agent collaborate on cluster management |
| **Data Pipeline** | Ingestion agent + transformation agent + validation agent + loading agent |
| **Code Review** | Style agent + security agent + performance agent each review and report findings |

---

## 9. Implementation SDKs

| SDK | Language | Repository |
|-----|----------|------------|
| **a2a-python** | Python | [a2aproject/a2a-python](https://github.com/a2aproject/a2a-python) |
| **trpc-a2a-go** | Go | [trpc-group/trpc-a2a-go](https://github.com/trpc-group/trpc-a2a-go) |
| **a2a-samples** | Multi | [a2aproject/a2a-samples](https://github.com/a2aproject/a2a-samples) |
| **python-a2a** | Python | [themanojdesai/python-a2a](https://github.com/themanojdesai/python-a2a) |
| **Google ADK** | Python | [google.github.io/adk-docs](https://google.github.io/adk-docs/a2a/quickstart-exposing/) |

---

## 10. Security

### Authentication Methods

| Method | Description |
|--------|-------------|
| API Key | Static key in header (`X-API-Key`) |
| Bearer Token | JWT or opaque token in Authorization header |
| OAuth 2.0 | Client credentials or authorization code flow |
| mTLS | Mutual TLS for service-to-service |
| OpenID Connect | Identity verification |

### In-Task Auth

Agents can request authentication mid-task by transitioning to `AUTH_REQUIRED` state. The client provides credentials, and the agent resumes processing.

### Agent Card Security

- **Public cards** — open at well-known URI
- **Extended cards** — authenticated endpoint with additional details
- **Caching** — `Cache-Control` + `ETag` headers for efficient re-fetching
