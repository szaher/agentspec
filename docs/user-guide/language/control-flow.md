# Control Flow

The `on input` block defines an agent's request processing flow using structured control flow statements. It supports conditional branching, iteration, skill invocation, agent delegation, and direct responses.

---

## On Input Block

The `on input` block is nested inside an `agent` block and is evaluated each time the agent receives a message.

<!-- novalidate -->
```ias
agent "<name>" {
  on input {
    <statements>
  }
}
```

---

## Statements

The following statements are available inside an `on input` block.

### use skill

Invokes a skill by name. The skill must be declared with `uses skill` in the agent.

<!-- novalidate -->
```ias
use skill greet_user
```

### delegate to

Hands off processing to another agent by name. The target agent must be defined in the same package or an imported file.

<!-- novalidate -->
```ias
delegate to escalation-agent
```

### respond

Returns a response string directly without invoking the LLM.

<!-- novalidate -->
```ias
respond "I'll connect you with a specialist."
```

---

## Conditional Branching

### if / else if / else

Conditional blocks use [expr](https://expr-lang.org/) expressions to branch execution.

<!-- novalidate -->
```ias
on input {
  if <condition> {
    <statements>
  } else if <condition> {
    <statements>
  } else {
    <statements>
  }
}
```

The `else if` and `else` clauses are optional. Conditions are evaluated in order; the first matching branch executes.

#### Available Variables

| Variable | Type   | Description                          |
|----------|--------|--------------------------------------|
| `input`  | string | The current user input message.      |

#### Expression Examples

```
input == "hello"                    # Exact match
input == "help"                     # Exact match
input contains "urgent"             # Substring check
len(input) > 100                    # Length check
```

---

## Iteration

### for each

Iterates over a collection, executing the body for each item.

<!-- novalidate -->
```ias
on input {
  for each item in collection {
    use skill process_item
  }
}
```

| Attribute    | Description                                        |
|--------------|----------------------------------------------------|
| `item`       | Loop variable name, available inside the body.      |
| `collection` | An expr expression that evaluates to a list.        |

---

## Full Example

A router agent that directs requests based on input content:

```ias
package "router-agent" version "1.0.0" lang "3.0"

prompt "router-system" {
  content "You are a routing agent that directs requests to the appropriate handler."
}

skill "greet_user" {
  description "Greet the user warmly"
  input {
    name string required
  }
  output {
    greeting string required
  }
  tool command {
    binary "echo"
    args ["Hello, welcome!"]
  }
}

skill "search_faq" {
  description "Search the FAQ database"
  input {
    query string required
  }
  output {
    answer string required
  }
  tool command {
    binary "echo"
    args ["FAQ result for query"]
  }
}

skill "escalate" {
  description "Escalate to human support"
  input {
    reason string required
  }
  output {
    ticket string required
  }
  tool command {
    binary "echo"
    args ["Escalated to human support"]
  }
}

agent "router" {
  uses prompt "router-system"
  uses skill "greet_user"
  uses skill "search_faq"
  uses skill "escalate"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 5
  on input {
    if input == "hello" {
      use skill greet_user
    } else if input == "help" {
      use skill search_faq
    } else {
      use skill escalate
    }
  }
}
```

---

## Combining with Validate

Control flow and validation work together. The `on input` block determines which skill handles the request, while `validate` rules check the resulting output:

<!-- novalidate -->
```ias
agent "support-agent" {
  uses prompt "support-system"
  uses skill "knowledge-search"
  model "claude-sonnet-4-5-20250514"
  validate {
    rule no_pii error
      "Response must not contain personally identifiable information"
      when output != ""
  }
  on input {
    if input == "urgent" {
      use skill knowledge-search
    } else {
      use skill knowledge-search
    }
  }
}
```

---

## See Also

- [Agent](agent.md) -- The parent block that contains `on input`
- [Skill](skill.md) -- Skills invoked with `use skill`
- [Validate](validate.md) -- Output validation rules applied after control flow
- [Router / Triage](../use-cases/router.md) -- The router pattern in practice
