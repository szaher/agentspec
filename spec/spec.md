# Agentz DSL Language Specification v1.0

## Overview

The Agentz DSL (`.az` files) is a declarative language for defining
AI agent configurations, prompts, skills, MCP server/client connections,
environment overlays, and deployment bindings.

## Syntax

### Package Declaration (required, must be first)

```
package "<name>" version "<semver>" lang "<version>"
```

### Resource Blocks

```
<resource-type> "<name>" {
  <attribute> <value>
  ...
}
```

### Keywords

| Keyword     | Usage                                    |
|-------------|------------------------------------------|
| package     | Package declaration                      |
| version     | Version specification                    |
| lang        | Language version                         |
| prompt      | Prompt resource declaration              |
| skill       | Skill resource declaration               |
| agent       | Agent resource declaration               |
| binding     | Adapter binding declaration              |
| uses        | Reference (agent uses prompt/skill)      |
| model       | Model identifier                         |
| input       | Input schema block                       |
| output      | Output schema block                      |
| execution   | Execution specification                  |
| description | Human-readable description               |
| content     | Content value (prompt text)              |
| default     | Default flag for bindings                |
| adapter     | Adapter identifier                       |
| secret      | Secret reference declaration             |
| environment | Environment overlay declaration          |
| policy      | Security policy declaration              |
| plugin      | Plugin reference                         |
| server      | MCP server declaration                   |
| client      | MCP client declaration                   |
| connects    | Connection reference                     |
| exposes     | Exposure reference                       |
| env         | Environment variable source              |
| store       | Secret store source                      |
| command     | Command execution type                   |
| require     | Policy rule: require                     |
| deny        | Policy rule: deny                        |
| allow       | Policy rule: allow                       |
| to          | Connection preposition                   |
| true/false  | Boolean literals                         |
| required    | Field requirement marker                 |
| import      | External package import                  |
| transport   | Transport protocol                       |
| url         | URL value                                |
| auth        | Authentication reference                 |
| args        | Command arguments                        |
| metadata    | Metadata block                           |

### Comments

Line comments start with `#` or `//`.

### Strings

Strings are enclosed in double quotes. Escape sequences: `\n`, `\t`, `\\`, `\"`.
Multiline strings are supported within quotes.

### Resource Types

- `agent` — AI agent with model, prompt, and skills
- `prompt` — Prompt template with content and variables
- `skill` — Capability with input/output schema and execution
- `server` — MCP server with transport configuration
- `client` — MCP client connecting to servers
- `secret` — Secret reference (env or store)
- `environment` — Environment overlay with attribute overrides
- `policy` — Security constraint rules
- `binding` — Adapter deployment target
- `plugin` — Plugin dependency reference

### Canonical Formatting

The `agentz fmt` command produces canonical output with:
- 2-space indentation
- One blank line between resource blocks
- Sorted metadata keys
- Consistent quoting of all string values
