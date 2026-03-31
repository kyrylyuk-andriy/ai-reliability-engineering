# Releases

Kubernetes manifests synced by Flux CD via GitOps. Changes pushed to this directory are automatically reconciled to the cluster.

## Two-Phase Deployment

```
releases-crds (phase 1, wait: true)    →    releases (phase 2, dependsOn: crds)
```

CRDs install first to ensure custom resource types exist before apps reference them.

## Manifests

| File | Components | Namespace |
|------|-----------|-----------|
| `agentgateway.yaml` | Namespace, OCIRepository, HelmRelease, Gateway | agentgateway-system |
| `kagent.yaml` | Namespace, OCIRepository, HelmRelease, HTTPRoute, ModelConfig, ReferenceGrant | kagent |
| `kmcp-server.yaml` | ServiceAccount, RBAC, MCPServer, Agent | kagent |
| `mcpg.yaml` | Namespace, RBAC, Controller, Dashboard, GovernancePolicy, Evaluation | mcp-governance |
| `agentregistry.yaml` | Namespace, PostgreSQL StatefulSet, Server Deployment | agentregistry |
| `phoenix.yaml` | Namespace, OCIRepository, HelmRelease (includes PostgreSQL) | phoenix |
| `qdrant.yaml` | Namespace, HelmRepository, HelmRelease | qdrant |
| `prompt-enrichment.yaml` | AgentgatewayBackend, HTTPRoute, AgentgatewayPolicy | agentgateway-system |

## CRDs (`crds/`)

| File | CRDs |
|------|------|
| `agentgateway-crds.yaml` | AgentGateway CRDs via Helm |
| `kagent-crds.yaml` | kagent CRDs via Helm |
| `mcpg-crds.yaml` | MCPGovernancePolicy, GovernanceEvaluation |
