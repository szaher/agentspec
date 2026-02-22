# Adapter Interface Contract

## Overview

Adapters translate the platform-neutral IR into platform-specific
artifacts and apply them to their target. Adapters MUST be thin:
mapping and deployment only, no business logic.

## Go Interface

```go
// Adapter translates IR resources into platform-specific
// artifacts and applies them.
type Adapter interface {
    // Name returns the adapter identifier
    // (e.g., "local-mcp", "docker-compose").
    Name() string

    // Validate checks whether the IR resources are compatible
    // with this adapter. Returns errors for unsupported
    // resource types or configurations.
    Validate(ctx context.Context, resources []ir.Resource) error

    // Plan computes the changes needed to reach desired state.
    // Returns a list of actions (create, update, delete) with
    // deterministic ordering.
    Plan(
        ctx context.Context,
        desired []ir.Resource,
        current []state.Entry,
    ) ([]Action, error)

    // Apply executes the planned actions. Returns results per
    // resource (success or failure with error).
    Apply(
        ctx context.Context,
        actions []Action,
    ) ([]Result, error)

    // Export generates platform-specific artifacts without
    // applying. Output MUST be byte-identical for identical
    // inputs.
    Export(
        ctx context.Context,
        resources []ir.Resource,
        outDir string,
    ) error
}
```

## Action Types

| Action   | Description                              |
|----------|------------------------------------------|
| `create` | Resource does not exist; create it       |
| `update` | Resource exists but hash differs; update |
| `delete` | Resource in state but not in desired     |
| `noop`   | Resource unchanged                       |

## Result

Each `Result` contains:
- `FQN`: Fully-qualified resource name
- `Action`: What was attempted
- `Status`: `success` or `failed`
- `Error`: Error message if failed
- `Artifact`: Path or reference to produced artifact

## Adapter Registration

Adapters register via an init function:

```go
func init() {
    adapters.Register("local-mcp", NewLocalMCPAdapter)
    adapters.Register("docker-compose", NewComposeAdapter)
}
```

## Constraints

- Adapters MUST accept IR as input, never raw DSL.
- Adapters MUST NOT contain business logic (validation,
  reference resolution, override merging).
- Adapter outputs MUST be deterministic.
- Adapters MUST handle partial failure gracefully (return
  per-resource results).

## MVP Adapters

### Local MCP (`local-mcp`)

Generates MCP server/client configuration files for local
runtimes. Writes JSON configuration to disk.

**Artifacts produced**:
- `mcp-servers.json` — MCP server configurations
- `mcp-clients.json` — MCP client configurations
- `agents.json` — Agent definitions with resolved references

### Docker Compose (`docker-compose`)

Generates Docker Compose services from agent and server
definitions.

**Artifacts produced**:
- `docker-compose.yml` — Service definitions
- `config/` — Configuration files mounted as volumes
- `.env` — Environment variable references (secret refs only)
