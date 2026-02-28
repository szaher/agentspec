# eval

Run evaluation test cases against agents.

## Usage

```bash
agentspec eval [file.ias]
```

## Description

The `eval` command runs declared evaluation test cases against agents defined in `.ias` files. This validates agent quality by comparing actual outputs against expected outputs using configurable scoring methods.

Evaluation cases are defined inline within the `.ias` file using `eval` blocks. Each case specifies an input, an expected output, and a scoring method. The command parses the spec, invokes each agent with the test inputs, and scores the results.

You can filter which agents and test cases to run using `--agent` and `--tags`. Results can be exported to a file and compared against a previous run to track quality over time.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--agent` | | | Evaluate a specific agent by name |
| `--tags` | | | Filter eval cases by tags (comma-separated) |
| `--output` | `-o` | *(stdout)* | Write report to a file |
| `--format` | | `table` | Output format: `table`, `json`, `markdown` |
| `--compare` | | | Path to a previous eval report (JSON) for comparison |

## Scoring Methods

Eval cases support the following scoring methods to compare actual output against expected output:

| Method | Description |
|--------|-------------|
| `exact` | Actual output must exactly match expected output |
| `contains` | Expected string must appear somewhere in actual output |
| `semantic` | Uses semantic similarity to compare meaning (tolerant of phrasing differences) |
| `regex` | Expected value is treated as a regular expression and matched against actual output |

## Eval Block in .ias Files

Evaluation test cases are defined inside the agent block in your `.ias` file:

```
agent "support-bot" {
  model = "gpt-4"
  prompt = "You are a helpful support agent."

  eval "greeting-test" {
    input    = "Hello"
    expected = "Hello! How can I help you today?"
    score    = "contains"
    tags     = ["smoke", "greeting"]
  }

  eval "refund-policy" {
    input    = "What is your refund policy?"
    expected = "30-day money-back guarantee"
    score    = "semantic"
    tags     = ["policy"]
  }
}
```

## Examples

```bash
# Run all eval cases in a spec
agentspec eval agent.ias

# Evaluate a specific agent
agentspec eval --agent support-bot agent.ias

# Filter by tags
agentspec eval --tags smoke,greeting agent.ias

# Output results as JSON
agentspec eval --format json agent.ias

# Write a markdown report to a file
agentspec eval --format markdown --output report.md agent.ias

# Compare against a previous run
agentspec eval --format json --output current.json agent.ias
agentspec eval --compare current.json agent.ias
```

## Output Formats

### Table (default)

```
Agent: support-bot

  Case              Score   Result
  greeting-test     PASS    contains match
  refund-policy     PASS    semantic similarity: 0.92

  2/2 passed (100%)
```

### JSON

Structured output suitable for CI pipelines and automated analysis. Use `--format json` to enable.

### Markdown

A formatted report suitable for documentation or pull request comments. Use `--format markdown` to enable.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All eval cases passed |
| `1` | One or more eval cases failed, or an error occurred |

## See Also

- [CLI: run](run.md) -- Run an agent interactively
- [CLI: compile](compile.md) -- Compile agents for deployment
