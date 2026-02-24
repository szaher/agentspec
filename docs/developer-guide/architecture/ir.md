# IR and Plan Engine

The Intermediate Representation (IR) is the fully-resolved, platform-independent description of all resources declared in an IntentLang file. The plan engine diffs the desired IR against the persisted state to produce a minimal set of actions.

## Packages

| Package | Path | Purpose |
|---------|------|---------|
| `ir` | `internal/ir/` | IR types, canonical serialization, content hashing |
| `plan` | `internal/plan/` | Desired-state diff engine, binding/target resolution |
| `state` | `internal/state/` | State persistence backend |

## IR Document Structure

The top-level IR container is `ir.Document`:

```go
type Document struct {
    IRVersion     string         `json:"ir_version"`
    LangVersion   string         `json:"lang_version"`
    Package       Package        `json:"package"`
    Resources     []Resource     `json:"resources"`
    Policies      []Policy       `json:"policies,omitempty"`
    Bindings      []Binding      `json:"bindings,omitempty"`
    DeployTargets []DeployTarget `json:"deploy_targets,omitempty"`
}
```

Each `Resource` is a generic container:

```go
type Resource struct {
    Kind       string                 `json:"kind"`
    Name       string                 `json:"name"`
    FQN        string                 `json:"fqn"`
    Attributes map[string]interface{} `json:"attributes"`
    References []string               `json:"references,omitempty"`
    Hash       string                 `json:"hash"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
```

Key fields:

- **Kind** -- The resource type: `agent`, `prompt`, `skill`, `secret`, `server`, `client`, `deploy`, `pipeline`, `type`, etc.
- **FQN** -- Fully Qualified Name in the format `package/kind/name` (e.g., `my-app/agent/assistant`).
- **Attributes** -- A map holding all resource-specific properties. The schema depends on the `Kind`.
- **References** -- FQNs of other resources this resource depends on.
- **Hash** -- SHA-256 hash of the canonical attribute serialization, used for change detection.

## IR Lowering from AST

The lowering process converts typed AST nodes into generic IR resources:

```text
AST Node (ast.Agent)
    |
    v
IR Resource {
    Kind: "agent",
    Name: "assistant",
    FQN:  "my-app/agent/assistant",
    Attributes: {
        "model": "claude-sonnet-4-20250514",
        "strategy": "react",
        "max_turns": 10,
        "prompt_ref": "my-app/prompt/system",
        ...
    },
    References: ["my-app/prompt/system", "my-app/skill/search"],
    Hash: "a1b2c3...",
}
```

The lowering stage:

1. Iterates over all AST statements.
2. Converts each typed node to a generic `ir.Resource` with its attributes flattened into a map.
3. Resolves references (e.g., `uses prompt "system"` becomes the FQN `pkg/prompt/system`).
4. Computes the content hash from the canonical serialization of attributes.

## Content Hashing (SHA-256)

Content hashing enables efficient change detection without field-by-field comparison.

The process:

1. **Canonical serialization** -- Attributes are serialized to JSON with sorted keys and no whitespace using `SerializeCanonical()`:

```go
func SerializeCanonical(attrs map[string]interface{}) ([]byte, error) {
    ordered := sortedMap(attrs)
    return json.Marshal(ordered)
}
```

2. **SHA-256 digest** -- The canonical JSON bytes are hashed with SHA-256. The hex-encoded hash string is stored in `Resource.Hash`.

3. **Determinism guarantee** -- `SortResources()` sorts the resource list by kind then name. Map keys are sorted recursively. This means two logically identical IR documents always produce the same serialized form and the same hashes.

## Plan Generation

The plan engine (`internal/plan/plan.go`) compares the desired IR resources against the current state entries and produces a set of actions.

```go
func ComputePlan(desired []ir.Resource, current []state.Entry) *Plan
```

The algorithm:

```text
For each desired resource:
    if not in current state     -> ActionCreate ("resource does not exist")
    if hash differs             -> ActionUpdate ("resource hash changed")
    if previously failed        -> ActionUpdate ("retrying previously failed resource")
    else                        -> ActionNoop   ("no changes")

For each current state entry:
    if not in desired resources -> ActionDelete ("resource no longer defined")
```

The `Plan` struct:

```go
type Plan struct {
    Actions       []adapters.Action
    TargetBinding string
    HasChanges    bool
}
```

### Action Types

Actions are defined in the `adapters` package:

```go
type ActionType string

const (
    ActionCreate ActionType = "create"
    ActionUpdate ActionType = "update"
    ActionDelete ActionType = "delete"
    ActionNoop   ActionType = "noop"
)

type Action struct {
    FQN      string
    Type     ActionType
    Resource *ir.Resource
    Reason   string
}
```

Actions are sorted deterministically by FQN before being returned, ensuring consistent plan output.

## Deploy Target Resolution

The plan engine resolves which adapter to use via deploy targets (IntentLang 2.0) or bindings (IntentLang 1.0):

```go
func ResolveDeployTarget(targets []ir.DeployTarget, targetName string) (*ir.DeployTarget, error)
```

Resolution order:

1. If `targetName` is specified, find the exact match.
2. Otherwise, find the target marked `default true`.
3. If there is exactly one target, use it implicitly.

The `DeployTargetAdapter()` function maps target types to adapter names:

| Target Type | Adapter Name |
|-------------|-------------|
| `process` | `local-mcp` |
| `docker` | `docker` |
| `docker-compose` | `docker-compose` |
| `kubernetes` | `kubernetes` |

## State Reconciliation

After apply, the state backend records each resource's FQN, hash, status, adapter, and timestamp:

```go
type Entry struct {
    FQN         string    `json:"fqn"`
    Hash        string    `json:"hash"`
    Status      Status    `json:"status"`      // "applied" or "failed"
    LastApplied time.Time `json:"last_applied"`
    Adapter     string    `json:"adapter"`
    Error       string    `json:"error,omitempty"`
}
```

On the next `plan` or `apply`, these entries are loaded and compared against the new IR to determine what changed. Failed resources are automatically retried on the next apply.

## Extending the IR

To add a new resource kind to the IR pipeline:

1. Define the AST node in `internal/ast/`.
2. Add lowering logic that converts the AST node to an `ir.Resource` with the appropriate `Kind` and `Attributes`.
3. The plan engine and adapters work generically on `ir.Resource`, so no changes are needed there unless the new kind requires special handling.
4. If the resource requires adapter-specific behavior, extend the adapter interface or add adapter-level logic.
