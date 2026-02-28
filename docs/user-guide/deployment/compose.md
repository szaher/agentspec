# Docker Compose Deployment

The `docker-compose` target deploys your agent as part of a multi-service Docker Compose stack. AgentSpec generates a `docker-compose.yml` file with service definitions, networking, health checks, and secret mappings. This target is ideal for local integration testing, multi-agent systems, and staging environments where agents need to communicate with each other or with supporting services.

---

## Prerequisites

- Docker and Docker Compose installed (`docker compose version` to verify).
- The `agentspec` CLI binary installed and available on your `PATH`.

---

## Deploy Block

A Docker Compose deployment specifies the image, port, replicas, and health check:

```ias
deploy "compose" target "docker-compose" {
  image "agentspec/my-agent:0.1.0"
  port 8080
  replicas 2
  resources {
    cpu "500m"
    memory "256Mi"
  }
  health {
    path "/healthz"
    interval "30s"
    timeout "5s"
  }
}
```

### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `default` | bool | Mark this as the default deploy target. |
| `image` | string | Docker image name and tag. |
| `port` | int | Port exposed by the service. |
| `replicas` | int | Number of service replicas to run. |
| `resources` | block | CPU and memory limits per replica. |
| `health` | block | Health check configuration for the service. |
| `env` | block | Environment variables injected into the service. |
| `secrets` | block | Secret mappings resolved at deploy time. |

!!! info "Attributes Not Applicable"
    The `namespace` and `autoscale` attributes have no effect on the `docker-compose` target. For namespace isolation and autoscaling, use [Kubernetes](kubernetes.md).

---

## How It Works

When you run `agentspec apply` with a `docker-compose` target, AgentSpec:

1. Generates a `docker-compose.yml` file from all `docker-compose` deploy blocks in the `.ias` file.
2. Builds Docker images for each service if needed.
3. Creates a Docker network for inter-service communication.
4. Starts all services with the configured replicas, resource limits, and environment variables.
5. Configures health checks for each service.
6. Records the deployment state in `.agentspec.state.json`.

---

## Multi-Agent Setup

A single `.ias` file can define multiple agents, each with its own `docker-compose` deploy block. AgentSpec generates a single `docker-compose.yml` containing all services:

```ias
deploy "frontend-compose" target "docker-compose" {
  image "agentspec/frontend-agent:0.1.0"
  port 8080
  health {
    path "/healthz"
    interval "30s"
    timeout "5s"
  }
}

deploy "backend-compose" target "docker-compose" {
  image "agentspec/backend-agent:0.1.0"
  port 8081
  health {
    path "/healthz"
    interval "30s"
    timeout "5s"
  }
}
```

Both services are placed on the same Docker network and can reach each other by their deploy block names (e.g., `frontend-compose:8080`, `backend-compose:8081`).

---

## Networking Between Agents

All services defined in `docker-compose` deploy blocks within the same package are placed on a shared Docker network. Services can communicate using the deploy block name as the hostname.

For example, if agent A needs to call agent B:

```ias
deploy "agent-a" target "docker-compose" {
  port 8080
  env {
    AGENT_B_URL "http://agent-b:8081"
  }
}

deploy "agent-b" target "docker-compose" {
  port 8081
}
```

Agent A can reach Agent B at `http://agent-b:8081` using the deploy block name as a DNS hostname within the Docker network.

---

## Replicas

The `replicas` attribute controls how many instances of the service Docker Compose runs:

```ias
deploy "compose" target "docker-compose" {
  image "agentspec/my-agent:0.1.0"
  port 8080
  replicas 3
}
```

Docker Compose distributes incoming connections across replicas. This is useful for load testing and simulating production-like scaling behavior locally.

---

## Resource Limits

The `resources` block sets CPU and memory constraints per service replica:

```ias
resources {
  cpu "500m"
  memory "256Mi"
}
```

These translate to Docker Compose `deploy.resources.limits` in the generated `docker-compose.yml`.

---

## Complete Example

A multi-agent customer support system deployed with Docker Compose:

```ias
package "support-stack" version "0.1.0" lang "2.0"

prompt "router-prompt" {
  content "You are a support request router. Analyze incoming requests\nand route them to the appropriate specialist agent based on\nthe type of issue."
}

prompt "billing-prompt" {
  content "You are a billing specialist. Help users with invoice\nquestions, payment issues, and subscription management."
}

prompt "technical-prompt" {
  content "You are a technical support specialist. Help users\nresolve technical issues, troubleshoot errors, and\nconfigure their systems."
}

skill "lookup-account" {
  description "Look up a customer account by email or ID"
  input {
    identifier string required
  }
  output {
    account string
  }
  tool command {
    binary "account-lookup"
  }
}

skill "search-docs" {
  description "Search technical documentation"
  input {
    query string required
  }
  output {
    results string
  }
  tool command {
    binary "doc-search"
  }
}

agent "router" {
  uses prompt "router-prompt"
  model "claude-sonnet-4-20250514"
  strategy "router"
  delegate to agent "billing-agent" when "billing or payment issue"
  delegate to agent "tech-agent" when "technical issue or error"
}

agent "billing-agent" {
  uses prompt "billing-prompt"
  uses skill "lookup-account"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 10
}

agent "tech-agent" {
  uses prompt "technical-prompt"
  uses skill "search-docs"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 15
}

secret "api-key" {
  env(ANTHROPIC_API_KEY)
}

# Local development
deploy "dev" target "process" {
  default true
  port 8080
}

# Docker Compose stack
deploy "router-svc" target "docker-compose" {
  image "agentspec/router:0.1.0"
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
    BILLING_AGENT_URL "http://billing-svc:8081"
    TECH_AGENT_URL "http://tech-svc:8082"
  }
  secrets {
    ANTHROPIC_API_KEY "api-key"
  }
}

deploy "billing-svc" target "docker-compose" {
  image "agentspec/billing-agent:0.1.0"
  port 8081
  replicas 2
  resources {
    cpu "500m"
    memory "256Mi"
  }
  health {
    path "/healthz"
    interval "30s"
    timeout "5s"
  }
  secrets {
    ANTHROPIC_API_KEY "api-key"
  }
}

deploy "tech-svc" target "docker-compose" {
  image "agentspec/tech-agent:0.1.0"
  port 8082
  replicas 2
  resources {
    cpu "1"
    memory "512Mi"
  }
  health {
    path "/healthz"
    interval "30s"
    timeout "5s"
  }
  secrets {
    ANTHROPIC_API_KEY "api-key"
  }
}
```

---

## Deploying

Validate, plan, and apply the Docker Compose stack:

```bash
# Validate the .ias file
agentspec validate support-stack.ias

# Preview the compose deployment
agentspec plan support-stack.ias --target router-svc

# Apply all docker-compose targets
agentspec apply support-stack.ias --target router-svc
```

AgentSpec generates a `docker-compose.yml` and runs `docker compose up` with the appropriate configuration.

---

## Verification

After applying, verify all services are running and healthy.

### Check Service Status

```bash
docker compose ps
```

Expected output:

```
NAME                 STATUS              PORTS
router-svc-1         Up (healthy)        0.0.0.0:8080->8080/tcp
billing-svc-1        Up (healthy)        0.0.0.0:8081->8081/tcp
billing-svc-2        Up (healthy)
tech-svc-1           Up (healthy)        0.0.0.0:8082->8082/tcp
tech-svc-2           Up (healthy)
```

### Health Check

```bash
curl http://localhost:8080/healthz
curl http://localhost:8081/healthz
curl http://localhost:8082/healthz
```

### View Logs

```bash
# All services
docker compose logs --follow

# Specific service
docker compose logs router-svc --follow
```

---

## Stopping the Stack

To stop and remove the Docker Compose stack:

```bash
agentspec destroy support-stack.ias --target router-svc
```

This runs `docker compose down`, stops all services, removes containers and networks, and updates `.agentspec.state.json`.

---

## See Also

- [Deployment Overview](index.md) -- Compare all deployment targets
- [Docker Deployment](docker.md) -- Standalone container deployment
- [Kubernetes Deployment](kubernetes.md) -- Production-grade orchestration
- [Deploy Block Reference](../language/deploy.md) -- Full attribute reference
- [Best Practices](best-practices.md) -- Production readiness guidance
