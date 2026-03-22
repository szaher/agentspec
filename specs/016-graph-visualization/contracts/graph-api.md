# Contract: Graph API

**Feature**: 016-graph-visualization
**Type**: HTTP JSON API (localhost only)
**Consumer**: Web UI (embedded `index.html`)

## Endpoint: GET /api/graph

Returns the complete graph data structure for rendering.

### Request

```
GET /api/graph HTTP/1.1
Host: 127.0.0.1:8686
Accept: application/json
```

No parameters. No authentication (localhost-only server).

### Response: 200 OK

```json
{
  "nodes": [
    {
      "id": "agent:router",
      "type": "agent",
      "name": "router",
      "file": "main.ias",
      "line": 15,
      "attributes": {
        "model": "claude-sonnet-4-20250514",
        "strategy": "router",
        "max_turns": "5",
        "timeout": "30s"
      }
    },
    {
      "id": "skill:search",
      "type": "skill",
      "name": "search",
      "file": "skills/search.ias",
      "line": 3,
      "attributes": {
        "description": "Search the knowledge base",
        "tool_type": "mcp",
        "server_tool": "search-server/query"
      }
    },
    {
      "id": "prompt:system",
      "type": "prompt",
      "name": "system",
      "file": "main.ias",
      "line": 5,
      "attributes": {
        "content_preview": "You are a helpful assistant that routes..."
      }
    }
  ],
  "edges": [
    {
      "source": "agent:router",
      "target": "prompt:system",
      "label": "uses prompt",
      "style": "solid"
    },
    {
      "source": "agent:router",
      "target": "skill:search",
      "label": "uses skill",
      "style": "solid"
    },
    {
      "source": "agent:router",
      "target": "agent:specialist",
      "label": "delegates to",
      "style": "dashed"
    }
  ],
  "package": {
    "name": "my-agents",
    "version": "1.0.0",
    "description": "My agent configuration"
  },
  "files": [
    "main.ias",
    "skills/search.ias"
  ],
  "stats": {
    "node_count": 8,
    "edge_count": 12,
    "file_count": 2,
    "type_counts": {
      "agent": 2,
      "prompt": 1,
      "skill": 3,
      "mcp_server": 1,
      "deploy": 1
    }
  },
  "errors": [
    "skills/broken.ias:15:3: unexpected token \"}\""
  ]
}
```

### Response Fields

| Field           | Type     | Required | Description |
|-----------------|----------|----------|-------------|
| nodes           | array    | Yes      | All graph nodes |
| nodes[].id      | string   | Yes      | Unique node identifier (`{type}:{name}`) |
| nodes[].type    | string   | Yes      | Entity type (one of 18 types) |
| nodes[].name    | string   | Yes      | Entity display name |
| nodes[].file    | string   | No       | Source file path (relative) |
| nodes[].line    | int      | No       | Source line number |
| nodes[].attributes | object | No     | Type-specific key-value pairs |
| edges           | array    | Yes      | All graph edges |
| edges[].source  | string   | Yes      | Source node ID |
| edges[].target  | string   | Yes      | Target node ID |
| edges[].label   | string   | Yes      | Relationship type label |
| edges[].style   | string   | No       | "solid" (default) or "dashed" |
| package         | object   | No       | Package metadata |
| package.name    | string   | No       | Package name |
| package.version | string   | No       | Package version |
| package.description | string | No    | Package description |
| files           | array    | Yes      | Source file paths |
| stats           | object   | Yes      | Aggregate statistics |
| stats.node_count | int     | Yes      | Total nodes |
| stats.edge_count | int     | Yes      | Total edges |
| stats.file_count | int     | Yes      | Total files |
| stats.type_counts | object | Yes     | Node count per type |
| errors          | array    | No       | Parse error messages (file:line:col: message) |

### Error Response: 500 Internal Server Error

```json
{
  "error": "failed to parse files: no .ias files found"
}
```

---

## Endpoint: GET /

Serves the embedded `index.html` single-page application.

### Response: 200 OK

Content-Type: `text/html; charset=utf-8`

The HTML page contains all CSS and JS inline. On load, it fetches `/api/graph` and renders the visualization.

---

## CLI Contract

### Command Signature

```
agentspec graph [file.ias|directory...] [flags]
```

### Arguments

| Argument | Required | Default | Description |
|----------|----------|---------|-------------|
| paths    | No       | `.`     | One or more `.ias` files or directories |

### Flags

| Flag         | Type   | Default | Description |
|--------------|--------|---------|-------------|
| --format     | string | web     | Output format: web, dot, mermaid |
| --port       | int    | 8686    | Port for web UI server |
| --open       | bool   | true    | Auto-open browser (web mode only) |
| --no-open    | bool   | false   | Disable auto-open browser |
| --theme      | string | dark    | Web UI theme: dark, light |
| --output     | string | ""      | Write output to file (dot/mermaid only) |
| --no-files   | bool   | false   | Hide file nodes |
| --no-orphans | bool   | false   | Hide entities with no connections |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0    | Success |
| 1    | Error (parse failure, no files found, port in use) |

### Output Behavior

| Format  | Output Destination | Blocking | Browser |
|---------|-------------------|----------|---------|
| web     | Localhost server  | Yes (until Ctrl+C) | Opens if --open |
| dot     | stdout or --output file | No | No |
| mermaid | stdout or --output file | No | No |
