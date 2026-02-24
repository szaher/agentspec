# Contract: Documentation Site Structure

**Version**: 1.0.0 | **Date**: 2026-02-23

## Overview

This contract defines the directory layout, file naming conventions, and configuration schema for the AgentSpec documentation site built with MkDocs + Material for MkDocs.

## Directory Layout

```text
docs/                           # Documentation source root
├── index.md                    # Site homepage
├── user-guide/                 # User Guide tab
│   ├── getting-started/
│   │   ├── index.md            # Installation & overview
│   │   ├── quickstart.md       # Quick start tutorial
│   │   └── concepts.md         # Core concepts
│   ├── language/
│   │   ├── index.md            # Language overview
│   │   ├── agent.md            # agent block reference
│   │   ├── prompt.md           # prompt block reference
│   │   ├── skill.md            # skill block reference
│   │   ├── tool.md             # tool block reference (mcp, http, command, inline)
│   │   ├── deploy.md           # deploy block reference
│   │   ├── pipeline.md         # pipeline block reference
│   │   ├── type.md             # type definition reference
│   │   ├── server.md           # server block reference
│   │   ├── client.md           # client block reference
│   │   ├── secret.md           # secret block reference
│   │   ├── environment.md      # environment block reference
│   │   ├── policy.md           # policy block reference
│   │   └── plugin.md           # plugin block reference
│   ├── configuration/
│   │   ├── runtime.md          # Agent runtime attributes
│   │   ├── prompt-variables.md # Template variables
│   │   ├── error-handling.md   # Error strategies
│   │   └── delegation.md       # Agent delegation
│   ├── use-cases/
│   │   ├── index.md            # Architecture overview & comparison table
│   │   ├── react.md            # ReAct agent
│   │   ├── plan-execute.md     # Plan-and-Execute
│   │   ├── reflexion.md        # Reflexion
│   │   ├── router.md           # Router / Triage
│   │   ├── map-reduce.md       # Map-Reduce
│   │   ├── pipeline.md         # Multi-Agent Pipeline
│   │   └── delegation.md       # Agent Delegation
│   ├── deployment/
│   │   ├── index.md            # Deployment overview
│   │   ├── process.md          # Local process target
│   │   ├── docker.md           # Docker target
│   │   ├── compose.md          # Docker Compose target
│   │   ├── kubernetes.md       # Kubernetes target
│   │   └── best-practices.md   # Production best practices
│   ├── cli/
│   │   ├── index.md            # CLI overview
│   │   ├── validate.md
│   │   ├── fmt.md
│   │   ├── plan.md
│   │   ├── apply.md
│   │   ├── run.md
│   │   ├── dev.md
│   │   ├── status.md
│   │   ├── logs.md
│   │   ├── destroy.md
│   │   ├── init.md
│   │   ├── migrate.md
│   │   ├── export.md
│   │   ├── diff.md
│   │   ├── sdk.md
│   │   └── version.md
│   ├── api/
│   │   ├── index.md            # API overview & authentication
│   │   ├── agents.md           # Agent endpoints (invoke, stream)
│   │   ├── sessions.md         # Session endpoints
│   │   ├── pipelines.md        # Pipeline endpoints
│   │   └── health-metrics.md   # Health check & metrics
│   ├── sdks/
│   │   ├── python.md
│   │   ├── typescript.md
│   │   └── go.md
│   ├── migration.md            # IntentLang 1.0 to 2.0
│   └── changelog.md
│
├── developer-guide/            # Developer Guide tab
│   ├── architecture/
│   │   ├── index.md            # Architecture overview
│   │   ├── parser.md           # Parser pipeline
│   │   ├── ir.md               # IR and plan engine
│   │   ├── runtime.md          # Runtime and agentic loop
│   │   ├── adapters.md         # Adapter system
│   │   └── plugins.md          # Plugin host
│   ├── contributing/
│   │   ├── index.md            # Build from source
│   │   ├── testing.md          # Running tests
│   │   ├── code-style.md       # Code conventions
│   │   └── pull-requests.md    # PR guidelines
│   ├── extending/
│   │   ├── adapters.md         # Writing custom adapters
│   │   └── plugins.md          # Writing WASM plugins
│   └── internals/
│       ├── state.md            # State management
│       ├── secrets.md          # Secret resolution
│       ├── telemetry.md        # Observability
│       └── sdk-generation.md   # SDK code generation
│
├── examples/                   # Shared example .ias files (validated during build)
│   ├── basic-agent.ias
│   ├── customer-support.ias
│   ├── code-review-pipeline.ias
│   ├── rag-chatbot.ias
│   ├── data-pipeline.ias
│   ├── react-agent.ias
│   ├── plan-execute-agent.ias
│   ├── reflexion-agent.ias
│   ├── router-agent.ias
│   ├── map-reduce-agent.ias
│   └── delegation-agent.ias
│
├── stylesheets/
│   └── extra.css               # Custom CSS overrides
│
└── overrides/                  # Theme overrides (if needed)

docs-tools/                     # Documentation tooling (not published)
├── pygments_intentlang/        # Custom Pygments lexer
│   ├── __init__.py
│   ├── lexer.py                # IntentLang token definitions
│   └── setup.py                # Pygments entry point registration
└── validate_examples.sh        # Optional CI helper script

mkdocs.yml                      # MkDocs configuration (site root)
```

## MkDocs Configuration Contract

The `mkdocs.yml` configuration MUST include:

### Required Settings

```yaml
site_name: AgentSpec Documentation
site_url: https://<org>.github.io/<repo>/
repo_url: https://github.com/<org>/<repo>

theme:
  name: material
  features:
    - navigation.tabs        # Separate User/Developer tabs
    - navigation.sections    # Group pages under section headings
    - navigation.expand      # Auto-expand navigation sections
    - navigation.path        # Breadcrumb navigation
    - navigation.top         # Back-to-top button
    - search.suggest         # Search suggestions
    - search.highlight       # Highlight search terms
    - content.code.copy      # Copy button on code blocks
    - content.tabs.link      # Linked content tabs

markdown_extensions:
  - pymdownx.superfences:      # Mermaid + custom fences
      custom_fences:
        - name: mermaid
          class: mermaid
          format: !!python/name:pymdownx.superfences.fence_mermaid
  - pymdownx.highlight:        # Syntax highlighting
      anchor_linenums: true
  - pymdownx.tabbed:           # Tabbed content (for SDK examples)
      alternate_style: true
  - admonition                 # Note/warning/tip boxes
  - pymdownx.details           # Collapsible sections
  - attr_list                  # Attribute lists for badges/buttons
  - def_list                   # Definition lists
  - tables                     # Tables

plugins:
  - search                     # Built-in search
  - social                     # Social cards (optional)
```

### Navigation Contract

The `nav` section MUST follow this two-tab structure:

```yaml
nav:
  - Home: index.md
  - User Guide:
    - Getting Started:
      - user-guide/getting-started/index.md
      - user-guide/getting-started/quickstart.md
      - user-guide/getting-started/concepts.md
    - Language Reference:
      - user-guide/language/index.md
      - ... (one entry per resource type)
    - ... (remaining sections)
  - Developer Guide:
    - Architecture:
      - developer-guide/architecture/index.md
      - ... (one entry per topic)
    - ... (remaining sections)
```

## File Naming Conventions

- All Markdown files use lowercase kebab-case: `prompt-variables.md`, not `PromptVariables.md`
- Section index files are named `index.md`
- CLI command pages match the command name exactly: `validate.md`, `fmt.md`, `apply.md`
- Language reference pages match the keyword name: `agent.md`, `prompt.md`, `skill.md`
- Example `.ias` files use kebab-case: `basic-agent.ias`, `customer-support.ias`

## Code Block Conventions

All IntentLang code blocks in Markdown MUST use the appropriate fence tag:

```markdown
<!-- Complete, valid example -->
```ias
package "example" version "1.0.0" lang "2.0"
agent "my-agent" { ... }
\```

<!-- Fragment (no package header) -->
```ias fragment
prompt "greeting" {
  content "Hello {{name}}"
}
\```

<!-- Intentionally invalid (for error docs) -->
```ias invalid
agent "bad" {
  model 42  # model must be a string
}
\```

<!-- Pseudocode / conceptual -->
```ias novalidate
agent "concept" {
  # Hypothetical future syntax
  memory persistent
}
\```
```

## CI Pipeline Contract

The GitHub Actions workflow for docs deployment MUST:

1. Build the `agentspec` binary from source
2. Run `go test ./integration_tests/ -run TestDocExamples` to validate all `.ias` code blocks
3. Install MkDocs and dependencies (`pip install mkdocs-material`)
4. Install the custom Pygments lexer
5. Build the site (`mkdocs build --strict`)
6. Deploy to GitHub Pages (on `main` branch pushes only)

Build MUST fail if any step fails. The `--strict` flag causes MkDocs to fail on warnings (broken links, missing pages).
