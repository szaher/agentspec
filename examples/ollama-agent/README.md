# Ollama Agent Example

A local coding assistant that runs entirely on your machine via [Ollama](https://ollama.com). No API keys required — no data leaves your network.

## What's included

| Agent | Purpose | Skills |
|-------|---------|--------|
| `coder` | Coding assistant with file and shell access | `list-files`, `read-file`, `run-command`, `get-time` |
| `reviewer` | Code reviewer for bugs, security, and style | `read-file` |

Both agents use `ollama/llama3.2` by default. Change the `model` line to use any Ollama model:

```
model "ollama/codellama:7b"
model "ollama/mistral"
model "ollama/deepseek-coder:6.7b"
model "ollama/qwen2.5-coder:7b"
```

## Prerequisites

1. Install Ollama: https://ollama.com/download
2. Pull a model:

```bash
ollama pull llama3.2
```

## Run

```bash
# Build the CLI (from repo root)
go build -o agentspec ./cmd/agentspec

# Start the dev server with hot reload + web UI
./agentspec dev examples/ollama-agent/ollama-agent.ias

# Open the chat UI
open http://localhost:8080
```

The web UI lets you switch between the `coder` and `reviewer` agents using the dropdown.

## One-shot usage

```bash
# Ask the coder agent
./agentspec run examples/ollama-agent/ollama-agent.ias \
  --agent coder \
  --input "What files are in the current directory?" \
  --stream

# Ask the reviewer agent
./agentspec run examples/ollama-agent/ollama-agent.ias \
  --agent reviewer \
  --input "Review this Python function: def add(a, b): return a + b"
```

## Custom Ollama host

If Ollama runs on a different machine or port, set `OLLAMA_HOST`:

```bash
OLLAMA_HOST=http://192.168.1.100:11434 ./agentspec dev examples/ollama-agent/ollama-agent.ias
```

## Validate and plan

```bash
./agentspec validate examples/ollama-agent/ollama-agent.ias
./agentspec plan examples/ollama-agent/ollama-agent.ias
```

## Model string format

AgentSpec auto-detects the provider from the model string:

| Format | Provider | Example |
|--------|----------|---------|
| `ollama/<model>` | Ollama (local) | `ollama/llama3.2` |
| `openai/<model>` | OpenAI | `openai/gpt-4o` |
| `anthropic/<model>` | Anthropic | `anthropic/claude-sonnet-4-20250514` |
| `claude-*` | Anthropic (auto) | `claude-sonnet-4-20250514` |
| `gpt-*` | OpenAI (auto) | `gpt-4o` |
