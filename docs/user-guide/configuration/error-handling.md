# Error Handling Configuration

Error handling attributes control how an agent responds when it encounters failures during execution. IntentLang provides three strategies -- retry, immediate failure, and fallback delegation -- that can be combined to build resilient agent systems.

---

## Syntax Overview

Error handling is configured inside the `agent` block using the `on_error`, `max_retries`, and `fallback` attributes:

```ias novalidate
agent "<name>" {
  on_error "<strategy>"
  max_retries <integer>
  fallback "<agent-name>"
}
```

---

## Attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `on_error` | string | `"fail"` | Error handling strategy. One of `"retry"`, `"fail"`, or `"fallback"`. |
| `max_retries` | integer | `0` | Number of retry attempts before giving up. Only used when `on_error` is `"retry"`. |
| `fallback` | string | *(none)* | Name of the agent to delegate to on failure. Required when `on_error` is `"fallback"`. |

---

## Strategies

### retry

When `on_error` is set to `"retry"`, the agent re-attempts execution up to `max_retries` times after a failure. Each retry starts a fresh invocation with the same input.

```ias novalidate
agent "resilient" {
  model "claude-sonnet-4-20250514"
  on_error "retry"
  max_retries 3
}
```

**Behavior:**

1. The agent encounters an error (model API failure, timeout, tool error).
2. The runtime waits briefly, then re-invokes the agent with the original input.
3. If the agent succeeds on any retry, the result is returned normally.
4. If all retries are exhausted, the invocation fails with the last error.

!!! tip "When to use retry"
    Use `"retry"` for transient failures such as network timeouts, rate limits, or intermittent API errors. It is not effective for deterministic failures like invalid input or missing resources.

### fail

When `on_error` is set to `"fail"` (or when `on_error` is not specified), the agent fails immediately upon encountering an error. No retries or fallbacks are attempted.

```ias novalidate
agent "strict" {
  model "claude-sonnet-4-20250514"
  on_error "fail"
}
```

**Behavior:**

1. The agent encounters an error.
2. The error is returned immediately to the caller.
3. No additional processing occurs.

!!! info "Default behavior"
    `"fail"` is the default strategy. If you omit `on_error` entirely, the agent behaves as if `on_error "fail"` were set.

### fallback

When `on_error` is set to `"fallback"`, the agent delegates to a different agent upon failure. The fallback agent receives the original input and handles the request independently.

```ias novalidate
agent "primary" {
  model "claude-sonnet-4-20250514"
  on_error "fallback"
  fallback "backup"
}
```

**Behavior:**

1. The primary agent encounters an error.
2. The runtime invokes the fallback agent with the same input.
3. The fallback agent's response is returned to the caller.
4. If the fallback agent also fails, its own `on_error` strategy applies.

!!! warning "Fallback requires a defined agent"
    The `fallback` attribute must reference an agent that exists in the same package. Validation fails if the referenced agent is not declared.

---

## Examples

### Retry with Limited Attempts

A common pattern for agents that call external APIs:

```ias
package "retry-example" version "0.1.0" lang "2.0"

prompt "api-caller" {
  content "You are an assistant that retrieves data from external APIs.
           If a request fails, report the error clearly."
}

skill "fetch-data" {
  description "Fetch data from an external API"
  input {
    endpoint string required
  }
  output {
    data string
  }
  tool command {
    binary "api-fetch"
  }
}

agent "data-fetcher" {
  uses prompt "api-caller"
  uses skill "fetch-data"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 5
  timeout "30s"
  on_error "retry"
  max_retries 3
}

deploy "local" target "process" {
  default true
}
```

### Fallback to a Simpler Agent

When the primary agent fails, a simpler fallback agent handles the request:

```ias
package "fallback-example" version "0.1.0" lang "2.0"

prompt "primary-system" {
  content "You are an advanced research assistant with access to
           multiple tools. Provide thorough, detailed answers."
}

prompt "fallback-system" {
  content "You are a simple assistant. The primary agent encountered
           an error. Provide the best answer you can without tools."
}

skill "web-search" {
  description "Search the web"
  input {
    query string required
  }
  output {
    results string
  }
  tool command {
    binary "search-tool"
  }
}

agent "advanced-researcher" {
  uses prompt "primary-system"
  uses skill "web-search"
  model "claude-sonnet-4-20250514"
  strategy "plan-and-execute"
  max_turns 15
  timeout "60s"
  on_error "fallback"
  fallback "simple-responder"
}

agent "simple-responder" {
  uses prompt "fallback-system"
  model "claude-haiku-latest"
  strategy "react"
  max_turns 3
  timeout "15s"
  on_error "fail"
}

deploy "local" target "process" {
  default true
}
```

!!! tip "Fallback agents should be simpler"
    Design fallback agents to be more reliable than the primary agent. Use a faster model, fewer tools, and shorter timeouts. The goal is to provide a degraded but functional response rather than a complete failure.

### Combining Retry and Fallback

For maximum resilience, you can chain strategies: the primary agent retries, and if all retries fail, a separate fallback agent with its own error handling takes over.

```ias
package "resilient-system" version "0.1.0" lang "2.0"

prompt "primary" {
  content "You are a resilient agent with access to external tools."
}

prompt "backup" {
  content "You are the fallback agent. The primary agent could not
           complete the request after multiple attempts."
}

skill "process-data" {
  description "Process incoming data"
  input {
    payload string required
  }
  output {
    result string
  }
  tool command {
    binary "data-processor"
  }
}

agent "primary-agent" {
  uses prompt "primary"
  uses skill "process-data"
  model "claude-sonnet-4-20250514"
  max_turns 10
  timeout "45s"
  on_error "retry"
  max_retries 2
}

agent "backup-agent" {
  uses prompt "backup"
  model "claude-haiku-latest"
  max_turns 3
  timeout "15s"
  on_error "fail"
}

deploy "local" target "process" {
  default true
}
```

!!! info "Retry then fallback"
    IntentLang does not have a built-in "retry then fallback" composite strategy in a single agent. To achieve this pattern, use a pipeline or delegation layer that first invokes the retrying agent and, upon final failure, routes to the fallback agent.

---

## Best Practices

| Scenario | Recommended Strategy | Notes |
|----------|---------------------|-------|
| External API calls | `retry` with `max_retries 2-3` | Handles transient network and rate-limit errors. |
| Strict correctness required | `fail` | Fail fast so the caller can handle the error explicitly. |
| User-facing chat | `fallback` | Avoid showing raw errors to users; provide a graceful degraded response. |
| Pipeline steps | `fail` | Let the pipeline orchestrator handle step failures. |
| Cost-sensitive workloads | `fallback` to a cheaper model | Use an expensive model first, fall back to a cheaper one on failure. |

---

## Validation Rules

The `agentspec validate` command enforces:

- If `on_error` is `"fallback"`, the `fallback` attribute must be present and must reference an existing agent in the package.
- If `on_error` is `"retry"`, `max_retries` should be set (defaults to `0` if omitted, which effectively means no retries).
- The `fallback` attribute is ignored if `on_error` is not `"fallback"`.
- Circular fallback chains (agent A falls back to B, which falls back to A) are detected and rejected.

---

## See Also

- [agent](../language/agent.md) -- Full agent block reference
- [Agent Runtime Configuration](runtime.md) -- Model, strategy, timeout, and other runtime attributes
- [Agent Delegation](delegation.md) -- Routing requests between agents
