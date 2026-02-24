# apply

Apply the changes defined in an IntentLang spec to the target environment.

## Usage

```bash
agentspec apply <file.ias>
```

## Description

The `apply` command executes the changes needed to bring the target environment in line with the desired state described in the spec file. It creates, updates, or destroys resources as necessary and records the resulting state in the state file.

By default, `apply` displays the planned changes and prompts for confirmation before proceeding. Use `--auto-approve` to skip the confirmation prompt, which is useful in CI/CD pipelines. You can also pass a previously saved plan file with `--plan-file` to apply a reviewed plan exactly as generated.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--auto-approve` | | `false` | Skip the interactive confirmation prompt |
| `--plan-file` | | | Apply a previously saved plan file instead of computing a new plan |
| `--target` | | | Limit changes to a specific deploy target |
| `--env` | | | Set the environment name (e.g., `staging`, `production`) |

## Examples

```bash
# Apply changes with interactive confirmation
agentspec apply agent.ias

# Apply without confirmation (for CI/CD)
agentspec apply --auto-approve agent.ias

# Apply a previously saved plan
agentspec apply --plan-file plan.json
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All changes applied successfully |
| `1` | An error occurred during apply |

## See Also

- [Deployment Overview](../deployment/index.md) -- Guides for all deployment targets (process, Docker, Compose, Kubernetes)
- [CLI: plan](plan.md) -- Preview changes before applying
- [CLI: destroy](destroy.md) -- Remove deployed resources
- [Deploy Block Reference](../language/deploy.md) -- Full syntax for `deploy` blocks in IntentLang
