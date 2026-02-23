# Customer Support

A customer support agent with ticket lookup, knowledge base search, and escalation capabilities. Demonstrates secrets management and environment overlays in a realistic IntentLang scenario.

## What This Demonstrates

- **Multi-skill agent** with domain-specific capabilities
- **Secret references** for API keys stored in environment variables
- **Environment overlays** switching models between dev (cheaper) and prod (higher quality)
- **Multi-line prompts** with detailed behavioral instructions

## AgentSpec Structure

### Prompt with Detailed Instructions

```
prompt "support" {
  content "You are a customer support agent for Acme Corp.
           Be empathetic, concise, and solution-oriented.
           Always greet the customer by name when available.
           If you cannot resolve an issue, escalate to a human agent."
}
```

Prompts support multi-line content. The formatter preserves the content as-is.

### Secret Declaration

```
secret "api-key" {
  store "env"
  env "ACME_API_KEY"
}
```

Secrets declare where sensitive values come from without embedding them in the AgentSpec. The `store "env"` and `env` attribute indicate the value is read from the `ACME_API_KEY` environment variable at runtime. The validator ensures secrets are referenced properly and rejects plaintext values in AgentSpec files.

### Environment-Specific Models

```
environment "dev" {
  agent "support-bot" {
    model "claude-haiku-latest"
  }
}

environment "prod" {
  agent "support-bot" {
    model "claude-sonnet-4-20250514"
  }
}
```

Dev uses a faster, cheaper model for iteration. Prod uses a higher-quality model for customer-facing interactions.

## How to Run

```bash
# Validate
./agentspec validate examples/customer-support.ias

# Plan for dev (haiku model)
./agentspec plan examples/customer-support.ias --env dev

# Plan for prod (sonnet model)
./agentspec plan examples/customer-support.ias --env prod

# Apply for prod
./agentspec apply examples/customer-support.ias --env prod --auto-approve

# Export
./agentspec export examples/customer-support.ias --out-dir ./output
```

## Resources Created

| Kind | Name | Description |
|------|------|-------------|
| Prompt | support | Customer support behavioral instructions |
| Skill | lookup-ticket | Ticket lookup by ID |
| Skill | search-kb | Knowledge base search |
| Skill | escalate | Escalation to human agent |
| Agent | support-bot | Support agent (model varies by environment) |
| Secret | api-key | API key for backend services |

## Next Steps

- Add policy enforcement: see [data-pipeline](../data-pipeline/)
- Expose skills via MCP: see [mcp-server-client](../mcp-server-client/)
