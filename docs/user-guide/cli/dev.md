# dev

Start a local development server with live reload.

## Usage

```bash
agentspec dev <file.ias>
```

## Description

The `dev` command launches a development server that loads the agent defined in the spec file and exposes it over HTTP. This provides a rapid feedback loop during authoring: you can send requests to the agent and see results immediately.

With `--watch` enabled, the server monitors the spec file for changes and reloads the agent automatically. The `--hot-reload` flag goes further by preserving session state across reloads so you can iterate without losing conversation context.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--watch` | | `false` | Watch the spec file for changes and reload automatically |
| `--port` | | `3000` | Port for the local development server |
| `--hot-reload` | | `false` | Preserve session state across reloads |

## Examples

```bash
# Start the dev server on the default port
agentspec dev agent.ias

# Start with file watching and hot reload
agentspec dev --watch --hot-reload agent.ias

# Use a custom port
agentspec dev --port 9090 --watch agent.ias
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Server shut down cleanly |
| `1` | An error occurred (port conflict, invalid spec, etc.) |
