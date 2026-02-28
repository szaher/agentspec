# Data Model: Agent Compilation & Deployment Framework

**Feature**: 006-agent-compile-deploy
**Date**: 2026-02-28

## Entity Relationship Diagram

```text
┌──────────────┐     imports     ┌──────────────┐
│  SourceFile  │────────────────▶│  SourceFile  │
│   (.ias)     │  1..*    0..*   │   (.ias)     │
└──────┬───────┘                 └──────────────┘
       │ contains 1..*
       ▼
┌──────────────┐
│  AgentDef    │
│              │─────┐ has 0..*
└──────┬───────┘     │
       │             ▼
       │      ┌──────────────┐
       │      │ ValidationRule│
       │      └──────────────┘
       │
       │ has 0..*    ┌──────────────┐
       ├────────────▶│  EvalCase    │
       │             └──────────────┘
       │
       │ has 1..*    ┌──────────────┐
       ├────────────▶│  ConfigParam │
       │             └──────────────┘
       │
       │ compiles to
       ▼
┌──────────────┐     packages to   ┌──────────────┐
│  CompiledIR  │──────────────────▶│   Artifact   │
│              │    1        1..*  │              │
└──────┬───────┘                   └──────┬───────┘
       │                                  │
       │ targets 1                        │ deploys as
       ▼                                  ▼
┌──────────────┐                   ┌──────────────┐
│  CompTarget  │                   │  Deployment  │
│              │                   │              │
└──────────────┘                   └──────────────┘

┌──────────────┐     contains 1..* ┌──────────────┐
│   Package    │──────────────────▶│  SourceFile  │
│  (registry)  │                   └──────────────┘
└──────┬───────┘
       │ depends on 0..*
       ▼
┌──────────────┐
│   Package    │
│  (dependency)│
└──────────────┘
```

## Entities

### SourceFile

An IntentLang file containing agent definitions, skills, prompts, and other resources.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| path | string | Required, unique within project | Relative path from project root |
| package_name | string | Required | From `package` declaration |
| package_version | string | Required, semver | From `version` declaration |
| lang_version | string | Required | IntentLang version (e.g., "3.0") |
| imports | []ImportRef | Optional | List of import references |
| resources | []Resource | Required, 1..* | Agent, Skill, Prompt, etc. |

### ImportRef

A reference to another .ias file or published package.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| source | string | Required | Path or package name |
| version | string | Required for packages, N/A for local | Semver constraint |
| kind | enum | `local` or `package` | Local file vs registry package |
| alias | string | Optional | Import alias for namespacing |
| resolved_path | string | Set during resolution | Absolute path after resolution |
| content_hash | string | Set during resolution | SHA-256 of resolved content |

**State transitions**: `declared` → `resolved` → `validated` → `lowered`

### AgentDef

An agent definition within a source file, extended with compilation-specific fields.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| name | string | Required, unique within package | Agent identifier |
| prompt_ref | string | Required | Reference to a Prompt resource |
| model | string | Required | LLM model identifier |
| skills | []string | Optional | References to Skill resources |
| loop_strategy | enum | Required | `react`, `plan-execute`, `reflexion`, `router`, `map-reduce` |
| max_turns | int | Optional, default 10 | Maximum reasoning turns |
| timeout | duration | Optional | Per-request timeout |
| config_params | []ConfigParam | Optional | Runtime configuration declarations |
| validation_rules | []ValidationRule | Optional | Output validation rules |
| eval_cases | []EvalCase | Optional | Evaluation test cases |
| control_flow | []ControlFlowBlock | Optional | If/else, for-each blocks |

### ConfigParam

A runtime configuration parameter declared in an agent definition.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| name | string | Required, unique within agent | Parameter name (maps to env var) |
| type | enum | `string`, `int`, `float`, `bool` | Value type |
| description | string | Required | Human-readable description |
| secret | bool | Default false | If true, value is never logged and cannot have a default |
| default | any | Optional | Default value (never when `secret` is true) |
| required | bool | Default true | Must be provided at runtime |
| env_var | string | Auto-generated | `AGENTSPEC_<AGENT>_<NAME>` convention |

### ValidationRule

A declarative output validation rule for an agent.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| name | string | Required, unique within agent | Rule identifier |
| expression | string | Required | Validation expression (expr syntax) |
| severity | enum | Default `error` | `error` (reject + retry) or `warning` (log only) |
| message | string | Required | Human-readable failure message |
| max_retries | int | Default 3 | Retries before failing (error severity only) |

### EvalCase

A golden input/output test case for evaluating agent quality.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| name | string | Required, unique within agent | Test case identifier |
| input | string | Required | Input message to the agent |
| expected_output | string | Required | Expected output (for comparison) |
| scoring | enum | Default `semantic` | `exact`, `contains`, `semantic`, `custom` |
| threshold | float | Default 0.8 | Minimum similarity score to pass (semantic) |
| tags | []string | Optional | For filtering eval runs |

### CompilationTarget

A named output format the compiler can produce.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| name | string | Required, unique | Target identifier (e.g., `standalone`, `crewai`) |
| kind | enum | Required | `native` or `framework` |
| plugin_ref | string | Required for `framework` | Reference to compilation plugin |
| output_type | enum | Required | `binary`, `source_code`, `bundle` |
| supported_features | []string | Required | List of AgentSpec features this target supports |

### CompiledArtifact

The output of a compilation run.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| id | string | Required, unique | Artifact identifier (content hash) |
| target | string | Required | Compilation target name |
| platform | string | Required for native | `linux/amd64`, `darwin/arm64`, etc. |
| source_hash | string | Required | SHA-256 of input .ias files |
| content_hash | string | Required | SHA-256 of output artifact |
| size_bytes | int64 | Required | Artifact size |
| agents | []string | Required | Names of agents in this artifact |
| config_ref | string | Required | Path to generated config reference |
| created_at | timestamp | Required | Compilation timestamp |
| warnings | []string | Optional | Compilation warnings |

### Package (Registry)

A versioned, distributable collection of .ias files.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| name | string | Required, unique in namespace | Package name |
| namespace | string | Required | Owner namespace (e.g., `github.com/user`) |
| version | string | Required, semver | Package version |
| description | string | Required | Human-readable description |
| author | string | Required | Package author |
| license | string | Optional | SPDX license identifier |
| dependencies | map[string]string | Optional | Package name → version constraint |
| checksum | string | Required | SHA-256 of package archive |
| published_at | timestamp | Required | Publication timestamp |
| deprecated | bool | Default false | Whether package is deprecated |
| deprecation_msg | string | Optional | Reason for deprecation |
| signature | string | Default `"unsigned"` | Package signature (MVP: always `"unsigned"`) |
| signer | string | Optional | Signer key ID (empty in MVP) |
| provenance | object | Optional | Build provenance metadata (source repo, commit SHA) |

**State transitions**: `draft` → `published` → `deprecated`

### FrontendSession

A user's interaction session with an agent through the built-in frontend.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| session_id | string | Required, unique | UUID session identifier |
| agent_name | string | Required | Agent being interacted with |
| messages | []Message | Required | Conversation history |
| activity_log | []ActivityEntry | Required | Agent reasoning trace |
| created_at | timestamp | Required | Session start |
| last_active | timestamp | Required | Last interaction |
| auth_token | string | Required unless auth disabled | API key hash for this session |

### Message

A single message in a frontend conversation.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| role | enum | `user` or `assistant` | Message sender |
| content | string | Required | Message text |
| timestamp | timestamp | Required | When sent |
| metadata | map | Optional | Input schema values, attachments |

### ActivityEntry

A single entry in the agent's reasoning trace.

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| type | enum | Required | `thought`, `tool_call`, `tool_result`, `validation`, `error` |
| content | string | Required | Activity description |
| timestamp | timestamp | Required | When occurred |
| duration_ms | int | Optional | How long this step took |
| metadata | map | Optional | Tool name, parameters, results |

## Indexes & Uniqueness

- SourceFile: unique by `(package_name, path)`
- AgentDef: unique by `(package_name, name)`
- ConfigParam: unique by `(agent_name, name)`, env_var globally unique
- Package: unique by `(namespace, name, version)`
- CompiledArtifact: unique by `content_hash`
- FrontendSession: unique by `session_id`

## Validation Rules

- ImportRef version constraints must be valid semver ranges
- ConfigParam with `secret: true` must not have a default value
- ValidationRule expressions must be valid `expr` syntax and compile without errors
- EvalCase threshold must be between 0.0 and 1.0
- AgentDef must reference at least one Prompt and specify a model
- Package dependencies must not form cycles (detected by Tarjan's SCC)
- CompiledArtifact source_hash must match the hash of the actual input files (determinism check)
