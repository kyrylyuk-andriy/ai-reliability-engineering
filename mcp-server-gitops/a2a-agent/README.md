# A2A Health Agent

Standalone A2A-compliant Kubernetes health checker agent built with the official [a2a-go SDK](https://github.com/a2aproject/a2a-go).

## What it does

Receives natural language requests via the [A2A protocol](https://a2a-protocol.org) and queries the Kubernetes cluster for health information using client-go.

## Skills

| Skill | Description |
|-------|-------------|
| pod-status | List pods with status in a namespace |
| node-status | List cluster nodes with conditions |
| deployment-status | List deployments with replica counts |
| warning-events | Get recent warning events |
| cluster-summary | Full cluster health summary |

## Build & Run

```bash
go build -o bin/a2a-agent .
PORT=9090 ./bin/a2a-agent
```

## Endpoints

- **Agent Card:** `GET /.well-known/agent-card.json`
- **A2A JSON-RPC:** `POST /`

## Test

```bash
# Discover agent
curl -s http://localhost:9090/.well-known/agent-card.json | jq .name

# Send task
curl -s -X POST http://localhost:9090/ -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"SendMessage","params":{"message":{"messageId":"test-1","role":"user","parts":[{"text":"Show node status"}]}},"id":1}' | jq .
```

## Tech

- **Language:** Go
- **SDK:** [github.com/a2aproject/a2a-go/v2](https://github.com/a2aproject/a2a-go)
- **K8s client:** client-go (in-cluster or kubeconfig fallback)
- **Transport:** JSON-RPC over HTTP

---
*Back to [mcp-server-gitops](../)*
