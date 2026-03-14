# Basic Agentic Infrastructure

This module covers the foundational infrastructure patterns for building reliable AI agent systems. The first task is setting up [AgentGateway](https://agentgateway.dev/) — an open-source agentic proxy that connects, secures, and observes agent-to-agent and agent-to-tool communication.

---

## AgentGateway Local Installation

Reference: https://agentgateway.dev/docs/standalone/latest/deployment/binary/

---

### Basic

Minimal steps to get AgentGateway running locally.

#### 1. Install the binary

```bash
curl -sL https://agentgateway.dev/install | bash
```

You may be prompted for your password to install to `/usr/local/bin`.

**Output:**

```
Downloading https://github.com/agentgateway/agentgateway/releases/download/v1.0.0-rc.1/agentgateway-darwin-arm64
Verifying checksum... Done.
Preparing to install agentgateway into /usr/local/bin
agentgateway installed into /usr/local/bin/agentgateway
```

#### 2. Verify installation

```bash
agentgateway --version
```

**Output:**

```json
{
  "version": "1.0.0-rc.1",
  "git_revision": "93be096804b37c7fc7836e8001d92a0b3abc33ac",
  "rust_version": "1.93.1",
  "build_profile": "release",
  "build_target": "aarch64-apple-darwin"
}
```

#### 3. Create a config file

Create `config.yaml`:

```yaml
# yaml-language-server: $schema=https://agentgateway.dev/schema/config
binds:
- port: 3000
  listeners:
  - routes:
    - policies:
        cors:
          allowOrigins:
          - "*"
          allowHeaders:
          - mcp-protocol-version
          - content-type
          - cache-control
          exposeHeaders:
          - "Mcp-Session-Id"
      backends:
      - mcp:
          targets:
          - name: everything
            stdio:
              cmd: npx
              args: ["@modelcontextprotocol/server-everything"]
```

#### 4. Run it

```bash
agentgateway -f basic/config.yaml
```

**Output:**

```
2026-03-14T00:01:47.514465Z    info    agentgateway_app::commands::run    version: {
  "version": "1.0.0-rc.1",
  "git_revision": "93be096804b37c7fc7836e8001d92a0b3abc33ac",
  "rust_version": "1.93.1",
  "build_profile": "release",
  "build_target": "aarch64-apple-darwin"
}
2026-03-14T00:01:47.525859Z    info    state_manager    Watching config file: basic/config.yaml
2026-03-14T00:01:47.531525Z    info    state_manager    loaded config from File("basic/config.yaml")
2026-03-14T00:01:47.534185Z    info    app    serving UI at http://localhost:15000/ui
2026-03-14T00:01:47.534516Z    info    proxy::gateway    started bind    bind="bind/3000"
2026-03-14T00:01:47.534735Z    info    agent_core::readiness    Task 'state manager' complete (21.670084ms), marking server ready
```

#### 5. Verify it works

- Open the UI: http://localhost:15000/ui
- The gateway listens on port 3000

---

## Configure Anthropic as LLM Provider

Reference: https://agentgateway.dev/docs/standalone/latest/llm/providers/anthropic/

#### 1. Set your API key

Get your key from the [Anthropic Console](https://console.anthropic.com/) and export it:

```bash
export ANTHROPIC_API_KEY="your-api-key-here"
```

#### Model Options

| Model | Best for | Trade-off |
|-------|----------|-----------|
| `claude-haiku-4-5-20251001` | Testing, high-throughput, low-cost tasks | Less capable on complex reasoning |
| `claude-sonnet-4-6-20250627` | Good balance of cost/capability | Mid-tier pricing |
| `claude-opus-4-6-20250627` | Complex reasoning, agentic workflows | Slowest, most expensive |

#### 2. Update config.yaml

Replace the config with the Anthropic LLM provider setup:

```yaml
# yaml-language-server: $schema=https://agentgateway.dev/schema/config
binds:
- port: 3000
  listeners:
  - routes:
    - backends:
      - ai:
          name: anthropic
          provider:
            anthropic:
              model: claude-haiku-4-5-20251001
          policies:
            backendAuth:
              key: "$ANTHROPIC_API_KEY"
      policies:
        ai:
          routes:
            /v1/messages: messages
            /v1/chat/completions: completions
            /v1/models: passthrough
            "*": passthrough
```

**Note:** There are two separate `policies` sections:
- **`ai.policies.backendAuth`** — authenticates outbound requests to Anthropic
- **`route.policies.ai.routes`** — tells the gateway how to handle each URL pattern

#### 3. Run AgentGateway

```bash
agentgateway -f config.yaml
```

#### 4. Test it

Send a request through the gateway using the native Anthropic messages format:

```bash
curl -X POST http://localhost:3000/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-haiku-4-5-20251001",
    "max_tokens": 128,
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

**Expected response:**

```json
{
  "id": "msg_01QfjTB3JKDZNp8C81A8w1US",
  "type": "message",
  "role": "assistant",
  "model": "claude-haiku-4-5-20251001",
  "stop_reason": "end_turn",
  "stop_sequence": null,
  "usage": {
    "input_tokens": 9,
    "output_tokens": 16
  },
  "content": [
    {
      "text": "Hello! 👋 How can I help you today?",
      "type": "text"
    }
  ]
}
```

#### 5. (Optional) Connect Claude Code through the gateway

```bash
export ANTHROPIC_BASE_URL="http://localhost:3000"
claude
```

---

## Configure Google Gemini as LLM Provider

Reference: https://agentgateway.dev/docs/standalone/latest/llm/providers/gemini/

#### 1. Set your API key

Get your key from [Google AI Studio](https://aistudio.google.com/api-keys) and export it:

```bash
export GEMINI_API_KEY="your-api-key-here"
```

> **Billing note:** The free tier has very limited quota. If you get a `RESOURCE_EXHAUSTED` error, you need to [enable billing](https://ai.google.dev/gemini-api/docs/rate-limits) on your Google AI Studio account.

#### 2. Add Gemini bind to config.yaml

Add a second bind for Gemini on port 3001:

```yaml
- port: 3001
  listeners:
  - routes:
    - backends:
      - ai:
          name: gemini
          provider:
            gemini:
              model: gemini-2.0-flash
          policies:
            backendAuth:
              key: "$GEMINI_API_KEY"
      policies:
        ai:
          routes:
            /v1/chat/completions: completions
            /v1/models: passthrough
            "*": passthrough
```

#### 3. Test it

```bash
curl -X POST http://localhost:3001/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.0-flash",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

**Example error response (free tier without billing):**

```json
{
  "error": {
    "type": "rate_limit_error",
    "message": "You exceeded your current quota, please check your plan and billing details.",
    "code": "RESOURCE_EXHAUSTED"
  }
}
```

This confirms the gateway is proxying requests to Gemini correctly — the error comes from Google's API, not the gateway.
