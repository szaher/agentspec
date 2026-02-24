# State Management

The state system tracks which resources have been applied and their current status. It enables the plan engine to compute minimal diffs between the desired configuration and the deployed reality.

## Package

| Package | Path | Purpose |
|---------|------|---------|
| `state` | `internal/state/` | State backend interface and types |

## State File

The default state backend persists to a local JSON file named `.agentspec.state.json` in the project directory. The toolchain automatically migrates from the legacy `.agentz.state.json` name if found.

### File Format

```json
[
  {
    "fqn": "my-app/agent/assistant",
    "hash": "a1b2c3d4e5f6...",
    "status": "applied",
    "last_applied": "2026-02-24T10:30:00Z",
    "adapter": "docker"
  },
  {
    "fqn": "my-app/prompt/system",
    "hash": "f6e5d4c3b2a1...",
    "status": "applied",
    "last_applied": "2026-02-24T10:30:00Z",
    "adapter": "docker"
  },
  {
    "fqn": "my-app/skill/search",
    "hash": "1a2b3c4d5e6f...",
    "status": "failed",
    "last_applied": "2026-02-24T10:30:00Z",
    "adapter": "docker",
    "error": "container image build failed: exit code 1"
  }
]
```

### Entry Schema

Each entry in the state file corresponds to one deployed resource:

```go
type Entry struct {
    FQN         string    `json:"fqn"`
    Hash        string    `json:"hash"`
    Status      Status    `json:"status"`
    LastApplied time.Time `json:"last_applied"`
    Adapter     string    `json:"adapter"`
    Error       string    `json:"error,omitempty"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `fqn` | string | Fully Qualified Name (`package/kind/name`) |
| `hash` | string | SHA-256 hash of the canonical resource attributes |
| `status` | string | `"applied"` or `"failed"` |
| `last_applied` | timestamp | When the resource was last applied |
| `adapter` | string | Which adapter processed this resource |
| `error` | string | Error message (only present when status is `"failed"`) |

## Status Values

```go
const (
    StatusApplied Status = "applied"
    StatusFailed  Status = "failed"
)
```

- **applied** -- The resource was successfully created or updated by the adapter.
- **failed** -- The adapter returned an error for this resource. The error message is stored in the `Error` field.

## Backend Interface

The `Backend` interface abstracts state persistence:

```go
type Backend interface {
    Load() ([]Entry, error)
    Save(entries []Entry) error
    Get(fqn string) (*Entry, error)
    List(status *Status) ([]Entry, error)
}
```

| Method | Purpose |
|--------|---------|
| `Load()` | Reads all state entries from the backend |
| `Save(entries)` | Writes all entries, replacing the previous state |
| `Get(fqn)` | Retrieves a single entry by its Fully Qualified Name |
| `List(status)` | Returns entries, optionally filtered by status |

The default implementation reads and writes `.agentspec.state.json` using JSON encoding.

## State Transitions

Resources move through states as they are applied:

```text
  (not in state)
       |
       | ActionCreate (success)
       v
   +----------+
   | applied  |
   +-----+----+
         |
         | ActionUpdate (success) -> stays "applied" (hash updated)
         | ActionUpdate (failure) -> moves to "failed"
         |
         | ActionDelete (success) -> removed from state
         |
         v
   +----------+
   |  failed  |
   +-----+----+
         |
         | Next apply retries automatically
         | ActionUpdate (success) -> moves back to "applied"
         |
         v
   +----------+
   | applied  |
   +----------+
```

### Transition Rules

1. **New resource** -- When a resource appears in the IR but not in state, the plan engine generates an `ActionCreate`. On success, an entry is added with `StatusApplied`.

2. **Changed resource** -- When the resource hash in the IR differs from the state, an `ActionUpdate` is generated. On success, the hash is updated and status remains `StatusApplied`. On failure, the status changes to `StatusFailed`.

3. **Removed resource** -- When a resource exists in state but not in the IR, an `ActionDelete` is generated. On success, the entry is removed from state.

4. **Failed resource** -- On the next apply, a failed resource automatically gets an `ActionUpdate` ("retrying previously failed resource"), regardless of whether its hash changed. This ensures failed deployments are retried.

## Reconciliation Flow

During `agentspec plan` or `agentspec apply`:

```text
  1. Load current state from .agentspec.state.json
                    |
                    v
  2. Lower .ias to IR (with hashes)
                    |
                    v
  3. ComputePlan(desired IR, current state)
                    |
                    v
  4. For each desired resource:
     - Not in state?     -> ActionCreate
     - Hash changed?     -> ActionUpdate
     - Previously failed? -> ActionUpdate (retry)
     - Same hash?         -> ActionNoop

  5. For each state entry not in desired:
     -> ActionDelete
                    |
                    v
  6. Apply actions via adapter
                    |
                    v
  7. Update state entries based on results
                    |
                    v
  8. Save state to .agentspec.state.json
```

## State File Location

The state file is always created in the current working directory. This means each project has its own independent state.

### Migration from Legacy Path

When the toolchain starts, it checks for the legacy `.agentz.state.json` file. If found and no `.agentspec.state.json` exists, the file is renamed automatically. A warning is printed to stderr:

```text
Warning: Migrated state file from '.agentz.state.json' to '.agentspec.state.json'
```

## Working with State

### Inspecting State

View the current state:

```bash
cat .agentspec.state.json | python3 -m json.tool
```

### Resetting State

To force a full re-apply, remove the state file:

```bash
rm .agentspec.state.json
agentspec apply my-app.ias
```

All resources will be treated as new and get `ActionCreate` actions.

### Partial State

If you manually edit the state file to remove entries, those resources will be re-created on the next apply. If you add entries that do not correspond to real deployments, the plan engine will generate `ActionNoop` for them (assuming hashes match).

## Extending the State Backend

To implement a custom state backend (e.g., backed by a database):

1. Implement the `Backend` interface.
2. Register your backend with the state package.
3. Configure the CLI to use the custom backend via a flag or configuration option.

This is useful for team environments where multiple developers need to share deployment state.
