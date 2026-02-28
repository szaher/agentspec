# Prompt Template Variables

Prompt templates in IntentLang support variable interpolation using the `{{variable}}` syntax. Variables allow you to create reusable prompts whose content is customized at runtime without modifying the `.ias` file.

---

## Syntax Overview

Variables are referenced in prompt `content` strings using double curly braces, and declared in a `variables` block within the prompt:

```ias
prompt "<name>" {
  content "Text with {{variable_name}} placeholders."
  variables {
    variable_name <type> <modifiers>
  }
}
```

---

## Variable Declaration

Each variable is declared on its own line inside the `variables` block. A declaration consists of a name, a type, and optional modifiers.

```ias
variables {
  name string required
  role string required default "assistant"
  language string default "English"
  verbose bool
}
```

### Declaration Syntax

```
<name> <type> [required] [default "<value>"]
```

| Component | Required | Description |
|-----------|----------|-------------|
| `name` | Yes | Variable identifier. Must be a valid IntentLang identifier (letters, digits, underscores). |
| `type` | Yes | Data type of the variable. Typically `string`, but `int`, `bool`, and custom types are also supported. |
| `required` | No | Marks the variable as mandatory. The runtime returns an error if a required variable is not provided. |
| `default` | No | Provides a fallback value used when the variable is not supplied at runtime. |

---

## Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `content` | string | The prompt text containing `{{variable}}` placeholders. |
| `variables` | block | Variable declarations with types and modifiers. |
| `required` | modifier | Marks a variable as mandatory at runtime. |
| `default` | modifier | Sets a fallback value for the variable (e.g., `default "value"`). |

---

## Variable Resolution

When an agent is invoked, variables are resolved in the following order:

1. **Explicit values** -- Variables provided directly in the invocation request take highest priority.
2. **Default values** -- If a variable is not provided but has a `default` modifier, the default value is used.
3. **Error** -- If a `required` variable has no explicit value and no default, the runtime returns a validation error before the agent executes.

!!! info "Variables without `required` or `default`"
    A variable declared without either modifier is optional and has no default. If not provided at runtime, the `{{variable}}` placeholder is replaced with an empty string.

---

## Examples

### Basic Variable Usage

A prompt with a single required variable:

```ias
package "greeter" version "0.1.0" lang "2.0"

prompt "welcome" {
  content "Hello {{name}}, welcome to our service."
  variables {
    name string required
  }
}

agent "greeter" {
  uses prompt "welcome"
  model "claude-sonnet-4-20250514"
}

deploy "local" target "process" {
  default true
}
```

When the agent is invoked, the caller must supply the `name` variable:

```json
{
  "variables": {
    "name": "Alice"
  }
}
```

The resolved prompt becomes: `Hello Alice, welcome to our service.`

### Required with Default

A variable can be both `required` and have a `default`. The `required` modifier documents that the variable is significant, while the `default` ensures the prompt is valid even without an explicit value:

```ias
prompt "greeting" {
  content "Hello {{name}}, you are a {{role}}."
  variables {
    name string required
    role string required default "assistant"
  }
}
```

If invoked with only `name = "Bob"`:

- `{{name}}` resolves to `"Bob"`
- `{{role}}` resolves to `"assistant"` (the default)

Result: `Hello Bob, you are a assistant.`

### Multiple Variables with Mixed Modifiers

```ias
prompt "report-generator" {
  content "Generate a {{report_type}} report for {{company}} covering
           the period {{period}}. Use {{language}} language.
           Detail level: {{detail_level}}."
  variables {
    report_type string required
    company string required
    period string required default "last quarter"
    language string default "English"
    detail_level string default "standard"
  }
}
```

In this example:

- `report_type` and `company` must always be provided.
- `period` defaults to `"last quarter"` if not specified.
- `language` defaults to `"English"`.
- `detail_level` defaults to `"standard"`.

### Persona Template

Variables are particularly useful for creating reusable persona prompts:

```ias
package "persona-template" version "0.1.0" lang "2.0"

prompt "persona" {
  content "You are {{agent_name}}, a {{specialty}} specialist at {{company}}.
           Your communication style is {{tone}}.
           Always respond in {{language}}."
  variables {
    agent_name string required default "Assistant"
    specialty string required
    company string required
    tone string default "professional and concise"
    language string default "English"
  }
}

skill "respond" {
  description "Generate a response"
  input {
    message string required
  }
  output {
    reply string
  }
  tool command {
    binary "respond-tool"
  }
}

agent "specialist" {
  uses prompt "persona"
  uses skill "respond"
  model "claude-sonnet-4-20250514"
}

deploy "local" target "process" {
  default true
}
```

!!! tip "Prompt reuse across agents"
    Multiple agents can reference the same prompt template with different variable values. Define the prompt once and let each agent's invocation supply the appropriate context.

---

## Validation

The `agentspec validate` command checks that:

- Every `{{variable}}` placeholder in the `content` string has a corresponding declaration in the `variables` block.
- Every variable declared in `variables` is referenced at least once in the `content` string.
- Variable names are valid identifiers (no spaces, no special characters other than underscores).
- Default values are compatible with the declared type.

!!! warning "Undeclared variables"
    If a `{{variable}}` appears in the content but is not declared in the `variables` block, validation fails. This prevents typos from producing unexpected empty substitutions at runtime.

---

## See Also

- [prompt](../language/prompt.md) -- Full prompt block reference
- [agent](../language/agent.md) -- Agents that use prompts
- [Agent Runtime Configuration](runtime.md) -- Other agent configuration attributes
