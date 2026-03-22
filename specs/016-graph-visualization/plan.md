# Implementation Plan: Interactive Dependency Graph Visualization

**Branch**: `016-graph-visualization` | **Date**: 2026-03-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/016-graph-visualization/spec.md`

## Summary

Add an `agentspec graph` command that parses `.ias` files and renders interactive dependency graphs. The command extracts all entities (agents, prompts, skills, MCP servers/clients, pipelines, secrets, policies, guardrails, users, deploy targets, state config, environments, types, plugins) and their relationships from the AST, then outputs as: (1) interactive web UI served on localhost, (2) Graphviz DOT, or (3) Mermaid markdown. Reuses the existing parser, AST types, `resolveFiles()`, and `go:embed` patterns already established in the project.

## Technical Context

**Language/Version**: Go 1.25+ (existing project language)
**Primary Dependencies**: cobra v1.10.2 (existing CLI), `embed` (stdlib, existing for web assets), `net/http` (stdlib, web server), `os/exec` (stdlib, browser launch), `encoding/json` (stdlib, API endpoint)
**Storage**: N/A (read-only command, no state)
**Testing**: `go test ./... -count=1` (existing test infrastructure)
**Target Platform**: macOS, Linux, Windows (cross-platform CLI)
**Project Type**: CLI command extension
**Performance Goals**: Graph construction <1s for 50 entities; web UI render <2s for 200 nodes; 60fps interactions
**Constraints**: Offline-capable (all assets embedded), localhost-only web server (127.0.0.1), zero external runtime dependencies, read-only (never modifies files)
**Scale/Scope**: Up to 200 nodes per graph, typical projects 10-50 entities

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Determinism | PASS | DOT and Mermaid outputs are sorted deterministically. Web UI renders the same graph data. |
| II. Idempotency | PASS | Read-only command — never mutates state or files. |
| III. Portability | PASS | No platform-specific behavior except browser launch (which uses OS-appropriate commands and is best-effort). |
| IV. Separation of Concerns | PASS | Graph extraction operates on AST nodes (surface syntax), not IR semantics. This is intentional — the graph visualizes the DSL structure, not the lowered IR. Graph model is independent of rendering format. |
| V. Reproducibility | PASS | DOT/Mermaid output is deterministic. Web UI serves the same data structure. |
| VI. Safe Defaults | PASS | Web server binds to localhost only. No secrets involved. |
| VII. Minimal Surface Area | PASS | One new command (`graph`), justified by the concrete use case of architecture visualization. No new keywords or DSL constructs. |
| VIII. English-Friendly Syntax | N/A | No syntax changes. |
| IX. Canonical Formatting | N/A | No formatting changes. Graph outputs have their own deterministic ordering. |
| X. Strict Validation | PASS | Parse errors include file/line info, displayed to stderr. Graph renders successfully parsed files. |
| XI. Explicit References | N/A | No new imports or dependencies requiring pinning. |
| XII. No Hidden Behavior | PASS | All graph extraction is explicit — entities and relationships map directly from AST fields. |
| Pre-Commit Validation | PASS | Standard `gofmt`, `go build`, `go test` apply. |

**Result**: All gates pass. No violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/016-graph-visualization/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── graph-api.md     # JSON API contract for web UI
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
cmd/agentspec/
├── graph.go             # CLI command: agentspec graph [files|dir] [flags]
└── main.go              # Register newGraphCmd() (one line addition)

internal/graph/
├── extract.go           # AST → GraphModel extraction (entity + relationship extraction)
├── extract_test.go      # Unit tests for extraction logic
├── model.go             # GraphNode, GraphEdge, Graph data types
├── dot.go               # Graph → DOT format renderer
├── dot_test.go          # Unit tests for DOT output
├── mermaid.go           # Graph → Mermaid format renderer
├── mermaid_test.go      # Unit tests for Mermaid output
├── server.go            # HTTP server for web UI mode (serves embedded assets + /api/graph JSON)
├── server_test.go       # Unit tests for server (API endpoint, graceful shutdown)
├── web/                 # Embedded web assets
│   └── index.html       # Single-file SPA: HTML + CSS + JS (vanilla, no framework)
├── embed.go             # go:embed directive for web/ directory
└── browser.go           # Cross-platform browser launch (open/xdg-open/start)

integration_tests/
└── graph_test.go        # Integration tests: parse real .ias files → verify graph output
```

**Structure Decision**: Follows the existing project pattern — CLI command file in `cmd/agentspec/`, new internal package `internal/graph/` for the graph logic (extraction, rendering, serving). Web assets embedded via `go:embed` following the same pattern as `internal/frontend/handler.go`. The `internal/graph/` package is self-contained with no dependencies on state, runtime, or apply — it only depends on `internal/ast` and `internal/parser`.

## Design Decisions

### AST vs IR for Graph Extraction

The graph extractor operates on **AST nodes** rather than the lowered IR. Rationale:

1. **Complete entity coverage**: The AST preserves all entity types (User, Guardrail, Environment, TypeDef, Plugin) that are flattened or lost in the IR's generic `Resource` type.
2. **Relationship fidelity**: AST nodes contain typed references (e.g., `agent.Delegates`, `agent.GuardrailRefs`, `pipeline.Steps`) while IR uses generic `References []string`.
3. **Source location**: AST nodes carry `StartPos`/`EndPos` for file/line display. IR nodes do not.
4. **No lowering side effects**: The graph is a read-only visualization of what the user wrote, not the resolved semantics. Users want to see their configuration structure, not the compiled output.

### Web UI Architecture

Single HTML file with embedded CSS and JS (following the existing `internal/frontend/web/` pattern):

- **Layout engine**: D3.js force simulation (vendored, embedded). D3 is the standard for graph visualization in the browser, well-tested, and small enough to embed (~100KB minified).
- **No build step**: Vanilla JS, no npm, no bundler — consistent with the existing frontend approach.
- **Data flow**: Server serves `index.html` on `/`, graph data on `/api/graph` as JSON. The HTML page fetches data on load and renders.
- **Graceful shutdown**: HTTP server listens for SIGINT/SIGTERM via `signal.NotifyContext`.

### Deterministic Output

DOT and Mermaid renderers sort nodes by ID and edges by (source, target, label) to ensure identical output for identical input. This enables meaningful `git diff` of generated graph files.

## Complexity Tracking

No constitution violations to justify. The design is straightforward:
- 1 new CLI command
- 1 new internal package with 6 source files
- 1 embedded HTML file
- Reuses existing parser and AST
