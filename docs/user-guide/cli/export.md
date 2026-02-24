# export

Export an IntentLang spec to JSON or YAML.

## Usage

```bash
agentspec export <file.ias>
```

## Description

The `export` command converts an IntentLang spec file into a structured data format such as JSON or YAML. This is useful for integrating with external tools, generating configuration for other systems, or inspecting the fully resolved spec structure.

The exported output includes all resolved values, defaults, and computed fields. Use `--target` to export the configuration for a specific deploy target only.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | | `json` | Output format: `json` or `yaml` |
| `--output` | `-o` | | Write output to a directory instead of stdout |
| `--target` | | | Export configuration for a specific deploy target |

## Examples

```bash
# Export a spec as JSON to stdout
agentspec export agent.ias

# Export as YAML to a directory
agentspec export --format yaml -o ./output agent.ias

# Export configuration for a specific target
agentspec export --target production agent.ias
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Export completed successfully |
| `1` | An error occurred (invalid spec, write failure, etc.) |
