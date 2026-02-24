# logs

Stream or retrieve logs from running agents.

## Usage

```bash
agentspec logs [agent-name]
```

## Description

The `logs` command retrieves log output from deployed or locally running agents. When an agent name is provided, logs are filtered to that specific agent. Without a name, logs from all agents are shown.

Use `--follow` to stream logs in real time, similar to `tail -f`. The `--since` flag limits output to entries within a given time window, and `--tail` shows only the most recent entries.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--follow` | `-f` | `false` | Stream new log entries as they arrive |
| `--since` | | | Show logs since a duration ago (e.g., `5m`, `1h`, `24h`) |
| `--tail` | | | Show only the last N log entries |
| `--format` | | `text` | Output format: `json` or `text` |

## Examples

```bash
# Show all logs
agentspec logs

# Follow logs for a specific agent
agentspec logs -f my-agent

# Show the last 50 log entries from the past hour
agentspec logs --since 1h --tail 50
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Logs retrieved successfully |
| `1` | An error occurred (agent not found, no state file, etc.) |
