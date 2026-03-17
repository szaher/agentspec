# run

Start the agent runtime server with hot reload and built-in web UI.

## Usage

```bash
agentspec run <file.ias>
```

## Description

The `run` command starts an HTTP server that hosts the agents defined in the given spec file. It watches `.ias` files for changes and automatically restarts the runtime when modifications are detected. A built-in web UI is available for interactive testing.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--port` | | `8080` | HTTP server port |
| `--ui` | | `true` | Enable built-in web frontend |
| `--no-auth` | | `false` | Explicitly allow unauthenticated access (WARNING: insecure) |
| `--cors-origins` | | | Comma-separated list of allowed CORS origins |
| `--tls-cert` | | | Path to TLS certificate file (enables HTTPS) |
| `--tls-key` | | | Path to TLS private key file |
| `--audit-log` | | | Path to audit log file for invocation tracking |

## Examples

```bash
# Start the server with default settings
agentspec run agent.ias

# Start on a custom port with UI disabled
agentspec run --port 9090 --ui=false agent.ias

# Allow unauthenticated access for local testing
agentspec run --no-auth agent.ias
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Server shut down cleanly |
| `1` | An error occurred (port conflict, invalid spec, etc.) |

## See Also

- [CLI: dev](dev.md) -- One-shot agent invocation for quick testing
- [HTTP API Overview](../api/index.md) -- API exposed by the runtime server
- [Agent Runtime Configuration](../configuration/runtime.md) -- Configure strategies, timeouts, and streaming
