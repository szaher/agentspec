# fmt

Format an IntentLang file to canonical style.

## Usage

```bash
agentspec fmt <file.ias>
```

## Description

The `fmt` command rewrites an IntentLang file using consistent indentation, key ordering, and whitespace rules. This ensures that all `.ias` files in a project follow the same formatting conventions, making diffs cleaner and code reviews easier.

By default, the formatted output is printed to stdout. Use `--write` to modify the file in place, or `--check` to verify formatting without changing anything.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--check` | | `false` | Check if the file is already formatted; exit 1 if not |
| `--write` | `-w` | `false` | Write the formatted result back to the file in place |
| `--diff` | | `false` | Show a unified diff of formatting changes |

## Examples

```bash
# Print formatted output to stdout
agentspec fmt agent.ias

# Format the file in place
agentspec fmt --write agent.ias

# Check formatting in CI (exits 1 if the file would change)
agentspec fmt --check agent.ias
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | File is already formatted, or formatting succeeded |
| `1` | File is not formatted (with `--check`), or an error occurred |
