# Adapter System

Adapters bridge the gap between the platform-independent plan engine and specific deployment targets. Each adapter translates IR resources and plan actions into platform-specific operations.

## Package

| Package | Path | Purpose |
|---------|------|---------|
| `adapters` | `internal/adapters/` | Adapter interface, registry, and built-in implementations |

Sub-packages for built-in adapters:

| Sub-package | Path | Target |
|-------------|------|--------|
| `process` | `internal/adapters/process/` | Local OS processes |
| `local` | `internal/adapters/local/` | Local MCP server processes |
| `docker` | `internal/adapters/docker/` | Docker containers |
| `compose` | `internal/adapters/compose/` | Docker Compose stacks |
| `kubernetes` | `internal/adapters/kubernetes/` | Kubernetes deployments |

## Adapter Interface

Every adapter implements the `Adapter` interface defined in `internal/adapters/adapter.go`:

```go
type Adapter interface {
    // Name returns the adapter identifier.
    Name() string

    // Validate checks whether the IR resources are compatible.
    Validate(ctx context.Context, resources []ir.Resource) error

    // Apply executes the planned actions.
    Apply(ctx context.Context, actions []Action) ([]Result, error)

    // Export generates platform-specific artifacts without applying.
    Export(ctx context.Context, resources []ir.Resource, outDir string) error

    // Status returns the runtime status of deployed resources.
    Status(ctx context.Context) ([]ResourceStatus, error)

    // Logs streams logs from deployed resources to the provided writer.
    Logs(ctx context.Context, w io.Writer, opts LogOptions) error

    // Destroy tears down all deployed resources.
    Destroy(ctx context.Context) ([]Result, error)
}
```

### Method Responsibilities

| Method | When Called | Purpose |
|--------|-----------|---------|
| `Name()` | Always | Returns the adapter identifier (e.g., `"docker"`, `"kubernetes"`) |
| `Validate()` | `agentspec validate`, `agentspec plan` | Checks platform prerequisites and resource compatibility |
| `Apply()` | `agentspec apply` | Executes create/update/delete actions |
| `Export()` | `agentspec export` | Generates artifacts (Dockerfiles, manifests) without deploying |
| `Status()` | `agentspec status` | Queries runtime state of deployed resources |
| `Logs()` | `agentspec logs` | Streams container/process logs |
| `Destroy()` | `agentspec destroy` | Tears down all managed resources |

## Action and Result Types

Actions flow from the plan engine to adapters:

```go
type ActionType string

const (
    ActionCreate ActionType = "create"
    ActionUpdate ActionType = "update"
    ActionDelete ActionType = "delete"
    ActionNoop   ActionType = "noop"
)

type Action struct {
    FQN      string        // Fully Qualified Name of the resource
    Type     ActionType    // What to do
    Resource *ir.Resource  // The desired resource (nil for delete)
    Reason   string        // Human-readable explanation
}
```

Results flow back from adapters:

```go
type ResultStatus string

const (
    ResultSuccess ResultStatus = "success"
    ResultFailed  ResultStatus = "failed"
)

type Result struct {
    FQN      string
    Action   ActionType
    Status   ResultStatus
    Error    string  // Error message if failed
    Artifact string  // Path to generated artifact (if any)
}
```

## Resource Status

The `Status()` method returns runtime information for each deployed resource:

```go
type ResourceStatus struct {
    FQN       string            `json:"fqn"`
    Name      string            `json:"name"`
    Kind      string            `json:"kind"`
    State     string            `json:"state"`     // running, stopped, failed, pending, unknown
    Endpoint  string            `json:"endpoint,omitempty"`
    Health    string            `json:"health,omitempty"` // healthy, unhealthy, unknown
    Uptime    string            `json:"uptime,omitempty"`
    Replicas  string            `json:"replicas,omitempty"`
    ExtraInfo map[string]string `json:"extra_info,omitempty"`
}
```

## Adapter Registry

Adapters register themselves via a global registry:

```go
// AdapterFactory is a function that creates a new adapter instance.
type AdapterFactory func() Adapter

// Register adds an adapter factory to the global registry.
func Register(name string, factory AdapterFactory)

// Get retrieves an adapter factory by name.
func Get(name string) (AdapterFactory, error)

// List returns the names of all registered adapters.
func List() []string
```

The registry is thread-safe (guarded by `sync.RWMutex`). Built-in adapters register themselves in their package `init()` functions.

## Built-in Adapters

### Process Adapter (`process`)

Runs agents as local OS processes. Used for development and testing.

- **Create** -- Spawns a new process with the configured command and environment.
- **Update** -- Stops the existing process and starts a new one.
- **Delete** -- Sends SIGTERM, then SIGKILL after a grace period.
- **Export** -- Generates a shell script with the process invocation.

### Docker Adapter (`docker`)

Deploys agents as Docker containers.

- **Create** -- Builds or pulls the image, creates and starts a container.
- **Update** -- Stops the old container, starts a new one with updated config.
- **Delete** -- Stops and removes the container.
- **Export** -- Generates a Dockerfile and docker-compose.yml fragment.

### Compose Adapter (`compose` / `docker-compose`)

Manages multi-container deployments via Docker Compose.

- **Create/Update** -- Generates or updates `docker-compose.yml` and runs `docker compose up -d`.
- **Delete** -- Runs `docker compose down` for the relevant services.
- **Export** -- Writes the complete `docker-compose.yml`.

### Kubernetes Adapter (`kubernetes`)

Deploys agents to a Kubernetes cluster.

- **Create** -- Generates and applies Deployment, Service, and ConfigMap manifests.
- **Update** -- Patches existing resources with updated specs.
- **Delete** -- Deletes the managed Kubernetes resources.
- **Export** -- Writes Kubernetes YAML manifests to disk.
- **Status** -- Queries pod status, replica counts, and endpoints.

## Adapter Lifecycle

When `agentspec apply` runs:

```text
  Plan (actions)
      |
      v
  Resolve deploy target -> adapter name
      |
      v
  adapters.Get(name) -> AdapterFactory
      |
      v
  factory() -> Adapter instance
      |
      v
  adapter.Validate(resources)
      |
      v
  adapter.Apply(actions) -> []Result
      |
      v
  Update state file with results
```

The adapter receives only the actions relevant to its resources. Noop actions are typically skipped by the adapter. Failed results are recorded in the state file and retried on the next apply.

## Log Streaming

The `Logs()` method accepts a `LogOptions` struct for flexible log retrieval:

```go
type LogOptions struct {
    Follow bool   // Tail the log stream
    Tail   int    // Number of recent lines to show
    Since  string // Show logs since a timestamp or duration
}
```

Output is written to the provided `io.Writer`, typically `os.Stdout`.
