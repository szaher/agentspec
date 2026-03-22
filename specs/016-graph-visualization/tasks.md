# Tasks: Interactive Dependency Graph Visualization

**Input**: Design documents from `/specs/016-graph-visualization/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Constitution mandates integration tests as the primary quality gate. Unit tests for the `internal/graph/` package and integration tests for end-to-end graph generation are included.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the new `internal/graph/` package structure and register the CLI command

- [x] T001 Create `internal/graph/` package directory and `internal/graph/model.go` with `GraphNode`, `GraphEdge`, `Graph`, `PackageInfo`, `GraphStats` types per data-model.md
- [x] T002 Create `internal/graph/web/` directory for embedded web assets (empty `index.html` placeholder)
- [x] T003 Create `internal/graph/embed.go` with `//go:embed web` directive exposing `WebFS embed.FS`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core extraction logic that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Implement `Extract(files []*ast.File) *Graph` function in `internal/graph/extract.go` that walks AST statements and builds GraphNode entries for 17 entity types (agent, prompt, skill, mcp_server, mcp_client, pipeline, step, secret, policy, guardrail, user, binding, deploy, state, env, type, plugin). File-level nodes (`file` type) are added by T022 (US3).
- [x] T005 Extend `Extract` in `internal/graph/extract.go` to build GraphEdge entries for 14 relationship types (uses prompt, uses skill, uses guardrail, uses client, delegates to, fallback, contains, invokes, depends on, connects to, uses tool, can access, governs, overrides) with correct source/target IDs and labels. File-level edges (`imports`, `defines`) are added by T022 (US3).
- [x] T006 Add `PopulateAttributes(node *GraphNode, stmt ast.Statement)` helper in `internal/graph/extract.go` to fill type-specific attributes for each entity (model, strategy, content preview, input/output schemas, description, etc.)
- [x] T007 Add `ComputeStats(g *Graph)` function in `internal/graph/extract.go` to populate `GraphStats` with node/edge/file counts and per-type breakdown
- [x] T008 Handle unresolved references in `internal/graph/extract.go`: when a relationship target is not found in parsed nodes, create a placeholder node with `missing: true` attribute
- [x] T009 Write unit tests for `Extract` covering all 18 entity types and 16 relationship types, unresolved references, and stats computation in `internal/graph/extract_test.go`

**Checkpoint**: Foundation ready — graph extraction from AST produces complete, correct Graph model. All user stories can now proceed.

---

## Phase 3: User Story 1 — Single-File Interactive Visualization (Priority: P1) MVP

**Goal**: Run `agentspec graph myfile.ias` and see an interactive graph in the browser

**Independent Test**: Run `agentspec graph examples/multi-agent-router.ias`, verify browser opens with correct nodes, edges, sidebar, and legend

### Implementation for User Story 1

- [x] T010 [P] [US1] Implement `internal/graph/server.go` with `Serve(g *Graph, port int, theme string) error` that starts an HTTP server on 127.0.0.1, serves embedded web assets on `/`, serves graph JSON on `/api/graph`, handles graceful shutdown on SIGINT/SIGTERM via `signal.NotifyContext`, and detects port-in-use errors returning a clear message suggesting `--port <alternative>`
- [x] T011 [P] [US1] Implement `internal/graph/browser.go` with `OpenBrowser(url string) error` that uses `os/exec` with `runtime.GOOS` switch: `open` (darwin), `xdg-open` (linux), `cmd /c start` (windows), failing silently on error
- [x] T012 [US1] Create `internal/graph/web/index.html` — single-file SPA with embedded CSS and JS: (a) dark theme styles with CSS variables for colors, (b) D3.js v7 force simulation vendored inline (~100KB minified), (c) SVG canvas with zoom/pan via d3-zoom, (d) color-coded nodes by entity type per data-model.md color mapping, (e) labeled directed edges, (f) click handler showing detail sidebar (right panel, ~300px) with entity name, type, file:line, attributes, and connected entities, (g) legend panel (bottom-left) with all entity types and colors, (h) header bar with package name, node/edge/file counts, (i) fetch `/api/graph` on load and render graph, (j) render nodes with `missing: true` attribute using dashed borders and a warning icon to indicate unresolved references, (k) if API response contains `errors` array, show a dismissible warning banner at the top listing parse error count and file names
- [x] T013 [US1] Create `cmd/agentspec/graph.go` with `newGraphCmd() *cobra.Command` that: accepts file/directory args (default `.`), adds `--format` (web|dot|mermaid, default web), `--port` (default 8686), `--open`/`--no-open` (default true), `--theme` (dark|light, default dark), `--output`, `--no-files`, `--no-orphans` flags, calls `resolveFiles(args)` and `parseFiles(files)` to get `[]*ast.File`, calls `graph.Extract(astFiles)`, applies `--no-files` (remove nodes where Type=="file" and all edges referencing them) and `--no-orphans` (remove nodes with zero remaining edges), then recomputes stats via `ComputeStats`, and for `--format web`: calls `graph.Serve()` then `graph.OpenBrowser()` if `--open`
- [x] T014 [US1] Add `parseFiles(files []string) ([]*ast.File, []string, error)` helper in `cmd/agentspec/graph.go` that parses each file via `parser.Parse()`, collects AST files and parse error strings, and returns both (errors reported to stderr, valid files continue)
- [x] T015 [US1] Register `newGraphCmd()` in `cmd/agentspec/main.go` by adding `root.AddCommand(newGraphCmd())`
- [x] T016 [US1] Write unit tests for `Serve` (verify `/api/graph` returns valid JSON matching Graph schema, verify `/` returns HTML) in `internal/graph/server_test.go`

**Checkpoint**: `agentspec graph myfile.ias` opens browser with interactive graph. Nodes are color-coded, edges are labeled, sidebar shows details on click, legend is visible. Ctrl+C stops the server.

---

## Phase 4: User Story 2 — Static Format Export (Priority: P1) MVP

**Goal**: Export graph as Mermaid or Graphviz DOT for documentation embedding

**Independent Test**: Run `agentspec graph myfile.ias --format mermaid` and verify valid Mermaid output; run `--format dot` and verify valid DOT output

### Implementation for User Story 2

- [x] T017 [P] [US2] Implement `RenderDOT(g *Graph) string` in `internal/graph/dot.go` that produces valid `digraph` with `rankdir=LR`, nodes with distinct shapes per entity type (per research.md shape mapping), labeled edges, subgraph clusters by source file, and deterministic output (nodes and edges sorted by ID)
- [x] T018 [P] [US2] Implement `RenderMermaid(g *Graph) string` in `internal/graph/mermaid.go` that produces valid `graph LR` with Mermaid node shapes per entity type (per data-model.md Mermaid Shape column), `-->|label|` edges, subgraph grouping by source file, `classDef`/`class` for color-coding, and deterministic output (sorted)
- [x] T019 [P] [US2] Write unit tests for `RenderDOT` (valid DOT syntax, correct shapes, sorted output, subgraph clustering) in `internal/graph/dot_test.go`
- [x] T020 [P] [US2] Write unit tests for `RenderMermaid` (valid Mermaid syntax, correct shapes, sorted output, classDef styling) in `internal/graph/mermaid_test.go`
- [x] T021 [US2] Extend `cmd/agentspec/graph.go` to handle `--format dot` and `--format mermaid`: call the respective renderer, write to stdout or `--output` file, and exit (non-blocking)

**Checkpoint**: `agentspec graph myfile.ias --format dot` prints valid DOT. `--format mermaid` prints valid Mermaid. Output is deterministic. `--output file` writes to file.

---

## Phase 5: User Story 3 — Multi-File Project Visualization (Priority: P2)

**Goal**: Visualize a directory of `.ias` files with cross-file import relationships

**Independent Test**: Run `agentspec graph examples/multi-file-agent/` and verify file nodes, import edges, and cross-file skill references appear

### Implementation for User Story 3

- [x] T022 [US3] Extend `Extract` in `internal/graph/extract.go` to accept multiple `*ast.File` entries, create synthetic `file` nodes for each source file, add `defines` edges from file nodes to their contained entities, and add `imports` edges between files based on `ast.Import` statements
- [x] T023 [US3] Extend `parseFiles` in `cmd/agentspec/graph.go` to handle import resolution: when a file contains import statements, parse the imported files and include them in the AST file list (with deduplication and cycle detection via visited path tracking). For directory traversal, track visited real paths (via `filepath.EvalSymlinks`) to detect and skip symlink loops.
- [x] T024 [US3] Update `internal/graph/web/index.html` to visually annotate nodes with their source file (small file label below node name) and optionally group by file using force simulation clustering
- [x] T025 [US3] Add multi-file test case in `internal/graph/extract_test.go` verifying file nodes, defines edges, and imports edges are correctly generated

**Checkpoint**: `agentspec graph dir/` shows file nodes, import edges, cross-file entity references, and per-file grouping.

---

## Phase 6: User Story 4 — Complex Graph Navigation (Priority: P2)

**Goal**: Zoom, pan, search, and filter within the web UI for large graphs

**Independent Test**: Load a large `.ias` file, verify zoom/pan/search/filter/double-click interactions work smoothly

### Implementation for User Story 4

- [x] T026 [US4] Add search box to `internal/graph/web/index.html`: text input in the header that filters nodes by partial name match — matching nodes keep full opacity, non-matching nodes dim to 0.2 opacity, edges to/from dimmed nodes also dim
- [x] T027 [US4] Add legend interactivity to `internal/graph/web/index.html`: clicking a legend item toggles visibility of that entity type (hide/show nodes and their edges)
- [x] T028 [US4] Add double-click-to-center behavior to `internal/graph/web/index.html`: double-clicking a node animates the viewport to center and zoom to that node
- [x] T029 [US4] Add node/edge/file count display in the header bar of `internal/graph/web/index.html` using data from the `stats` field of the API response

**Checkpoint**: Web UI supports search filtering, legend toggling, double-click centering, and shows graph statistics.

---

## Phase 7: User Story 5 — Pipeline Flow Visualization (Priority: P2)

**Goal**: Pipeline steps rendered in execution order with parallelism shown

**Independent Test**: Load `examples/research-swarm.ias` and verify pipeline steps appear in DAG order with dependency edges

### Implementation for User Story 5

- [x] T030 [US5] Enhance `internal/graph/web/index.html` to detect pipeline subgraphs (nodes of type `pipeline` and `step`) and apply hierarchical positioning: steps with no `depends on` incoming edges at the top, subsequent steps below their dependencies, parallel steps at the same Y level
- [x] T031 [US5] Style pipeline subgraphs in `internal/graph/web/index.html` with a distinct visual boundary (dashed rectangle background behind pipeline nodes) and render pipeline `contains` edges as thin connector lines
- [x] T032 [US5] Update `RenderDOT` in `internal/graph/dot.go` to place pipeline steps in a subgraph with `rank=same` for parallel steps (steps sharing the same set of dependencies)
- [x] T033 [US5] Update `RenderMermaid` in `internal/graph/mermaid.go` to group pipeline steps in a `subgraph` block with steps ordered by dependency level

**Checkpoint**: Pipeline steps render in execution order. Parallel steps appear at the same level. Pipeline area is visually distinct.

---

## Phase 8: User Story 6 — Theme Customization (Priority: P3)

**Goal**: Light and dark themes for the web UI

**Independent Test**: Run with `--theme light` and `--theme dark`, verify both have correct contrast and color-coding

### Implementation for User Story 6

- [x] T034 [US6] Add light theme CSS variables to `internal/graph/web/index.html`: white background, dark text, adjusted entity type colors (per data-model.md Light Theme column), and dark edge/arrow colors
- [x] T035 [US6] Pass theme selection from `graph.Serve()` to the HTML page via a query parameter or template variable, and toggle between dark/light CSS variable sets on page load in `internal/graph/web/index.html`

**Checkpoint**: `--theme dark` and `--theme light` both render with correct contrast. Entity type colors are distinguishable in both themes.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Integration validation, testing, and pre-commit checks

- [x] T036 Write integration test in `integration_tests/graph_test.go` that parses `examples/multi-agent-router.ias`, calls `Extract`, and verifies: correct node count/types, correct edge count/labels, deterministic DOT output, deterministic Mermaid output. Include a scale test that programmatically builds a Graph with 200 nodes and verifies Extract + RenderDOT + RenderMermaid complete within 1 second.
- [x] T037 Write integration test in `integration_tests/graph_test.go` for error resilience: create a temp directory with one valid and one invalid `.ias` file, verify graph is generated from the valid file and errors are returned for the invalid file
- [x] T038 Verify `--no-files` flag correctly removes file nodes and `defines` edges from graph output; `--no-orphans` flag removes nodes with zero edges
- [x] T039 Run `golangci-lint run ./...`, `gofmt -l .`, `go build ./...`, `go test ./... -count=1` pre-commit checks and fix any issues
- [x] T040 Run quickstart.md scenarios manually and verify expected output

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational (Phase 2) — MVP web UI
- **User Story 2 (Phase 4)**: Depends on Foundational (Phase 2) — can proceed in parallel with US1
- **User Story 3 (Phase 5)**: Depends on US1 or US2 (needs graph.go CLI structure from T013)
- **User Story 4 (Phase 6)**: Depends on US1 (enhances existing web UI from T012)
- **User Story 5 (Phase 7)**: Depends on US1 and US2 (enhances web UI and static renderers)
- **User Story 6 (Phase 8)**: Depends on US1 (enhances existing web UI from T012)
- **Polish (Phase 9)**: Depends on all user stories being complete

### Within Each User Story

- Data model (model.go) before extraction (extract.go)
- Extraction before rendering (dot.go, mermaid.go, server.go)
- Rendering before CLI integration (graph.go)
- CLI integration before registration (main.go)

### Parallel Opportunities

```text
# Phase 2: These can run sequentially (same file extract.go):
T004 → T005 → T006 → T007 → T008 → T009

# Phase 3 (US1): Server and browser in parallel (different files):
T010 (server.go) || T011 (browser.go)
# Then HTML, then CLI:
T012 (index.html) → T013 (graph.go) → T014 (graph.go) → T015 (main.go)

# Phase 4 (US2): DOT and Mermaid in parallel (different files):
T017 (dot.go) || T018 (mermaid.go) || T019 (dot_test.go) || T020 (mermaid_test.go)
# Then CLI integration:
T021 (graph.go)

# Phase 6 (US4): All UI enhancements in same file (sequential):
T026 → T027 → T028 → T029

# Phase 7 (US5): Web and static renderers can partially parallel:
T030 + T031 (index.html) then T032 (dot.go) || T033 (mermaid.go)
```

---

## Implementation Strategy

### MVP First (User Story 1 + 2)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T009)
3. Complete Phase 3: US1 — Web UI (T010-T016)
4. Complete Phase 4: US2 — DOT + Mermaid (T017-T021)
5. **STOP and VALIDATE**: Test with `examples/multi-agent-router.ias` in all 3 formats
6. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Graph extraction works
2. User Story 1 → Interactive web UI → MVP!
3. User Story 2 → DOT + Mermaid export → Documentation-ready!
4. User Story 3 → Multi-file support
5. User Story 4 → Navigation for large graphs
6. User Story 5 → Pipeline flow visualization
7. User Story 6 → Light theme
8. Polish → Integration tests, pre-commit validation

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Unit tests included for extraction (foundation) and renderers (US2) — these are the most logic-heavy components
- US1 and US2 are both P1 and form the MVP together
- US3-US5 are P2 and can proceed in any order after MVP
- US6 is P3 (cosmetic)
- The HTML file (T012) is the largest single task — it contains the full web UI
- D3.js v7 minified source must be embedded inline in the HTML file
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
