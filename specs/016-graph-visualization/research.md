# Research: Interactive Dependency Graph Visualization

**Feature**: 016-graph-visualization
**Date**: 2026-03-22

## Research Topics

### 1. Client-Side Graph Visualization Library

**Decision**: D3.js force simulation (vendored/embedded)

**Rationale**: D3.js is the industry standard for data-driven visualizations in the browser. Its force simulation module provides exactly the force-directed graph layout needed. The library is mature (v7), well-documented, and has been used in production for over a decade. It works with vanilla JS (no framework required), which matches the project's existing frontend approach in `internal/frontend/web/`.

**Alternatives considered**:
- **Cytoscape.js**: Full-featured graph library, but heavier (~500KB vs ~100KB for D3 core). Better for complex graph analysis use cases but overkill for visualization-only.
- **vis.js Network**: Easy graph visualization, but the project is less actively maintained and the API is less flexible for custom rendering.
- **dagre + d3-graphviz**: Dagre provides hierarchical layout which would be ideal for pipelines, but requires a separate layout pass. Could be used as a complement to D3 force layout.
- **Sigma.js**: Optimized for large graphs (10k+ nodes), but API is more complex and the project has had maintenance gaps.
- **Custom SVG**: No library, just manual SVG. Too much work for the interaction features (zoom, pan, drag, click).

**Implementation note**: Embed the minified D3 v7 core + d3-force + d3-zoom modules (~100KB total) directly in the HTML file as an inline `<script>` block. No CDN, no external fetch.

### 2. Graph Layout Strategy

**Decision**: Force-directed (D3 force simulation) as default, with optional hierarchical layout for pipelines

**Rationale**: Force-directed layout works well for general dependency graphs — it naturally clusters related nodes and spreads unrelated ones apart. For pipeline subgraphs specifically, a simple top-to-bottom ordering based on `depends_on` edges provides the hierarchical flow visualization needed.

**Alternatives considered**:
- **Dagre (hierarchical-only)**: Great for strict DAGs but poor for general entity relationship graphs where there's no single direction of flow.
- **Elk.js (Eclipse Layout Kernel)**: Very powerful but very heavy (~2MB). Overkill for this use case.
- **Manual positioning**: Not feasible for dynamic graphs of varying sizes.

### 3. DOT Output Format

**Decision**: Standard Graphviz DOT with `digraph`, `rankdir=LR`, distinct shapes per entity type

**Rationale**: Graphviz DOT is the most widely supported graph description format. Every major OS has Graphviz available, and it's commonly used in documentation pipelines. Using `rankdir=LR` (left-to-right) produces readable horizontal flow diagrams. Node shapes (box, ellipse, diamond, hexagon, etc.) are sufficient to distinguish entity types without color (since DOT files are often rendered without custom color support).

**Shape mapping**:
- Agent: `box` (rounded via `style=rounded`)
- Prompt: `note`
- Skill: `component`
- MCP Server/Client: `hexagon`
- Pipeline: `tab`
- Pipeline Step: `circle`
- Secret: `diamond`
- Policy: `house`
- Guardrail: `shield` (or `octagon` if shield unavailable)
- User: `invhouse`
- Deploy Target: `folder`
- State Config: `cylinder`
- Environment: `parallelogram`
- Type Def: `rect`
- Plugin: `pentagon`
- File: `folder`

### 4. Mermaid Output Format

**Decision**: `graph LR` with Mermaid node shapes and `classDef` for styling

**Rationale**: Mermaid is the de facto standard for diagrams in markdown documentation, especially on GitHub (native rendering support). The `graph LR` direction matches the DOT `rankdir=LR` for consistency. Mermaid's node shape syntax (`()`, `[]`, `{}`, `[()]`, `(())`) provides 5+ distinct shapes for entity types.

**Limitations**: Mermaid has fewer shape options than Graphviz, so some entity types will share shapes but be distinguished by `classDef` color classes. This is acceptable since Mermaid is primarily for embedding in markdown where colors render correctly.

### 5. Web Server Pattern

**Decision**: Standard `net/http` with `signal.NotifyContext` for graceful shutdown

**Rationale**: The project already uses `net/http` for the `run` command's web UI. The pattern is proven: embed assets via `go:embed`, serve static files with `http.FileServer`, add API endpoints, and handle signals for clean shutdown. No need for a web framework.

**Endpoints**:
- `GET /` → serves `index.html` (embedded)
- `GET /api/graph` → returns JSON graph data

### 6. Browser Launch

**Decision**: OS-detection with `open` (macOS), `xdg-open` (Linux), `start` (Windows)

**Rationale**: Standard approach used by many CLI tools. The `--open` flag defaults to true and fails silently if no browser is available (SSH/headless). The URL is always printed to stdout regardless.

**Implementation**: Simple `runtime.GOOS` switch with `exec.Command`. No external package needed.

### 7. Entity Extraction from AST

**Decision**: Walk `ast.File.Statements` and type-switch on each `ast.Statement` to extract nodes and edges

**Rationale**: The AST preserves all entity types and relationships in their typed form. A single pass through `File.Statements` with type assertions gives access to every entity and its references. This is simpler and more complete than working with the generic IR `Resource` type.

**Entity type coverage** (from `internal/ast/ast.go`):
- `*ast.Agent` → agent node + edges for prompt, skills, guardrails, client, delegates, fallback
- `*ast.Prompt` → prompt node
- `*ast.Skill` → skill node + edge for MCP server tool
- `*ast.MCPServer` → mcp_server node
- `*ast.MCPClient` → mcp_client node + edges for servers
- `*ast.Pipeline` → pipeline node + step nodes + edges for contains, depends_on, invokes
- `*ast.Secret` → secret node
- `*ast.Policy` → policy node + edges for governed resources
- `*ast.Guardrail` → guardrail node
- `*ast.User` → user node + edges for accessible agents
- `*ast.Binding` → binding node
- `*ast.DeployTarget` → deploy node
- `*ast.StateConfig` → state node
- `*ast.Environment` → env node + edges for overridden resources
- `*ast.TypeDef` → type node
- `*ast.PluginRef` → plugin node
- File (synthetic) → file node + edges for contains, imports
