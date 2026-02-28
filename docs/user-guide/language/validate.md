# Validate

The `validate` block declares output validation rules for an agent. Rules are evaluated against the agent's response at runtime and can trigger errors or warnings based on configurable conditions.

---

## Syntax

<!-- novalidate -->
```ias
agent "<name>" {
  validate {
    rule <name> <severity>
      "<message>"
      when <expression>
  }
}
```

The `validate` block is nested inside an `agent` block and contains one or more `rule` declarations.

---

## Rule Attributes

| Attribute | Required | Description                                                        |
|-----------|----------|--------------------------------------------------------------------|
| name      | Yes      | Rule identifier. Must be unique within the validate block.          |
| severity  | Yes      | `error` or `warning`. Determines behavior when the rule triggers.   |
| message   | Yes      | Human-readable description of the validation (quoted string).       |
| `when`    | No       | Expression that determines when the rule is evaluated.              |

---

## Severity Levels

| Severity  | Behavior                                                                          |
|-----------|-----------------------------------------------------------------------------------|
| `error`   | The agent's response is rejected. If `max_retries` is set, the agent retries.      |
| `warning` | The validation result is logged but the response is still delivered to the user.    |

!!! warning "Error Severity and Retries"
    When a rule with `error` severity triggers, the agent is prompted to fix the issue and regenerate its response. The `max_retries` attribute on the validation rule controls how many times this can happen before the response is rejected entirely.

---

## When Expressions

The `when` clause uses [expr](https://expr-lang.org/) syntax to define conditions. The following variables are available in the expression context:

| Variable | Type   | Description                           |
|----------|--------|---------------------------------------|
| `output` | string | The agent's generated response text.  |
| `input`  | string | The original user input.              |

Common expression patterns:

```
when output != ""           # Run on any non-empty output
when len(output) > 1000     # Run when output exceeds length
when input contains "urgent" # Run for specific input patterns
```

If no `when` clause is provided, the rule is always evaluated.

---

## Examples

### Warning Rule

A rule that logs a warning when the agent's response is non-empty:

<!-- novalidate -->
```ias
agent "coder" {
  uses prompt "coder-system"
  model "ollama/llama3.1"
  validate {
    rule no_secrets warning
      "Response should not expose secrets or credentials"
      when output != ""
  }
}
```

### Multiple Rules with Mixed Severity

<!-- novalidate -->
```ias
agent "support-agent" {
  uses prompt "support-system"
  uses skill "knowledge-search"
  model "claude-sonnet-4-5-20250514"
  strategy "react"
  max_turns 10
  validate {
    rule no_pii error
      "Response must not contain personally identifiable information"
      when output != ""
    rule response_length warning
      "Response exceeds maximum length"
      when output != ""
    rule professional_tone warning
      "Response should maintain professional tone"
      when output != ""
  }
}
```

In this example, `no_pii` is an `error` rule -- if PII is detected in the output, the response is rejected and the agent must regenerate it. The `response_length` and `professional_tone` rules are warnings that are logged but do not block the response.

---

## Runtime Behavior

1. The agent generates a response.
2. Each validation rule's `when` expression is evaluated against the response.
3. Rules whose `when` condition is true are checked.
4. `warning` rules log the message and continue.
5. `error` rules reject the response. The agent receives the error message and regenerates, up to `max_retries` times.
6. If all retries are exhausted for an `error` rule, the agent fails.

---

## See Also

- [Agent](agent.md) -- The parent block that contains `validate`
- [Config](config.md) -- Runtime configuration parameters within agents
- [Eval](eval.md) -- Evaluation test cases within agents
- [Error Handling](../configuration/error-handling.md) -- Retry and fallback strategies
