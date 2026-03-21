# Support Bot

A customer support agent for Acme Corp with ticket lookup, knowledge base search, and human escalation capabilities. Demonstrates secrets management, environment overlays, and multi-skill agent composition in IntentLang.

## Architecture Overview

The support bot is a single agent backed by three specialized skills:

```
User Message
    |
    v
support-bot (agent)
    |
    +---> lookup-ticket   -- retrieves ticket details by ID
    +---> search-kb       -- searches the knowledge base for articles
    +---> escalate        -- hands off to a human agent when needed
```

The agent uses a detailed system prompt that enforces empathetic, solution-oriented behavior. Environment overlays allow switching between a cheaper model for development and a higher-quality model for production.

## Prerequisites

1. Build the AgentSpec CLI from the repository root:

   ```bash
   go build -o agentspec ./cmd/agentspec
   ```

2. Set the required environment variable:

   ```bash
   export ACME_API_KEY="your-acme-api-key"
   ```

3. Ensure the following tool binaries are available on `$PATH` (or stubbed for testing):
   - `ticket-lookup`
   - `kb-search`
   - `escalate-tool`

## Step-by-Step Run Instructions

```bash
# 1. Validate the AgentSpec
./agentspec validate examples/support-bot/support-bot.ias

# 2. Preview changes for the dev environment (uses claude-haiku-latest)
./agentspec plan examples/support-bot/support-bot.ias --env dev

# 3. Preview changes for the prod environment (uses claude-sonnet-4-20250514)
./agentspec plan examples/support-bot/support-bot.ias --env prod

# 4. Apply for production
./agentspec apply examples/support-bot/support-bot.ias --env prod --auto-approve

# 5. Export artifacts
./agentspec export examples/support-bot/support-bot.ias --out-dir ./output
```

## Customization Tips

- **Add more skills**: Define additional `skill` blocks (e.g., `refund-order`, `check-status`) and wire them into the agent with `uses skill`.
- **Change models per environment**: Edit the `environment` blocks to swap in different models for staging, QA, or cost optimization.
- **Extend the prompt**: Add domain-specific instructions (tone, prohibited topics, compliance rules) directly in the `prompt` block's `content` field.
- **Add MCP transport**: Wrap the skills in a `server` block to expose them over stdio or HTTP for integration with external systems.
- **Add policies**: See the [data-pipeline](../data-pipeline/) example for policy enforcement patterns.
