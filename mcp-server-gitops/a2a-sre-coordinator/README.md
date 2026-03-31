# A2A SRE Coordinator

An SRE orchestrator agent that delegates health checks to the [K8s Health Agent](../a2a-agent/) via A2A protocol and compiles SRE reports.

## How it works

```
User → SRE Coordinator (port 9091)
         ├── Discovers Health Agent via /.well-known/agent-card.json
         ├── Sends A2A tasks based on request type
         └── Compiles results into an SRE Health Report
```

## Skills

| Skill | Description |
|-------|-------------|
| health-assessment | Comprehensive cluster health check (nodes, pods, events) |
| incident-triage | Initial incident triage (events, nodes, system pods) |

## Build & Run

```bash
go build -o bin/sre-coordinator .

# Start health agent first
cd ../a2a-agent && PORT=9090 ./bin/a2a-agent &

# Start coordinator
./bin/sre-coordinator
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `:9091` | Server port |
| `HEALTH_AGENT_URL` | `http://localhost:9090/` | K8s Health Agent A2A endpoint |

## Test

```bash
curl -s -X POST http://localhost:9091/ -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"SendMessage","params":{"message":{"messageId":"sre-1","role":"user","parts":[{"text":"Run a full health check"}]}},"id":1}' | jq .
```

## Tech

- **Language:** Go
- **SDK:** [github.com/a2aproject/a2a-go/v2](https://github.com/a2aproject/a2a-go)
- **Pattern:** Agent-to-agent delegation via A2A protocol

---
*Back to [mcp-server-gitops](../)*
