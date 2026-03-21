# Support Bot

A customer support agent with order lookup and knowledge base search capabilities. This template demonstrates a single agent with multiple MCP-connected skills for handling support workflows.

## Prerequisites

- AgentSpec CLI installed
- `ANTHROPIC_API_KEY` environment variable set
- An MCP-compatible order service running (for the `lookup-order` skill)
- An MCP-compatible knowledge base service running (for the `search-kb` skill)

## Configuration

Set required environment variables:

```bash
export ANTHROPIC_API_KEY="your-key-here"
```

## Run

```bash
agentspec validate support-bot.ias
agentspec run support-bot.ias
```

## Customization

- Update the `prompt "support-system"` block to match your company's tone and policies.
- Change the MCP `server` values in each skill to point to your actual service endpoints.
- Add more skills (e.g., refund processing, ticket creation) by defining additional `skill` blocks.
- Adjust `max_turns` if your support workflows require more reasoning steps.
