# Contract: State Backend Interface

## Core Interface (existing — extended)

```go
// Backend is the interface for state persistence.
type Backend interface {
    // Load reads all state entries from the backend.
    Load() ([]Entry, error)

    // Save writes all state entries to the backend.
    Save(entries []Entry) error

    // Get retrieves a single entry by FQN.
    Get(fqn string) (*Entry, error)

    // List returns all entries, optionally filtered by status.
    List(status *Status) ([]Entry, error)
}
```

## Extended Interface (new)

```go
// HealthChecker is an optional interface for backends that support health checks.
type HealthChecker interface {
    // Ping validates connectivity and access permissions.
    Ping(ctx context.Context) error
}

// Closer is an optional interface for backends that hold connections.
type Closer interface {
    // Close releases backend resources (connections, pools).
    Close() error
}

// Locker is an optional interface for backends that support distributed locking.
type Locker interface {
    // Lock acquires an exclusive lock for state operations.
    Lock(ctx context.Context) error
    // Unlock releases the exclusive lock.
    Unlock() error
}

// BudgetStore is an optional interface for backends that support budget persistence.
type BudgetStore interface {
    LoadBudgets() ([]BudgetState, error)
    SaveBudgets(budgets []BudgetState) error
}

// VersionStore is an optional interface for backends that support version history.
type VersionStore interface {
    SaveVersion(agent string, entry VersionEntry) error
    GetVersions(agent string) ([]VersionEntry, error)
}
```

## Registry

```go
// BackendFactory creates a Backend from configuration properties.
type BackendFactory func(props map[string]string) (Backend, error)

// Register registers a backend factory for a given type name.
func Register(typeName string, factory BackendFactory)

// New creates a Backend by type name and properties.
func New(typeName string, props map[string]string) (Backend, error)

// Available returns the list of registered backend type names.
func Available() []string
```

## Backend Implementations

| Type         | Implements                                    |
|--------------|-----------------------------------------------|
| `local`      | Backend, Locker, BudgetStore, VersionStore    |
| `kubernetes` | Backend, HealthChecker, Closer                |
| `etcd`       | Backend, HealthChecker, Locker, Closer        |
| `postgres`   | Backend, HealthChecker, Locker, Closer, BudgetStore, VersionStore |
| `s3`         | Backend, HealthChecker, Closer                |

## Test Contract

Every backend MUST pass the same conformance test suite:
1. Save N entries, Load returns same N entries
2. Get by FQN returns correct entry
3. List with nil status returns all; List with status filter returns subset
4. Save is idempotent (save same entries twice, Load returns same result)
5. Concurrent Save from two goroutines does not corrupt state
6. HealthChecker.Ping returns nil when backend is reachable
7. HealthChecker.Ping returns error when backend is unreachable
