# Pipeline

A **pipeline** defines a multi-step workflow that orchestrates one or more agents in a
deterministic execution order. Each step invokes an agent, and steps can declare
dependencies on other steps to control sequencing. Steps without mutual dependencies
run in parallel by default.

---

## Syntax

<!-- novalidate -->
```ias
pipeline "<name>" {
  step "<step-name>" {
    agent   "<agent-ref>"
    input   "<description>"       # optional
    output  "<variable-name>"     # optional
    depends_on ["<step>", ...]    # optional
    parallel true                 # optional
  }
  # ... more steps
}
```

`<name>` is a unique identifier for the pipeline within the package.
Each `step` block declares a unit of work inside the pipeline.

---

## Step Attributes

| Attribute    | Type        | Required | Default | Description                                                        |
|:-------------|:------------|:---------|:--------|:-------------------------------------------------------------------|
| `agent`      | string      | Yes      | --      | Reference to the agent that executes this step.                    |
| `input`      | string      | No       | --      | Human-readable description of the input fed to the agent.          |
| `output`     | string      | No       | --      | Name of the output variable produced by this step.                 |
| `depends_on` | string list | No       | `[]`    | List of step names that must complete before this step can start.  |
| `parallel`   | bool        | No       | `false` | When `true`, this step is explicitly marked for parallel execution.|

!!! info "Implicit parallelism"
    Steps that share no `depends_on` relationship are eligible for parallel
    execution automatically. The `parallel` attribute provides an explicit
    annotation when you want to make intent clear to readers or to tooling.

---

## Rules

- Step names must be **unique within the pipeline**.
- A step may reference any agent defined in the same package.
- Circular dependencies are a validation error.
- A pipeline must contain **at least one step**.

!!! warning "Circular dependencies"
    `agentspec validate` will reject pipelines that contain cycles.
    For example, step A depending on step B while step B depends on step A
    is invalid.

---

## Examples

### Sequential Pipeline

A basic ETL pipeline where each step runs after the previous one completes.

```ias
package "etl" version "0.1.0" lang "2.0"

prompt "etl" {
  content "You are a data engineering assistant."
}

skill "extract" {
  description "Extract data from a source system"
  input  { source string required }
  output { raw_data string }
  tool command { binary "data-extract" }
}

skill "transform" {
  description "Transform data according to mapping rules"
  input  { data string required }
  output { transformed string }
  tool command { binary "data-transform" }
}

skill "load" {
  description "Load data into the target database"
  input  { data string required }
  output { rows_inserted string }
  tool command { binary "data-load" }
}

agent "etl-bot" {
  uses prompt "etl"
  uses skill "extract"
  uses skill "transform"
  uses skill "load"
  model "claude-sonnet-4-20250514"
}

pipeline "etl-pipeline" {
  step "extract" {
    agent "etl-bot"
    input "data source configuration"
    output "raw data"
  }
  step "transform" {
    agent "etl-bot"
    depends_on ["extract"]
    output "transformed data"
  }
  step "load" {
    agent "etl-bot"
    depends_on ["transform"]
    output "load confirmation"
  }
}

deploy "local" target "process" {
  default true
}
```

In this pipeline, `transform` waits for `extract`, and `load` waits for `transform`.
The execution order is always: extract, transform, load.

### Parallel Pipeline

Two independent analysis steps run concurrently before a final merge step.

<!-- fragment -->
```ias
pipeline "dual-analysis" {
  step "sentiment" {
    agent "sentiment-analyzer"
    input "customer feedback text"
    output "sentiment scores"
    parallel true
  }
  step "topic" {
    agent "topic-classifier"
    input "customer feedback text"
    output "topic labels"
    parallel true
  }
  step "merge" {
    agent "report-generator"
    depends_on ["sentiment", "topic"]
    output "combined report"
  }
}
```

`sentiment` and `topic` have no dependencies on each other, so they execute in
parallel. `merge` waits for both to finish before it runs.

### Fan-Out / Fan-In Pattern

A code review pipeline that fans out to multiple reviewers and fans in to a
summarizer.

<!-- fragment -->
```ias
pipeline "code-review" {
  step "analyze" {
    agent "code-analyzer"
    input "pull request URL"
    output "code analysis findings"
  }
  step "security" {
    agent "security-scanner"
    input "pull request URL"
    output "security findings"
  }
  step "performance" {
    agent "perf-reviewer"
    input "pull request URL"
    output "performance findings"
  }
  step "summarize" {
    agent "review-summarizer"
    depends_on ["analyze", "security", "performance"]
    output "final review summary"
  }
}
```

The first three steps (analyze, security, performance) are the **fan-out** phase --
they all start simultaneously because none depends on another. The `summarize`
step is the **fan-in** point -- it waits for all three to complete, then produces
a consolidated review.

!!! tip "Combining patterns"
    You can nest sequential chains inside fan-out branches. For example, the
    `analyze` step could itself depend on a `fetch-diff` step, creating a
    two-stage branch while other branches run independently.

---

## CLI Usage

Pipelines are validated and executed as part of the standard workflow:

```bash
# Validate pipeline structure and dependencies
agentspec validate my-pipeline.ias

# Preview what the pipeline will do
agentspec plan my-pipeline.ias

# Execute the pipeline
agentspec apply my-pipeline.ias
```

---

## See Also

- [Agent](agent.md) -- the resource type invoked by pipeline steps
- [Environment](environment.md) -- override agent attributes per environment
