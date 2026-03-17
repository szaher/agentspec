# dev

Invoke an agent and print the response (one-shot).

## Usage

```bash
agentspec dev <file.ias> --input "your message"
```

## Description

The `dev` command performs a one-shot agent invocation: it parses and validates the spec, invokes the agent with the configured LLM, prints the response, and exits. This is useful for quick testing of agent behavior without starting a persistent server.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--input` | | | Message to send to the agent (required) |
| `--agent` | | | Agent name (defaults to first agent in spec) |
| `--stream` | | `false` | Stream response tokens as they are generated |

## Examples

```bash
# Invoke an agent with a message
agentspec dev agent.ias --input "Summarize this document"

# Stream the response
agentspec dev agent.ias --input "What is the weather?" --stream

# Target a specific agent in a multi-agent spec
agentspec dev multi-agent.ias --agent researcher --input "Find recent papers on AI safety"
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Agent executed successfully |
| `1` | An error occurred during execution |

## See Also

- [CLI: run](run.md) -- Start the runtime server for persistent agent hosting
- [CLI: eval](eval.md) -- Evaluate agents against test cases
