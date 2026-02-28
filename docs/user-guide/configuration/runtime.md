# Agent Runtime Configuration

Agent runtime attributes control how an agent executes: which model it uses, how it reasons, how long it can run, and how it handles its token budget. These attributes are set inside an `agent` block.

---

## Syntax Overview

<!-- novalidate -->
```ias
agent "<name>" {
  uses prompt "<prompt-name>"
  model "<model-identifier>"
  strategy "<strategy-name>"
  max_turns <integer>
  timeout "<duration>"
  token_budget <integer>
  temperature <float>
  stream <boolean>
}
```

All runtime attributes are optional except `model`. If omitted, each attribute falls back to its default value.

---

## Attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `model` | string | *(required)* | LLM model identifier for this agent. |
| `strategy` | string | `"react"` | Execution strategy that governs the agent's reasoning loop. |
| `max_turns` | integer | `10` | Maximum number of conversation turns before the agent stops. |
| `timeout` | string | `"30s"` | Maximum wall-clock duration for a single invocation. |
| `token_budget` | integer | `100000` | Maximum number of tokens the agent may consume per invocation. |
| `temperature` | float | `1.0` | LLM sampling temperature, controlling response randomness. |
| `stream` | boolean | `false` | Whether the agent streams response tokens incrementally. |

---

## model

The `model` attribute specifies which LLM the agent uses. The value is a model identifier string recognized by the underlying provider.

<!-- novalidate -->
```ias
agent "analyst" {
  model "claude-sonnet-4-20250514"
}
```

### Common Model Identifiers

| Model | Description | Use Case |
|-------|-------------|----------|
| `claude-sonnet-4-20250514` | Claude Sonnet 4 | General-purpose, balanced cost and quality |
| `claude-haiku-latest` | Claude Haiku (latest) | Fast responses, lower cost, simpler tasks |
| `claude-opus-4-20250514` | Claude Opus 4 | Highest quality, complex reasoning |

!!! tip "Environment overrides"
    Use `environment` blocks to swap models per deployment stage. For example, use `claude-haiku-latest` in development for speed and `claude-sonnet-4-20250514` in production for quality.

<!-- novalidate -->
    ```ias
    environment "dev" {
      agent "analyst" {
        model "claude-haiku-latest"
      }
    }
    ```

---

## strategy

The `strategy` attribute determines how the agent reasons and acts. Each strategy implements a different execution loop suited to particular task types.

<!-- novalidate -->
```ias
agent "planner" {
  model "claude-sonnet-4-20250514"
  strategy "plan-and-execute"
}
```

### Available Strategies

| Strategy | Description | Best For |
|----------|-------------|----------|
| `react` | Reason-Act loop. The agent alternates between reasoning about the current state and taking actions (calling skills). This is the default strategy. | General-purpose tasks, interactive conversations, tool-use scenarios. |
| `plan-and-execute` | The agent first creates a complete plan, then executes each step sequentially. Produces more structured, predictable results for multi-step tasks. | Complex tasks with clear sub-steps, research workflows, document generation. |
| `reflexion` | Self-critique and revision loop. After producing an initial result, the agent evaluates its own output and iterates to improve quality. | Tasks requiring high accuracy, code generation, writing where quality matters. |
| `router` | Routes incoming requests to specialist agents based on the request content. Does not perform tasks itself. | Triage systems, multi-department support, request classification. |
| `map-reduce` | Splits work into parallel sub-tasks (map), processes each independently, then aggregates results (reduce). | Batch processing, document analysis across multiple files, parallel data processing. |

!!! info "Strategy and delegation"
    The `router` strategy is typically combined with `delegate` rules to direct requests to specialist agents. See [Agent Delegation](delegation.md) for details.

---

## max_turns

The `max_turns` attribute limits how many conversation turns (request-response cycles) the agent can take in a single invocation. When the limit is reached, the agent returns its current best response.

<!-- novalidate -->
```ias
agent "researcher" {
  model "claude-sonnet-4-20250514"
  max_turns 20
}
```

| Value | Behavior |
|-------|----------|
| Low (1-5) | Quick, direct responses. Suitable for simple Q&A or routing agents. |
| Medium (5-15) | Balanced. Allows multi-step reasoning and several tool calls. Good default range. |
| High (15-50) | Extended reasoning. For complex research, multi-tool workflows, or iterative refinement. |

!!! warning "Cost implications"
    Higher `max_turns` values allow the agent to consume more tokens and make more tool calls. Set this value based on the expected complexity of the task to avoid unnecessary cost.

---

## timeout

The `timeout` attribute sets the maximum wall-clock duration for a single agent invocation. The value is a duration string with a numeric amount followed by a unit suffix.

<!-- novalidate -->
```ias
agent "fast-responder" {
  model "claude-haiku-latest"
  timeout "10s"
}
```

### Duration Format

| Suffix | Unit | Example |
|--------|------|---------|
| `s` | Seconds | `"30s"` |
| `m` | Minutes | `"5m"` |

When the timeout is reached, the agent stops execution and returns whatever partial result is available. The invocation is marked as timed out in the response metadata.

!!! tip "Matching timeout to strategy"
    Set longer timeouts for strategies that involve multiple steps or iterations:

    - `react`: `"30s"` to `"60s"` is typical
    - `plan-and-execute`: `"60s"` to `"5m"` for complex plans
    - `reflexion`: `"60s"` to `"2m"` to allow revision cycles

---

## token_budget

The `token_budget` attribute caps the total number of tokens (input + output) the agent may consume in a single invocation. This provides cost control independent of the turn limit.

<!-- novalidate -->
```ias
agent "budget-conscious" {
  model "claude-sonnet-4-20250514"
  token_budget 50000
}
```

When the token budget is exhausted, the agent completes its current response and stops. Unlike `max_turns`, which counts discrete interactions, `token_budget` provides fine-grained cost control regardless of how many turns occur.

| Range | Use Case |
|-------|----------|
| 10,000 - 50,000 | Simple tasks, short conversations |
| 50,000 - 200,000 | Multi-step reasoning, moderate tool use |
| 200,000 - 500,000 | Extended research, complex analysis, long documents |

---

## temperature

The `temperature` attribute controls the randomness of LLM responses. The value is a float between `0` and `2`.

<!-- novalidate -->
```ias
agent "creative-writer" {
  model "claude-sonnet-4-20250514"
  temperature 1.5
}
```

| Range | Behavior |
|-------|----------|
| 0.0 - 0.3 | Highly deterministic. Nearly identical outputs for identical inputs. Best for factual tasks, code generation, structured data extraction. |
| 0.3 - 0.7 | Balanced. Some variation while remaining focused. Good default range for most tasks. |
| 0.7 - 1.2 | More creative and varied responses. Suitable for brainstorming, creative writing, generating diverse options. |
| 1.2 - 2.0 | Highly random. Responses may be unexpected or less coherent. Use with caution. |

!!! warning "Temperature and reproducibility"
    Lower temperatures produce more reproducible results. If your workflow relies on deterministic outputs (e.g., structured extraction, code generation), set the temperature to `0` or close to it.

---

## stream

The `stream` attribute controls whether the agent sends response tokens incrementally as they are generated, rather than waiting for the complete response.

<!-- novalidate -->
```ias
agent "interactive-bot" {
  model "claude-sonnet-4-20250514"
  stream true
}
```

| Value | Behavior |
|-------|----------|
| `true` | Tokens are sent to the client as they are generated. The user sees the response appear word by word. |
| `false` | The complete response is buffered and sent as a single payload when generation finishes. |

**When to enable streaming:**

- Interactive chat interfaces where users expect real-time feedback
- Long-running responses where early visibility is valuable
- Applications that display partial progress

**When to disable streaming:**

- Pipeline steps where the full output is needed before the next step begins
- Batch processing where latency per token is irrelevant
- Downstream systems that expect complete payloads

---

## Complete Example

An agent with all runtime attributes configured:

```ias
package "configured-agent" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a research assistant with access to web search
           and document analysis tools. Provide thorough, well-cited
           answers."
}

skill "web-search" {
  description "Search the web for information"
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

agent "researcher" {
  uses prompt "system"
  uses skill "web-search"
  model "claude-sonnet-4-20250514"
  strategy "plan-and-execute"
  max_turns 20
  timeout "2m"
  token_budget 200000
  temperature 0.3
  stream true
}

deploy "local" target "process" {
  default true
}
```

---

## See Also

- [agent](../language/agent.md) -- Full agent block reference
- [Error Handling](error-handling.md) -- Configuring `on_error`, `max_retries`, and `fallback`
- [Agent Delegation](delegation.md) -- Routing and delegating between agents
- [environment](../language/environment.md) -- Overriding runtime attributes per deployment stage
