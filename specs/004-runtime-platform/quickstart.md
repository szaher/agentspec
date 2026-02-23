# Quickstart: AgentSpec Runtime Platform

**Branch**: `004-runtime-platform` | **Date**: 2026-02-23

## Prerequisites

- Go 1.25+ installed
- Anthropic API key (set `ANTHROPIC_API_KEY` environment variable)
- Docker (optional, for container deployments)
- kubectl configured (optional, for Kubernetes deployments)

## 1. Build the CLI

```bash
go build -o agentspec ./cmd/agentspec
```

## 2. Create an Agent Definition

Create a file `my-bot.ias`:

```
package "my-bot" version "1.0.0" lang "2.0"

prompt "system" {
  content "You are a helpful assistant that can search the web."
}

skill "web-search" {
  description "Search the web for information"
  input { query string required }
  output { results string }
  tool mcp "search-server/search"
}

agent "assistant" {
  uses prompt "system"
  uses skill "web-search"
  model "claude-sonnet-4-20250514"
  max_turns 5
  strategy "react"
}

deploy "local" target "process" {
  port 8080
  default true
}
```

## 3. Validate

```bash
./agentspec validate my-bot.ias
```

## 4. Plan

```bash
./agentspec plan my-bot.ias
```

## 5. Apply (Start the Agent)

```bash
export ANTHROPIC_API_KEY="your-key-here"
./agentspec apply my-bot.ias
```

The runtime starts and reports:

```
✓ Runtime started on http://localhost:8080
✓ Agent "assistant" ready at /v1/agents/assistant/invoke
```

## 6. Invoke the Agent

```bash
curl -X POST http://localhost:8080/v1/agents/assistant/invoke \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the capital of France?"}'
```

## 7. One-Shot Invocation (Alternative)

```bash
./agentspec run assistant --input "What is the capital of France?" my-bot.ias
```

## 8. Development Mode

```bash
./agentspec dev my-bot.ias
```

Hot-reloads on `.ias` file changes. Use for iterative development.

## 9. Deploy to Docker

```bash
./agentspec apply my-bot.ias --target staging
```

(Requires a `deploy "staging" target "docker" { ... }` block in the `.ias` file.)

## 10. Check Status

```bash
./agentspec status
./agentspec logs assistant
./agentspec destroy
```
