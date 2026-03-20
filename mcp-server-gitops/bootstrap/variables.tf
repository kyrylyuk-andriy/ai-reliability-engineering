variable "cluster_name" {
  description = "Cluster Name"
  type        = string
  default     = "mcp-gitops"
}

variable "git_repo_url" {
  description = "Git repository URL for Flux to sync from"
  type        = string
  default     = "https://github.com/kyrylyuk-andriy/ai-reliability-engineering"
}

variable "git_repo_branch" {
  description = "Git branch to sync"
  type        = string
  default     = "main"
}

variable "releases_path" {
  description = "Path in the Git repo to the releases directory"
  type        = string
  default     = "mcp-server-gitops/releases"
}

variable "anthropic_api_key" {
  description = "Anthropic API key for kagent"
  type        = string
  sensitive   = true
  default     = ""
}
