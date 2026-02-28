# Docker Deployment

The `docker` target packages your agent as a standalone Docker container. AgentSpec generates a Dockerfile, builds the image, and runs the container with the configured resource limits, health checks, and environment variables.

---

## Prerequisites

- Docker installed and running (`docker --version` to verify).
- The `agentspec` CLI binary installed and available on your `PATH`.

---

## Deploy Block

A Docker deployment specifies the image name, port, resource constraints, and health check:

```ias
deploy "staging" target "docker" {
  image "agentspec/my-agent:0.1.0"
  port 8080
  resources {
    cpu "500m"
    memory "256Mi"
  }
  health {
    path "/healthz"
    interval "30s"
    timeout "5s"
  }
  env {
    LOG_LEVEL "info"
  }
}
```

### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `default` | bool | Mark this as the default deploy target. |
| `image` | string | Docker image name and tag (e.g. `"agentspec/my-agent:0.1.0"`). |
| `port` | int | Port exposed by the container and mapped to the host. |
| `resources` | block | CPU and memory limits applied to the container. |
| `health` | block | Docker health check configuration. |
| `env` | block | Environment variables injected into the container. |
| `secrets` | block | Secret mappings resolved and injected at deploy time. |

!!! info "Attributes Not Applicable"
    The `namespace`, `replicas`, and `autoscale` attributes have no effect on the `docker` target. For multi-container setups, use [Docker Compose](compose.md). For replicas and autoscaling, use [Kubernetes](kubernetes.md).

---

## How It Works

When you run `agentspec apply` with a `docker` target, AgentSpec:

1. Generates a `Dockerfile` based on the agent definition and deploy configuration.
2. Builds a Docker image with the specified name and tag.
3. Creates and starts a container with the configured port mapping, resource limits, and environment variables.
4. Configures a Docker health check if a `health` block is present.
5. Records the deployment state in `.agentspec.state.json`.

The generated Dockerfile uses a multi-stage build to produce a minimal runtime image.

---

## Resource Limits

The `resources` block sets CPU and memory constraints on the container. These map directly to Docker's `--cpus` and `--memory` flags.

```ias
resources {
  cpu "500m"
  memory "256Mi"
}
```

| Attribute | Format | Examples |
|-----------|--------|----------|
| `cpu` | Millicores or whole cores | `"250m"` (0.25 cores), `"1"` (1 core), `"2"` (2 cores) |
| `memory` | Mebibytes or gibibytes | `"128Mi"`, `"256Mi"`, `"1Gi"`, `"2Gi"` |

!!! tip "Sizing Resources"
    For a typical agent using the `react` strategy, start with `"500m"` CPU and `"256Mi"` memory. For agents with `plan-and-execute` or `reflexion` strategies that handle longer sessions, consider `"1"` CPU and `"512Mi"` memory.

---

## Health Checking

The `health` block generates a Docker `HEALTHCHECK` instruction in the Dockerfile and configures the container health check:

```ias
health {
  path "/healthz"
  interval "30s"
  timeout "5s"
}
```

Docker periodically runs a health check against the configured path. The container's health status is visible via `docker ps` and `docker inspect`:

```bash
docker ps --format "table {{.Names}}\t{{.Status}}"
```

Output:

```
NAMES               STATUS
my-agent            Up 5 minutes (healthy)
```

---

## Environment Variables

Use the `env` block to inject environment variables into the container:

```ias
env {
  LOG_LEVEL "info"
  ENVIRONMENT "staging"
  MAX_CONNECTIONS "100"
}
```

These variables are set at container creation time and are available to the agent process inside the container.

---

## Volume Mounts

For agents that need persistent storage or access to host files, use the `volumes` attribute:

```ias
deploy "staging" target "docker" {
  image "agentspec/my-agent:0.1.0"
  port 8080
  env {
    DATA_DIR "/data"
  }
}
```

!!! note "Volume Configuration"
    Volume mounts can be configured through environment variables that the agent reads at runtime to determine storage paths. For production data persistence, consider using [Kubernetes](kubernetes.md) with persistent volume claims.

---

## Secret Management

Use the `secrets` block to inject secret values into the container:

```ias
secret "api-key" {
  env(API_KEY)
}

secret "db-password" {
  store(production/database/password)
}

deploy "staging" target "docker" {
  image "agentspec/my-agent:0.1.0"
  port 8080
  secrets {
    API_KEY "api-key"
    DB_PASSWORD "db-password"
  }
}
```

Secrets are resolved at deploy time and injected as environment variables inside the container. They are never baked into the Docker image.

---

## Complete Example

A fully configured agent with Docker deployment:

```ias
package "docker-agent" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a customer support agent. Help users resolve issues\nby searching the knowledge base and creating support tickets."
}

skill "search-kb" {
  description "Search the knowledge base for relevant articles"
  input {
    query string required
  }
  output {
    articles string
  }
  tool command {
    binary "kb-search"
  }
}

skill "create-ticket" {
  description "Create a support ticket"
  input {
    subject string required
    description string required
    priority string required
  }
  output {
    ticket_id string
  }
  tool command {
    binary "ticket-tool"
  }
}

agent "support-bot" {
  uses prompt "system"
  uses skill "search-kb"
  uses skill "create-ticket"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 15
  timeout "60s"
  on_error "retry"
  max_retries 2
}

secret "api-key" {
  env(ANTHROPIC_API_KEY)
}

# Local development
deploy "dev" target "process" {
  default true
  port 8080
}

# Docker staging
deploy "staging" target "docker" {
  image "agentspec/support-bot:0.1.0"
  port 8080
  resources {
    cpu "500m"
    memory "256Mi"
  }
  health {
    path "/healthz"
    interval "30s"
    timeout "5s"
  }
  env {
    LOG_LEVEL "info"
    ENVIRONMENT "staging"
  }
  secrets {
    ANTHROPIC_API_KEY "api-key"
  }
}
```

---

## Deploying

Validate, plan, and apply the Docker deployment:

```bash
# Validate the .ias file
agentspec validate support-bot.ias

# Preview changes for the staging target
agentspec plan support-bot.ias --target staging

# Apply the Docker deployment
agentspec apply support-bot.ias --target staging
```

---

## Verification

After applying, verify the container is running and healthy.

### Check Container Status

```bash
docker ps --filter name=support-bot
```

### Health Check

```bash
curl http://localhost:8080/healthz
```

Expected response:

```json
{"status": "healthy"}
```

### View Logs

```bash
docker logs support-bot --follow
```

### Inspect Resource Usage

```bash
docker stats support-bot --no-stream
```

---

## Updating the Deployment

To update a running Docker deployment, modify your `.ias` file and re-apply:

```bash
agentspec plan support-bot.ias --target staging
agentspec apply support-bot.ias --target staging
```

AgentSpec computes the diff between the current state and the desired state. If the image tag or configuration has changed, it rebuilds and replaces the container.

---

## Stopping the Deployment

To stop and remove the Docker container:

```bash
agentspec destroy support-bot.ias --target staging
```

This stops the container, removes it, and updates `.agentspec.state.json`.

---

## See Also

- [Deployment Overview](index.md) -- Compare all deployment targets
- [Process Deployment](process.md) -- Simpler local development
- [Docker Compose Deployment](compose.md) -- Multi-agent container stacks
- [Deploy Block Reference](../language/deploy.md) -- Full attribute reference
- [Best Practices](best-practices.md) -- Production readiness guidance
