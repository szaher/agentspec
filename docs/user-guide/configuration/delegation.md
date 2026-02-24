# Agent Delegation

Agent delegation allows one agent to route requests to other specialist agents based on conditions. This enables multi-agent architectures where a coordinator (often called a router or triage agent) analyzes incoming requests and dispatches them to the most appropriate handler.

---

## Syntax

Delegation rules are declared inside the `agent` block using the `delegate` keyword:

```ias novalidate
delegate to agent "<agent-name>" when "<condition>"
```

| Component | Description |
|-----------|-------------|
| `agent "<agent-name>"` | The name of the target agent to delegate to. Must reference an agent defined in the same package. |
| `when "<condition>"` | A natural-language description of when this delegation should occur. The routing agent uses this description to decide which delegate to invoke. |

An agent can have multiple `delegate` rules:

```ias novalidate
agent "router" {
  uses prompt "triage"
  model "claude-sonnet-4-20250514"
  delegate to agent "tech-agent" when "user has a technical issue"
  delegate to agent "billing-agent" when "user has a billing question"
  delegate to agent "general-agent" when "user has a general inquiry"
}
```

---

## Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `delegate` | rule | No | Delegation rule. Can appear multiple times within a single agent block. |
| `to agent` | string | Yes (per rule) | Name of the target agent. |
| `when` | string | Yes (per rule) | Natural-language condition describing when to delegate. |

---

## How Delegation Works

1. The router agent receives a user request.
2. The router's LLM evaluates the request against all `when` conditions.
3. The LLM selects the most appropriate delegate based on the condition descriptions.
4. The runtime invokes the selected delegate agent with the original request.
5. The delegate agent processes the request and returns its response.
6. The response is returned to the caller.

!!! info "Condition matching is LLM-driven"
    The `when` conditions are natural-language descriptions, not programmatic rules. The routing agent's LLM interprets the request and matches it against the conditions. Write conditions that are clear and distinct to avoid ambiguous routing.

---

## Router Pattern

The most common delegation pattern is the **router** (or triage) pattern: a central agent that classifies incoming requests and dispatches them to specialists.

```ias
package "support-router" version "0.1.0" lang "2.0"

prompt "triage" {
  content "You are a request router. Analyze the user's request and
           delegate it to the most appropriate specialist agent.
           Consider the topic, complexity, and required expertise
           when making routing decisions."
}

prompt "tech-support" {
  content "You are a technical support specialist. Help users
           resolve technical issues with software and hardware.
           Ask clarifying questions when needed."
}

prompt "billing" {
  content "You are a billing specialist. Help users with
           invoices, payments, refunds, and subscription management.
           Always verify account details before making changes."
}

prompt "general" {
  content "You are a general assistant. Handle inquiries that
           do not require specialized expertise. Be helpful and
           direct the user to specialists if needed."
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

skill "check-system-status" {
  description "Check the status of internal systems and services"
  input {
    service string required
  }
  output {
    status string
  }
  tool command {
    binary "status-check"
  }
}

skill "process-refund" {
  description "Process a customer refund"
  input {
    order_id string required
    reason string required
  }
  output {
    refund_id string
  }
  tool command {
    binary "refund-tool"
  }
}

agent "tech-agent" {
  uses prompt "tech-support"
  uses skill "check-system-status"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 10
}

agent "billing-agent" {
  uses prompt "billing"
  uses skill "lookup-account"
  uses skill "process-refund"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 10
}

agent "general-agent" {
  uses prompt "general"
  model "claude-sonnet-4-20250514"
  max_turns 5
}

agent "router" {
  uses prompt "triage"
  model "claude-sonnet-4-20250514"
  delegate to agent "tech-agent" when "user has a technical issue"
  delegate to agent "billing-agent" when "user has a billing question"
  delegate to agent "general-agent" when "user has a general inquiry"
}

deploy "local" target "process" {
  default true
  port 8080
}
```

!!! tip "Router agent strategy"
    While the router agent works with the default `"react"` strategy, you can also set `strategy "router"` to use the optimized routing execution loop, which is designed specifically for classification and dispatch.

---

## Condition Strings

Condition strings are natural-language descriptions that the routing agent's LLM uses to determine which delegate to invoke. Write conditions that are:

- **Specific** -- Clearly describe the type of request each delegate handles.
- **Distinct** -- Avoid overlap between conditions to minimize ambiguous routing.
- **Action-oriented** -- Describe what the user is trying to do, not implementation details.

### Good Conditions

```ias novalidate
delegate to agent "tech-agent" when "user reports a bug or needs help with software"
delegate to agent "billing-agent" when "user asks about invoices, payments, or subscriptions"
delegate to agent "hr-agent" when "user has a question about company policies or benefits"
```

### Avoid Vague Conditions

```ias novalidate
# Too vague -- these overlap significantly
delegate to agent "agent-a" when "user needs help"
delegate to agent "agent-b" when "user has a question"
```

!!! warning "Ambiguous conditions"
    If conditions overlap, the routing LLM may make inconsistent routing decisions. Test your delegation rules with a variety of inputs to verify correct classification.

---

## Multiple Delegates

An agent can delegate to any number of specialist agents. There is no hard limit on the number of `delegate` rules per agent.

```ias novalidate
agent "project-manager" {
  uses prompt "manager"
  model "claude-sonnet-4-20250514"
  delegate to agent "research-agent" when "research is needed"
  delegate to agent "writing-agent" when "writing is needed"
  delegate to agent "review-agent" when "review is needed"
  delegate to agent "design-agent" when "design work is needed"
  delegate to agent "qa-agent" when "testing or quality assurance is needed"
}
```

Each delegate agent operates independently with its own prompt, skills, model, and runtime configuration.

---

## Delegation vs. Pipeline

IntentLang provides two mechanisms for multi-agent coordination: delegation and pipelines. Choose the right one based on your use case.

| Aspect | Delegation | Pipeline |
|--------|-----------|----------|
| **Control flow** | Dynamic, LLM-driven routing based on request content. | Static, predefined sequence of steps. |
| **Use case** | Request classification, triage, routing to specialists. | Multi-step workflows with known stages. |
| **Execution order** | Single delegate selected per request. | All steps execute in declared order. |
| **Data flow** | Original request forwarded to delegate. | Each step receives output from previous step(s). |
| **Parallelism** | One agent at a time. | Steps can run in parallel with `depends_on`. |

### When to Use Delegation

- The system must classify requests and route them to different handlers.
- The set of possible actions depends on user intent.
- Only one specialist should handle any given request.

### When to Use Pipeline

- A task requires multiple sequential processing stages.
- Every input must go through the same set of steps.
- Steps have explicit data dependencies.

```ias novalidate
# Delegation: route to one specialist
agent "router" {
  delegate to agent "analyst" when "data analysis needed"
  delegate to agent "writer" when "content creation needed"
}

# Pipeline: every request goes through all steps
pipeline "review-workflow" {
  step "analyze" {
    agent "code-analyzer"
    input "source code"
    output "analysis"
  }
  step "report" {
    agent "report-writer"
    depends_on ["analyze"]
    output "review report"
  }
}
```

---

## Delegation with Error Handling

Delegate agents can have their own error handling configuration. If a delegate fails, its `on_error` strategy applies independently:

```ias novalidate
agent "primary-handler" {
  uses prompt "primary"
  model "claude-sonnet-4-20250514"
  on_error "retry"
  max_retries 2
}

agent "router" {
  uses prompt "triage"
  model "claude-sonnet-4-20250514"
  delegate to agent "primary-handler" when "standard request"
  delegate to agent "fallback-handler" when "request cannot be classified"
}
```

!!! tip "Combine delegation with fallback"
    For critical systems, configure delegate agents with their own fallback agents. This creates a layered resilience model: the router selects the right specialist, and each specialist has its own error recovery strategy.

---

## Validation Rules

The `agentspec validate` command enforces:

- Every agent referenced in a `delegate to agent` rule must exist in the same package.
- Circular delegation chains (agent A delegates to B, which delegates back to A) are detected and rejected.
- The `when` condition must be a non-empty string.

---

## See Also

- [agent](../language/agent.md) -- Full agent block reference
- [pipeline](../language/pipeline.md) -- Sequential multi-step workflows
- [Agent Runtime Configuration](runtime.md) -- Model, strategy, and other runtime attributes
- [Error Handling](error-handling.md) -- Configuring retry, fail, and fallback strategies
- [Router / Triage Use Case](../use-cases/router.md) -- Detailed router architecture walkthrough
