# Basic Agent

The simplest possible IntentLang AgentSpec: a single agent with a system prompt and one binding.

## What This Demonstrates

- **Package declaration** with name, version, and language version
- **Prompt** resource defining the agent's system instructions
- **Agent** resource that references the prompt and specifies a model
- **Deploy target** specifying the default deployment platform

## AgentSpec Structure

```
package "basic-agent" version "0.1.0" lang "2.0"
```

Every `.ias` file starts with a package header. The `lang "2.0"` field pins the IntentLang language version for forward compatibility.

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
deploy "local" target "process" {
  default true
}
```

A deploy target declares where the agent is deployed. The `default true` flag means `agentspec plan` and `agentspec apply` use this target when no `--target` flag is provided.

## How to Run

```bash
# Format
./agentspec fmt examples/basic-agent.ias

# Validate
./agentspec validate examples/basic-agent.ias

# Plan (shows 3 resources: Prompt, Agent, DeployTarget)
./agentspec plan examples/basic-agent.ias

# Apply
./agentspec apply examples/basic-agent.ias --auto-approve

# Verify idempotency
./agentspec apply examples/basic-agent.ias --auto-approve
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
