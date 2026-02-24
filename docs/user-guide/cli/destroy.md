# destroy

Tear down all resources created by an IntentLang spec.

## Usage

```bash
agentspec destroy <file.ias>
```

## Description

The `destroy` command removes all resources that were previously created by applying the given spec file. It reads the state file to determine what exists, plans the deletions, and prompts for confirmation before proceeding.

This is a destructive operation. Use `--force` to skip the confirmation prompt. If the spec defines multiple deploy targets, use `--target` to limit destruction to a single target.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | | `false` | Skip the interactive confirmation prompt |
| `--target` | | | Limit destruction to a specific deploy target |

## Examples

```bash
# Destroy resources with confirmation prompt
agentspec destroy agent.ias

# Force destroy without confirmation
agentspec destroy --force agent.ias

# Destroy only resources for a specific target
agentspec destroy --target staging agent.ias
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All resources destroyed successfully |
| `1` | An error occurred during destruction |
