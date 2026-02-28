# Quick Start

This tutorial walks you through creating, validating, and applying your first AgentSpec file. By the end, you will have a working agent definition and understand the four core CLI commands.

---

## Step 1: Create Your First `.ias` File

Create a file named `hello.ias` with the following content:

```ias
package "hello" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a helpful assistant."
}

skill "greet" {
  description "Greet the user by name"
  input {
    name string required
  }
  output {
    message string
  }
  tool command {
    binary "greet-tool"
  }
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

This file defines a complete agent system:

- A **package** header declaring the name, version, and IntentLang version.
- A **prompt** that sets the agent's system instructions.
- A **skill** with typed input/output and a command-based tool implementation.
- An **agent** that binds the prompt and skill to a language model.
- A **deploy** target that runs the agent as a local process.

---

## Step 2: Validate

Run the validator to check your file for syntax errors, missing references, and invalid values:

```bash
agentspec validate hello.ias
```

If the file is valid, you will see:

```
hello.ias: OK
```

The validator checks that:

- The package header is present and well-formed.
- All resource names are unique within their type.
- All references resolve (e.g., the agent's `uses prompt "system"` points to a declared prompt).
- Required attributes are present on every resource.
- Values are within allowed ranges (e.g., valid strategy names, valid model formats).

---

## Step 3: Format

Format the file to apply consistent indentation and whitespace:

```bash
agentspec fmt hello.ias
```

The formatter rewrites the file in place with canonical formatting. Running `fmt` on an already-formatted file produces no changes -- the operation is idempotent.

!!! tip "Format on Save"
    Run `agentspec fmt` as part of your editor's save hook or CI pipeline to keep all `.ias` files consistently formatted across your team.

---

## Step 4: Plan

Preview the changes AgentSpec will make without applying them:

```bash
agentspec plan hello.ias
```

The plan output shows each resource that will be created, updated, or removed. For a new file, all resources are marked for creation:

```
Plan: 4 to add, 0 to change, 0 to destroy.

  + prompt "system"
  + skill "greet"
  + agent "assistant"
  + deploy "local" (target: process)
```

The plan command is read-only. It compares the desired state in your `.ias` file against the current state (stored in `.agentspec.state.json`) and computes the minimal set of operations to reconcile the two. No resources are modified during planning.

!!! info "State File"
    AgentSpec stores the current state of applied resources in `.agentspec.state.json` in the working directory. This file is auto-created on the first `apply` and should be committed to version control.

---

## Step 5: Apply

Apply the desired state to create the agent system:

```bash
agentspec apply hello.ias
```

The `apply` command executes the plan:

```
Applying changes...

  + prompt "system" ... created
  + skill "greet" ... created
  + agent "assistant" ... created
  + deploy "local" (target: process) ... created

Apply complete! 4 added, 0 changed, 0 destroyed.
```

AgentSpec follows a **desired-state model**: you declare the end state you want in your `.ias` file, and the tool computes and applies the minimal changes to reach that state. This means:

- **Creating** resources that exist in the file but not in the current state.
- **Updating** resources whose definition has changed since the last apply.
- **Removing** resources that are in the state file but no longer in the `.ias` file.

Running `apply` again with no changes to the file produces no operations -- the system is already in the desired state. This property is called **idempotency**.

---

## What's Next

Now that you have created and applied your first agent, explore these topics:

- [Core Concepts](concepts.md) -- Understand the mental model behind packages, agents, prompts, skills, and the desired-state architecture.
- [Language Reference](../language/index.md) -- Full syntax and attribute reference for all 13 IntentLang resource types.
- [Deployment Targets](../language/deploy.md) -- Deploy to Docker, Docker Compose, and Kubernetes.
- [Agent Runtime Configuration](../configuration/runtime.md) -- Configure strategies, timeouts, token budgets, and streaming.
- [Error Handling](../configuration/error-handling.md) -- Set up retries, fallbacks, and failure modes.
- [Agent Delegation](../configuration/delegation.md) -- Build multi-agent systems with routing and delegation.

---

## Troubleshooting

This section covers common errors you may encounter when writing `.ias` files and how to fix them.

### Missing Package Header

**Error:**

```
error: missing package header
  --> hello.ias:1:1
  |
  | expected: package "<name>" version "<version>" lang "<lang-version>"
```

**Fix:** Every `.ias` file must begin with a package header as the first non-comment line. Add one at the top of your file:

```ias
package "hello" version "0.1.0" lang "2.0"
```

### Unknown Keyword

**Error:**

```
error: unknown keyword "agents"
  --> hello.ias:8:1
  |
8 | agents "assistant" {
  | ^^^^^^ unknown keyword
  |
  = help: did you mean "agent"?
```

**Fix:** Check the spelling of the resource type. IntentLang resource types are singular nouns: `agent`, `prompt`, `skill`, `tool`, `deploy`, `pipeline`, `type`, `server`, `client`, `secret`, `environment`, `policy`, `plugin`.

### Invalid Strategy Value

**Error:**

```
error: invalid strategy value "chain-of-thought"
  --> hello.ias:10:13
   |
10 |   strategy "chain-of-thought"
   |            ^^^^^^^^^^^^^^^^^^^ invalid value
   |
   = help: valid strategies are: react, plan-and-execute, reflexion, router, map-reduce
```

**Fix:** Use one of the five supported strategy values: `react` (the default), `plan-and-execute`, `reflexion`, `router`, or `map-reduce`.

### Missing Required Attribute

**Error:**

```
error: missing required attribute "model" on agent "assistant"
  --> hello.ias:8:1
  |
8 | agent "assistant" {
  | ^^^^^^^^^^^^^^^^^^
  |
  = help: add: model "<model-identifier>"
```

**Fix:** Add the missing attribute. In this case, every `agent` block requires a `model` attribute:

```ias
agent "assistant" {
  uses prompt "system"
  model "claude-sonnet-4-20250514"
}
```

### Invalid Model Format

**Error:**

```
error: invalid model identifier ""
  --> hello.ias:10:9
   |
10 |   model ""
   |         ^^ model identifier must not be empty
```

**Fix:** The `model` attribute requires a non-empty string containing a valid model identifier. Use a recognized model name such as `"claude-sonnet-4-20250514"`, `"claude-haiku-latest"`, or `"claude-opus-4-20250514"`.
