# Basic Agent

The simplest possible Agentz definition: a single agent with a system prompt and one binding.

## What This Demonstrates

- **Package declaration** with name, version, and language version
- **Prompt** resource defining the agent's system instructions
- **Agent** resource that references the prompt and specifies a model
- **Binding** to the `local-mcp` adapter as the default deployment target

## Definition Structure

```
package "basic-agent" version "0.1.0" lang "1.0"
```

Every `.az` file starts with a package header. The `lang "1.0"` field pins the DSL language version for forward compatibility.

```
prompt "system" {
  content "You are a helpful assistant."
}
```

A prompt defines reusable instructions. Agents reference prompts by name with `uses prompt`.

```
agent "helper" {
  uses prompt "system"
  model "claude-sonnet-4-20250514"
}
```

The agent ties together a prompt and a model. The `uses` keyword creates a reference — the validator checks that `"system"` exists as a prompt and will suggest corrections if you mistype the name.

```
binding "local" adapter "local-mcp" {
  default true
}
```

A binding declares where the agent is deployed. The `default true` flag means `agentz plan` and `agentz apply` use this binding when no `--target` flag is provided.

## How to Run

```bash
# Format
./agentz fmt examples/basic-agent.az

# Validate
./agentz validate examples/basic-agent.az

# Plan (shows 3 resources: Prompt, Agent, Binding)
./agentz plan examples/basic-agent.az

# Apply
./agentz apply examples/basic-agent.az --auto-approve

# Verify idempotency
./agentz apply examples/basic-agent.az --auto-approve
# Output: No changes. Infrastructure is up-to-date.
```

## Resources Created

| Kind | Name | Description |
|------|------|-------------|
| Prompt | system | System instructions for the agent |
| Agent | helper | The agent itself |

## Next Steps

- Add skills to the agent: see [multi-skill-agent](../multi-skill-agent/)
- Deploy to multiple platforms: see [multi-binding](../multi-binding/)
- Add environment overlays: see [multi-environment](../multi-environment/)
