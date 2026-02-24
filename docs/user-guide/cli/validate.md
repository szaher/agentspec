# validate

Validate an IntentLang file for syntax and semantic correctness.

## Usage

```bash
agentspec validate <file.ias>
```

## Description

The `validate` command parses an IntentLang file and checks it against the language specification. It reports syntax errors, undefined references, type mismatches, and other problems that would prevent successful planning or deployment.

By default, warnings are reported but do not cause a non-zero exit code. Use `--strict` to treat warnings as errors.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--strict` | | `false` | Treat warnings as errors and exit with code 1 |
| `--format` | | `text` | Output format: `json` or `text` |
| `--quiet` | `-q` | `false` | Suppress all output; rely on exit code only |

## Examples

```bash
# Validate a single spec file
agentspec validate agent.ias

# Validate with strict mode (fail on warnings)
agentspec validate --strict agent.ias

# Validate and output results as JSON for CI integration
agentspec validate --format json agent.ias
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | File is valid (no errors; warnings may be present unless `--strict`) |
| `1` | Validation failed with one or more errors |
