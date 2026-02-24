# init

Scaffold a new IntentLang project.

## Usage

```bash
agentspec init [name]
```

## Description

The `init` command creates a new IntentLang project directory with starter files. If a name is provided, a directory with that name is created in the current working directory. If no name is given, the current directory is initialized.

The generated project includes a sample `.ias` spec file, a `.agentspec.state.json` state file, and a `.gitignore` configured for AgentSpec projects. Use `--template` to choose from different starter templates depending on your use case.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--template` | | `basic` | Starter template: `basic`, `pipeline`, or `multi-agent` |

## Examples

```bash
# Create a new project with the default template
agentspec init my-agent

# Initialize the current directory with a pipeline template
agentspec init --template pipeline

# Scaffold a multi-agent project
agentspec init --template multi-agent my-project
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Project created successfully |
| `1` | An error occurred (directory already exists, etc.) |
