# Getting Started

Welcome to AgentSpec, a declarative toolchain for defining, validating, and deploying AI agent systems. This guide walks you through installation, your first agent definition, and the core concepts you need to build production-ready agents.

---

## Prerequisites

!!! note "System Requirements"
    - **Go 1.25+** -- Required to build the `agentspec` CLI from source.
    - **git** -- Required to clone the repository.

Verify that both tools are available:

```bash
go version    # Should print go1.25 or later
git --version
```

---

## Installation

AgentSpec is installed by building the CLI binary from source.

### 1. Clone the Repository

```bash
git clone https://github.com/szaher/designs.git
cd designs/agentz
```

### 2. Build the CLI

```bash
go build -o agentspec ./cmd/agentspec
```

This produces an `agentspec` binary in the current directory.

!!! tip "Add to PATH"
    Move the binary to a directory on your `$PATH` so you can run it from anywhere:

    ```bash
    sudo mv agentspec /usr/local/bin/
    ```

### 3. Verify the Installation

```bash
agentspec version
```

You should see the current version printed to the terminal. If you see `command not found`, ensure the binary is on your `$PATH` or use `./agentspec version` from the build directory.

---

## What You Will Learn

This Getting Started section is organized into three pages:

<div class="grid cards" markdown>

-   **Quick Start**

    ---

    Create, validate, and apply your first `.ias` file in five minutes. A hands-on tutorial that covers every CLI command.

    [:octicons-arrow-right-24: Quick Start](quickstart.md)

-   **Core Concepts**

    ---

    Understand the mental model behind AgentSpec: packages, agents, prompts, skills, tools, pipelines, deployment targets, and the desired-state model.

    [:octicons-arrow-right-24: Core Concepts](concepts.md)

</div>

---

## Next Steps

After completing the Getting Started section, explore these areas:

- [Language Reference](../language/index.md) -- Full syntax and semantics for all 13 IntentLang resource types.
- [Agent Runtime Configuration](../configuration/runtime.md) -- Tune strategy, timeouts, token budgets, and more.
- [Error Handling](../configuration/error-handling.md) -- Configure retries, fallbacks, and failure modes.
