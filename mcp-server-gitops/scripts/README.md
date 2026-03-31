# Scripts

Helper scripts for bootstrapping the infrastructure.

## setup.sh

Main bootstrap script called by `make run`. It:

1. Checks prerequisites (terraform, kubectl, helm, kind)
2. Verifies `ANTHROPIC_API_KEY` environment variable is set
3. Runs `terraform init` in the bootstrap directory
4. Runs `terraform apply` with the API key variable

### Usage

```bash
export ANTHROPIC_API_KEY="your-key"
./scripts/setup.sh
```

Or via Makefile:

```bash
make run
```

---
*Back to [mcp-server-gitops](../)*
