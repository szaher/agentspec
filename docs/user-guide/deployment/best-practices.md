# Production Best Practices

This page covers operational best practices for deploying AgentSpec agents to production environments. Following these guidelines helps you build secure, reliable, and observable agent systems.

---

## Secret Management

Never store credentials, API keys, or sensitive values directly in `.ias` files. Use `secret` resources to reference values from external sources.

### Use `env()` for Environment Variables

```ias novalidate
secret "api-key" {
  env(ANTHROPIC_API_KEY)
}
```

### Use `store()` for Secret Managers

```ias novalidate
secret "db-password" {
  store(production/database/password)
}
```

### Reference Secrets in Deploy Blocks

```ias novalidate
deploy "production" target "kubernetes" {
  secrets {
    ANTHROPIC_API_KEY "api-key"
    DB_PASSWORD "db-password"
  }
}
```

!!! warning "Never Hardcode Secrets"
    Do not place credentials in `env` blocks, prompt content, or anywhere else in your `.ias` files. These files are committed to version control and should not contain sensitive values.

### Checklist

- All API keys referenced through `secret` resources.
- All database credentials referenced through `secret` resources.
- No credentials in `env` blocks, prompt `content`, or tool arguments.
- `.ias` files are safe to commit to version control.
- Secret rotation requires only `agentspec apply`, not file changes.

---

## Monitoring and Health Checks

Always configure a `health` block for production deployments. Health checks enable the deployment target to detect unhealthy agents and take corrective action.

### Recommended Configuration

```ias novalidate
health {
  path "/healthz"
  interval "30s"
  timeout "5s"
}
```

| Target | Health Check Behavior |
|--------|----------------------|
| `process` | AgentSpec polls the endpoint and logs warnings on failure. |
| `docker` | Docker marks the container as unhealthy. Orchestrators can restart it. |
| `docker-compose` | Docker Compose marks the service as unhealthy. Dependent services wait. |
| `kubernetes` | Kubernetes restarts unhealthy pods (liveness) and removes them from traffic (readiness). |

### Guidelines

- Always set `interval` to at least `"10s"` to avoid excessive polling.
- Set `timeout` lower than `interval` to ensure checks complete between polls.
- Use a dedicated health endpoint (`/healthz`) that checks internal readiness (model connectivity, tool availability).
- Monitor health check failures in your logging and alerting systems.

---

## Resource Planning

Right-sizing CPU and memory prevents both over-provisioning (wasting resources) and under-provisioning (causing failures under load).

### Sizing by Strategy

| Strategy | CPU | Memory | Rationale |
|----------|-----|--------|-----------|
| `react` | `"500m"` - `"1"` | `"256Mi"` - `"512Mi"` | Moderate: sequential reason-act cycles. |
| `plan-and-execute` | `"1"` - `"2"` | `"512Mi"` - `"1Gi"` | Higher: builds and maintains a plan in memory. |
| `reflexion` | `"1"` - `"2"` | `"512Mi"` - `"1Gi"` | Higher: multiple revision passes over output. |
| `router` | `"250m"` - `"500m"` | `"128Mi"` - `"256Mi"` | Lower: classifies and delegates, minimal processing. |
| `map-reduce` | `"1"` - `"4"` | `"1Gi"` - `"4Gi"` | Highest: parallel chunk processing requires more memory. |

### Guidelines

- Start with the lower end of each range and increase based on observed usage.
- Monitor actual CPU and memory consumption over time with `kubectl top pods` or `docker stats`.
- Set Kubernetes resource requests and limits to the same value (AgentSpec does this by default) for guaranteed QoS.

---

## Scaling Strategies

### Horizontal Scaling (Replicas and Autoscale)

Add more instances of the same agent to handle increased load:

```ias novalidate
deploy "production" target "kubernetes" {
  replicas 3
  autoscale {
    min 3
    max 10
    metric "cpu"
    target 80
  }
}
```

- Use `replicas` for a fixed number of instances.
- Use `autoscale` for dynamic scaling based on metrics.
- Set `autoscale.min` to at least `2` for high availability (one pod can restart while the other serves traffic).
- Set `autoscale.target` to `70-80` for a balance between efficiency and headroom.

### Vertical Scaling (Resources)

Give each instance more CPU and memory:

```ias novalidate
resources {
  cpu "2"
  memory "2Gi"
}
```

Vertical scaling is appropriate when individual requests are resource-intensive (e.g., long-running `plan-and-execute` or `map-reduce` strategies) rather than when request volume is high.

### When to Use Each

| Scenario | Approach |
|----------|----------|
| High request volume, fast per-request | Horizontal (more replicas) |
| Low request volume, long per-request | Vertical (more resources per pod) |
| Variable traffic patterns | Horizontal with autoscaling |
| Memory-intensive strategies | Vertical with higher memory limits |

---

## CI/CD Integration

Integrate AgentSpec into your CI/CD pipeline for automated validation, planning, and deployment.

### Recommended Pipeline Stages

```
validate  -->  plan  -->  approve  -->  apply
```

### Example Pipeline

```yaml
# .github/workflows/deploy.yml
name: Deploy Agent
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install AgentSpec
        run: |
          go install github.com/szaher/designs/agentz/cmd/agentspec@latest

      - name: Validate
        run: agentspec validate my-agent.ias

      - name: Plan
        run: agentspec plan my-agent.ias --target production

      - name: Apply
        run: agentspec apply my-agent.ias --target production
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

### Guidelines

- Always run `validate` before `plan`. Catch syntax and reference errors early.
- Always run `plan` before `apply`. Review the diff to confirm expected changes.
- Store secrets in your CI/CD platform's secret manager (e.g., GitHub Actions secrets), not in the repository.
- Commit `.agentspec.state.json` to version control so that the plan accurately reflects drift.
- Run `agentspec fmt` in CI to enforce consistent formatting across the team.

---

## Logging and Observability

Configure logging through environment variables in the deploy block:

```ias novalidate
deploy "production" target "kubernetes" {
  env {
    LOG_LEVEL "warn"
    LOG_FORMAT "json"
  }
}
```

### Recommended Log Levels by Environment

| Environment | Log Level | Rationale |
|-------------|-----------|-----------|
| Development | `"debug"` | Maximum visibility for debugging. |
| Staging | `"info"` | Enough detail to diagnose integration issues. |
| Production | `"warn"` | Minimal noise. Alerts on warnings and errors only. |

### Guidelines

- Use structured JSON logging (`LOG_FORMAT "json"`) in production for easier parsing by log aggregation systems.
- Forward logs to a centralized system (e.g., Elasticsearch, Datadog, CloudWatch).
- Include request correlation IDs in logs to trace requests across multi-agent systems.
- Monitor key metrics: request latency, error rate, token consumption, and turn count.

---

## Environment Management

Use `environment` blocks to vary configuration across deployment stages without duplicating `.ias` files:

```ias novalidate
environment "dev" {
  agent "assistant" {
    model "claude-haiku-latest"
    max_turns 5
  }
}

environment "staging" {
  agent "assistant" {
    model "claude-sonnet-4-20250514"
    max_turns 10
  }
}

environment "prod" {
  agent "assistant" {
    model "claude-sonnet-4-20250514"
    max_turns 15
    timeout "60s"
  }
}
```

Apply with a specific environment:

```bash
agentspec apply my-agent.ias --env prod --target production
```

### Common Overrides

| Attribute | Dev | Staging | Production |
|-----------|-----|---------|------------|
| `model` | `claude-haiku-latest` | `claude-sonnet-4-20250514` | `claude-sonnet-4-20250514` |
| `max_turns` | 5 | 10 | 15 |
| `timeout` | `"10s"` | `"30s"` | `"60s"` |
| `token_budget` | 10000 | 100000 | 200000 |
| `temperature` | 0.7 | 0.5 | 0.3 |

!!! tip "Cost Control in Development"
    Use `claude-haiku-latest` with lower `max_turns` and `token_budget` in development. This reduces cost and iteration time without affecting the structure of your agent system.

---

## Rollback Strategies

AgentSpec's desired-state model makes rollbacks straightforward. To roll back, apply a previous version of the `.ias` file.

### Using Version Control

```bash
# Revert to the previous commit's agent definition
git checkout HEAD~1 -- my-agent.ias

# Preview the rollback
agentspec plan my-agent.ias --target production

# Apply the rollback
agentspec apply my-agent.ias --target production
```

### Using Image Tags

If the rollback only requires a different container image, update the `image` attribute:

```ias novalidate
deploy "production" target "kubernetes" {
  image "agentspec/assistant:0.9.0"  # Roll back to previous version
}
```

Then apply:

```bash
agentspec apply my-agent.ias --target production
```

Kubernetes performs a rolling update to the previous image version with zero downtime.

### Guidelines

- Tag every release with a semantic version in both the package header and the image tag.
- Keep the `.agentspec.state.json` file in version control for accurate diff computation.
- Test rollbacks in staging before executing them in production.
- Monitor health checks after a rollback to confirm the previous version is healthy.

---

## Security Policies

Use `policy` blocks to enforce security and governance constraints across your agent system:

```ias novalidate
policy "production-safety" {
  deny model claude-haiku-latest
  require secret api-key
  require secret db-password
  allow model claude-sonnet-4-20250514
  allow model claude-opus-4-20250514
}
```

### Common Policies

| Rule | Purpose |
|------|---------|
| `deny model claude-haiku-latest` | Prevent lower-capability models in production. |
| `require secret api-key` | Ensure API key secrets are declared before deployment. |
| `require secret db-password` | Ensure database credentials are managed through secrets. |
| `allow model claude-sonnet-4-20250514` | Explicitly permit approved models for compliance. |

### Guidelines

- Define at least one policy for production deployments.
- Use `deny` rules to prevent insecure or inappropriate configurations.
- Use `require` rules to mandate that secrets are declared for all sensitive values.
- Policies are evaluated after environment overrides, so they apply to the final resolved configuration.
- Run `agentspec validate` in CI to catch policy violations before deployment.

---

## Complete Production Example

A production-ready `.ias` file incorporating all best practices:

```ias
package "production-best-practices" version "1.0.0" lang "2.0"

prompt "system" {
  content "You are a production assistant. Provide accurate responses,\nfollow security best practices, and never expose sensitive data."
}

skill "search" {
  description "Search the knowledge base"
  input { query string required }
  output { results string }
  tool command { binary "search-tool" }
}

agent "assistant" {
  uses prompt "system"
  uses skill "search"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 15
  timeout "60s"
  token_budget 200000
  temperature 0.3
  on_error "retry"
  max_retries 3
}

secret "api-key" {
  env(ANTHROPIC_API_KEY)
}

policy "production-safety" {
  deny model claude-haiku-latest
  require secret api-key
  allow model claude-sonnet-4-20250514
}

environment "dev" {
  agent "assistant" {
    model "claude-haiku-latest"
  }
}

environment "prod" {
  agent "assistant" {
    model "claude-sonnet-4-20250514"
  }
}

deploy "dev" target "process" {
  default true
  port 8080
  env {
    LOG_LEVEL "debug"
  }
}

deploy "production" target "kubernetes" {
  namespace "agents"
  image "agentspec/assistant:1.0.0"
  replicas 3
  port 8080
  resources {
    cpu "1"
    memory "1Gi"
  }
  health {
    path "/healthz"
    interval "30s"
    timeout "5s"
  }
  autoscale {
    min 3
    max 10
    metric "cpu"
    target 80
  }
  env {
    LOG_LEVEL "warn"
    LOG_FORMAT "json"
    ENVIRONMENT "production"
  }
  secrets {
    ANTHROPIC_API_KEY "api-key"
  }
}
```

---

## See Also

- [Deployment Overview](index.md) -- Compare all deployment targets
- [Process Deployment](process.md) -- Local development target
- [Docker Deployment](docker.md) -- Container deployment
- [Docker Compose Deployment](compose.md) -- Multi-agent stacks
- [Kubernetes Deployment](kubernetes.md) -- Production orchestration
- [Deploy Block Reference](../language/deploy.md) -- Full syntax reference
- [Environment Reference](../language/environment.md) -- Environment overlays
- [Secret Reference](../language/secret.md) -- Secret management
- [Policy Reference](../language/policy.md) -- Security and governance policies
