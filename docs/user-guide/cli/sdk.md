# sdk

Generate typed SDK bindings from an IntentLang spec.

## Usage

```bash
agentspec sdk <file.ias>
```

## Description

The `sdk` command reads an IntentLang spec file and generates typed client code for interacting with the defined agent. The generated SDK includes request and response types, client initialization, and method stubs for each capability exposed by the agent.

Supported target languages are Python, TypeScript, and Go. The generated code is written to the specified output directory, or to stdout if no directory is given.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--language` | | | Target language: `python`, `typescript`, or `go` |
| `--output` | `-o` | | Output directory for generated files |

## Examples

```bash
# Generate a Python SDK to stdout
agentspec sdk --language python agent.ias

# Generate a TypeScript SDK into a directory
agentspec sdk --language typescript -o ./sdk agent.ias

# Generate a Go client package
agentspec sdk --language go -o ./pkg/client agent.ias
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | SDK generated successfully |
| `1` | An error occurred (unsupported language, invalid spec, etc.) |
