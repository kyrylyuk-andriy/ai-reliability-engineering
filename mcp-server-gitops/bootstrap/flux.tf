# ==========================================
# Bootstrap Flux Operator
# ==========================================
resource "helm_release" "flux_operator" {
  depends_on = [kind_cluster.this]

  name             = "flux-operator"
  namespace        = "flux-system"
  repository       = "oci://ghcr.io/controlplaneio-fluxcd/charts"
  chart            = "flux-operator"
  create_namespace = true
}

# ==========================================
# Bootstrap Flux Instance
# ==========================================
resource "helm_release" "flux_instance" {
  depends_on = [helm_release.flux_operator]

  name       = "flux-instance"
  namespace  = "flux-system"
  repository = "oci://ghcr.io/controlplaneio-fluxcd/charts"
  chart      = "flux-instance"
  wait       = true

  set {
    name  = "distribution.version"
    value = "=2.x"
  }
}

# ==========================================
# Gateway API CRDs (installed before Flux syncs)
# ==========================================
resource "kubectl_manifest" "gateway_api_crds" {
  depends_on = [kind_cluster.this]

  yaml_body = <<-YAML
    apiVersion: v1
    kind: Namespace
    metadata:
      name: gateway-api-system
  YAML
}

resource "null_resource" "gateway_api_crds" {
  depends_on = [kind_cluster.this]

  provisioner "local-exec" {
    command = "kubectl apply --server-side --force-conflicts -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.5.0/standard-install.yaml"
  }
}

# ==========================================
# Anthropic API key secret for kagent
# ==========================================
resource "kubectl_manifest" "kagent_namespace" {
  depends_on = [kind_cluster.this]

  yaml_body = <<-YAML
    apiVersion: v1
    kind: Namespace
    metadata:
      name: kagent
  YAML
}

resource "kubectl_manifest" "anthropic_secret" {
  depends_on = [kubectl_manifest.kagent_namespace]

  yaml_body = <<-YAML
    apiVersion: v1
    kind: Secret
    metadata:
      name: kagent-anthropic
      namespace: kagent
    type: Opaque
    stringData:
      ANTHROPIC_API_KEY: ${var.anthropic_api_key}
  YAML
}

# ==========================================
# GitRepository source (syncs from Git repo)
# ==========================================
resource "kubectl_manifest" "git_repo" {
  depends_on = [helm_release.flux_instance]

  yaml_body = <<-YAML
    apiVersion: source.toolkit.fluxcd.io/v1
    kind: GitRepository
    metadata:
      name: mcp-server-gitops
      namespace: flux-system
    spec:
      interval: 1m
      url: ${var.git_repo_url}
      ref:
        branch: ${var.git_repo_branch}
  YAML
}

# ==========================================
# Kustomization for CRDs (applied first)
# ==========================================
resource "kubectl_manifest" "kustomization_crds" {
  depends_on = [kubectl_manifest.git_repo]

  yaml_body = <<-YAML
    apiVersion: kustomize.toolkit.fluxcd.io/v1
    kind: Kustomization
    metadata:
      name: releases-crds
      namespace: flux-system
    spec:
      interval: 2m
      sourceRef:
        kind: GitRepository
        name: mcp-server-gitops
      path: ./${var.releases_path}/crds
      prune: true
      wait: true
  YAML
}

# ==========================================
# Kustomization for releases (depends on CRDs)
# ==========================================
resource "kubectl_manifest" "kustomization_releases" {
  depends_on = [kubectl_manifest.kustomization_crds]

  yaml_body = <<-YAML
    apiVersion: kustomize.toolkit.fluxcd.io/v1
    kind: Kustomization
    metadata:
      name: releases
      namespace: flux-system
    spec:
      interval: 2m
      dependsOn:
        - name: releases-crds
      sourceRef:
        kind: GitRepository
        name: mcp-server-gitops
      path: ./${var.releases_path}
      prune: true
      wait: true
      retryInterval: 30s
  YAML
}
