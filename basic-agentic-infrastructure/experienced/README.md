# Experienced Agentic Infrastructure

This module builds on the [basic setup](../basic/README.md) by deploying AgentGateway and kagent in a Kubernetes cluster with proper secrets management.

---

## Plan

1. Deploy AgentGateway as a Helm chart in a Kubernetes cluster
2. Configure Secrets and ConfigMap for API keys and gateway configuration
3. Deploy [kagent](https://kagent.dev/docs/kagent/getting-started/quickstart)
4. Configure model route through AgentGateway
5. Verify any built-in agent works end-to-end

---

## 1. Deploy AgentGateway in Kubernetes

Reference: https://agentgateway.dev/docs/kubernetes/main/quickstart/install/

### Prerequisites

- A Kubernetes cluster (e.g. [Kind](https://kind.sigs.k8s.io/))
- `kubectl` configured
- `helm` installed

### 1.1 Deploy Kubernetes Gateway API CRDs

```bash
kubectl apply --server-side --force-conflicts -f \
  https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.5.0/standard-install.yaml
```

**Expected output:**

```
customresourcedefinition.apiextensions.k8s.io/backendtlspolicies.gateway.networking.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/gatewayclasses.gateway.networking.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/gateways.gateway.networking.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/grpcroutes.gateway.networking.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/httproutes.gateway.networking.k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/referencegrants.gateway.networking.k8s.io serverside-applied
```

### 1.2 Deploy AgentGateway CRDs via Helm

```bash
helm upgrade -i agentgateway-crds \
  oci://cr.agentgateway.dev/charts/agentgateway-crds \
  --create-namespace --namespace agentgateway-system \
  --version v1.0.0-rc.1 \
  --set controller.image.pullPolicy=Always
```

**Expected output:**

```
Release "agentgateway-crds" does not exist. Installing it now.
Pulled: cr.agentgateway.dev/charts/agentgateway-crds:v1.0.0-rc.1
NAME: agentgateway-crds
LAST DEPLOYED: ...
NAMESPACE: agentgateway-system
STATUS: deployed
```

Verify CRDs:

```bash
kubectl get crds | grep agentgateway.dev
```

**Expected output:**

```
agentgatewaybackends.agentgateway.dev
agentgatewayparameters.agentgateway.dev
agentgatewaypolicies.agentgateway.dev
```

### 1.3 Install AgentGateway Control Plane

```bash
helm upgrade -i agentgateway \
  oci://cr.agentgateway.dev/charts/agentgateway \
  --namespace agentgateway-system \
  --version v1.0.0-rc.1 \
  --set controller.image.pullPolicy=Always \
  --set controller.extraEnv.KGW_ENABLE_GATEWAY_API_EXPERIMENTAL_FEATURES=true
```

Verify the control plane is running:

```bash
kubectl get pods -n agentgateway-system
```

**Expected output:**

```
NAME                            READY   STATUS    RESTARTS   AGE
agentgateway-5f76d6dddb-vvqtj   1/1     Running   0          21s
```

### 1.4 Create the Gateway Resource

```bash
kubectl apply -f- <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: agentgateway-proxy
  namespace: agentgateway-system
spec:
  gatewayClassName: agentgateway
  listeners:
  - protocol: HTTP
    port: 80
    name: http
    allowedRoutes:
      namespaces:
        from: All
EOF
```

Verify the gateway and proxy deployment:

```bash
kubectl get gateway agentgateway-proxy -n agentgateway-system
kubectl get deployment agentgateway-proxy -n agentgateway-system
```

**Expected output:**

```
NAME                 CLASS          ADDRESS   PROGRAMMED   AGE
agentgateway-proxy   agentgateway             True         22s

NAME                 READY   UP-TO-DATE   AVAILABLE   AGE
agentgateway-proxy   1/1     1            1           22s
```

### 1.5 Access the Gateway

Port-forward both the API traffic port and the admin UI:

```bash
kubectl port-forward deployment/agentgateway-proxy \
  -n agentgateway-system 8080:80 15000:15000
```

- **API traffic:** http://localhost:8080
- **Admin UI:** http://localhost:15000/ui

---

## 2. Configure Secrets for API Keys

### 2.1 Create a Kubernetes Secret for Anthropic

```bash
kubectl apply -f- <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: anthropic-secret
  namespace: agentgateway-system
type: Opaque
stringData:
  Authorization: $ANTHROPIC_API_KEY
EOF
```

---

## 3. Configure Model Route through AgentGateway

### 3.1 Create an AgentgatewayBackend

```bash
kubectl apply -f- <<EOF
apiVersion: agentgateway.dev/v1alpha1
kind: AgentgatewayBackend
metadata:
  name: anthropic
  namespace: agentgateway-system
spec:
  ai:
    provider:
      anthropic:
        model: claude-haiku-4-5-20251001
  policies:
    auth:
      secretRef:
        name: anthropic-secret
EOF
```

### 3.2 Create an HTTPRoute

```bash
kubectl apply -f- <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: anthropic
  namespace: agentgateway-system
spec:
  parentRefs:
    - name: agentgateway-proxy
      namespace: agentgateway-system
  rules:
    - backendRefs:
      - name: anthropic
        namespace: agentgateway-system
        group: agentgateway.dev
        kind: AgentgatewayBackend
EOF
```

### 3.3 Test it

```bash
curl "localhost:8080/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-haiku-4-5-20251001",
    "messages": [{"role": "user", "content": "Hello!"}]
  }' | jq
```

**Expected response:**

```json
{
  "model": "claude-haiku-4-5-20251001",
  "usage": {
    "prompt_tokens": 9,
    "completion_tokens": 23,
    "total_tokens": 32
  },
  "choices": [
    {
      "message": {
        "content": "Hello! 👋 It's nice to meet you. How can I help you today?",
        "role": "assistant"
      },
      "index": 0,
      "finish_reason": "stop"
    }
  ],
  "id": "msg_0155Xgme2nASqPrZPyW2L1qi",
  "created": 1773499451,
  "object": "chat.completion"
}
```

---

## 4. Deploy kagent

Reference: https://kagent.dev/docs/kagent/getting-started/quickstart

### 4.1 Install kagent CLI

```bash
brew install kagent
```

### 4.2 Deploy to cluster

The installer requires `OPENAI_API_KEY` — use a dummy value since we'll reconfigure to Anthropic:

```bash
OPENAI_API_KEY=dummy kagent install --profile demo
```

Verify pods are running:

```bash
kubectl get pods -n kagent
```

**Expected output (demo profile includes many built-in agents):**

```
k8s-agent-...          1/1     Running
helm-agent-...         1/1     Running
istio-agent-...        1/1     Running
kagent-controller-...  1/1     Running
kagent-ui-...          1/1     Running
...
```

### 4.3 Route kagent through AgentGateway

Reference: https://kagent.dev/docs/kagent/supported-providers/byo-openai

Create a secret for the Anthropic API key in the kagent namespace (AgentGateway handles the actual auth, but kagent requires a secret):

```bash
kubectl create secret generic kagent-anthropic \
  --namespace kagent \
  --from-literal=ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY
```

Replace the default ModelConfig to use AgentGateway as a BYO OpenAI-compatible endpoint:

```bash
kubectl replace --force -f- <<EOF
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: default-model-config
  namespace: kagent
spec:
  provider: OpenAI
  model: claude-haiku-4-5-20251001
  apiKeySecret: kagent-anthropic
  apiKeySecretKey: ANTHROPIC_API_KEY
  openAI:
    baseUrl: "http://agentgateway-proxy.agentgateway-system.svc.cluster.local"
EOF
```

**Why `provider: OpenAI` when we're using Anthropic?** The kagent [Anthropic provider](https://kagent.dev/docs/kagent/supported-providers/anthropic) doesn't support a custom `baseUrl` field, so there's no way to point it at AgentGateway. The [BYO OpenAI-compatible provider](https://kagent.dev/docs/kagent/supported-providers/byo-openai) does support `baseUrl`, and AgentGateway exposes an OpenAI-compatible `/v1/chat/completions` endpoint. AgentGateway handles the translation from OpenAI format to Anthropic's native API on the backend side.

This routes all kagent LLM traffic through AgentGateway, which translates requests and forwards them to Anthropic. The flow is:

```
kagent agent → AgentGateway (OpenAI-compatible) → Anthropic API
```

You can observe all traffic in the AgentGateway admin UI at http://localhost:15000/ui

### 4.4 Access the Dashboard

```bash
kagent dashboard
```

Open http://localhost:8082

---

## 5. Verify a Built-in Agent

1. Open the kagent dashboard at http://localhost:8082
2. Select a built-in agent (e.g. **k8s-agent**)
3. Send a prompt: "What pods are running in my cluster?"
4. The agent should use Anthropic (claude-haiku-4-5) via AgentGateway to reason and execute kubectl commands
5. Verify traffic appears in the AgentGateway admin UI at http://localhost:15000/ui

---

## Cleanup

```bash
kagent uninstall
helm uninstall agentgateway agentgateway-crds -n agentgateway-system
```
