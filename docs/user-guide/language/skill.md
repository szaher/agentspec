# Skill

The `skill` block defines a capability that an agent can invoke. A skill has a description (used by the LLM to understand when to call it), typed input and output schemas, and a `tool` block that specifies the underlying implementation.

---

## Syntax

```ias novalidate
skill "<name>" {
  description "<text>"
  input {
    <field-name> <type> [required]
  }
  output {
    <field-name> <type>
  }
  tool <variant> {
    ...
  }
}
```

---

## Attributes

| Name          | Type   | Required | Description                                                            |
|---------------|--------|----------|------------------------------------------------------------------------|
| `description` | string | Yes      | Human-readable description of what the skill does. The LLM uses this to decide when to invoke the skill. |
| `input`       | block  | Yes      | Input schema. Defines the fields the skill expects.                    |
| `output`      | block  | Yes      | Output schema. Defines the fields the skill returns.                   |
| `tool`        | block  | Yes      | Tool implementation. One of four variants: `mcp`, `http`, `command`, or `inline`. |

---

## Input and Output Schemas

The `input` and `output` blocks define typed field schemas. Each field has a name, a type, and an optional `required` modifier.

### Field Syntax

```ias novalidate
input {
  <name> <type> [required]
}
```

### Supported Field Types

| Type     | Description                          |
|----------|--------------------------------------|
| `string` | Text value                           |
| `int`    | Integer number                       |
| `float`  | Floating-point number                |
| `bool`   | Boolean (`true`/`false`)             |

### Required Fields

Adding the `required` keyword after the type makes the field mandatory. The runtime will reject skill invocations that omit required fields.

```ias novalidate
input {
  query string required    # Must be provided
  limit int                # Optional
  verbose bool             # Optional
}
```

!!! tip "Schema Design"
    Keep input schemas minimal. Only mark fields as `required` when the skill cannot function without them. This gives the LLM more flexibility in how it invokes the skill.

---

## Tool Variants

The `tool` block specifies how the skill is executed. IntentLang supports four tool variants, each suited to different integration patterns.

| Variant   | Syntax                          | Use Case                                          |
|-----------|---------------------------------|---------------------------------------------------|
| `mcp`     | `tool mcp "<server/tool>"`      | Call a tool exposed by an MCP server.              |
| `http`    | `tool http { ... }`            | Make an HTTP request to an external API.           |
| `command` | `tool command { ... }`         | Execute a local binary or script.                  |
| `inline`  | `tool inline { ... }`          | Run embedded code in a sandboxed WASM environment. |

For full details on each variant, including all attributes and examples, see the [Tool Reference](tool.md).

### Quick Reference

**MCP tool** -- delegates to an MCP server:

```ias novalidate
tool mcp "file-server/read-file"
```

**HTTP tool** -- calls an external API:

```ias novalidate
tool http {
  method "POST"
  url "https://api.example.com/search"
  headers {
    Authorization "Bearer {{API_KEY}}"
    Content-Type "application/json"
  }
  body_template "{\"query\": \"{{query}}\"}"
  timeout "10s"
}
```

**Command tool** -- runs a local binary:

```ias novalidate
tool command {
  binary "search-tool"
  args ["--format", "json"]
  timeout "15s"
}
```

**Inline tool** -- executes embedded code:

```ias novalidate
tool inline {
  language "python"
  code "import json\nresult = json.dumps({'message': f'Hello, {input[\"name\"]}!'})\nprint(result)"
  timeout "5s"
  memory 64
}
```

---

## Examples

### Skill with Command Tool

A skill that searches the web using a local binary:

```ias
package "search-agent" version "0.1.0" lang "2.0"

prompt "researcher" {
  content "You are a research assistant. Use available tools to\ngather information and synthesize clear answers."
}

skill "web-search" {
  description "Search the web for information on a given query"
  input {
    query string required
  }
  output {
    results string
  }
  tool command {
    binary "search-tool"
    args ["--format", "json"]
    timeout "15s"
  }
}

agent "researcher" {
  uses prompt "researcher"
  uses skill "web-search"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 10
}

deploy "local" target "process" {
  default true
}
```

### Multiple Skills

An agent with several skills demonstrating different tool types:

```ias
package "multi-tool" version "0.1.0" lang "2.0"

prompt "assistant" {
  content "You are a versatile assistant with access to search,\nsummarization, and translation tools."
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

skill "summarize" {
  description "Summarize a body of text into key points"
  input {
    text string required
    max_length int
  }
  output {
    summary string
  }
  tool command {
    binary "summarize-tool"
  }
}

skill "translate" {
  description "Translate text to a target language"
  input {
    text string required
    target_language string required
  }
  output {
    translated string
  }
  tool command {
    binary "translate-tool"
  }
}

agent "assistant" {
  uses prompt "assistant"
  uses skill "web-search"
  uses skill "summarize"
  uses skill "translate"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 15
  timeout "60s"
}

deploy "local" target "process" {
  default true
}
```

!!! note "Skill Naming"
    Skill names must be unique within a package. Use descriptive, hyphenated names (e.g. `"web-search"`, `"lookup-ticket"`, `"process-refund"`) to make agent logs and traces easy to follow.
