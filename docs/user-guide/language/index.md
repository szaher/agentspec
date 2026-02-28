# Language Reference

IntentLang 2.0 is the declarative language at the heart of AgentSpec. Every `.ias` file describes a complete agent system -- prompts, skills, tools, deployment targets, and more -- in a single, human-readable source of truth.

This reference covers the syntax, semantics, and available resource types of IntentLang 2.0.

---

## Package Header

Every `.ias` file **must** begin with a package header. The header declares the package name, its semantic version, and the IntentLang version it targets.

<!-- novalidate -->
```ias
package "my-agent" version "1.0.0" lang "2.0"
```

| Component   | Type   | Required | Description                                        |
|-------------|--------|----------|----------------------------------------------------|
| `package`   | string | Yes      | Unique package name. Must be a valid identifier.    |
| `version`   | string | Yes      | Semantic version of the package (`MAJOR.MINOR.PATCH`). |
| `lang`      | string | Yes      | IntentLang specification version. Currently `"2.0"`. |

!!! warning "Version Compatibility"
    The `lang` field must be `"2.0"` for all features documented in this reference. Files that omit the `lang` field or use an unsupported version will fail validation.

---

## Resource Types

IntentLang 2.0 defines **13 resource types**. Each resource is declared as a named block at the top level of an `.ias` file.

| Resource                                    | Description                                                        |
|---------------------------------------------|--------------------------------------------------------------------|
| [`agent`](agent.md)                         | An AI agent with a model, prompt, skills, and behavior settings.    |
| [`prompt`](prompt.md)                       | A reusable prompt template with optional variables.                 |
| [`skill`](skill.md)                         | A capability an agent can invoke, backed by a tool implementation.  |
| [`tool`](tool.md)                           | The underlying implementation of a skill (MCP, HTTP, command, inline). |
| [`deploy`](deploy.md)                       | A deployment target (process, Docker, Kubernetes, etc.).            |
| `pipeline`                                  | A multi-step workflow that chains agents in sequence or parallel.   |
| `type`                                      | A custom data type definition for structured input/output schemas.  |
| `server`                                    | An MCP server that exposes skills over a transport.                 |
| `client`                                    | An MCP client that connects to a server.                           |
| `secret`                                    | A secret value sourced from an environment variable.                |
| `environment`                               | An environment overlay that overrides agent settings per environment.|
| `policy`                                    | A governance rule that constrains model usage or requires secrets.   |
| `plugin`                                    | A WASM plugin that extends AgentSpec with custom hooks.             |

---

## File Structure

An `.ias` file follows a consistent structure:

1. **Package header** (required, must be the first non-comment line)
2. **Resource blocks** (zero or more, in any order)

Resources can appear in any order, but a common convention is:

```ias
package "example" version "0.1.0" lang "2.0"

# 1. Prompts
prompt "system" {
  content "You are a helpful assistant."
}

# 2. Skills
skill "search" {
  description "Search for information"
  input { query string required }
  output { results string }
  tool command { binary "search-tool" }
}

# 3. Agents
agent "assistant" {
  uses prompt "system"
  uses skill "search"
  model "claude-sonnet-4-20250514"
}

# 4. Deployment targets
deploy "local" target "process" {
  default true
}
```

!!! tip "Resource Ordering"
    While resource order does not affect semantics, placing prompts and skills before the agents that reference them improves readability.

---

## Comments

IntentLang supports two comment styles. Comments are ignored by the parser.

**Line comments** begin with `#` or `//` and extend to the end of the line:

<!-- novalidate -->
```ias
# This is a comment
// This is also a comment

agent "my-agent" {  # Inline comment
  model "claude-sonnet-4-20250514"  // Another inline comment
}
```

!!! note
    Block comments (`/* ... */`) are **not** supported in IntentLang 2.0.

---

## Primitive Types

IntentLang has four primitive types used in attribute values and schema definitions.

### Strings

String values are enclosed in double quotes. Strings support the following escape sequences:

<!-- novalidate -->
```ias
content "Line one\nLine two"
content "She said \"hello\""
content "Path: C:\\Users\\agent"
```

| Escape | Meaning         |
|--------|-----------------|
| `\n`   | Newline          |
| `\t`   | Tab              |
| `\\`   | Literal backslash|
| `\"`   | Literal quote    |

### Numbers

IntentLang supports integer and floating-point number literals. Numbers are unquoted.

<!-- novalidate -->
```ias
max_turns 10          # integer
temperature 0.7       # float
replicas 3            # integer
token_budget 100000   # integer
```

### Booleans

Boolean values are the unquoted keywords `true` and `false`.

<!-- novalidate -->
```ias
default true
stream false
```

### Arrays

Array literals are enclosed in square brackets with comma-separated elements. Elements can be strings, numbers, or booleans.

<!-- novalidate -->
```ias
depends_on ["step-1", "step-2"]
args ["-v", "--output", "json"]
```

---

## Identifiers and References

Resource names are always double-quoted strings. References to other resources use the same quoted name preceded by the resource type keyword.

<!-- novalidate -->
```ias
# Declaring a prompt named "system"
prompt "system" {
  content "You are a helpful assistant."
}

# Referencing it from an agent
agent "my-agent" {
  uses prompt "system"   # References the prompt declared above
}
```

!!! warning "Name Uniqueness"
    Resource names must be unique within their type. You cannot declare two resources of the same type with the same name in a single package.

---

## What's Next?

Explore the individual resource type references:

- [Agent](agent.md) -- Define AI agents with models, strategies, and delegation
- [Prompt](prompt.md) -- Create reusable prompt templates with variables
- [Skill](skill.md) -- Declare agent capabilities backed by tool implementations
- [Tool](tool.md) -- Configure MCP, HTTP, command, and inline tool backends
- [Deploy](deploy.md) -- Target processes, Docker, Compose, or Kubernetes

Or see these resources in action:

- [Agentic Architecture Patterns](../use-cases/index.md) -- Complete walkthroughs of ReAct, Plan-and-Execute, Reflexion, Router, Map-Reduce, Pipeline, and Delegation patterns
- [Quick Start](../getting-started/quickstart.md) -- Build and apply your first agent
