# diff

Show differences between an IntentLang spec and the current deployed state.

## Usage

```bash
agentspec diff <file.ias>
```

## Description

The `diff` command compares the desired state defined in an IntentLang spec file against the actual state recorded in the state file. It highlights what has changed, what would be added, and what would be removed if `apply` were run.

Unlike `plan`, which produces a full execution plan, `diff` focuses on presenting a concise, human-readable summary of the differences. This is useful for quick inspections before committing spec changes or running a plan.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | | `text` | Output format: `text` or `json` |
| `--color` | | `auto` | Force colored output even when not writing to a terminal |

## Examples

```bash
# Show differences in text format
agentspec diff agent.ias

# Output differences as JSON
agentspec diff --format json agent.ias

# Force colored output when piping to less
agentspec diff --color agent.ias | less -R
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Diff computed successfully (differences may or may not exist) |
| `1` | An error occurred (invalid spec, missing state file, etc.) |
