# Eval

The `eval` block declares evaluation test cases for an agent. Each case defines an input/expected-output pair with a scoring method, enabling automated quality checks via the `agentspec eval` command.

---

## Syntax

<!-- novalidate -->
```ias
agent "<name>" {
  eval {
    case <name>
      input "<input-text>"
      expect "<expected-text>"
      scoring <method>
      [threshold <float>]
      [tags "<tag1>,<tag2>"]
  }
}
```

The `eval` block is nested inside an `agent` block and contains one or more `case` declarations.

---

## Case Attributes

| Attribute   | Required | Description                                                        |
|-------------|----------|--------------------------------------------------------------------|
| name        | Yes      | Case identifier. Must be unique within the eval block.              |
| `input`     | Yes      | The input text sent to the agent (quoted string).                   |
| `expect`    | Yes      | The expected output or pattern to match against (quoted string).    |
| `scoring`   | Yes      | Scoring method used to compare actual output to expected.           |
| `threshold` | No       | Similarity threshold for `semantic` scoring. Default: `0.8`.       |
| `tags`      | No       | Comma-separated tags for filtering eval runs (quoted string).       |

---

## Scoring Methods

| Method     | Description                                                                     |
|------------|---------------------------------------------------------------------------------|
| `exact`    | The output must exactly match the `expect` string.                               |
| `contains` | The output must contain the `expect` string as a substring.                      |
| `semantic` | The output is compared to `expect` using embedding similarity. Passes if the similarity score meets or exceeds the `threshold`. |
| `regex`    | The `expect` string is treated as a regular expression. The output must match it. |

!!! tip "Choosing a Scoring Method"
    - Use `exact` for deterministic outputs like classifications or fixed responses.
    - Use `contains` for checking that key information appears in free-form responses.
    - Use `semantic` when the wording may vary but the meaning should match.
    - Use `regex` for structured output validation (e.g., JSON patterns, formatted strings).

---

## Running Evaluations

Run all eval cases defined across your agents:

```bash
agentspec eval
```

Filter by tags:

```bash
agentspec eval --tags "greeting,basic"
```

Run evals for a specific agent:

```bash
agentspec eval --agent "support-agent"
```

The command reports pass/fail status for each case along with the score.

---

## Examples

### Basic Eval Cases

<!-- novalidate -->
```ias
agent "coder" {
  uses prompt "coder-system"
  uses skill "list-files"
  model "ollama/llama3.1"
  eval {
    case help_request
      input "What files are in the current directory?"
      expect "list"
      scoring contains
    case code_question
      input "How do I read a file in Python?"
      expect "open"
      scoring contains
  }
}
```

### Eval Cases for a Support Agent

<!-- novalidate -->
```ias
agent "support-agent" {
  uses prompt "support-system"
  uses skill "knowledge-search"
  model "claude-sonnet-4-5-20250514"
  eval {
    case greeting
      input "Hello, I need help"
      expect "welcome"
      scoring contains
    case product_question
      input "What are your pricing plans?"
      expect "pricing"
      scoring contains
    case complaint_handling
      input "Your product is broken and I want a refund"
      expect "apologize"
      scoring contains
  }
}
```

### Semantic Scoring with Threshold

<!-- novalidate -->
```ias
agent "advisor" {
  uses prompt "advisor-system"
  model "claude-sonnet-4-20250514"
  eval {
    case investment_advice
      input "Should I invest in index funds?"
      expect "Index funds provide diversified, low-cost exposure to the market"
      scoring semantic
      threshold 0.7
      tags "finance,basic"
  }
}
```

---

## See Also

- [Agent](agent.md) -- The parent block that contains `eval`
- [Config](config.md) -- Runtime configuration parameters within agents
- [Validate](validate.md) -- Output validation rules within agents
