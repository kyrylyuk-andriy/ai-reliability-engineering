#!/bin/bash
set -euo pipefail

LOG=/tmp/mcp-gitops-setup.log
exec > >(tee -a "$LOG") 2>&1

log() { echo "[$(date '+%H:%M:%S')] $*"; }

log "=== mcp-server-gitops setup start ==="

# Check prerequisites
for cmd in terraform kubectl helm kind; do
  if ! command -v "$cmd" &>/dev/null; then
    log "ERROR: $cmd is not installed. Run 'make tools' first."
    exit 1
  fi
done

# Check Anthropic API key
if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
  log "ERROR: ANTHROPIC_API_KEY is not set."
  log "Run: export ANTHROPIC_API_KEY='your-api-key'"
  exit 1
fi

# Initialize Terraform
log "Running terraform init..."
cd bootstrap
terraform init
log "terraform init done"

# Apply Terraform (creates Kind cluster + bootstraps Flux)
log "Running terraform apply..."
terraform apply -auto-approve -var="anthropic_api_key=$ANTHROPIC_API_KEY"
log "terraform apply done"

export KUBECONFIG=~/.kube/config

cd ..

log "Waiting for Flux to reconcile..."
sleep 30

log "Checking Flux status..."
kubectl get kustomizations -n flux-system

log "=== setup complete ==="
log "Access UIs with:"
log "  kubectl port-forward -n agentgateway-system deployment/agentgateway-external 8080:80 15000:15000"
log "  AgentGateway UI: http://localhost:15000/ui"
log "  kagent UI:       http://localhost:8080"
