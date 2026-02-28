# Agent

The `agent` block defines an AI agent -- the central resource type in IntentLang. An agent binds a language model to a prompt and one or more skills, with configurable behavior for strategy, error handling, and multi-agent delegation.

---

## Syntax

<!-- novalidate -->
```ias
agent "<name>" {
  model "<model-identifier>"
  uses prompt "<prompt-name>"
  uses skill "<skill-name>"
  strategy "<strategy>"
  max_turns <int>
  timeout "<duration>"
  token_budget <int>
  temperature <float>
  stream <bool>
  on_error "<error-action>"
  max_retries <int>
  fallback "<agent-name>"
  delegate to agent "<agent-name>" when "<condition>"
}
```

---

## Attributes

| Name          | Type     | Required   | Description                                                                 |
|---------------|----------|------------|-----------------------------------------------------------------------------|
| `model`       | string   | Yes        | LLM model identifier (e.g. `"claude-sonnet-4-20250514"`).                   |
| `uses prompt` | ref      | Yes        | Reference to a `prompt` resource. Exactly one prompt per agent.              |
| `uses skill`  | ref      | No         | Reference to a `skill` resource. Repeatable for multiple skills.             |
| `strategy`    | string   | No         | Agent execution strategy. Default: `"react"`.                                |
| `max_turns`   | int      | No         | Maximum number of conversation turns before the agent stops.                 |
| `timeout`     | string   | No         | Maximum wall-clock time for the agent session (e.g. `"30s"`, `"2m"`).        |
| `token_budget`| int      | No         | Maximum total tokens (input + output) the agent may consume.                 |
| `temperature` | float    | No         | Sampling temperature for the LLM. Range: `0` to `2`.                        |
| `stream`      | bool     | No         | Enable streaming output from the LLM. Default: `false`.                      |
| `on_error`    | string   | No         | Error handling strategy. Default: `"fail"`.                                  |
| `max_retries` | int      | No         | Number of retry attempts when `on_error` is `"retry"`.                       |
| `fallback`    | string   | No         | Name of the fallback agent. Required when `on_error` is `"fallback"`.        |
| `delegate to agent` | ref | No      | Delegation rule. Repeatable. Routes to another agent based on a condition.   |

---

## Strategy Values

The `strategy` attribute controls how the agent reasons and acts. Each strategy implements a different agentic architecture.

| Value               | Description                                                                  |
|---------------------|------------------------------------------------------------------------------|
| `"react"`           | **ReAct** (Reason + Act). The agent observes, reasons about available tools, acts, and repeats. This is the default strategy. |
| `"plan-and-execute"` | The agent first creates a structured plan, then executes each step sequentially, verifying results along the way. |
| `"reflexion"`       | The agent drafts output, self-critiques, and iteratively revises. Useful for high-quality writing and complex analysis. |
| `"router"`          | The agent acts as a dispatcher, analyzing requests and routing them to specialized sub-agents. Typically used with `delegate to agent` rules. |
| `"map-reduce"`      | The agent processes data in parallel chunks (map phase) and then combines results (reduce phase). Used in pipeline contexts. |

!!! tip "Choosing a Strategy"
    - Use `"react"` for general-purpose agents that need tool access.
    - Use `"plan-and-execute"` when tasks require multi-step planning with verification.
    - Use `"reflexion"` when output quality is critical and iterative refinement is acceptable.
    - Use `"router"` with delegation rules for multi-agent triage patterns.

---

## Error Handling Values

The `on_error` attribute determines what happens when the agent encounters an error during execution.

| Value        | Description                                                                 |
|--------------|-----------------------------------------------------------------------------|
| `"fail"`     | Stop execution immediately and report the error. This is the default.        |
| `"retry"`    | Retry the failed operation up to `max_retries` times.                        |
| `"fallback"` | Hand off to the agent specified by the `fallback` attribute.                 |

!!! warning "Fallback Requires a Target"
    When `on_error` is set to `"fallback"`, you **must** also specify the `fallback` attribute with the name of an existing agent. Validation will fail otherwise.

---

## Delegation

Delegation allows an agent to route subtasks to specialized agents based on natural-language conditions. Each `delegate to agent` directive specifies a target agent and a condition string that the agent evaluates at runtime.

<!-- novalidate -->
```ias
agent "manager" {
  uses prompt "manager-prompt"
  model "claude-sonnet-4-20250514"
  delegate to agent "research-agent" when "research is needed"
  delegate to agent "writing-agent" when "writing is needed"
  delegate to agent "review-agent" when "review is needed"
}
```

Delegation rules are evaluated in order. The first matching condition triggers routing to the corresponding agent. A delegating agent typically uses the `"router"` strategy but can use any strategy.

---

## Examples

### Basic Agent

A minimal agent with a prompt, a single skill, and sensible defaults:

```ias
package "basic-agent" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a helpful assistant. Answer questions clearly and concisely."
}

skill "greet" {
  description "Greet the user by name"
  input {
    name string required
  }
  output {
    message string
  }
  tool command {
    binary "greet-tool"
  }
}

agent "assistant" {
  uses prompt "system"
  uses skill "greet"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 10
  timeout "30s"
  token_budget 100000
  stream true
  on_error "retry"
  max_retries 2
}

deploy "local" target "process" {
  default true
  port 8080
  health {
    path "/healthz"
  }
}
```

### Multi-Agent Delegation

A project manager agent that delegates to specialist agents:

```ias
package "delegation" version "0.1.0" lang "2.0"

prompt "manager" {
  content "You are a project manager agent. Break down complex tasks\nand delegate them to specialist agents. Monitor progress\nand coordinate between agents to ensure quality results."
}

prompt "researcher" {
  content "You are a research specialist. Gather information from\navailable sources and produce thorough, well-cited reports."
}

prompt "writer" {
  content "You are a writing specialist. Produce clear, well-structured\ndocuments based on research findings and requirements."
}

skill "search" {
  description "Search for information on a topic"
  input {
    topic string required
  }
  output {
    findings string
  }
  tool command {
    binary "search-tool"
  }
}

skill "write-doc" {
  description "Write a document section"
  input {
    topic string required
    findings string required
  }
  output {
    document string
  }
  tool command {
    binary "write-tool"
  }
}

agent "research-agent" {
  uses prompt "researcher"
  uses skill "search"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 10
}

agent "writing-agent" {
  uses prompt "writer"
  uses skill "write-doc"
  model "claude-sonnet-4-20250514"
  max_turns 5
}

agent "project-manager" {
  uses prompt "manager"
  model "claude-sonnet-4-20250514"
  delegate to agent "research-agent" when "research is needed"
  delegate to agent "writing-agent" when "writing is needed"
}

deploy "local" target "process" {
  default true
  port 8080
}
```

### Reflexion Agent

An agent that iteratively improves its output through self-critique:

```ias
package "reflexion" version "0.1.0" lang "2.0"

prompt "writer" {
  content "You are a writing assistant that produces high-quality text.\nAfter each draft, reflect on your work: identify weaknesses,\nmissing information, and areas for improvement. Then revise\nyour output based on your self-critique."
}

skill "draft" {
  description "Generate an initial draft of text"
  input {
    topic string required
    requirements string required
  }
  output {
    draft string
  }
  tool command {
    binary "draft-tool"
  }
}

skill "critique" {
  description "Critically evaluate a piece of text"
  input {
    text string required
  }
  output {
    feedback string
  }
  tool command {
    binary "critique-tool"
  }
}

skill "revise" {
  description "Revise text based on feedback"
  input {
    text string required
    feedback string required
  }
  output {
    revised string
  }
  tool command {
    binary "revise-tool"
  }
}

agent "reflective-writer" {
  uses prompt "writer"
  uses skill "draft"
  uses skill "critique"
  uses skill "revise"
  model "claude-sonnet-4-20250514"
  strategy "reflexion"
  max_turns 10
  timeout "90s"
  token_budget 250000
  on_error "retry"
  max_retries 2
}

deploy "local" target "process" {
  default true
}
```

---

## See Also

- [Core Concepts](../getting-started/concepts.md) -- Mental model for agents, strategies, and composition
- [Agentic Architecture Patterns](../use-cases/index.md) -- Choosing the right pattern for your use case
- [ReAct Agent](../use-cases/react.md) -- The default `strategy "react"` in practice
- [Plan-and-Execute](../use-cases/plan-execute.md) -- Using `strategy "plan-and-execute"` for multi-step tasks
- [Reflexion](../use-cases/reflexion.md) -- Using `strategy "reflexion"` for iterative self-improvement
- [Router / Triage](../use-cases/router.md) -- Using `delegate to agent` for request routing
- [Agent Delegation](../use-cases/delegation.md) -- Using `delegate to agent` for dynamic multi-agent coordination
- [Agent Runtime Configuration](../configuration/runtime.md) -- Tuning strategy, timeouts, and token budgets
- [Error Handling](../configuration/error-handling.md) -- Retry and fallback strategies
