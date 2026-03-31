# A2A Team Coordinator

Orchestrates a team of AI agents — both custom standalone agents and [kagent](https://kagent.dev) built-in agents — via the A2A protocol.

## Architecture

```
User → A2A Team Coordinator (port 9092)
         ├── K8s Health Agent (custom, port 9090)     — cluster health via client-go
         ├── Helm Agent (kagent, port 8083)            — Helm releases via kagent A2A
         └── K8s Agent (kagent, port 8083)             — K8s diagnostics via kagent A2A
```

## How it works

1. Receives a request via A2A
2. Plans which agents to consult based on the request
3. Discovers each agent's card via `/.well-known/agent-card.json`
4. Sends A2A tasks to each agent (handles both official SDK and kagent formats)
5. Compiles responses into a unified Team Report

## Build & Run

```bash
go build -o bin/a2a-team .

# Start all team members
cd ../a2a-agent && PORT=9090 ./bin/a2a-agent &
kubectl port-forward svc/kagent-controller -n kagent 8083:8083 &

# Start coordinator
./bin/a2a-team
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `:9092` | Server port |
| `HEALTH_AGENT_URL` | `http://localhost:9090` | Custom health agent |
| `KAGENT_A2A_URL` | `http://localhost:8083/api/a2a/kagent` | kagent A2A base URL |

## Test

```bash
curl -s -X POST http://localhost:9092/ -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"SendMessage","params":{"message":{"messageId":"team-1","role":"user","parts":[{"text":"Run a full health check"}]}},"id":1}' | jq .
```

## Tech

- **Language:** Go
- **SDK:** [github.com/a2aproject/a2a-go/v2](https://github.com/a2aproject/a2a-go)
- **Pattern:** Multi-agent team coordination via A2A, bridging custom and kagent agents

---
*Back to [mcp-server-gitops](../)*
