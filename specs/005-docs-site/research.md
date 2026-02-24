# Research: AgentSpec Documentation Site

**Branch**: `005-docs-site` | **Date**: 2026-02-23

## R1: Static Site Generator Choice

### Decision: MkDocs + Material for MkDocs

### Rationale

MkDocs with Material for MkDocs scored highest across the weighted evaluation criteria:

- **Lowest setup complexity**: Single `mkdocs.yml` configuration file. No npm dependencies, no Git submodules, no theme compilation. `pip install mkdocs-material` installs everything.
- **Best multi-section navigation**: The `navigation.tabs` feature gives User Guide and Developer Guide each their own top-level tab with independent sidebar trees. This is a YAML config option, not a structural workaround.
- **Best built-in search**: Client-side lunr.js search with zero configuration, keyboard shortcuts (`/` to search), suggestions, and match highlighting. No external service required.
- **First-class Mermaid support**: A few lines in `mkdocs.yml` via `pymdownx.superfences` custom fences. No plugins to install.
- **Custom syntax highlighting**: Pygments lexer for IntentLang can be defined as a small Python package. Simpler than forking Chroma (Hugo) and comparable to Prism.js (Docusaurus).
- **Fast CI builds**: ~30 seconds for medium-sized sites. With pip caching, total CI pipeline under 2 minutes. Easily meets <5 min requirement.
- **Built-in sitemap generation**: Zero configuration.
- **Proven at scale**: Used by FastAPI, Pydantic, Cloudflare Workers, and many infrastructure/CLI tool projects.

### Alternatives Considered

| Option | Why Rejected |
|--------|-------------|
| Hugo + Docsy | Custom syntax highlighting with Chroma is significantly harder. Multi-section navigation requires manual configuration. Docsy's npm dependency undermines the "pure Go" advantage. Viable fallback if team prefers Go tooling. |
| Docusaurus | No built-in search (requires Algolia or third-party plugin). Node.js/React dependency is heavy and misaligned with Go project. Slowest build times of all three. |

---

## R2: Code Example Validation Approach

### Decision: Go Integration Test with Line Scanner

### Rationale

A Go test file in `integration_tests/` that scans documentation Markdown files, extracts fenced code blocks tagged with `ias`, and validates them using the project's own `parser.Parse()` and `validate.ValidateStructural()` / `validate.ValidateSemantic()` functions.

- **Uses existing patterns**: The project already has integration tests calling `parser.Parse()` + validators. A doc-example test is just another integration test.
- **Already in CI**: Runs via `go test ./... -count=1` — no new CI step needed.
- **No version skew**: Calls Go functions directly instead of shelling out to a binary.
- **No external dependency required**: A simple line-by-line scanner reliably extracts fenced code blocks. No need for `gomarkdown/markdown` or any external tool.

### Fence Tag Convention

| Tag | Behavior |
|-----|----------|
| ` ```ias ` | Complete, valid `.ias` file. Must pass `agentspec validate`. |
| ` ```ias fragment ` | Pedagogical fragment. Test wraps with synthetic `package` header before validation. |
| ` ```ias invalid ` | Intentionally invalid example. Test asserts validation *fails*. |
| ` ```ias novalidate ` | Pseudocode or conceptual. Test skips entirely. |
| ` ``` ` (no tag) | Not IntentLang (bash, ASCII, etc.). Skipped. |

### Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Shell script extraction | Fragile regex parsing, poor error messages, requires external tool. |
| MkDocs plugin hook | Couples validation to SSG choice. Requires Python for validation. Separate CI concern. |
| Separate `.ias` files with includes | Cannot handle fragments. Forces documentation restructuring. |
| Pre-commit hook | Not a substitute for CI. Developers can skip hooks. |

---

## R3: Existing Content Inventory

### High-Value Content for Reuse

| Source | Content | Target Section |
|--------|---------|---------------|
| `README.md` | Quick start, CLI overview, IntentLang syntax | Getting Started, CLI Reference |
| `ARCHITECTURE.md` | System design, data flow, 28 components, threat model | Developer Guide: Architecture |
| `spec/spec.md` | Complete IntentLang 2.0 language spec, 100+ keywords | Language Reference |
| `CHANGELOG.md` | v0.1.0, v0.2.0, v0.3.0 release notes | Changelog section |
| `examples/` (10 dirs) | 10 complete examples with README.md and .ias files | Tutorials, Use Cases |
| `cmd/agentspec/*.go` | 16 CLI commands with flags | CLI Reference |
| `integration_tests/testdata/` | 11 validated .ias test files | Code examples |
| `internal/templates/` | 5 project templates | Templates guide |

### Statistics

- 50,000+ lines of existing documentation across 20+ Markdown files
- 10 complete, self-contained examples (600+ lines of .ias code)
- 16 fully implemented CLI commands
- 27 internal packages documented in ARCHITECTURE.md
- 100+ language keywords in spec/spec.md

---

## R4: GitHub Pages Deployment

### Decision: GitHub Actions with `peaceiris/actions-gh-pages`

### Rationale

- MkDocs has a built-in `mkdocs gh-deploy` command, but using a dedicated GitHub Action provides more control over the deployment process.
- `peaceiris/actions-gh-pages` is the standard community action (4.9k+ stars) for deploying static sites to GitHub Pages.
- The workflow builds the site, validates `.ias` examples via `go test`, and deploys the built static files.
- Deployment triggers on pushes to `main` branch.

### Alternative: `mkdocs gh-deploy`

Simpler but less flexible. Doesn't integrate cleanly with the Go test validation step. The GitHub Action approach allows the full pipeline (build Go binary → validate examples → build site → deploy) in a single workflow.

---

## R5: Custom Pygments Lexer for IntentLang

### Decision: Ship a minimal Pygments lexer as part of the docs site

### Rationale

- Pygments lexers are defined as Python classes with regex-based token rules.
- IntentLang has a small, regular syntax: ~50 keywords, double-quoted strings, `#` and `//` comments, `{ }` block delimiters, numbers, and booleans.
- The lexer can be defined in a single Python file (~80 lines) and registered via a `setup.py` entry point or installed as part of the docs build.
- This enables proper syntax highlighting in all documentation code blocks.

### Token Categories

| Token Type | Patterns |
|-----------|----------|
| Keywords | `package`, `version`, `lang`, `prompt`, `skill`, `agent`, `deploy`, `target`, `tool`, `pipeline`, `step`, `type`, `secret`, `environment`, `policy`, `plugin`, `server`, `client`, `uses`, `model`, `strategy`, etc. |
| Strings | `"..."` with escape sequences |
| Comments | `#` and `//` line comments |
| Numbers | Integer and decimal literals |
| Booleans | `true`, `false` |
| Operators | `{`, `}`, `[`, `]` |
| Attributes | `required`, `default`, `from`, `to`, `when` |
