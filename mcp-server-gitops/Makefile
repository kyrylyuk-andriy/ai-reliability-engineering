help:
	@echo "Available targets:"
	@echo "  run    - Bootstrap the full environment (install tools, provision cluster)"
	@echo "  down   - Destroy the cluster and all resources"
	@echo "  tools  - Install Terraform and k9s only"
	@echo "  init   - Initialize Terraform"
	@echo "  apply  - Apply Terraform configuration"

run:
	@bash scripts/setup.sh

tools:
	@brew install terraform k9s

init:
	@cd bootstrap && terraform init

apply:
	@cd bootstrap && terraform apply -auto-approve

down:
	@cd bootstrap && terraform destroy -auto-approve
