# Data Model: Declarative Agent Packaging DSL

**Date**: 2026-02-22
**Source**: spec.md (Key Entities) + clarifications

## Entity Relationships

```text
Package 1──* Agent
Package 1──* Prompt
Package 1──* Skill
Package 1──* MCPServer
Package 1──* MCPClient
Package 1──* Environment
Package 1──* Binding
Package 1──* Policy
Package 0──* Plugin (declared as dependency)

Agent *──1 Prompt (uses)
Agent *──* Skill (uses)
Agent *──0..1 MCPClient (connects to)

MCPClient *──* MCPServer (connects to)
MCPServer 1──* Skill (exposes)

Binding *──1 Adapter (targets)
Environment 1──* Override (contains)

Plugin 1──* CustomResourceType (declares)
Plugin 1──* Hook (declares)
Plugin 1──* Validator (declares)
Plugin 1──* Transform (declares)
```

## Core Entities

### Package

The top-level container. Identity scope for all resources.

| Field        | Type     | Required | Description                              |
|--------------|----------|----------|------------------------------------------|
| name         | string   | yes      | Unique package identifier                |
| version      | semver   | yes      | Package version                          |
| lang_version | string   | yes      | DSL language version                     |
| description  | string   | no       | Human-readable description               |
| imports      | []Import | no       | External package references (pinned)     |
| plugins      | []PluginRef | no    | Plugin dependencies (versioned)          |

**Identity**: Globally unique by `name`.
**Validation**: `lang_version` MUST match a supported version.
All `imports` MUST be pinned (version, SHA, or content hash).

### Agent

| Field       | Type        | Required | Description                            |
|-------------|-------------|----------|----------------------------------------|
| name        | string      | yes      | Unique within package (type+name)      |
| prompt      | ref(Prompt) | yes      | Reference to a Prompt                  |
| skills      | []ref(Skill)| no       | Skills this agent can invoke           |
| model       | string      | yes      | Model identifier                       |
| parameters  | map         | no       | Model parameters (temperature, etc.)   |
| client      | ref(MCPClient) | no    | MCP client connection                  |
| metadata    | map         | no       | User-defined key-value pairs           |

**Identity**: `Agent/<name>` within package scope.
**References**: Prompt and Skill refs may be local or
fully-qualified (`package/Prompt/name`).

### Prompt

| Field       | Type     | Required | Description                             |
|-------------|----------|----------|-----------------------------------------|
| name        | string   | yes      | Unique within package (type+name)       |
| content     | string   | yes      | Prompt template text                    |
| variables   | []Variable | no     | Declared template variables             |
| version     | semver   | no       | Independent version (for reuse)         |
| metadata    | map      | no       | User-defined key-value pairs            |

**Identity**: `Prompt/<name>` within package scope.
**Validation**: All `{{variable}}` references in content MUST
have a corresponding entry in `variables`.

### Skill

| Field         | Type     | Required | Description                           |
|---------------|----------|----------|---------------------------------------|
| name          | string   | yes      | Unique within package (type+name)     |
| description   | string   | yes      | Human-readable description            |
| input_schema  | Schema   | yes      | Input parameter schema                |
| output_schema | Schema   | yes      | Output value schema                   |
| execution     | Execution| yes      | How the skill runs (binding)          |
| metadata      | map      | no       | User-defined key-value pairs          |

**Identity**: `Skill/<name>` within package scope.

### MCPServer

| Field       | Type        | Required | Description                          |
|-------------|-------------|----------|--------------------------------------|
| name        | string      | yes      | Unique within package (type+name)    |
| transport   | enum        | yes      | `stdio`, `sse`, `streamable-http`    |
| command     | string      | cond     | Required for stdio transport         |
| args        | []string    | no       | Command arguments                    |
| url         | string      | cond     | Required for HTTP transports         |
| auth        | ref(Secret) | no       | Authentication reference             |
| skills      | []ref(Skill)| no       | Skills this server exposes           |
| env         | map         | no       | Environment variables                |
| metadata    | map         | no       | User-defined key-value pairs         |

**Identity**: `MCPServer/<name>` within package scope.
**Validation**: `command` required when transport is `stdio`;
`url` required when transport is `sse` or `streamable-http`.

### MCPClient

| Field       | Type            | Required | Description                      |
|-------------|-----------------|----------|----------------------------------|
| name        | string          | yes      | Unique within package (type+name)|
| servers     | []ref(MCPServer)| yes      | Servers this client connects to  |
| metadata    | map             | no       | User-defined key-value pairs     |

**Identity**: `MCPClient/<name>` within package scope.

### Environment

| Field       | Type        | Required | Description                          |
|-------------|-------------|----------|--------------------------------------|
| name        | string      | yes      | Environment name (dev, staging, prod)|
| overrides   | []Override  | yes      | Attribute overrides                  |

**Identity**: `Environment/<name>` within package scope.
**Validation**: Each attribute MUST resolve to a single unambiguous
value. Conflicting overlay values MUST be rejected at validation
time.

### Override

| Field       | Type     | Required | Description                             |
|-------------|----------|----------|-----------------------------------------|
| resource    | ref      | yes      | Reference to the resource being overridden |
| attribute   | string   | yes      | Attribute path to override              |
| value       | any      | yes      | Override value                          |

### Secret

| Field       | Type     | Required | Description                             |
|-------------|----------|----------|-----------------------------------------|
| name        | string   | yes      | Secret reference name                   |
| source      | enum     | yes      | `env` or `store`                        |
| key         | string   | yes      | Environment variable name or store path |

**Validation**: Plaintext literal values MUST be rejected.
Only `env(KEY)` or `store(path)` references permitted.

### Policy

| Field       | Type     | Required | Description                             |
|-------------|----------|----------|-----------------------------------------|
| name        | string   | yes      | Policy name                             |
| rules       | []Rule   | yes      | Security constraint rules               |

Each rule specifies a resource type pattern and a constraint
(e.g., `deny network *`, `require pinned imports`).

### Binding

| Field       | Type     | Required | Description                             |
|-------------|----------|----------|-----------------------------------------|
| name        | string   | yes      | Binding name                            |
| adapter     | string   | yes      | Adapter identifier (e.g., `local-mcp`)  |
| default     | bool     | no       | Whether this is the default binding     |
| config      | map      | no       | Adapter-specific configuration          |

**Identity**: `Binding/<name>` within package scope.
**Validation**: At most one binding per package MAY be marked
`default`. A sole binding is implicitly default.

### Plugin (dependency reference)

| Field       | Type     | Required | Description                             |
|-------------|----------|----------|-----------------------------------------|
| name        | string   | yes      | Plugin package name                     |
| version     | string   | yes      | Pinned version                          |
| hooks_order | []string | no       | Explicit hook execution order           |

## IR Representation

The IR is the canonical, platform-neutral representation produced
after parsing and lowering the AST. All references are resolved,
all environment overrides are merged, and all validation has passed.

### IR Document

| Field       | Type          | Description                            |
|-------------|---------------|----------------------------------------|
| version     | string        | IR schema version                      |
| lang_version| string        | DSL language version                   |
| package     | IRPackage     | Resolved package metadata              |
| resources   | []IRResource  | Flat list of fully-resolved resources  |
| policies    | []IRPolicy    | Resolved policy rules                  |
| bindings    | []IRBinding   | Target bindings                        |

### IRResource

| Field       | Type     | Description                                |
|-------------|----------|--------------------------------------------|
| kind        | string   | Resource type (Agent, Prompt, Skill, etc.) |
| name        | string   | Resource name                              |
| fqn         | string   | Fully-qualified name (package/kind/name)   |
| attributes  | map      | Resolved attribute key-value pairs         |
| references  | []string | FQNs of resources this one depends on      |
| hash        | string   | Content hash of this resource definition   |

**Determinism**: Resources are serialized in a stable order:
sorted by `kind`, then by `name`. All map keys are sorted
alphabetically. This guarantees byte-identical IR output for
identical inputs.

## State Transitions

Resources in the state file track lifecycle:

```text
                    ┌─────────┐
     plan(create) → │ planned │
                    └────┬────┘
                         │ apply
                    ┌────▼────┐
                    │ applied │ ◄── plan(update) → apply
                    └────┬────┘
                         │ plan(delete)
                    ┌────▼────┐
                    │ removed │
                    └─────────┘
```

Each state entry records:
- `fqn`: Fully-qualified resource name
- `hash`: Content hash at last apply
- `status`: `applied` or `failed`
- `last_applied`: Timestamp
- `adapter`: Which adapter applied it
- `error`: Error message if `status == failed`
