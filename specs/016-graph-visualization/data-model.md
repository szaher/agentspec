# Data Model: Interactive Dependency Graph Visualization

**Feature**: 016-graph-visualization
**Date**: 2026-03-22

## Entities

### GraphNode

Represents a single entity extracted from a parsed `.ias` file.

| Field      | Type              | Required | Description |
|------------|-------------------|----------|-------------|
| ID         | string            | Yes      | Unique composite key: `{type}:{name}` (e.g., `agent:router`, `skill:search`) |
| Type       | string            | Yes      | Entity type enum: agent, prompt, skill, mcp_server, mcp_client, pipeline, step, secret, policy, guardrail, user, binding, deploy, state, env, type, plugin, file |
| Name       | string            | Yes      | The entity's declared name from the `.ias` source |
| File       | string            | No       | Relative path to the source `.ias` file |
| Line       | int               | No       | Line number in the source file (from AST Pos) |
| Attributes | map[string]string | No       | Type-specific configuration key-value pairs for sidebar display |

**Identity**: Unique by `ID` (type + name composite). For multi-file projects, nodes from different files with the same type and name are disambiguated by appending the filename: `agent:router@main.ias`.

**Validation**:
- ID must not be empty
- Type must be one of the 18 allowed values
- Name must not be empty

### GraphEdge

Represents a directed relationship between two entities.

| Field  | Type   | Required | Description |
|--------|--------|----------|-------------|
| Source | string | Yes      | Source node ID |
| Target | string | Yes      | Target node ID |
| Label  | string | Yes      | Relationship description (e.g., "uses prompt", "delegates to") |
| Style  | string | No       | Visual style: "solid" (default) or "dashed" (for conditional/optional) |

**Identity**: Unique by (Source, Target, Label) triple.

**Validation**:
- Source and Target must reference existing node IDs (or be marked as "missing")
- Label must be one of the 16 defined relationship types

### Graph

Top-level container for the complete visualization data.

| Field   | Type          | Required | Description |
|---------|---------------|----------|-------------|
| Nodes   | []GraphNode   | Yes      | All extracted entities |
| Edges   | []GraphEdge   | Yes      | All extracted relationships |
| Package | PackageInfo   | No       | Package metadata from the `.ias` header |
| Files   | []string      | Yes      | List of all source file paths |
| Stats   | GraphStats    | Yes      | Aggregate counts |

### PackageInfo

| Field       | Type   | Required | Description |
|-------------|--------|----------|-------------|
| Name        | string | No       | Package name |
| Version     | string | No       | Package version |
| Description | string | No       | Package description |

### GraphStats

| Field      | Type           | Required | Description |
|------------|----------------|----------|-------------|
| NodeCount  | int            | Yes      | Total number of nodes |
| EdgeCount  | int            | Yes      | Total number of edges |
| FileCount  | int            | Yes      | Number of source files |
| TypeCounts | map[string]int | Yes      | Node count per entity type |

## Relationships

### Entity Relationships (Edges)

| Relationship                    | Source Type    | Target Type    | Label            | Style  |
|---------------------------------|---------------|----------------|------------------|--------|
| Agent uses Prompt               | agent         | prompt         | uses prompt      | solid  |
| Agent uses Skill                | agent         | skill          | uses skill       | solid  |
| Agent uses Guardrail            | agent         | guardrail      | uses guardrail   | solid  |
| Agent uses MCP Client           | agent         | mcp_client     | uses client      | solid  |
| Agent delegates to Agent        | agent         | agent          | delegates to     | dashed |
| Agent fallback to Agent         | agent         | agent          | fallback         | dashed |
| Pipeline contains Step          | pipeline      | step           | contains         | solid  |
| Pipeline Step invokes Agent     | step          | agent          | invokes          | solid  |
| Pipeline Step depends on Step   | step          | step           | depends on       | solid  |
| MCP Client connects to Server   | mcp_client    | mcp_server     | connects to      | solid  |
| Skill uses MCP Server tool      | skill         | mcp_server     | uses tool        | solid  |
| User can access Agent           | user          | agent          | can access       | solid  |
| Policy governs Resource         | policy        | (any)          | governs          | solid  |
| Environment overrides Resource  | env           | (any)          | overrides        | dashed |
| File imports File               | file          | file           | imports          | solid  |
| File contains Entity            | file          | (any)          | defines          | solid  |

### Unresolved References

When a relationship target cannot be found in the parsed files, a placeholder node is created with:
- Type: the expected type (e.g., `skill`)
- Name: the referenced name
- A `missing: true` attribute
- Visual indicator: dashed border in web UI, `style=dashed` in DOT, comment annotation in Mermaid

## Entity Type Visual Mapping

| Entity Type   | Color (Dark Theme) | Color (Light Theme) | DOT Shape     | Mermaid Shape |
|---------------|-------------------|---------------------|---------------|---------------|
| agent         | #4A9EFF (blue)    | #2563EB             | box (rounded) | ([name])      |
| prompt        | #4ADE80 (green)   | #16A34A             | note          | [name]        |
| skill         | #A78BFA (purple)  | #7C3AED             | component     | [name]        |
| mcp_server    | #FB923C (orange)  | #EA580C             | hexagon       | {{name}}      |
| mcp_client    | #FBBF24 (amber)   | #D97706             | hexagon       | {{name}}      |
| pipeline      | #22D3EE (cyan)    | #0891B2             | tab           | ([name])      |
| step          | #2DD4BF (teal)    | #0D9488             | circle        | ((name))      |
| secret        | #F87171 (red)     | #DC2626             | diamond       | {name}        |
| policy        | #F472B6 (pink)    | #DB2777             | house         | [name]        |
| guardrail     | #FB7185 (rose)    | #E11D48             | octagon       | [name]        |
| user          | #818CF8 (indigo)  | #4F46E5             | invhouse      | [name]        |
| deploy        | #94A3B8 (slate)   | #475569             | folder        | [name]        |
| binding       | #94A3B8 (slate)   | #475569             | folder        | [name]        |
| state         | #34D399 (emerald) | #059669             | cylinder      | [(name)]      |
| env           | #A3E635 (lime)    | #65A30D             | parallelogram | [/name/]      |
| type          | #9CA3AF (gray)    | #6B7280             | rect          | [name]        |
| plugin        | #C084FC (violet)  | #9333EA             | pentagon      | [name]        |
| file          | #E5E7EB (light)   | #374151 (dark)      | folder        | [name]        |
