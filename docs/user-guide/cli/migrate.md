# migrate

Migrate specs and state files to newer formats.

## Usage

```bash
agentspec migrate [path]
```

## Description

The `migrate` command updates IntentLang spec files and state files to the latest format. This includes renaming legacy file extensions (`.az` to `.ias`), updating state file names (`.agentz.state.json` to `.agentspec.state.json`), and rewriting spec syntax to newer language versions.

If no path is provided, the command operates on the current directory. Use `--dry-run` to preview what changes would be made without actually modifying any files.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--to-v2` | | `false` | Rewrite spec files to IntentLang 2.0 syntax |
| `--dry-run` | | `false` | Show what changes would be made without applying them |

## Examples

```bash
# Migrate files in the current directory
agentspec migrate

# Preview migration changes without modifying files
agentspec migrate --dry-run ./my-project

# Migrate specs to IntentLang 2.0 syntax
agentspec migrate --to-v2 ./my-project
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Migration completed successfully (or dry run finished) |
| `1` | An error occurred during migration |
