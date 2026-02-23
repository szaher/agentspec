# Code Review Pipeline

A multi-agent code review system with separate agents for static analysis, security scanning, and review summarization. Agents collaborate through MCP transport and can be deployed to both local and CI environments.

## What This Demonstrates

- **Multiple agents** in a single AgentSpec, each with a specialized role
- **Shared skills** referenced by multiple agents (`read-diff` is used by both analyzer and scanner)
- **MCP server/client** wiring for inter-agent communication
- **Dual deploy targets** for local development and CI/CD deployment

## AgentSpec Structure

### Specialized Agents

```
agent "code-analyzer" {
  uses prompt "analyzer"
  uses skill "read-diff"
  uses skill "analyze-code"
  model "claude-sonnet-4-20250514"
}

agent "security-scanner" {
  uses prompt "security-reviewer"
  uses skill "read-diff"
  uses skill "scan-security"
  model "claude-sonnet-4-20250514"
}

agent "review-summarizer" {
  uses prompt "summarizer"
  uses skill "post-review"
  model "claude-sonnet-4-20250514"
}
```

Each agent has a distinct role defined by its prompt and the skills it can use. Multiple agents can reference the same skill — here, both `code-analyzer` and `security-scanner` use `read-diff`.

### MCP Transport Layer

```
server "review-server" {
  transport "stdio"
  command "review-mcp-server"
  exposes skill "read-diff"
  exposes skill "analyze-code"
  exposes skill "scan-security"
  exposes skill "post-review"
}

client "review-client" {
  connects to server "review-server"
}
```

A single MCP server exposes all skills. The client connects to this server, providing the transport layer for skill execution.

### Dual Deployment Targets

```
deploy "local" target "process" {
  default true
}

deploy "ci" target "docker-compose" {
  output_dir "./ci-deploy"
}
```

- `local` — for development and testing on your machine
- `ci` — generates Docker Compose artifacts for CI/CD pipelines

## How to Run

```bash
# Validate
./agentspec validate examples/code-review-pipeline.ias

# Plan for local
./agentspec plan examples/code-review-pipeline.ias

# Plan for CI
./agentspec plan examples/code-review-pipeline.ias --target ci

# Apply locally
./agentspec apply examples/code-review-pipeline.ias --auto-approve

# Export CI artifacts
./agentspec export examples/code-review-pipeline.ias --target ci --out-dir ./ci-deploy
```

## Resources Created

| Kind | Name | Description |
|------|------|-------------|
| Prompt | analyzer | Code analysis instructions |
| Prompt | security-reviewer | Security review instructions |
| Prompt | summarizer | Review summarization instructions |
| Skill | read-diff | Read git diffs / PRs |
| Skill | analyze-code | Static analysis |
| Skill | scan-security | Security vulnerability scanning |
| Skill | post-review | Post review comments |
| Agent | code-analyzer | Identifies code quality issues |
| Agent | security-scanner | Identifies security vulnerabilities |
| Agent | review-summarizer | Combines findings into actionable review |
| MCPServer | review-server | Exposes all skills over stdio |
| MCPClient | review-client | Connects to the review server |

## CI Export Artifacts

```
ci-deploy/
  docker-compose.yml   # Three agent services + MCP server
  config/              # Per-service configuration
  .env                 # Environment variables template
```

## Next Steps

- Add environment overlays for staging vs prod: see [multi-environment](../multi-environment/)
- Add a RAG-based knowledge agent to the pipeline: see [rag-chatbot](../rag-chatbot/)
