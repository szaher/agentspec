# plan

Preview the changes that would be applied from an IntentLang spec.

## Usage

```bash
agentspec plan <file.ias>
```

## Description

The `plan` command compares the desired state described in an IntentLang file against the current state recorded in the state file. It produces a detailed execution plan showing which resources will be created, updated, or destroyed.

No changes are made to the target environment. The plan output can be saved to a file and later passed to `agentspec apply --plan-file` for a controlled deployment workflow.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--out` | | | Save the plan to a file for later use with `apply --plan-file` |
| `--format` | | `text` | Output format: `json` or `text` |
| `--target` | | | Limit the plan to a specific deploy target |
| `--env` | | | Set the environment name (e.g., `staging`, `production`) |

## Examples

```bash
# Preview changes for a spec
agentspec plan agent.ias

# Save the plan to a file for review before applying
agentspec plan --out plan.json agent.ias

# Plan changes for a specific target and environment
agentspec plan --target cloud --env staging agent.ias
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Plan generated successfully |
| `1` | An error occurred (invalid spec, state file not found, etc.) |
