# Process Deployment

The `process` target runs your agent as a local OS process. It is the simplest deployment target, ideal for development, testing, and debugging before moving to containerized or orchestrated environments.

---

## Prerequisites

- The `agentspec` CLI binary installed and available on your `PATH`.
- Go runtime (1.25+) if the agent uses command-based tools that require compilation.

---

## Deploy Block

A minimal process deployment requires only the target type:

```ias novalidate
deploy "local" target "process" {
  default true
  port 8080
  health {
    path "/healthz"
    interval "10s"
    timeout "3s"
  }
}
```

### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `default` | bool | Mark this as the default deploy target. |
| `port` | int | Port the agent listens on. |
| `health` | block | Health check configuration. |
| `env` | block | Environment variables passed to the process. |
| `secrets` | block | Secret mappings resolved at deploy time. |

!!! info "Attributes Not Applicable"
    The `image`, `namespace`, `replicas`, `resources`, and `autoscale` attributes have no effect on the `process` target. The agent runs as a single local process with access to all host resources.

---

## How It Works

When you run `agentspec apply` with a `process` target, AgentSpec:

1. Compiles or prepares the agent runtime based on the `.ias` definition.
2. Starts the agent as a local OS process bound to the configured port.
3. Begins health checking if a `health` block is configured.
4. Records the deployment state in `.agentspec.state.json`.

The process runs in the foreground by default. It can be backgrounded using standard shell techniques or managed by a process supervisor.

---

## Health Checking

The `health` block configures an HTTP health check endpoint. AgentSpec periodically sends a GET request to the configured path and expects an HTTP 200 response.

```ias novalidate
health {
  path "/healthz"
  interval "10s"
  timeout "3s"
}
```

- **`path`** -- The HTTP path to check (e.g. `"/healthz"`).
- **`interval`** -- How often AgentSpec polls the endpoint (e.g. `"10s"`).
- **`timeout`** -- Maximum time to wait for a response before marking the check as failed (e.g. `"3s"`).

If the health check fails repeatedly, AgentSpec logs a warning but does not automatically restart the process.

---

## Environment Variables

Use the `env` block to pass environment variables to the agent process:

```ias novalidate
deploy "local" target "process" {
  default true
  port 8080
  env {
    LOG_LEVEL "debug"
    ENVIRONMENT "development"
  }
}
```

Environment variables set in the `env` block are merged with the host environment. If a variable is defined in both, the `env` block value takes precedence.

---

## Secret Management

Use the `secrets` block to map environment variable names to declared `secret` resources:

```ias novalidate
secret "api-key" {
  env(API_KEY)
}

deploy "local" target "process" {
  default true
  port 8080
  secrets {
    AGENT_API_KEY "api-key"
  }
}
```

At deploy time, AgentSpec resolves the secret value from its source (environment variable or secret store) and injects it into the process environment as `AGENT_API_KEY`.

---

## Complete Example

A fully configured agent with a process deployment target:

```ias
package "local-agent" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a development assistant. Help users debug code,\nanswer questions about the codebase, and suggest improvements."
}

skill "analyze" {
  description "Analyze a code snippet for issues"
  input {
    code string required
    language string required
  }
  output {
    issues string
    suggestions string
  }
  tool command {
    binary "code-analyzer"
  }
}

agent "dev-helper" {
  uses prompt "system"
  uses skill "analyze"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 10
  timeout "30s"
}

secret "api-key" {
  env(ANTHROPIC_API_KEY)
}

deploy "local" target "process" {
  default true
  port 8080
  health {
    path "/healthz"
    interval "10s"
    timeout "3s"
  }
  env {
    LOG_LEVEL "debug"
  }
  secrets {
    ANTHROPIC_API_KEY "api-key"
  }
}
```

---

## Deploying

Validate, plan, and apply the deployment:

```bash
# Validate the .ias file
agentspec validate dev-helper.ias

# Preview changes
agentspec plan dev-helper.ias

# Apply the deployment
agentspec apply dev-helper.ias
```

---

## Verification

After applying, verify the agent is running:

### Health Check

```bash
curl http://localhost:8080/healthz
```

Expected response:

```json
{"status": "healthy"}
```

### Status Command

Check the deployment state using the AgentSpec CLI:

```bash
agentspec status
```

This reads `.agentspec.state.json` and reports the current state of all deployed resources.

---

## Stopping the Agent

To stop a process-based deployment, use the `destroy` command:

```bash
agentspec destroy dev-helper.ias
```

This stops the running process and removes the deployment state from `.agentspec.state.json`.

---

## When to Use Process Deployment

| Scenario | Recommended |
|----------|-------------|
| Local development and debugging | Yes |
| Running quick tests | Yes |
| CI pipeline test steps | Yes |
| Staging environment | No -- use [Docker](docker.md) |
| Production | No -- use [Kubernetes](kubernetes.md) |

---

## See Also

- [Deployment Overview](index.md) -- Compare all deployment targets
- [Docker Deployment](docker.md) -- Containerize your agent
- [Deploy Block Reference](../language/deploy.md) -- Full attribute reference
- [Best Practices](best-practices.md) -- Production readiness guidance
