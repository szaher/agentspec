# status

Show the current state of deployed agents.

## Usage

```bash
agentspec status
```

## Description

The `status` command reads the state file and displays the current status of all agents and resources managed by AgentSpec. It shows which agents are deployed, their health, and the last time each was updated.

Use `--watch` to continuously refresh the status display, which is useful for monitoring deployments in progress.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | | `text` | Output format: `json` or `text` |
| `--watch` | | `false` | Continuously refresh the status display |

## Examples

```bash
# Show status of all deployed agents
agentspec status

# Output status as JSON for scripting
agentspec status --format json

# Watch status updates in real time
agentspec status --watch
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Status retrieved successfully |
| `1` | An error occurred (state file not found, etc.) |
