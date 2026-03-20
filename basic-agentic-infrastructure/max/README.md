# Max Agentic Infrastructure

This module builds on the [experienced setup](../experienced/README.md) by using the kagent Gateway API integration instead of standalone AgentGateway Helm charts.

Reference: https://agentgateway.dev/docs/kubernetes/main/

---

## Plan

1. Deploy AgentGateway using kagent's built-in Gateway API integration
2. Configure Secrets and model routes via Gateway API resources
3. Deploy kagent with AgentGateway as the LLM proxy
4. Verify built-in agents work end-to-end through the gateway
