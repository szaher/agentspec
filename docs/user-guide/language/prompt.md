# Prompt

The `prompt` block defines a reusable prompt template that agents reference via `uses prompt`. Prompts contain the system instructions that shape an agent's behavior, personality, and constraints. They support template variables for dynamic content injection at runtime.

---

## Syntax

<!-- novalidate -->
```ias
prompt "<name>" {
  content "<prompt-text>"
  version "<version>"
  variables {
    <name> <type> [required] [default "<value>"]
  }
}
```

---

## Attributes

| Name        | Type   | Required | Description                                                              |
|-------------|--------|----------|--------------------------------------------------------------------------|
| `content`   | string | Yes      | The prompt text. Supports `\n` for newlines and `{{variable}}` template syntax. |
| `version`   | string | No       | Version identifier for the prompt, useful for A/B testing and audit trails. |
| `variables` | block  | No       | Template variable definitions. Each variable has a name, type, and optional modifiers. |

---

## Template Variables

Prompts can include `{{variable}}` placeholders that are resolved at runtime. Each variable referenced in the `content` string should be declared in the `variables` block.

### Variable Definition Syntax

<!-- novalidate -->
```ias
variables {
  <name> <type> [required] [default "<value>"]
}
```

| Component  | Description                                                                     |
|------------|---------------------------------------------------------------------------------|
| `name`     | The variable name. Must match the `{{name}}` placeholder in the content string. |
| `type`     | The data type: `string`, `int`, `float`, or `bool`.                             |
| `required` | Optional modifier. Marks the variable as mandatory -- the runtime will reject execution if not provided. |
| `default`  | Optional modifier. Provides a fallback value when the variable is not supplied.  |

!!! note "Required vs Default"
    A variable can be `required`, have a `default`, or neither (optional with no default). A variable cannot be both `required` and have a `default` -- if a default value exists, the variable is always satisfiable.

### Supported Variable Types

| Type     | Description                     | Example Default         |
|----------|---------------------------------|-------------------------|
| `string` | Text value                      | `default "en"`          |
| `int`    | Integer value                   | `default "10"`          |
| `float`  | Floating-point value            | `default "0.7"`         |
| `bool`   | Boolean value (`true`/`false`)  | `default "true"`        |

---

## Content String

The `content` attribute is a double-quoted string that forms the system prompt for the agent. It supports the standard IntentLang escape sequences:

- `\n` -- newline (for multi-line prompts)
- `\t` -- tab
- `\"` -- literal double quote
- `\\` -- literal backslash

Multi-line prompts are written as a single string with embedded `\n` characters:

<!-- novalidate -->
```ias
content "You are a customer support agent for Acme Corp.\nBe empathetic, concise, and solution-oriented.\nAlways greet the customer by name when available."
```

!!! tip "Readability"
    For very long prompts, consider breaking the content across multiple `\n`-separated lines within a single string. This keeps the prompt readable while remaining a valid single-line string literal.

---

## Examples

### Simple Prompt

A prompt with no variables:

```ias
package "simple-prompt" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a helpful assistant. Answer questions clearly and concisely."
}

agent "assistant" {
  uses prompt "system"
  model "claude-sonnet-4-20250514"
}

deploy "local" target "process" {
  default true
}
```

### Prompt with Template Variables

A prompt that adapts its behavior based on runtime variables:

```ias
package "dynamic-prompt" version "0.1.0" lang "2.0"

prompt "support" {
  content "You are a {{role}} for {{company}}.\nYou speak {{language}}.\nTone: {{tone}}.\nAlways greet the customer by name when available."
  version "1.2.0"
  variables {
    role string required
    company string required
    language string default "English"
    tone string default "professional and empathetic"
  }
}

skill "lookup-ticket" {
  description "Look up a customer support ticket by ID"
  input {
    ticket_id string required
  }
  output {
    ticket string
  }
  tool command {
    binary "ticket-lookup"
  }
}

agent "support-bot" {
  uses prompt "support"
  uses skill "lookup-ticket"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 15
  timeout "60s"
}

deploy "local" target "process" {
  default true
}
```

In this example, `role` and `company` must be provided at runtime, while `language` and `tone` have sensible defaults that can be overridden.

### Multi-Line Prompt

A detailed prompt for a code review agent:

```ias
package "code-review-prompt" version "0.1.0" lang "2.0"

prompt "analyzer" {
  content "You are a code analysis expert. Identify code smells,\nanti-patterns, and suggest improvements. Focus on\nreadability, maintainability, and performance.\n\nWhen reviewing code:\n1. Start with a high-level summary\n2. List issues by severity (critical, high, medium, low)\n3. Provide specific, actionable suggestions\n4. Include code examples for recommended fixes"
}

agent "code-analyzer" {
  uses prompt "analyzer"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 10
}

deploy "local" target "process" {
  default true
}
```

!!! warning "Variable Resolution"
    If a `content` string contains `{{variable}}` placeholders but the corresponding variables are not declared in the `variables` block, validation will emit a warning. Undeclared variables are treated as literal text and will not be substituted at runtime.
