# Basic Chatbot

A minimal single-agent chatbot template. Includes a system prompt, one skill (web search via MCP), and a local deployment target. This is the simplest starting point for building an AgentSpec project.

## Prerequisites

- AgentSpec CLI installed
- `ANTHROPIC_API_KEY` environment variable set

## Configuration

Set required environment variables:

```bash
export ANTHROPIC_API_KEY="your-key-here"
```

## Run

```bash
agentspec validate basic-chatbot.ias
agentspec run basic-chatbot.ias
```

## Customization

- Edit the `prompt "chatbot-system"` block to change the agent's personality and instructions.
- Replace the `skill "web-search"` with your own MCP server or HTTP tool.
- Adjust `max_turns`, `temperature`, and `strategy` in the agent block to tune behavior.
- Change the `port` in the deploy block to serve on a different port.
