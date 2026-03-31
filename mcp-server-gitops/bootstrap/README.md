# Bootstrap

Terraform configuration that provisions the Kind cluster and bootstraps Flux CD for GitOps.

## What it creates

1. **Kind cluster** — 1 control-plane + 2 worker nodes with IPVS kube-proxy
2. **Flux Operator** — installed via Helm
3. **Flux Instance** — configured to sync from this Git repo
4. **Gateway API CRDs** — experimental channel (includes TLSRoute for AgentGateway)
5. **Secrets** — Anthropic API key in `kagent` and `agentgateway-system` namespaces
6. **GitRepository** — points at this repo's `mcp-server-gitops/releases` directory
7. **Kustomizations** — two-phase: `releases-crds` (CRDs first) → `releases` (apps)

## Files

| File | Purpose |
|------|---------|
| `providers.tf` | Terraform providers (kind, helm, kubectl) |
| `variables.tf` | Variables (git repo URL, branch, Anthropic API key) |
| `cluster.tf` | Kind cluster definition |
| `flux.tf` | Flux operator, instance, secrets, GitRepository, Kustomizations |

## Usage

```bash
cd bootstrap
terraform init
terraform apply -var="anthropic_api_key=$ANTHROPIC_API_KEY"
```

Or via the Makefile from the parent directory:

```bash
export ANTHROPIC_API_KEY="your-key"
make run
```

## Destroy

```bash
terraform destroy
# or
make down
```

---
*Back to [mcp-server-gitops](../)*
