# Deployment Overview

AgentSpec supports deploying agents to multiple targets from a single `.ias` file. Each deployment target is defined using a `deploy` block, and a single package can contain multiple deploy blocks targeting different environments -- from a local process for development to a Kubernetes cluster for production.

---

## Deployment Targets

AgentSpec supports four deployment targets. Choose the one that matches your operational needs.

| Target | Best For | Prerequisites | Scaling |
|--------|----------|---------------|---------|
| [`process`](process.md) | Local development, testing, debugging | `agentspec` binary, Go runtime | Single instance |
| [`docker`](docker.md) | Standalone containers, staging environments | Docker installed | Single container |
| [`docker-compose`](compose.md) | Multi-agent stacks, local integration testing | Docker Compose installed | `replicas` per service |
| [`kubernetes`](kubernetes.md) | Production workloads, auto-scaling, high availability | `kubectl`, cluster access | HPA autoscaling |

!!! tip "Start Simple"
    Begin with the `process` target for local development. Once your agent is working correctly, add a `docker` or `kubernetes` deploy block to the same `.ias` file for staging and production.

---

## Deployment Workflow

Every deployment follows the same four-step workflow, regardless of the target.

### 1. Write the Deploy Block

Add a `deploy` block to your `.ias` file specifying the target type and its configuration:

<!-- novalidate -->
```ias
deploy "production" target "kubernetes" {
  namespace "agents"
  replicas 3
  port 8080
}
```

### 2. Validate and Plan

Validate the file for syntax and reference errors, then preview what AgentSpec will do:

```bash
agentspec validate my-agent.ias
agentspec plan my-agent.ias
```

The plan output shows which resources will be created, updated, or removed:

```
Plan: 5 to add, 0 to change, 0 to destroy.

  + prompt "system"
  + skill "search"
  + agent "assistant"
  + deploy "local" (target: process)
  + deploy "production" (target: kubernetes)
```

### 3. Apply

Execute the plan to deploy:

```bash
agentspec apply my-agent.ias
```

To target a specific deploy block by name:

```bash
agentspec apply my-agent.ias --target production
```

### 4. Verify

Confirm the deployment is healthy:

```bash
curl http://localhost:8080/healthz
```

---

## Deploy Block Syntax

The `deploy` block has the following general structure:

<!-- novalidate -->
```ias
deploy "<name>" target "<type>" {
  default <bool>
  port <int>
  image "<docker-image>"
  namespace "<namespace>"
  replicas <int>
  resources {
    cpu "<cpu-spec>"
    memory "<memory-spec>"
  }
  health {
    path "<endpoint>"
    interval "<duration>"
    timeout "<duration>"
  }
  autoscale {
    min <int>
    max <int>
    metric "<metric-name>"
    target <int>
  }
  env {
    <VAR_NAME> "<value>"
  }
  secrets {
    <VAR_NAME> "<secret-name>"
  }
}
```

---

## Common Attributes

These attributes are available across all deployment targets.

### port

The port the agent service listens on:

<!-- novalidate -->
```ias
deploy "local" target "process" {
  port 8080
}
```

### health

Health check configuration. AgentSpec uses this to verify the agent is running correctly after deployment:

<!-- novalidate -->
```ias
health {
  path "/healthz"
  interval "30s"
  timeout "5s"
}
```

| Attribute | Description |
|-----------|-------------|
| `path` | HTTP endpoint for health checks (e.g. `"/healthz"`). |
| `interval` | How often to check (e.g. `"30s"`, `"1m"`). |
| `timeout` | Maximum wait time for a response (e.g. `"5s"`). |

### env

Environment variables passed to the deployed service:

<!-- novalidate -->
```ias
env {
  LOG_LEVEL "info"
  ENVIRONMENT "staging"
}
```

### secrets

Maps environment variable names to declared `secret` resources. Secrets are resolved at deploy time from their configured source (environment variable or secret store):

<!-- novalidate -->
```ias
secrets {
  API_KEY "api-key"
  DATABASE_URL "db-url"
}
```

!!! warning "Never hardcode secrets"
    Always use `secret` resources and the `secrets` block in deploy. Never put credentials directly in `env` blocks or anywhere else in your `.ias` file.

### default

Mark one deploy block as the default target. This is the target used when you run `agentspec apply` without the `--target` flag:

<!-- novalidate -->
```ias
deploy "local" target "process" {
  default true
}
```

Only one deploy block per package may be marked as default.

---

## Multiple Deploy Targets

A single `.ias` file can define multiple deploy targets for different environments:

```ias
package "multi-target" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a helpful assistant."
}

skill "search" {
  description "Search for information"
  input { query string required }
  output { results string }
  tool command { binary "search-tool" }
}

agent "assistant" {
  uses prompt "system"
  uses skill "search"
  model "claude-sonnet-4-20250514"
}

# Development: local process
deploy "dev" target "process" {
  default true
  port 8080
}

# Staging: Docker container
deploy "staging" target "docker" {
  image "agentspec/assistant:0.1.0"
  port 8080
  resources {
    cpu "500m"
    memory "256Mi"
  }
}

# Production: Kubernetes cluster
deploy "production" target "kubernetes" {
  namespace "agents"
  replicas 3
  port 8080
  resources {
    cpu "1"
    memory "1Gi"
  }
  autoscale {
    min 3
    max 10
    metric "cpu"
    target 80
  }
}
```

---

## What's Next

- [Process Deployment](process.md) -- Run agents as local processes for development
- [Docker Deployment](docker.md) -- Containerize agents with Docker
- [Docker Compose Deployment](compose.md) -- Orchestrate multi-agent stacks
- [Kubernetes Deployment](kubernetes.md) -- Deploy to production Kubernetes clusters
- [Best Practices](best-practices.md) -- Production readiness, security, and operational guidance
- [Deploy Block Reference](../language/deploy.md) -- Full syntax and attribute reference
- [CLI: apply](../cli/apply.md) -- The `agentspec apply` command that executes deployments
- [CLI: plan](../cli/plan.md) -- Preview deployment changes before applying
