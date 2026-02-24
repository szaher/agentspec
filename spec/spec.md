# IntentLang Language Specification v2.0

## Overview

IntentLang (`.ias` files) is a declarative language for defining
AI agent configurations, prompts, skills with tool backends, MCP
server/client connections, deployment targets, multi-agent pipelines,
custom types, environment overlays, and security policies.

IntentLang 2.0 replaces `execution command` with typed `tool` blocks,
`binding` with `deploy` targets, and adds agent runtime configuration,
prompt variables, type definitions, pipelines, and agent delegation.

Files using `lang "1.0"` are rejected. Use `agentspec migrate --to-v2`
to convert 1.0 files.

## Syntax

### Package Declaration (required, must be first)

```
package "<name>" version "<semver>" lang "2.0"
```

### Resource Blocks

```
<resource-type> "<name>" {
  <attribute> <value>
  ...
}
```

### Keywords

| Keyword       | Usage                                         |
|---------------|-----------------------------------------------|
| package       | Package declaration                           |
| version       | Version specification                         |
| lang          | Language version (must be "2.0")               |
| prompt        | Prompt resource declaration                   |
| skill         | Skill resource declaration                    |
| agent         | Agent resource declaration                    |
| deploy        | Deployment target declaration (replaces binding) |
| target        | Deploy target type (process, docker, kubernetes) |
| pipeline      | Multi-step agent workflow declaration         |
| step          | Pipeline step declaration                     |
| type          | Custom type definition                        |
| tool          | Tool backend specification (inside skill)     |
| delegate      | Agent delegation rule                         |
| uses          | Reference (agent uses prompt/skill)           |
| model         | LLM model identifier                          |
| input         | Input schema block                            |
| output        | Output schema block                           |
| description   | Human-readable description                    |
| content       | Content value (prompt text)                   |
| variables     | Prompt variable declarations block            |
| default       | Default value or default deploy flag          |
| required      | Field/variable requirement marker             |
| enum          | Enumeration type declaration                  |
| list          | List type declaration                         |
| secret        | Secret reference declaration                  |
| environment   | Environment overlay declaration               |
| policy        | Security policy declaration                   |
| plugin        | Plugin reference                              |
| server        | MCP server declaration                        |
| client        | MCP client declaration                        |
| connects      | Connection reference                          |
| exposes       | Exposure reference                            |
| env           | Environment variable source                   |
| store         | Secret store source                           |
| command       | Command execution type / tool type            |
| require       | Policy rule: require                          |
| deny          | Policy rule: deny                             |
| allow         | Policy rule: allow                            |
| to            | Connection/delegation preposition             |
| when          | Delegation condition                          |
| from          | Data source reference                         |
| true/false    | Boolean literals                              |
| import        | External package import                       |
| transport     | Transport protocol                            |
| url           | URL value                                     |
| auth          | Authentication reference                      |
| args          | Command arguments                             |
| metadata      | Metadata block                                |
| strategy      | Agent execution strategy                      |
| max_turns     | Maximum agentic loop iterations               |
| timeout       | Invocation timeout duration                   |
| token_budget  | Maximum token consumption per invocation      |
| temperature   | LLM sampling temperature                      |
| stream        | Enable streaming responses                    |
| on_error      | Error handling strategy (retry, fail, fallback) |
| max_retries   | Maximum retry count                           |
| fallback      | Fallback agent reference                      |
| parallel      | Pipeline step parallel execution flag         |
| depends_on    | Pipeline step dependency list                 |
| health        | Health check configuration                    |
| autoscale     | Auto-scaling configuration                    |
| resources     | Resource limits block                         |
| memory        | Memory limit / memory strategy                |

### Comments

Line comments start with `#` or `//`.

### Strings

Strings are enclosed in double quotes. Escape sequences: `\n`, `\t`, `\\`, `\"`.
Multiline strings are supported within quotes.
Template variables use `{{name}}` syntax inside prompt content strings.

### Resource Types

#### Core Resources

- `agent` — AI agent with model, prompt, skills, execution strategy, and runtime limits
- `prompt` — Prompt template with content and optional variable declarations
- `skill` — Capability with typed input/output schema and a tool backend
- `server` — MCP server with transport configuration
- `client` — MCP client connecting to servers

#### Deployment & Configuration

- `deploy` — Deployment target (process, docker, kubernetes, docker-compose)
- `secret` — Secret reference (env or store)
- `environment` — Environment overlay with attribute overrides
- `policy` — Security constraint rules
- `plugin` — Plugin dependency reference

#### IntentLang 2.0 Additions

- `type` — Custom data type with fields, enums, or lists
- `pipeline` — Multi-agent workflow with ordered/parallel steps

### Tool Blocks (inside `skill`)

Skills declare their backing tool using one of four variants:

```
skill "name" {
  tool mcp "server-name/tool-name"        # MCP server tool
  tool http { method "GET" url "..." }     # HTTP API call
  tool command { binary "cmd" args [...] } # Local subprocess
  tool inline { language "python" code "..." } # Inline code (sandboxed)
}
```

### Deploy Blocks

```
deploy "name" target "type" {
  port 8080
  default true
  # target-specific attributes
}
```

Target types: `process`, `docker`, `kubernetes`, `docker-compose`.

### Agent Runtime Configuration

```
agent "name" {
  uses prompt "system"
  uses skill "search"
  model "claude-sonnet-4-20250514"
  strategy "react"          # react | plan-execute | reflexion | router | map-reduce
  max_turns 10
  timeout "60s"
  token_budget 50000
  temperature 0.7
  stream true
  on_error "retry"          # retry | fail | fallback
  max_retries 3
  fallback "backup-agent"
  delegate to agent "other" when "condition"
}
```

### Prompt Variables

```
prompt "greeting" {
  content "Hello {{name}}, you are a {{role}}."
  variables {
    name string required
    role string required default "assistant"
  }
}
```

### Type Definitions

```
type "user" {
  name string required
  email string required
  age int
  active bool default "true"
}

type "status" enum ["pending", "active", "archived"]

type "tags" list string
```

### Pipeline Blocks

```
pipeline "workflow" {
  step "first" {
    agent "analyzer"
    input "raw data"
    output "analysis"
  }
  step "second" {
    agent "reporter"
    depends_on ["first"]
    output "final report"
  }
}
```

### Canonical Formatting

The `agentspec fmt` command produces canonical output with:
- 2-space indentation
- One blank line between resource blocks
- Sorted metadata keys
- Consistent quoting of all string values

### Deprecations in 2.0

- `binding` block — replaced by `deploy` block
- `execution command "..."` — replaced by `tool command { binary "..." }`
- `lang "1.0"` — rejected; use `agentspec migrate --to-v2` to convert
