# Multi-Agent Router

A router agent using IntentLang 3.0 control flow to classify incoming messages and delegate them to specialized handler skills. Uses `on input` with `if/else` branching to match message content against domain-specific keywords (billing, technical, sales) and route to the appropriate handler.

## Prerequisites

- AgentSpec CLI installed
- `ANTHROPIC_API_KEY` environment variable set
- MCP-compatible handler services for each domain (billing, technical, sales, general)

## Configuration

Set required environment variables:

```bash
export ANTHROPIC_API_KEY="your-key-here"
```

## Run

```bash
agentspec validate multi-agent-router.ias
agentspec run multi-agent-router.ias
```

## Customization

- Add new routing branches by adding `else if` clauses in the `on input` block.
- Add new handler skills for additional domains (e.g., HR, legal, shipping).
- Replace keyword matching with more sophisticated classification logic as needed.
- Change the MCP `server` values to point to your actual backend services.
- Adjust `temperature` (currently 0.1) -- low values keep routing deterministic.
