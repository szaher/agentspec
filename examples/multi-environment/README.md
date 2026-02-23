# Multi-Environment

Maintain a single AgentSpec with environment-specific overrides for dev and prod.

## What This Demonstrates

- **Environment overlays** that override specific resource attributes
- **Base AgentSpec inheritance** — unspecified attributes carry over from the base
- **Per-environment planning** with the `--env` flag

## AgentSpec Structure

### Base AgentSpec

```
agent "assistant" {
  uses prompt "greeting"
  uses skill "search"
  model "claude-sonnet-4-20250514"
}
```

The base AgentSpec is the source of truth. All attributes are set here.

### Environment Overlays

```
environment "dev" {
  agent "assistant" {
    model "claude-haiku-latest"
  }
}

environment "prod" {
  agent "assistant" {
    model "claude-sonnet-4-20250514"
  }
}
```

Each environment block targets a resource by name and overrides specific attributes. Attributes not mentioned in the overlay are inherited from the base. In this example:
- **dev** uses `claude-haiku-latest` (cheaper, faster for development)
- **prod** uses `claude-sonnet-4-20250514` (higher quality for production)

The validator detects conflicting overlays — for example, if two environments tried to set the same attribute on the same resource to different values within a single environment block.

## How to Run

```bash
# Validate (checks both base and environment overlays)
./agentspec validate examples/multi-environment.ias

# Plan for dev (shows haiku model)
./agentspec plan examples/multi-environment.ias --env dev

# Plan for prod (shows sonnet model)
./agentspec plan examples/multi-environment.ias --env prod

# Plan without --env (uses base AgentSpec, no overlay applied)
./agentspec plan examples/multi-environment.ias

# Apply for a specific environment
./agentspec apply examples/multi-environment.ias --env dev --auto-approve
```

## Resources Created

| Kind | Name | Description |
|------|------|-------------|
| Prompt | greeting | Shared prompt (same across environments) |
| Skill | search | Shared skill (same across environments) |
| Agent | assistant | Model varies by environment |

## How Overlays Work

1. The base AgentSpec is parsed and lowered to IR
2. When `--env dev` is specified, the environment overlay is applied
3. The overlay finds the target resource (`agent "assistant"`) and replaces the specified attribute (`model`)
4. A new content hash is computed for the modified resource
5. Environment resources themselves are filtered out of the final IR (they are metadata, not deployable)

This means the plan for `--env dev` and `--env prod` will show different content hashes for the agent, even though the underlying AgentSpec is the same file.

## Next Steps

- Combine environments with secrets: see [customer-support](../customer-support/)
- Add policy enforcement per environment: see [data-pipeline](../data-pipeline/)
