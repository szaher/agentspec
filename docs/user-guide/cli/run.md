# run

Execute an agent locally from an IntentLang spec file.

## Usage

```bash
agentspec run <file.ias>
```

## Description

The `run` command starts a local agent runtime and executes the agent defined in the given spec file. This is useful for testing agent behavior against real inputs without deploying to a remote environment.

You can pass input directly with `--input`, or the agent will read from stdin if no input is provided. Use `--stream` to see token-by-token output as the agent generates a response. Sessions can be resumed by passing a `--session` ID from a previous run.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--input` | | | Input text to send to the agent |
| `--stream` | | `false` | Stream output tokens as they are generated |
| `--session` | | | Resume a previous session by ID |
| `--verbose` | `-v` | `false` | Enable verbose runtime output |
| `--port` | | | Start an HTTP server on the specified port instead of running once |

## Examples

```bash
# Run an agent with inline input
agentspec run --input "Summarize this document" agent.ias

# Run with streaming output
agentspec run --stream --input "What is the weather?" weather-agent.ias

# Start the agent as a local HTTP server
agentspec run --port 8080 agent.ias
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Agent executed successfully |
| `1` | An error occurred during execution |

## See Also

- [HTTP API Overview](../api/index.md) -- For production use, agents expose an HTTP API that can be invoked programmatically as an alternative to the CLI
- [CLI: apply](apply.md) -- Deploy agents to remote targets
- [CLI: dev](dev.md) -- Run an agent in development mode with live reloading
- [Agent Runtime Configuration](../configuration/runtime.md) -- Configure strategies, timeouts, and streaming
