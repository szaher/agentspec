# Agentz Examples

This directory contains example `.az` definition files demonstrating the features of the Agentz DSL. Each example is self-contained and can be validated, planned, and applied independently.

## Prerequisites

Build the `agentz` CLI from the repository root:

```bash
go build -o agentz ./cmd/agentz
```

Verify the build:

```bash
./agentz version
# agentz version 0.1.0 (lang 1.0, ir 1.0)
```

## Examples

| Example | File | Key Concepts |
|---------|------|--------------|
| [Basic Agent](basic-agent/) | `basic-agent.az` | Minimal agent definition, prompt, binding |
| [Multi-Skill Agent](multi-skill-agent/) | `multi-skill-agent.az` | Multiple skills with input/output schemas |
| [MCP Server/Client](mcp-server-client/) | `mcp-server-client.az` | MCP transport, server/client connectivity |
| [Multi-Environment](multi-environment/) | `multi-environment.az` | Environment overlays (dev/prod) |
| [Plugin Usage](plugin-usage/) | `plugin-usage.az` | WASM plugin references |
| [Multi-Binding](multi-binding/) | `multi-binding.az` | Deploying to multiple adapters |
| [Customer Support](customer-support/) | `customer-support.az` | Secrets, environments, multi-skill agent |
| [Code Review Pipeline](code-review-pipeline/) | `code-review-pipeline.az` | Multi-agent collaboration, MCP, dual bindings |
| [Data Pipeline](data-pipeline/) | `data-pipeline.az` | Policies, secrets, three environments |
| [RAG Chatbot](rag-chatbot/) | `rag-chatbot.az` | Vector search, MCP transport, secrets |

## Running Any Example

Every example follows the same workflow. From the repository root:

```bash
# 1. Format the definition (canonical output)
./agentz fmt examples/<name>/<name>.az

# 2. Validate the definition
./agentz validate examples/<name>/<name>.az

# 3. Preview what will change
./agentz plan examples/<name>/<name>.az

# 4. Apply the changes
./agentz apply examples/<name>/<name>.az --auto-approve

# 5. Verify idempotency (should report no changes)
./agentz apply examples/<name>/<name>.az --auto-approve

# 6. Export artifacts
./agentz export examples/<name>/<name>.az --out-dir ./output
```

For example, to run the basic-agent example:

```bash
./agentz validate examples/basic-agent/basic-agent.az
./agentz plan examples/basic-agent/basic-agent.az
./agentz apply examples/basic-agent/basic-agent.az --auto-approve
```

For examples with environment overlays, add the `--env` flag:

```bash
./agentz plan examples/multi-environment/multi-environment.az --env dev
./agentz plan examples/multi-environment/multi-environment.az --env prod
```

For examples with multiple bindings, use the `--target` flag:

```bash
./agentz export examples/multi-binding/multi-binding.az --target compose --out-dir ./output
```

## Important Notes

- **State file**: Each `apply` writes to `.agentz.state.json` in the current directory. Use `--state-file` to specify a different location if running multiple examples.
- **Adapters**: The `local-mcp` adapter produces JSON manifests (agents, servers, clients). The `docker-compose` adapter produces `docker-compose.yml` and supporting config files.
- **Secrets**: Examples that declare `secret` blocks expect environment variables to be set. The tool validates that secrets are referenced correctly but does not read their values at plan/apply time.
- **Plugins**: The `plugin-usage` example references a WASM plugin. The plugin manifest must be present in `./plugins/` or `~/.agentz/plugins/` for full plugin functionality.
- **Determinism**: All outputs (IR, plan, export artifacts) are deterministic. Running the same command twice produces byte-identical results.
- **Cleanup**: To reset state and start fresh, delete `.agentz.state.json` from the working directory.
