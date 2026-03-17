# history

Show version history for an agent.

## Usage

```bash
agentspec history --agent <name>
```

## Description

The `history` command displays the version history of an agent as a table showing version number, timestamp, and change summary.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--agent` | | *(required)* | Name of the agent to show history for |
| `--state` | | `.agentspec.state.json` | Path to the state file |

## Examples

```bash
# Show history for the assistant agent
agentspec history --agent assistant
```

## Output

```
VERSION  TIMESTAMP                  SUMMARY
1        2026-03-15T10:30:00Z       Applied version 1
2        2026-03-16T14:20:00Z       Applied version 2
3        2026-03-17T09:00:00Z       Rollback to version 1
```
