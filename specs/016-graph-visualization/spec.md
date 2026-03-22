# Feature Specification: Interactive Dependency Graph Visualization

**Feature Branch**: `016-graph-visualization`
**Created**: 2026-03-22
**Status**: Draft
**Input**: User description: "Add an `agentspec graph` command that visualizes .ias files as interactive dependency graphs with web UI, DOT, and Mermaid output formats."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Single-File Interactive Visualization (Priority: P1)

As a developer working on an AgentSpec project, I want to run a single command and see an interactive graph of my `.ias` file so that I can understand the architecture at a glance without reading every line of configuration.

**Why this priority**: This is the core value proposition — understanding complex multi-agent architectures visually. Without this, the feature has no purpose.

**Independent Test**: Can be fully tested by running the command against any `.ias` file and verifying the browser opens with a correct, interactive graph showing all entities and relationships.

**Acceptance Scenarios**:

1. **Given** a valid `.ias` file with agents, skills, prompts, and a deploy target, **When** the user runs `agentspec graph myfile.ias`, **Then** a browser opens with an interactive graph showing all entities as color-coded nodes connected by labeled edges
2. **Given** the web UI is displayed, **When** the user clicks on any node, **Then** a sidebar panel shows the entity's configuration details (name, type, source file/line, attributes, connected entities)
3. **Given** the web server is running, **When** the user presses Ctrl+C in the terminal, **Then** the server shuts down gracefully with no orphaned processes
4. **Given** a file with up to 50 entities, **When** the graph loads, **Then** the visualization renders within 2 seconds
5. **Given** the web UI, **When** viewing the graph, **Then** a legend shows all entity types with their assigned colors

---

### User Story 2 - Static Format Export (Priority: P1)

As a developer writing documentation or reviewing in CI, I want to export the dependency graph as Mermaid or Graphviz DOT so that I can embed it in README files, pull requests, or generate images.

**Why this priority**: Static export enables documentation workflows and CI integration — essential for team adoption alongside the interactive mode.

**Independent Test**: Can be fully tested by running the command with `--format mermaid` or `--format dot` and verifying the output is valid syntax that renders correctly in their respective tools.

**Acceptance Scenarios**:

1. **Given** a valid `.ias` file, **When** the user runs `agentspec graph myfile.ias --format mermaid`, **Then** valid Mermaid markdown is printed to stdout with entity types distinguished by node shapes and edges labeled with relationship types
2. **Given** a valid `.ias` file, **When** the user runs `agentspec graph myfile.ias --format dot`, **Then** valid Graphviz DOT is printed to stdout with distinct node shapes per entity type
3. **Given** mermaid output, **When** pasted into a GitHub markdown code block, **Then** GitHub renders the diagram correctly
4. **Given** DOT output, **When** processed with `dot -Tpng`, **Then** a valid PNG image is generated without errors
5. **Given** the same input file, **When** the command is run twice, **Then** the output is identical (deterministic/sorted for reproducible diffs)
6. **Given** a `--output filename` flag, **When** the command runs, **Then** output is written to the specified file instead of stdout

---

### User Story 3 - Multi-File Project Visualization (Priority: P2)

As a developer working on a multi-file AgentSpec project with imports, I want to visualize the entire directory and see how files connect so that I can understand cross-file dependencies and the overall project structure.

**Why this priority**: Production projects typically span multiple files. Cross-file visualization is essential for understanding real-world architectures but requires the single-file foundation first.

**Independent Test**: Can be fully tested by creating a directory with multiple `.ias` files containing import statements and verifying the graph shows inter-file relationships.

**Acceptance Scenarios**:

1. **Given** a directory with multiple `.ias` files, **When** the user runs `agentspec graph ./`, **Then** all `.ias` files are discovered recursively and included in the graph
2. **Given** files with import statements, **When** the graph is rendered, **Then** import relationships between files are shown as labeled edges
3. **Given** an agent in one file using a skill defined in another file, **When** the graph is rendered, **Then** the cross-file reference is correctly resolved and shown as an edge
4. **Given** nodes from multiple files, **When** the graph is rendered, **Then** nodes are visually annotated or grouped by their source file

---

### User Story 4 - Complex Graph Navigation (Priority: P2)

As a developer working on a large AgentSpec project with 20+ entities, I want to zoom, pan, filter, and search within the graph so that I can focus on specific parts of the architecture.

**Why this priority**: Large projects produce dense graphs that are unusable without navigation controls. This extends the web UI's value to production-scale configurations.

**Independent Test**: Can be fully tested by loading a large `.ias` file and verifying zoom, pan, search, and type filtering all work correctly.

**Acceptance Scenarios**:

1. **Given** the web UI, **When** the user scrolls the mouse wheel, **Then** the graph zooms in/out centered on the cursor
2. **Given** the web UI, **When** the user click-drags on the canvas, **Then** the graph pans in the drag direction
3. **Given** a search box in the web UI, **When** the user types a partial entity name, **Then** matching nodes are highlighted and non-matching nodes are dimmed
4. **Given** a legend with entity type colors, **When** the user clicks a legend item, **Then** that entity type's visibility is toggled on/off
5. **Given** the web UI header, **When** viewing the graph, **Then** node count, relationship count, and file count are displayed
6. **Given** a node in the graph, **When** the user double-clicks it, **Then** the view centers and zooms to that node
7. **Given** a graph with up to 200 nodes, **When** interacting, **Then** the UI maintains smooth interactions (60fps)

---

### User Story 5 - Pipeline Flow Visualization (Priority: P2)

As a developer building multi-step agent pipelines, I want to see the pipeline execution flow with step ordering and parallelism so that I can verify the DAG is correct and identify bottlenecks.

**Why this priority**: Pipeline workflows have specific execution ordering that is critical to understand. Dedicated pipeline visualization adds significant value for workflow-heavy projects.

**Independent Test**: Can be fully tested by loading a `.ias` file with a pipeline and verifying steps are rendered in execution order with dependency edges.

**Acceptance Scenarios**:

1. **Given** a `.ias` file with a pipeline, **When** the graph is rendered, **Then** pipeline steps appear in execution order (top-to-bottom or left-to-right)
2. **Given** steps with `depends_on` relationships, **When** the graph is rendered, **Then** dependency edges connect the steps
3. **Given** steps that can run in parallel, **When** the graph is rendered, **Then** they appear at the same hierarchical level
4. **Given** a pipeline step, **When** it is displayed, **Then** it shows which agent it invokes
5. **Given** a pipeline subgraph, **When** the graph is rendered, **Then** it is visually distinct from the rest of the entity graph

---

### User Story 6 - Theme Customization (Priority: P3)

As a developer who prefers light or dark mode, I want to choose a visual theme for the web UI so that the graph is comfortable to view in my environment.

**Why this priority**: Aesthetic preference — nice to have but not blocking core functionality.

**Independent Test**: Can be fully tested by running the command with `--theme dark` and `--theme light` and verifying both render with appropriate contrast.

**Acceptance Scenarios**:

1. **Given** the `--theme dark` flag (default), **When** the web UI renders, **Then** the graph has a dark background with light text and edges
2. **Given** the `--theme light` flag, **When** the web UI renders, **Then** the graph has a white background with dark text and edges
3. **Given** either theme, **When** viewing entity nodes, **Then** entity type color-coding is maintained with appropriate contrast adjustments
4. **Given** a DOT or Mermaid format export, **When** a theme flag is provided, **Then** the theme selection has no effect on the output

---

### Edge Cases

- **Empty file**: A valid `.ias` file with only a `package` declaration and no entities renders a single file node with no edges
- **Self-referencing agent**: An agent that delegates to itself shows a self-loop edge
- **Duplicate entity names**: Two entities of the same type with the same name in different files render both with file annotations to distinguish them
- **Very long prompt content**: Content is truncated to 200 characters in the sidebar with a "..." indicator
- **Non-`.ias` files in directory**: Silently ignored; only `.ias` files are processed
- **Symlink loops**: Directory traversal follows symlinks but tracks visited paths to avoid infinite loops
- **Port already in use**: Clear error message naming the port and suggesting `--port <alternative>`
- **No browser available (headless/SSH)**: `--open` fails silently; the URL is always printed to stdout
- **Unicode entity names**: Render correctly in all output formats
- **100+ entities**: Web UI remains interactive; DOT/Mermaid output remains valid
- **Parse errors in some files**: Report errors to stderr, render successfully parsed files, show warning banner in web UI
- **Circular imports**: Detect and report, render what was successfully parsed
- **Unresolved references**: Render the node with a "missing" visual indicator (dashed border) and include it in the graph

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept one or more `.ias` file paths or a directory path as arguments, defaulting to the current directory when no argument is provided
- **FR-002**: System MUST parse all specified `.ias` files using the existing parser and extract all entity types: agents, prompts, skills, MCP servers, MCP clients, pipelines (with steps), secrets, policies, guardrails, users, bindings, deploy targets, state config, environments, type definitions, and plugins
- **FR-003**: System MUST extract all directed relationships between entities: uses prompt, uses skill, uses guardrail, uses client, delegates to, fallback, pipeline step invokes agent, step depends on step, pipeline contains step, MCP client connects to server, skill uses MCP tool, user can access agent, policy governs resource, environment overrides resource, file imports file, file contains entity
- **FR-004**: System MUST provide an interactive web UI output mode (default) that serves a single-page visualization on localhost with force-directed or hierarchical layout, color-coded nodes by entity type, labeled edges showing relationship types, a clickable sidebar for entity details, and a legend
- **FR-005**: System MUST provide a Graphviz DOT output mode that prints valid `digraph` syntax to stdout with distinct shapes per entity type, labeled edges, subgraph clustering by source file, and deterministic (sorted) output
- **FR-006**: System MUST provide a Mermaid markdown output mode that prints valid `graph LR` syntax to stdout with Mermaid node shapes per entity type, labeled edges, subgraph grouping by source file, and deterministic (sorted) output
- **FR-007**: System MUST embed all web UI assets in the binary with zero external runtime dependencies — no CDN, no external fonts, fully offline-capable
- **FR-008**: System MUST support CLI flags: `--format` (web|dot|mermaid, default web), `--port` (default 8686), `--open`/`--no-open` (auto-open browser, default true), `--theme` (dark|light, default dark), `--output` (write to file for dot/mermaid), `--no-files` (hide file nodes), `--no-orphans` (hide unconnected entities)
- **FR-009**: System MUST handle errors gracefully: report parse errors to stderr while rendering successfully parsed files, report port-in-use errors with a suggestion, exit with code 1 on failure, and show unresolved references with a "missing" visual indicator
- **FR-010**: System MUST be strictly read-only — it MUST NOT modify any `.ias` files or any files on disk
- **FR-011**: System MUST provide a data endpoint (for web mode) returning the graph as a structured data payload with nodes (id, type, name, file, line, attributes), edges (source, target, label, style), package metadata, file list, and aggregate statistics
- **FR-012**: System MUST serve the web UI only on localhost (127.0.0.1) by default for security — no binding to 0.0.0.0
- **FR-013**: System MUST gracefully shut down the web server on SIGINT/SIGTERM with no orphaned processes

### Key Entities

- **GraphNode**: Represents a single entity extracted from parsed `.ias` file(s) — has a unique composite ID (type + name), entity type, display name, source file path, line number, and type-specific attribute map
- **GraphEdge**: Represents a directed relationship between two entities — has source node ID, target node ID, human-readable relationship label, and visual style (solid for direct, dashed for conditional)
- **Graph**: Top-level container for the complete visualization data — contains a list of nodes, list of edges, package metadata, source file list, and aggregate statistics (counts by type)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can visualize any valid `.ias` file or directory with a single command and see all entities and relationships within 3 seconds
- **SC-002**: Users can identify the purpose and connections of any entity in under 5 seconds via the click-to-inspect sidebar
- **SC-003**: Generated Mermaid output renders correctly when pasted into a GitHub README without manual editing
- **SC-004**: Generated DOT output produces valid images via standard graph rendering tools without errors
- **SC-005**: Multi-file projects correctly show cross-file import relationships and resolved references
- **SC-006**: The graph remains interactive and usable for projects with up to 200 entities
- **SC-007**: 90% of first-time users can understand the graph without reading documentation (intuitive color/shape/label scheme)
- **SC-008**: All 18 entity types from the AgentSpec language are correctly extracted and visualized
- **SC-009**: All 16 relationship types are correctly extracted and displayed with appropriate labels
- **SC-010**: Static format outputs (DOT and Mermaid) are deterministic — same input always produces identical output, enabling meaningful version control diffs

## Dependencies

- Existing file parser for reading and interpreting `.ias` files
- Existing AST types for entity and relationship extraction
- Existing file discovery utilities for locating `.ias` files in directories

## Assumptions

- The command operates on source files only — it does not need runtime state, a running operator, or an active state backend
- The web UI uses no build step or package manager — a single embedded page with inline styles and scripts
- A client-side graph layout library is acceptable as a vendored/embedded dependency for the web visualization
- The `--open` flag is best-effort — if browser opening fails (e.g., headless server), the command continues and prints the URL
- Entity attributes in the sidebar are formatted for readability (e.g., prompt content shown as text, not escaped strings)
- The graph visualization library is vendored/embedded, not fetched from a CDN at runtime
- OS-appropriate browser launching is used (`open` on macOS, `xdg-open` on Linux, `start` on Windows)
