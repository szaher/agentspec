# Core Concepts

This page explains the mental model behind AgentSpec. Understanding these concepts will help you design agent systems that are composable, reproducible, and production-ready.

---

## Package

A **package** is the unit of distribution in AgentSpec. Every `.ias` file begins with a package header that declares the package name, its semantic version, and the IntentLang version it targets.

<!-- novalidate -->
```ias
package "my-agent" version "1.0.0" lang "2.0"
```

The package header serves three purposes:

1. **Identity** -- The package name uniquely identifies the agent system.
2. **Versioning** -- Semantic versioning (`MAJOR.MINOR.PATCH`) tracks changes over time.
3. **Compatibility** -- The `lang` field pins the file to a specific IntentLang specification, ensuring toolchain compatibility.

A package contains one or more resource definitions. All resources within a package can reference each other by name.

---

## Agents

The **agent** is the central resource type in IntentLang. An agent binds a language model to a system prompt and a set of skills, with configurable behavior for reasoning strategy, error handling, and multi-agent delegation.

<!-- novalidate -->
```ias
agent "assistant" {
  uses prompt "system"
  uses skill "greet"
  model "claude-sonnet-4-20250514"
  strategy "react"
}
```

An agent does not contain business logic directly. Instead, it composes other resources -- prompts define what the agent knows, skills define what the agent can do, and the strategy defines how the agent reasons.

### Strategies

The `strategy` attribute controls the agent's reasoning loop. AgentSpec supports five strategies:

| Strategy | How It Works |
|----------|-------------|
| `react` | Reason-Act loop. Observe, think, act, repeat. The default for general-purpose agents. |
| `plan-and-execute` | Create a structured plan first, then execute each step with verification. |
| `reflexion` | Draft, self-critique, and revise iteratively. Best when output quality is critical. |
| `router` | Analyze the request and dispatch to specialized sub-agents. Used with delegation rules. |
| `map-reduce` | Split work into parallel chunks, process independently, then combine results. |

[:octicons-arrow-right-24: Agent Reference](../language/agent.md)

---

## Prompts

A **prompt** contains the system instructions that shape an agent's behavior. Prompts are declared as standalone resources and referenced by agents via `uses prompt`, making them reusable across multiple agents.

<!-- novalidate -->
```ias
prompt "support" {
  content "You are a {{role}} for {{company}}.\nBe empathetic and solution-oriented."
  variables {
    role string required
    company string required
  }
}
```

Prompts support **template variables** -- `{{variable}}` placeholders that are resolved at runtime. Variables can be marked `required` or given `default` values, enabling the same prompt to adapt to different contexts without duplication.

[:octicons-arrow-right-24: Prompt Reference](../language/prompt.md)

---

## Skills

A **skill** defines a capability that an agent can invoke. Each skill has a description (used by the LLM to decide when to call it), typed input and output schemas, and a tool implementation that specifies how the skill executes.

<!-- novalidate -->
```ias
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
```

Skills are the bridge between what an agent decides to do and how that action is carried out. The input/output schemas give the LLM a structured contract for invocation, while the tool block handles execution details.

[:octicons-arrow-right-24: Skill Reference](../language/skill.md)

---

## Tools

A **tool** is the execution backend for a skill. It specifies how a skill's action is actually performed. IntentLang supports four tool variants:

| Variant | Description | Use Case |
|---------|-------------|----------|
| `mcp` | Delegates to a tool exposed by an MCP (Model Context Protocol) server. | Integrating with MCP-compatible tool servers. |
| `http` | Makes an HTTP request to an external API. | Calling REST APIs, webhooks, and third-party services. |
| `command` | Executes a local binary or script. | Running existing tools, scripts, and CLIs. |
| `inline` | Runs embedded code in a sandboxed WASM environment. | Lightweight transformations and computations. |

The tool variant is chosen based on how the underlying capability is implemented. An agent does not know or care which variant a skill uses -- it interacts with skills through their typed schemas.

[:octicons-arrow-right-24: Tool Reference](../language/tool.md)

---

## Pipelines

A **pipeline** defines a multi-step workflow that orchestrates agents in a deterministic execution order. Each step invokes an agent and can declare dependencies on other steps to control sequencing.

<!-- novalidate -->
```ias
pipeline "review" {
  step "analyze" {
    agent "code-analyzer"
    output "analysis"
  }
  step "summarize" {
    agent "summarizer"
    depends_on ["analyze"]
    output "summary"
  }
}
```

Steps without mutual dependencies run in parallel by default. Steps with `depends_on` relationships run sequentially, creating fan-out/fan-in patterns for complex workflows.

[:octicons-arrow-right-24: Pipeline Reference](../language/pipeline.md)

---

## Deploy Targets

A **deploy** block defines where and how an agent system runs. A single `.ias` file can contain multiple deploy targets, allowing the same agent definition to target different environments without modification.

| Target | Description |
|--------|-------------|
| `process` | Run as a local OS process. The simplest target, ideal for development. |
| `docker` | Run as a standalone Docker container. |
| `docker-compose` | Run as part of a multi-service Docker Compose stack. |
| `kubernetes` | Deploy to a Kubernetes cluster with namespaces, replicas, and autoscaling. |

<!-- novalidate -->
```ias
deploy "dev" target "process" {
  default true
}

deploy "production" target "kubernetes" {
  namespace "agents"
  replicas 3
}
```

Mark one deploy target as `default true` -- this is the target used when you run `agentspec apply` without specifying a target name.

[:octicons-arrow-right-24: Deploy Reference](../language/deploy.md)

---

## Desired-State Model

AgentSpec follows a **desired-state model**, inspired by infrastructure-as-code tools. You declare the end state you want in your `.ias` file, and the toolchain computes the minimal set of changes to reach that state.

The workflow is:

```
Write .ias file  -->  validate  -->  plan  -->  apply
                                      |          |
                                      v          v
                                   Preview    State File
                                   changes    (.agentspec.state.json)
```

1. **Validate** checks syntax, semantics, and references.
2. **Plan** compares the desired state against the current state and shows what will change.
3. **Apply** executes the plan, creating, updating, or removing resources as needed.

This model has two important properties:

- The plan is always computed as a diff between desired and current state, so you never need to write imperative migration scripts.
- The state file (`.agentspec.state.json`) tracks what has been applied, enabling accurate change detection across runs.

---

## Idempotency

Every AgentSpec operation is **idempotent**: running the same command twice with the same input produces the same result. Specifically:

- `validate` always returns the same result for the same file.
- `fmt` on an already-formatted file produces no changes.
- `plan` with no file changes shows zero operations.
- `apply` with no file changes makes no modifications.

Idempotency means you can safely re-run `agentspec apply` in CI/CD pipelines, cron jobs, or automated workflows without worrying about duplicate or conflicting operations.

---

## Environments

An **environment** block defines an overlay that overrides resource attributes for a specific deployment context. Environments let you maintain a single `.ias` source file while varying configuration across development, staging, and production.

<!-- novalidate -->
```ias
environment "dev" {
  agent "assistant" {
    model "claude-haiku-latest"
  }
}

environment "prod" {
  agent "assistant" {
    model "claude-sonnet-4-20250514"
  }
}
```

Apply with a specific environment using the `--env` flag:

```bash
agentspec apply my-agent.ias --env dev
```

Overrides are applied on top of the base resource definitions. Attributes not mentioned in the override retain their original values.

[:octicons-arrow-right-24: Environment Reference](../language/environment.md)

---

## Secrets

A **secret** block defines a reference to a sensitive value such as an API key or database credential. Secrets are never stored in the `.ias` source file -- they are resolved at runtime from an external source.

<!-- novalidate -->
```ias
secret "api-key" {
  env(API_KEY)
}

secret "db-password" {
  store(production/database/password)
}
```

IntentLang supports two secret sources:

- **Environment variables** (`env`) -- Resolved from the shell environment at apply time.
- **Secure stores** (`store`) -- Resolved from a secrets manager (e.g., HashiCorp Vault, AWS Secrets Manager) using a path.

Secrets can be referenced by deploy blocks, server auth, and policy blocks, keeping credentials out of your source files.

[:octicons-arrow-right-24: Secret Reference](../language/secret.md)

---

## Policies

A **policy** block defines security and governance constraints that are enforced during validation and deployment. Policies let you prohibit certain configurations, mandate secrets, and explicitly permit approved resources.

<!-- novalidate -->
```ias
policy "production-safety" {
  deny model claude-haiku-latest
  require secret db-connection
  allow model claude-sonnet-4-20250514
}
```

Policies support three actions:

| Action | Effect |
|--------|--------|
| `deny` | Validation fails if the specified resource is used anywhere in the package. |
| `require` | Validation fails if the specified resource is not declared in the package. |
| `allow` | Explicitly permits a resource (useful for compliance documentation). |

Policies are evaluated after environment overrides are applied, so they enforce constraints on the final, resolved configuration.

[:octicons-arrow-right-24: Policy Reference](../language/policy.md)

---

## Plugins

A **plugin** declares a dependency on a sandboxed WebAssembly (WASM) module that extends AgentSpec's behavior. Plugins run inside a wazero sandbox and can hook into three stages of the toolchain pipeline:

| Hook | Stage | Purpose |
|------|-------|---------|
| `validator` | `agentspec validate` | Run custom validation rules against the parsed AST. |
| `transform` | `agentspec plan` | Transform the intermediate representation before plan generation. |
| `pre_deploy` | `agentspec apply` | Execute pre-flight checks before deployment. |

<!-- novalidate -->
```ias
plugin "security-scanner" version "2.1.0"
```

Plugins are loaded from `~/.agentspec/plugins/` (with a fallback to `~/.agentz/plugins/`). Each plugin is a `.wasm` file whose version must match the declared version exactly.

[:octicons-arrow-right-24: Plugin Reference](../language/plugin.md)

---

## How It All Fits Together

The following diagram shows how the core resource types relate to each other:

```
                        Package
                   ┌──────────────────────────────────┐
                   │                                  │
                   │  prompt ──────┐                  │
                   │               ├──> agent ──┐     │
                   │  skill ───────┘       |    |     │
                   │    |            strategy   |     │
                   │    v                       |     │
                   │  tool (mcp|http|           |     │
                   │        command|inline)     |     │
                   │                           v     │
                   │  pipeline ──> step ──> agent     │
                   │                                  │
                   │  environment ──> overrides        │
                   │  secret ──> external values       │
                   │  policy ──> constraints            │
                   │  plugin ──> WASM extensions        │
                   │                                  │
                   │  deploy ──> process | docker      │
                   │             compose | kubernetes  │
                   └──────────────────────────────────┘
```

- **Agents** are the central resource. They compose prompts and skills.
- **Skills** are backed by tools, which handle execution.
- **Pipelines** orchestrate agents into multi-step workflows.
- **Deploy targets** determine where the system runs.
- **Environments**, **secrets**, **policies**, and **plugins** layer on configuration, security, governance, and extensibility.

---

## What's Next

- [Quick Start](quickstart.md) -- Build and apply your first agent (if you have not done so already).
- [Language Reference](../language/index.md) -- Detailed syntax and semantics for every resource type.
- [Agentic Architecture Patterns](../use-cases/index.md) -- See how strategies and delegation map to real-world patterns (ReAct, Plan-and-Execute, Reflexion, Router, Map-Reduce, Pipeline, Delegation).
- [CLI Reference](../cli/index.md) -- Full reference for `validate`, `fmt`, `plan`, `apply`, and other commands.
- [Agent Runtime Configuration](../configuration/runtime.md) -- Tune strategy, timeouts, and token budgets.
