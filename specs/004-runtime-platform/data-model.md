# Data Model: AgentSpec Runtime Platform

**Branch**: `004-runtime-platform` | **Date**: 2026-02-23

## Entity Overview

```
┌──────────┐     uses      ┌──────────┐     backed by    ┌──────────┐
│  Agent   │──────────────▶│  Prompt  │                  │   Type   │
│          │               └──────────┘                  └──────────┘
│          │     uses      ┌──────────┐     references        │
│          │──────────────▶│  Skill   │◀──────────────────────┘
│          │               │          │     input/output schema
│          │               │          │     backed by    ┌──────────┐
│          │               │          │────────────────▶│   Tool   │
│          │               └──────────┘                  └──────────┘
│          │     deployed via                               │
│          │               ┌──────────────┐                 │ mcp/http/
│          │──────────────▶│ Deploy Target│                 │ command/
└──────────┘               └──────────────┘                 │ inline
      │                                                     │
      │ participates in   ┌──────────┐     connects to  ┌──────────┐
      └──────────────────▶│ Pipeline │                  │MCP Server│
                          └──────────┘                  └──────────┘
                               │
                          ┌──────────┐
                          │   Step   │
                          └──────────┘
```

## Language-Level Entities (Stored in IR)

### Package

Top-level container for agent definitions.

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| name | string | yes | Package identifier |
| version | semver | yes | Package version |
| lang | string | yes | Language version (`"2.0"`) |
| imports | []Import | no | External package references |
| plugins | []PluginRef | no | Plugin declarations |

### Agent

Primary deployable unit. References prompts, skills, and deployment config.

| Field | Type | Required | Default | Description |
| ----- | ---- | -------- | ------- | ----------- |
| name | string | yes | — | Agent identifier (unique within package) |
| model | string | yes | — | LLM model identifier |
| prompt_refs | []string | yes | — | Referenced prompt names (`uses prompt`) |
| skill_refs | []string | no | [] | Referenced skill names (`uses skill`) |
| strategy | enum | no | `"react"` | Execution strategy: react, plan-and-execute, reflexion, router, map-reduce |
| max_turns | int | no | 10 | Maximum agentic loop iterations |
| timeout | duration | no | `"120s"` | Per-invocation timeout |
| token_budget | int | no | 100000 | Maximum tokens per invocation |
| temperature | float | no | 0.7 | LLM sampling temperature |
| stream | bool | no | true | Enable streaming responses |
| on_error | enum | no | `"retry"` | Error strategy: retry, fail, fallback |
| max_retries | int | no | 3 | Max retry attempts (when on_error=retry) |
| fallback_ref | string | no | — | Fallback agent name (when on_error=fallback) |
| memory | MemoryConfig | no | — | Conversation memory configuration |
| delegates | []Delegate | no | [] | Delegation rules |

**FQN pattern**: `{package-name}/Agent/{agent-name}`

**State transitions**: `pending` → `deploying` → `running` → `updating` → `running` / `failed` → `destroying` → `destroyed`

### Prompt

System prompt template with optional variable interpolation.

| Field | Type | Required | Default | Description |
| ----- | ---- | -------- | ------- | ----------- |
| name | string | yes | — | Prompt identifier |
| content | string | yes | — | Prompt text with `{{variable}}` placeholders |
| variables | []Variable | no | [] | Variable declarations |

**FQN pattern**: `{package-name}/Prompt/{prompt-name}`

### Variable

Prompt template variable declaration.

| Field | Type | Required | Default | Description |
| ----- | ---- | -------- | ------- | ----------- |
| name | string | yes | — | Variable name (matches `{{name}}` in content) |
| type | enum | yes | — | string, number, bool |
| required | bool | no | false | Whether variable must be provided at invocation |
| default | any | no | — | Default value if not provided |

### Skill

Agent capability backed by a tool execution mechanism.

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| name | string | yes | Skill identifier |
| description | string | yes | Human-readable description (sent to LLM) |
| input | []Field | no | Input schema fields |
| output | []Field | no | Output schema fields |
| tool | ToolConfig | yes | Tool execution configuration |

**FQN pattern**: `{package-name}/Skill/{skill-name}`

### ToolConfig

Defines how a skill is executed at runtime.

| Variant | Fields | Description |
| ------- | ------ | ----------- |
| mcp | server_tool: `"server/tool"` | Calls a tool on a named MCP server |
| http | method, url, headers, body_template | Makes an HTTP request |
| command | binary, args, timeout, env, secrets | Spawns a subprocess |
| inline | language, code, timeout, memory_limit, env, secrets | Runs embedded code in sandboxed subprocess |

### Type

User-defined data type for skill I/O schemas.

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| name | string | yes | Type identifier |
| fields | []TypeField | yes | Field definitions |

**FQN pattern**: `{package-name}/Type/{type-name}`

### TypeField

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| name | string | yes | Field name |
| type | string | yes | string, number, bool, timestamp, or user-defined type name |
| required | bool | no | Whether field is mandatory |
| default | any | no | Default value |
| enum | []string | no | Allowed values (if type=enum) |
| list | TypeField | no | List item schema (if type=list) |

### DeployTarget (replaces Binding)

Deployment configuration for a target environment.

| Field | Type | Required | Default | Description |
| ----- | ---- | -------- | ------- | ----------- |
| name | string | yes | — | Deploy target identifier (e.g., "production") |
| target | enum | yes | — | process, docker, docker-compose, kubernetes |
| default | bool | no | false | Whether this is the default deploy target |
| port | int | no | 8080 | HTTP server port |
| namespace | string | no | — | Kubernetes namespace |
| replicas | int | no | 1 | Replica count |
| image | string | no | — | Container image (docker/k8s) |
| resources | ResourceLimits | no | — | CPU/memory limits |
| health | HealthConfig | no | — | Health check configuration |
| autoscale | AutoscaleConfig | no | — | Horizontal scaling rules |
| env | map[string]string | no | — | Environment variables |
| secrets | map[string]SecretRef | no | — | Secret references |

### Pipeline

Multi-agent workflow with step dependencies and parallel execution.

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| name | string | yes | Pipeline identifier |
| steps | []PipelineStep | yes | Ordered list of steps |

**FQN pattern**: `{package-name}/Pipeline/{pipeline-name}`

### PipelineStep

| Field | Type | Required | Default | Description |
| ----- | ---- | -------- | ------- | ----------- |
| name | string | yes | — | Step identifier |
| agent_ref | string | yes | — | Agent to execute |
| input | []StepInput | no | — | Input mappings (from trigger or previous steps) |
| output | []string | no | — | Named output fields |
| parallel | bool | no | false | Can run concurrently with other parallel steps |
| depends_on | []string | no | [] | Steps that must complete before this one starts |

### Delegate

Agent delegation rule.

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| agent_ref | string | yes | Target agent name |
| condition | string | yes | Natural language condition for delegation |

## Runtime Entities (Not in IR — created at runtime)

### Session

Stateful conversation context. Created when a caller opens a session with an agent.

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | string (UUID) | Session identifier |
| agent_fqn | string | FQN of the agent this session belongs to |
| messages | []Message | Conversation history |
| created_at | timestamp | Session creation time |
| last_active | timestamp | Last message time |
| memory_strategy | enum | sliding_window or summary |
| max_messages | int | Maximum messages before compression |
| metadata | map[string]any | Caller-provided metadata |

**State transitions**: `active` → `expired` (timeout) / `closed` (explicit)

### Invocation

Single request-response cycle. Created per HTTP request to an agent.

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | string (UUID) | Invocation identifier |
| agent_fqn | string | FQN of the invoked agent |
| session_id | string | Associated session (if any) |
| input | string | User message |
| variables | map[string]string | Prompt template variables |
| output | string | Final agent response |
| tool_calls | []ToolCallRecord | Audit trail of tool invocations |
| tokens_used | TokenUsage | Token consumption breakdown |
| turns | int | Number of agentic loop iterations |
| duration | duration | Total invocation time |
| status | enum | running, completed, failed, timeout |
| error | string | Error message (if failed) |
| created_at | timestamp | Invocation start time |

### Message

Single message in a conversation.

| Field | Type | Description |
| ----- | ---- | ----------- |
| role | enum | user, assistant, tool_result |
| content | string | Message text |
| tool_calls | []ToolCall | Tool calls (assistant messages only) |
| tool_call_id | string | Tool call this result corresponds to (tool_result only) |
| timestamp | timestamp | When the message was created |

### ToolCallRecord

Audit record of a single tool invocation during an agentic loop.

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | string | Tool call identifier |
| tool_name | string | Name of the skill/tool |
| input | any (JSON) | Input sent to the tool |
| output | any (JSON) | Output received from the tool |
| duration | duration | Execution time |
| error | string | Error message (if failed) |

### TokenUsage

Token consumption for an invocation.

| Field | Type | Description |
| ----- | ---- | ----------- |
| input_tokens | int | Tokens in LLM input |
| output_tokens | int | Tokens in LLM output |
| cache_read | int | Tokens read from prompt cache |
| cache_write | int | Tokens written to prompt cache |
| total | int | Total tokens consumed |

## Identity and Uniqueness Rules

- **Package**: Unique by name (globally, within a registry)
- **All language entities**: Unique by name within a package. FQN (`{package}/{Kind}/{name}`) is globally unique.
- **Session**: Unique by UUID. Scoped to an agent FQN.
- **Invocation**: Unique by UUID. Scoped to an agent FQN.
- **Deploy targets**: Unique by name within a package. At most one may have `default true`.

## Relationships

| From | To | Cardinality | Constraint |
| ---- | -- | ----------- | ---------- |
| Agent | Prompt | many-to-many | Via `uses prompt` references. Must resolve within package. |
| Agent | Skill | many-to-many | Via `uses skill` references. Must resolve within package. |
| Agent | DeployTarget | one-to-many | Implicit (all agents deploy to all targets in package). |
| Skill | Type | many-to-one | Input/output fields may reference a named Type. |
| Skill | ToolConfig | one-to-one | Each skill has exactly one tool binding. |
| Pipeline | PipelineStep | one-to-many | Steps are ordered within a pipeline. |
| PipelineStep | Agent | many-to-one | Each step references one agent. |
| Session | Agent | many-to-one | Each session belongs to one agent. |
| Invocation | Session | many-to-one | Optional. Stateless invocations have no session. |
| Invocation | ToolCallRecord | one-to-many | Each invocation may produce zero or more tool calls. |
