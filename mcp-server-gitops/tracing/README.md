# Tracing & Evaluation

Python scripts that instrument the A2A agent team with OpenTelemetry and send traces to [Phoenix](https://arize.com/docs/phoenix) for observability and evaluation.

## Setup

```bash
pip3 install -r requirements.txt
```

## Scripts

### trace_team.py

Traces A2A communication across the agent team and sends spans to Phoenix.

**What it traces:**
- Agent card discovery (`GET /.well-known/agent-card.json`)
- A2A message send (`POST /` with JSON-RPC)
- Response parsing and timing

```bash
# Start agents + Phoenix port-forward
cd ../a2a-agent && ./bin/a2a-agent &
kubectl port-forward svc/kagent-controller -n kagent 8083:8083 &
cd ../a2a-team && ./bin/a2a-team &
kubectl port-forward svc/phoenix-svc -n phoenix 6006:6006 &

# Run tracing
python3 trace_team.py
```

Results visible in Phoenix under the `a2a-team-tracer` project.

### evaluate_team.py

Evaluates the K8s Health Agent against test cases using keyword matching and reports scores to Phoenix.

```bash
cd ../a2a-agent && ./bin/a2a-agent &
kubectl port-forward svc/phoenix-svc -n phoenix 6006:6006 &

python3 evaluate_team.py
```

**Test results:**

| Test | Input | Score |
|------|-------|-------|
| node_status | Show cluster node status | 1.0 |
| pod_status_kagent | Show pods in kagent namespace | 1.0 |
| pod_status_default | Show pods in default namespace | 1.0 |
| events_check | Show warning events | 0.67 |
| deployment_status | Show deployments in kube-system | 1.0 |
| cluster_summary | Full cluster health summary | 1.0 |

**Average score: 0.94** (6/6 passed)

Results visible in Phoenix under the `a2a-team-evaluator` project.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PHOENIX_URL` | `http://localhost:6006` | Phoenix UI URL |
| `PHOENIX_COLLECTOR` | `http://localhost:6006/v1/traces` | OTLP HTTP endpoint |
| `PHOENIX_API_KEY` | (empty) | Bearer token if Phoenix auth is enabled |
| `HEALTH_AGENT_URL` | `http://localhost:9090` | K8s Health Agent |
| `TEAM_URL` | `http://localhost:9092` | A2A Team Coordinator |

## References

- [MCP Tracing with Phoenix](https://arize.com/docs/phoenix/integrations/python/mcp-tracing)
- [Pydantic Evals](https://arize.com/docs/phoenix/integrations/python/pydantic/pydantic-evals)
- [Phoenix Self-Hosting](https://arize.com/docs/phoenix/self-hosting)
