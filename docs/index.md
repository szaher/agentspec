# AgentSpec Documentation

**A declarative toolchain for defining, validating, and deploying AI agent systems.**

AgentSpec uses **IntentLang**, a purpose-built DSL, to describe agents, prompts, skills, MCP servers, and deployment targets in a single source of truth. Write what you want your agents to do — AgentSpec handles the rest.

---

## Who Is This For?

<div class="grid cards" markdown>

-   **:material-account-hard-hat: AI Engineers & Data Scientists**

    ---

    Build and deploy AI agents using a declarative language. Define agent behavior, tool integrations, and deployment targets without writing infrastructure code.

    [:octicons-arrow-right-24: User Guide](user-guide/getting-started/index.md)

-   **:material-code-braces: Contributors & Developers**

    ---

    Understand the AgentSpec architecture, extend the platform with custom adapters and WASM plugins, and contribute to the project.

    [:octicons-arrow-right-24: Developer Guide](developer-guide/architecture/index.md)

</div>

---

## What Is IntentLang?

IntentLang is a declarative language (`.ias` files) for defining complete AI agent systems. Every file starts with a package header and contains resource definitions:

```ias
package "hello" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a helpful assistant."
}

skill "greet" {
  description "Greet the user"
  input { name string required }
  output { message string }
  tool command { binary "greet-tool" }
}

agent "assistant" {
  uses prompt "system"
  uses skill "greet"
  model "claude-sonnet-4-20250514"
}

deploy "local" target "process" {
  default true
}
```

IntentLang 2.0 supports **13 resource types**: `agent`, `prompt`, `skill`, `tool`, `deploy`, `pipeline`, `type`, `server`, `client`, `secret`, `environment`, `policy`, and `plugin`.

[:octicons-arrow-right-24: Language Reference](user-guide/language/index.md)

---

## Key Features

### Desired-State Model

Declare the end state you want. AgentSpec computes the minimal set of changes to get there — creating, updating, or removing resources as needed. Every operation is **idempotent**.

### Validate Before You Deploy

```bash
agentspec validate my-agent.ias   # Check syntax and semantics
agentspec plan my-agent.ias       # Preview changes
agentspec apply my-agent.ias      # Apply idempotently
```

### Multiple Deployment Targets

Deploy the same agent definition to local processes, Docker, Docker Compose, or Kubernetes — without changing your `.ias` file.

```ias fragment
deploy "local" target "process" { default true }
deploy "staging" target "docker-compose" {}
deploy "production" target "kubernetes" {
  namespace "agents"
  replicas 3
}
```

### Agentic Architectures

Build ReAct agents, plan-and-execute workflows, multi-agent pipelines, router/triage patterns, and more — all declaratively.

[:octicons-arrow-right-24: Use Cases & Architectures](user-guide/use-cases/index.md)

### MCP Integration

Connect to Model Context Protocol servers and expose skills through standard transports.

### WASM Plugin System

Extend AgentSpec with sandboxed WebAssembly plugins for custom validation, transformation, and pre-deploy hooks.

---

## Getting Started

1. **Install AgentSpec** — Build from source with Go 1.25+
2. **Write your first `.ias` file** — Define an agent with a prompt and deployment target
3. **Validate and deploy** — Use the CLI to validate, plan, and apply

[:octicons-arrow-right-24: Quick Start Tutorial](user-guide/getting-started/quickstart.md)

---

## Architecture at a Glance

```
.ias source → Lexer → Parser → AST → Validator → IR → Plan → Apply
                                                    |          |
                                                    v          v
                                                 Export    State File
```

The toolchain processes `.ias` files through a deterministic pipeline. Running the same command twice produces byte-identical results.

[:octicons-arrow-right-24: Architecture Overview](developer-guide/architecture/index.md)
