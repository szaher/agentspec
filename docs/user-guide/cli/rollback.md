# rollback

Rollback an agent to its previous version.

## Usage

```bash
agentspec rollback --agent <name>
```

## Description

The `rollback` command restores an agent to its previous version by loading the version history from the state file and creating a new version entry that contains the previous version's snapshot.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--agent` | | *(required)* | Name of the agent to rollback |
| `--state` | | `.agentspec.state.json` | Path to the state file |

## Examples

```bash
# Rollback the assistant agent
agentspec rollback --agent assistant

# Rollback using a custom state file
agentspec rollback --agent assistant --state /path/to/state.json
```

## Behavior

- Requires at least two versions in history (current + previous)
- Creates a new version entry (incremented version number) with the previous version's snapshot
- The rollback itself is recorded in the version history with a summary like "Rollback to version N"
- Maximum of 10 versions are retained in history
