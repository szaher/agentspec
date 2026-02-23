# Data Pipeline

A data processing pipeline agent with extract, transform, load, and validate capabilities. Demonstrates policy enforcement, secret management, and three-environment configuration.

## What This Demonstrates

- **Policy rules** that restrict model usage and require secrets
- **Multiple secrets** for database and API credentials
- **Three environments** (dev, staging, prod) with different model configurations
- **Dual bindings** for local development and containerized deployment
- **ETL workflow** modeled as skills

## AgentSpec Structure

### Policy Enforcement

```
policy "production-safety" {
  deny model "claude-haiku-latest"
  require secret "db-connection"
}
```

Policies declare constraints that the validator enforces:
- `deny model` — blocks specific models from being used (prevents cheaper models in production)
- `require secret` — ensures a secret is declared before the configuration can be applied

When you run `agentspec validate`, the policy engine evaluates these rules against all resources and reports violations.

### Secret Management

```
secret "db-connection" {
  store "env"
  env "DATABASE_URL"
}

secret "source-api-key" {
  store "env"
  env "SOURCE_API_KEY"
}
```

Secrets reference external credential stores. The `store "env"` attribute means the value is read from an environment variable at runtime. The tool never stores secret values in the state file -- only the reference.

### Three-Environment Configuration

```
environment "dev" {
  agent "etl-bot" {
    model "claude-haiku-latest"
  }
}

environment "staging" {
  agent "etl-bot" {
    model "claude-sonnet-4-20250514"
  }
}

environment "prod" {
  agent "etl-bot" {
    model "claude-sonnet-4-20250514"
  }
}
```

Note: The `deny model "claude-haiku-latest"` policy applies globally. When validating with `--env dev`, the policy engine would flag the haiku model as denied. This demonstrates how policies and environments interact — you may want different policies per environment in a real deployment.

## How to Run

```bash
# Validate
./agentspec validate examples/data-pipeline.ias

# Plan for each environment
./agentspec plan examples/data-pipeline.ias --env dev
./agentspec plan examples/data-pipeline.ias --env staging
./agentspec plan examples/data-pipeline.ias --env prod

# Apply
./agentspec apply examples/data-pipeline.ias --env prod --auto-approve

# Export to Docker Compose
./agentspec export examples/data-pipeline.ias --target compose --out-dir ./pipeline-deploy
```

## Resources Created

| Kind | Name | Description |
|------|------|-------------|
| Prompt | etl | Data engineering instructions |
| Skill | extract | Extract data from sources |
| Skill | transform | Transform data with mapping rules |
| Skill | load | Load data into target database |
| Skill | validate | Run data quality checks |
| Agent | etl-bot | ETL pipeline agent (model varies by env) |
| Secret | db-connection | Database connection string |
| Secret | source-api-key | Source system API key |
| Policy | production-safety | Restricts models and requires secrets |

## Next Steps

- Add MCP transport for the data tools: see [mcp-server-client](../mcp-server-client/)
- Extend with a custom plugin: see [plugin-usage](../plugin-usage/)
