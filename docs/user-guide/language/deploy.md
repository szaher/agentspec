# Deploy

The `deploy` block defines a deployment target for your agent system. A single `.ias` file can contain multiple `deploy` blocks, allowing you to target local processes, Docker containers, Docker Compose stacks, and Kubernetes clusters from the same source definition.

---

## Syntax

```ias novalidate
deploy "<name>" target "<type>" {
  default <bool>
  port <int>
  namespace "<namespace>"
  replicas <int>
  image "<docker-image>"
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

## Top-Level Attributes

| Name        | Type   | Required | Description                                                               |
|-------------|--------|----------|---------------------------------------------------------------------------|
| `target`    | string | Yes      | The deployment target type. Specified inline with the declaration.         |
| `default`   | bool   | No       | Mark this deploy target as the default. Only one target per package may be default. |
| `port`      | int    | No       | Port number the service listens on.                                       |
| `namespace` | string | No       | Kubernetes namespace. Only applicable to the `"kubernetes"` target.        |
| `replicas`  | int    | No       | Number of replicas to run. Applicable to `"kubernetes"` and `"docker-compose"`. |
| `image`     | string | No       | Docker image name and optional tag (e.g. `"myapp:latest"`).               |
| `resources` | block  | No       | CPU and memory resource constraints.                                      |
| `health`    | block  | No       | Health check configuration.                                               |
| `autoscale` | block  | No       | Autoscaling policy. Only applicable to `"kubernetes"`.                     |
| `env`       | block  | No       | Environment variables passed to the deployed service as key-value pairs.   |
| `secrets`   | block  | No       | Secret mappings. Maps environment variable names to `secret` resource names.|

---

## Target Values

| Value              | Description                                                                  |
|--------------------|------------------------------------------------------------------------------|
| `"process"`        | Run as a local OS process. Simplest target for development and testing.       |
| `"docker"`         | Run as a standalone Docker container.                                        |
| `"docker-compose"` | Run as part of a Docker Compose stack with service orchestration.             |
| `"kubernetes"`     | Deploy to a Kubernetes cluster with full support for namespaces, replicas, resource limits, and autoscaling. |

---

## Resources Block

The `resources` block sets CPU and memory constraints for the deployed service.

```ias novalidate
resources {
  cpu "<cpu-spec>"
  memory "<memory-spec>"
}
```

| Name     | Type   | Required | Description                                                        |
|----------|--------|----------|--------------------------------------------------------------------|
| `cpu`    | string | No       | CPU allocation (e.g. `"500m"` for 0.5 cores, `"2"` for 2 cores).  |
| `memory` | string | No      | Memory allocation (e.g. `"256Mi"`, `"1Gi"`).                       |

---

## Health Block

The `health` block configures health checking for the deployed service.

```ias novalidate
health {
  path "<endpoint>"
  interval "<duration>"
  timeout "<duration>"
}
```

| Name       | Type   | Required | Description                                                      |
|------------|--------|----------|------------------------------------------------------------------|
| `path`     | string | No       | HTTP path for the health check endpoint (e.g. `"/healthz"`).     |
| `interval` | string | No       | How often to run the health check (e.g. `"30s"`, `"1m"`).        |
| `timeout`  | string | No       | Maximum time to wait for a health check response (e.g. `"5s"`).  |

---

## Autoscale Block

The `autoscale` block defines horizontal pod autoscaling rules. Only applicable to the `"kubernetes"` target.

```ias novalidate
autoscale {
  min <int>
  max <int>
  metric "<metric-name>"
  target <int>
}
```

| Name     | Type   | Required | Description                                                           |
|----------|--------|----------|-----------------------------------------------------------------------|
| `min`    | int    | No       | Minimum number of replicas.                                           |
| `max`    | int    | No       | Maximum number of replicas.                                           |
| `metric` | string | No      | The metric to scale on (e.g. `"cpu"`, `"memory"`, `"requests"`).      |
| `target` | int    | No       | Target utilization percentage that triggers scaling (e.g. `80`).      |

### Valid Metric Values

| Value        | Description                              |
|--------------|------------------------------------------|
| `"cpu"`      | Scale based on CPU utilization            |
| `"memory"`   | Scale based on memory utilization         |
| `"requests"` | Scale based on request rate               |

---

## Examples

### Process Target

The simplest deployment target. Runs the agent as a local process, ideal for development and testing.

```ias
package "local-deploy" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a helpful assistant."
}

skill "greet" {
  description "Greet the user"
  input { name string required }
  output { message string }
  tool command { binary "greet-tool" }
}

agent "assistant" {
  uses prompt "system"
  uses skill "greet"
  model "claude-sonnet-4-20250514"
}

deploy "local" target "process" {
  default true
  port 8080
  health {
    path "/healthz"
  }
}
```

### Docker Target

Run the agent as a standalone Docker container with resource limits.

```ias
package "docker-deploy" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a containerized assistant."
}

agent "assistant" {
  uses prompt "system"
  model "claude-sonnet-4-20250514"
}

deploy "dev" target "process" {
  default true
}

deploy "staging" target "docker" {
  image "agentspec/assistant:0.1.0"
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

### Docker Compose Target

Deploy as part of a multi-service Docker Compose stack.

```ias
package "compose-deploy" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a production assistant."
}

skill "query" {
  description "Query the database"
  input { q string required }
  output { data string }
  tool command { binary "query-tool" }
}

agent "data-bot" {
  uses prompt "system"
  uses skill "query"
  model "claude-sonnet-4-20250514"
}

secret "db-url" {
  env(DATABASE_URL)
}

deploy "local" target "process" {
  default true
}

deploy "compose" target "docker-compose" {
  image "agentspec/data-bot:0.1.0"
  port 8080
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
    DATABASE_URL "db-url"
  }
}
```

### Kubernetes Target

Full Kubernetes deployment with namespaces, replicas, resource limits, and autoscaling.

```ias
package "k8s-deploy" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a production-grade assistant running on Kubernetes."
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
  strategy "react"
  max_turns 15
  timeout "60s"
  on_error "retry"
  max_retries 3
}

secret "api-key" {
  env(API_KEY)
}

deploy "dev" target "process" {
  default true
  port 8080
}

deploy "production" target "kubernetes" {
  namespace "agents"
  image "agentspec/assistant:0.1.0"
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
    ENVIRONMENT "production"
  }
  secrets {
    API_KEY "api-key"
  }
}
```

!!! tip "Default Target"
    Mark exactly one `deploy` block as `default true`. This is the target used when you run `agentspec apply` without specifying a target name. Typically the `"process"` target is set as default for local development.

!!! warning "Target-Specific Attributes"
    Some attributes are only meaningful for certain targets. For example, `namespace` and `autoscale` only apply to `"kubernetes"`. Using them with other targets will not cause an error but will have no effect.

---

## See Also

- [Deployment Overview](../deployment/index.md) -- Deployment workflow, target comparison, and getting started
- [Docker Deployment](../deployment/docker.md) -- Detailed guide for the `"docker"` target
- [Docker Compose Deployment](../deployment/compose.md) -- Detailed guide for the `"docker-compose"` target
- [Kubernetes Deployment](../deployment/kubernetes.md) -- Detailed guide for the `"kubernetes"` target
- [Deployment Best Practices](../deployment/best-practices.md) -- Production readiness and operational guidance
- [CLI: apply](../cli/apply.md) -- The `agentspec apply` command that executes deployments
