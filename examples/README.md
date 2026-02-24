# Agentz Examples

This directory contains example `.ias` AgentSpec files demonstrating the features of IntentLang. Each example is self-contained and can be validated, planned, and applied independently.

## Prerequisites

Build the `agentspec` CLI from the repository root:

```bash
go build -o agentspec ./cmd/agentspec
```

Verify the build:

```bash
./agentspec version
# agentspec version 0.1.0 (lang 2.0, ir 1.0)
```

## Examples

| Example | File | Key Concepts |
|---------|------|--------------|
| [Basic Agent](basic-agent/) | `basic-agent.ias` | Minimal AgentSpec, prompt, deploy target |
| [Multi-Skill Agent](multi-skill-agent/) | `multi-skill-agent.ias` | Multiple skills with input/output schemas |
| [MCP Server/Client](mcp-server-client/) | `mcp-server-client.ias` | MCP transport, server/client connectivity |
| [Multi-Environment](multi-environment/) | `multi-environment.ias` | Environment overlays (dev/prod) |
| [Plugin Usage](plugin-usage/) | `plugin-usage.ias` | WASM plugin references |
| [Multi-Binding](multi-binding/) | `multi-binding.ias` | Deploying to multiple targets |
| [Customer Support](customer-support/) | `customer-support.ias` | Secrets, environments, multi-skill agent |
| [Code Review Pipeline](code-review-pipeline/) | `code-review-pipeline.ias` | Multi-agent collaboration, MCP, dual deploy targets |
| [Data Pipeline](data-pipeline/) | `data-pipeline.ias` | Policies, secrets, three environments |
| [RAG Chatbot](rag-chatbot/) | `rag-chatbot.ias` | Vector search, MCP transport, secrets |

## Running Any Example

Every example follows the same workflow. From the repository root:

```bash
# 1. Format the AgentSpec (canonical output)
./agentspec fmt examples/<name>/<name>.ias

# 2. Validate the AgentSpec
./agentspec validate examples/<name>/<name>.ias

# 3. Preview what will change
./agentspec plan examples/<name>/<name>.ias

# 4. Apply the changes
./agentspec apply examples/<name>/<name>.ias --auto-approve

# 5. Verify idempotency (should report no changes)
./agentspec apply examples/<name>/<name>.ias --auto-approve

# 6. Export artifacts
./agentspec export examples/<name>/<name>.ias --out-dir ./output
```

For example, to run the basic-agent example:

```bash
./agentspec validate examples/basic-agent/basic-agent.ias
./agentspec plan examples/basic-agent/basic-agent.ias
./agentspec apply examples/basic-agent/basic-agent.ias --auto-approve
```

For examples with environment overlays, add the `--env` flag:

```bash
./agentspec plan examples/multi-environment/multi-environment.ias --env dev
./agentspec plan examples/multi-environment/multi-environment.ias --env prod
```

For examples with multiple bindings, use the `--target` flag:

```bash
./agentspec export examples/multi-binding/multi-binding.ias --target compose --out-dir ./output
```

## Important Notes

- **State file**: Each `apply` writes to `.agentspec.state.json` in the current directory. Use `--state-file` to specify a different location if running multiple examples.
- **Deploy targets**: The `process` target produces JSON manifests (agents, servers, clients). The `docker-compose` target produces `docker-compose.yml` and supporting config files.
- **Secrets**: Examples that declare `secret` blocks expect environment variables to be set. The tool validates that secrets are referenced correctly but does not read their values at plan/apply time.
- **Plugins**: The `plugin-usage` example references a WASM plugin. The plugin manifest must be present in `./plugins/` or `~/.agentspec/plugins/` for full plugin functionality.
- **Determinism**: All outputs (IR, plan, export artifacts) are deterministic. Running the same command twice produces byte-identical results.
- **Cleanup**: To reset state and start fresh, delete `.agentspec.state.json` from the working directory.
