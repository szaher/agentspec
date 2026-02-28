# IntentLang Extensions Contract

**Feature**: 006-agent-compile-deploy

## Overview

IntentLang 3.0 introduces imports, control flow, configuration parameters, validation rules, and evaluation test cases. This contract defines the syntax and semantics of these language extensions.

## Import Statements

Imports appear at the top of an `.ias` file, after the `package` declaration and before any resource definitions.

### Local File Import

```
import "./skills/search.ias"
import "./prompts/support.ias"
import "../shared/types.ias"
```

- Path is relative to the importing file
- Must use `./` or `../` prefix (no bare names for local files)
- File extension `.ias` is required
- Imported definitions are available by their declared names

### Package Import

```
import "github.com/agentspec/web-tools" version "1.2.0"
import "github.com/user/custom-skills" version "^2.0.0"
```

- Package path follows `host/namespace/name` pattern
- Version is required (Constitution Principle XI: Explicit References)
- Version supports semver constraints: exact (`1.2.0`), caret (`^1.2.0`), tilde (`~1.2.0`)
- Imported definitions are namespaced by package name

### Aliased Import

```
import "github.com/agentspec/web-tools" version "1.2.0" as web
```

- Alias provides a short name for referencing imported definitions
- Usage: `web.search_skill` instead of full qualified name

### Import Resolution Order

1. Check local cache (`~/.agentspec/cache/`)
2. Check lock file (`.agentspec.lock`) for pinned version
3. Resolve from registry or Git repository
4. Verify checksum
5. Cache locally

---

## Control Flow Blocks

Control flow operates at runtime within agent definitions, pipeline steps, or skill orchestration.

### Conditional (if/else if/else)

```
agent support_agent {
  prompt system_prompt
  model "claude-sonnet-4-20250514"

  on input {
    if input.category == "billing" {
      use skill billing_handler
    } else if input.category == "technical" {
      use skill tech_support
    } else {
      use skill general_support
    }
  }
}
```

**Syntax**:
```
if <expression> {
  <statements>
} else if <expression> {
  <statements>
} else {
  <statements>
}
```

- `else` block is required (or compiler emits a warning per FR-011)
- Expressions use `expr` syntax: property access, comparisons, boolean logic, `in` operator
- Expressions evaluate against runtime context: `input`, `session`, `output`, `steps`

### For-Each Loop

```
agent data_processor {
  prompt processor_prompt
  model "claude-sonnet-4-20250514"

  on input {
    for each source in input.data_sources {
      use skill fetch_data with { url: source.url }
      use skill transform with { format: source.format }
    }
    use skill aggregate_results
  }
}
```

**Syntax**:
```
for each <variable> in <collection_expression> {
  <statements>
}
```

- Collection expression must resolve to a list/array at runtime
- Loop variable is scoped to the loop body
- Results from each iteration are collected and available after the loop

### Expression Syntax

Expressions follow `expr-lang/expr` syntax, restricted to safe operations:

| Operation | Syntax | Example |
|-----------|--------|---------|
| Property access | `.` | `input.category` |
| Comparison | `==`, `!=`, `>`, `<`, `>=`, `<=` | `input.priority >= 5` |
| Boolean logic | `and`, `or`, `not` | `input.urgent and not input.resolved` |
| Containment | `in` | `input.type in ["A", "B"]` |
| Type check | `is` | `input.data is string` |
| Null check | `== nil` | `input.optional == nil` |
| String match | `matches` | `input.email matches ".*@example.com"` |

**Not allowed**: Function calls, arithmetic (except comparisons), variable assignment, I/O operations, loops within expressions.

---

## Configuration Parameters

Declared within agent or package-level blocks.

```
agent support_agent {
  config {
    anthropic_api_key string required secret
      "Anthropic API key for LLM access"

    escalation_email string default "support@example.com"
      "Email address for escalated tickets"

    max_response_length int default 2000
      "Maximum response length in characters"

    debug_mode bool default false
      "Enable verbose debug logging"
  }

  prompt system_prompt
  model "claude-sonnet-4-20250514"
}
```

**Syntax**:
```
config {
  <name> <type> [required] [secret] [default <value>]
    "<description>"
}
```

**Types**: `string`, `int`, `float`, `bool`

**Modifiers**: `required` (must be provided at runtime), `secret` (never logged, never has defaults)

**Rules**:
- Parameters with the `secret` modifier never have defaults and are never logged
- `required` parameters without defaults cause fast-fail at startup
- Parameters map to environment variables: `AGENTSPEC_<AGENT>_<PARAM>` (uppercase, underscores)
- Parameters can also be provided via config file (YAML or JSON)

---

## Validation Rules

Declared within agent definitions. Run automatically on every agent response.

```
agent support_agent {
  prompt system_prompt
  model "claude-sonnet-4-20250514"

  validate {
    rule no_pii error max_retries 3
      "Response must not contain PII"
      when not (output matches "\\b\\d{3}-\\d{2}-\\d{4}\\b")
        and not (output matches "\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}\\b")

    rule tone_check warning
      "Response should maintain professional tone"
      when output.sentiment != "negative"

    rule max_length error max_retries 1
      "Response must be under configured character limit"
      when len(output) <= config.max_response_length
  }
}
```

**Syntax**:
```
validate {
  rule <name> <severity> [max_retries <n>]
    "<description>"
    when <expression>
}
```

**Severity levels**:
- `error`: Fail and retry (up to `max_retries`). After max retries, return best response with validation failure details.
- `warning`: Log the warning but return the response.

---

## Evaluation Test Cases

Declared within agent definitions or in companion `.ias` files.

```
agent support_agent {
  prompt system_prompt
  model "claude-sonnet-4-20250514"

  eval {
    case greeting_test
      input "Hello, I need help"
      expect "greeting response with offer to help"
      scoring semantic threshold 0.8
      tags ["smoke", "greeting"]

    case refund_request
      input "I want a refund for order #12345"
      expect "acknowledgment of refund request with next steps"
      scoring semantic threshold 0.8
      tags ["billing", "refund"]

    case empty_input
      input ""
      expect "polite request for more information"
      scoring semantic threshold 0.7
      tags ["edge-case"]
  }
}
```

**Syntax**:
```
eval {
  case <name>
    input "<input text>"
    expect "<expected output description>"
    scoring <method> [threshold <0.0-1.0>]
    [tags [<tag list>]]
}
```

**Scoring methods**:
- `exact`: Exact string match
- `contains`: Expected string is substring of output
- `semantic`: Semantic similarity using embedding comparison (default threshold: 0.8)
- `custom`: Custom scoring expression using `expr` syntax

---

## On-Input Block

The `on input` block defines the agent's request processing flow. It replaces the implicit "invoke agent with prompt + skills" model with explicit control flow.

```
agent router_agent {
  prompt router_prompt
  model "claude-sonnet-4-20250514"

  on input {
    use skill classify with { text: input.content }

    if steps.classify.output.category == "technical" {
      delegate to tech_agent
    } else {
      use skill respond with { context: steps.classify.output }
    }
  }
}
```

**Available statements within `on input`**:
- `use skill <name> [with { <params> }]` — invoke a skill
- `delegate to <agent_name>` — hand off to another agent
- `if/else if/else` — conditional branching
- `for each` — iteration
- `respond <expression>` — return a response directly

**Runtime context variables**:
- `input` — the current request input
- `session` — session state and memory
- `config` — configuration parameter values
- `steps` — results from previously executed skills in this request
- `output` — the current response being built
