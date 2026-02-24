# Data Model: AgentSpec Documentation Site

**Branch**: `005-docs-site` | **Date**: 2026-02-23

## Entities

### Page

A single documentation page rendered as an HTML document.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| title | string | yes | Page title displayed in browser tab and navigation |
| path | string | yes | URL path relative to site root (e.g., `user-guide/language/agent/`) |
| audience | enum | yes | `user` or `developer` — determines navigation section |
| section | string | yes | Parent navigation section (e.g., "Language Reference", "Deployment") |
| content | markdown | yes | Page body in Markdown format |
| nav_order | integer | no | Sort order within section (lower = earlier) |
| description | string | no | Meta description for search engines |

**Validation Rules**:
- `path` must be unique across all pages
- `path` must use lowercase kebab-case
- `audience` must be one of: `user`, `developer`

### CodeExample

An IntentLang code snippet embedded within a Page.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| source_file | string | yes | Path to the Markdown file containing this example |
| block_index | integer | yes | Zero-based index of the code block within the file |
| tag | enum | yes | Fence tag: `ias`, `ias fragment`, `ias invalid`, `ias novalidate` |
| content | string | yes | Raw IntentLang source code |
| validation_status | enum | computed | `pass`, `fail`, `skip` — computed during build |
| heading_context | string | computed | Nearest preceding heading for error reporting |

**Validation Rules**:
- `tag=ias`: content must pass `agentspec validate` as a complete file
- `tag=ias fragment`: content must pass validation when wrapped with a synthetic package header
- `tag=ias invalid`: content must fail validation (test asserts failure)
- `tag=ias novalidate`: validation skipped

### UseCase

A documented agentic architecture pattern.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | yes | Pattern name (e.g., "ReAct Agent", "Multi-Agent Pipeline") |
| strategy | enum | yes | `react`, `plan-execute`, `reflexion`, `router`, `map-reduce`, `pipeline`, `delegation` |
| problem | string | yes | Problem statement describing when to use this pattern |
| diagram | mermaid | yes | Architecture diagram in Mermaid format |
| example_file | string | yes | Path to complete `.ias` example file |
| deploy_target | string | yes | At least one deployment target configuration |
| trade_offs | object | yes | Complexity, latency, cost, and recommended use case |

### NavigationSection

A grouping of Pages in the site navigation.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| title | string | yes | Section display name |
| audience | enum | yes | `user` or `developer` |
| nav_order | integer | yes | Sort order among sections |
| pages | Page[] | yes | Ordered list of pages in this section |
| icon | string | no | Material icon identifier for navigation display |

### CLICommand

A documented CLI subcommand.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | yes | Command name (e.g., `validate`, `apply`, `run`) |
| usage | string | yes | Usage syntax string |
| description | string | yes | Brief description |
| flags | Flag[] | yes | List of available flags |
| examples | string[] | yes | At least one usage example |
| output_success | string | yes | Example output for success case |
| output_error | string | no | Example output for error case |

### Flag

A CLI command flag.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | yes | Flag name including `--` prefix |
| short | string | no | Short form (e.g., `-f`) |
| type | enum | yes | `string`, `bool`, `int` |
| default | string | no | Default value |
| description | string | yes | Flag description |

## Relationships

```
NavigationSection (1) ──── contains ────> (N) Page
Page (1) ──── contains ────> (N) CodeExample
UseCase (1) ──── is-a ────> (1) Page
CLICommand (1) ──── is-a ────> (1) Page
```

## Navigation Structure

```
Site Root
├── User Guide (audience=user)
│   ├── Getting Started (section)
│   │   ├── Installation
│   │   ├── Quick Start
│   │   └── Concepts
│   ├── Language Reference (section)
│   │   ├── Overview
│   │   ├── agent
│   │   ├── prompt
│   │   ├── skill
│   │   ├── tool (mcp, http, command, inline)
│   │   ├── deploy
│   │   ├── pipeline
│   │   ├── type
│   │   ├── server
│   │   ├── client
│   │   ├── secret
│   │   ├── environment
│   │   ├── policy
│   │   └── plugin
│   ├── Agent Configuration (section)
│   │   ├── Runtime Attributes
│   │   ├── Prompt Variables
│   │   ├── Error Handling
│   │   └── Agent Delegation
│   ├── Use Cases (section)
│   │   ├── Architecture Overview & Comparison
│   │   ├── ReAct Agent
│   │   ├── Plan-and-Execute
│   │   ├── Reflexion
│   │   ├── Router / Triage
│   │   ├── Map-Reduce
│   │   ├── Multi-Agent Pipeline
│   │   └── Agent Delegation
│   ├── Deployment (section)
│   │   ├── Overview
│   │   ├── Local Process
│   │   ├── Docker
│   │   ├── Docker Compose
│   │   ├── Kubernetes
│   │   └── Production Best Practices
│   ├── CLI Reference (section)
│   │   ├── validate
│   │   ├── fmt
│   │   ├── plan
│   │   ├── apply
│   │   ├── run
│   │   ├── dev
│   │   ├── status
│   │   ├── logs
│   │   ├── destroy
│   │   ├── init
│   │   ├── migrate
│   │   ├── export
│   │   ├── diff
│   │   ├── sdk
│   │   └── version
│   ├── HTTP API Reference (section)
│   │   ├── Authentication
│   │   ├── Agent Invocation
│   │   ├── Streaming
│   │   ├── Sessions
│   │   ├── Pipelines
│   │   ├── Health & Metrics
│   │   └── Error Responses
│   ├── SDKs (section)
│   │   ├── Python
│   │   ├── TypeScript
│   │   └── Go
│   ├── Migration Guide (section)
│   │   └── IntentLang 1.0 to 2.0
│   └── Changelog (section)
│
└── Developer Guide (audience=developer)
    ├── Architecture (section)
    │   ├── Overview
    │   ├── Parser Pipeline
    │   ├── IR and Plan Engine
    │   ├── Runtime and Agentic Loop
    │   ├── Adapter System
    │   └── Plugin Host
    ├── Contributing (section)
    │   ├── Build from Source
    │   ├── Running Tests
    │   ├── Code Style
    │   └── Pull Request Guidelines
    ├── Extending AgentSpec (section)
    │   ├── Writing a Custom Adapter
    │   └── Writing a WASM Plugin
    └── Internals (section)
        ├── State Management
        ├── Secret Resolution
        ├── Telemetry
        └── SDK Generation
```

## Page Count Estimate

| Section | Audience | Pages |
|---------|----------|-------|
| Getting Started | User | 3 |
| Language Reference | User | 14 |
| Agent Configuration | User | 4 |
| Use Cases | User | 8 |
| Deployment | User | 6 |
| CLI Reference | User | 15 |
| HTTP API Reference | User | 7 |
| SDKs | User | 3 |
| Migration Guide | User | 1 |
| Changelog | User | 1 |
| Architecture | Developer | 5 |
| Contributing | Developer | 4 |
| Extending AgentSpec | Developer | 2 |
| Internals | Developer | 4 |
| **Total** | | **77** |
