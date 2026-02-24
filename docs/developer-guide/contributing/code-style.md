# Code Style

This guide documents the Go coding conventions, linting rules, and patterns followed in the AgentSpec codebase.

## General Principles

- Follow standard Go conventions as documented in [Effective Go](https://go.dev/doc/effective_go) and the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
- Write clear, simple code. Avoid premature abstraction.
- Keep packages focused. Each package should have a single, well-defined responsibility.
- Write comprehensive tests alongside implementation code.

## Linting

The project uses golangci-lint v2.10+ with the configuration in `.golangci.yml`.

Run the linter:

```bash
golangci-lint run ./...
```

All code must pass linting before merge. The CI pipeline enforces this automatically.

### Key Linter Rules

The following linters are typically enabled:

- **errcheck** -- All error return values must be handled.
- **govet** -- Static analysis for common bugs.
- **staticcheck** -- Advanced static analysis.
- **unused** -- No unused code.
- **gofmt** -- Canonical formatting.
- **goimports** -- Import grouping and ordering.

## Naming Conventions

### Packages

- Use short, lowercase, single-word package names: `parser`, `ir`, `plan`, `state`.
- Avoid stuttering: `parser.Parser` is acceptable, `parser.ParserEngine` is not.

### Types

- Use PascalCase for exported types: `Document`, `Resource`, `Adapter`.
- Use descriptive names: `ActionType` not `AT`, `ResultStatus` not `RS`.
- Interface names should describe behavior, not just append "er" blindly. `Backend` is better than `Storer`.

### Functions

- Use PascalCase for exported functions: `ComputePlan`, `SerializeCanonical`.
- Constructor functions: `New<Type>` (e.g., `NewHost`, `NewRegistry`, `NewManager`).
- Factory functions: `<Type>Factory` as a type name (e.g., `AdapterFactory`).

### Constants

- Use PascalCase for exported constants: `ActionCreate`, `StatusApplied`.
- Group related constants with `const (...)` blocks.
- Use typed constants with `iota` or string values as appropriate.

### Variables

- Use camelCase for local variables: `currentMap`, `desiredMap`.
- Avoid single-letter names except for:
  - Loop variables: `i`, `j`, `k`
  - Receivers: short form of type name (e.g., `p` for `Parser`, `h` for `Host`)
  - Context: `ctx`

### File Names

- Use snake_case for Go files: `adapter.go`, `deploy_target.go`.
- Test files: `<name>_test.go`.

## Error Handling

### Always Handle Errors

Every error return value must be checked:

```go
// Good
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doing something: %w", err)
}

// Bad -- linter will catch this
result, _ := doSomething()
```

### Error Wrapping

Use `fmt.Errorf` with `%w` to wrap errors with context:

```go
if err := adapter.Apply(ctx, actions); err != nil {
    return fmt.Errorf("apply to %s: %w", adapter.Name(), err)
}
```

The wrapping pattern: `<what you were doing>: %w`.

### Sentinel Errors

Define sentinel errors when callers need to check for specific conditions:

```go
var ErrNotFound = errors.New("resource not found")
```

### Error Types

Use custom error types when the caller needs access to structured error data:

```go
type ParseError struct {
    File    string
    Line    int
    Column  int
    Message string
    Hint    string
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("%s:%d:%d: error: %s", e.File, e.Line, e.Column, e.Message)
}
```

## Package Organization

### Internal Packages

All core logic lives under `internal/`. This prevents external consumers from depending on unstable APIs.

### Import Grouping

Organize imports in three groups separated by blank lines:

```go
import (
    // Standard library
    "context"
    "fmt"

    // Third-party
    "github.com/tetratelabs/wazero"

    // Internal
    "github.com/szaher/designs/agentz/internal/ir"
    "github.com/szaher/designs/agentz/internal/state"
)
```

### Package Documentation

Every package must have a doc comment on the `package` line:

```go
// Package plan implements the desired-state diff engine for the
// AgentSpec toolchain.
package plan
```

## Interface Design

### Small Interfaces

Prefer small, focused interfaces:

```go
// Good -- focused
type Resolver interface {
    Resolve(ctx context.Context, ref string) (string, error)
}

// Avoid -- too broad
type Everything interface {
    Resolve(ctx context.Context, ref string) (string, error)
    List(ctx context.Context) ([]string, error)
    Create(ctx context.Context, name, value string) error
    Delete(ctx context.Context, name string) error
}
```

### Accept Interfaces, Return Structs

Functions should accept interfaces and return concrete types:

```go
// Good
func NewManager(store Store, mem memory.Store) *Manager

// Avoid
func NewManager(store Store, mem memory.Store) ManagerInterface
```

## Concurrency

### Thread Safety

Use `sync.RWMutex` for registries and shared maps:

```go
var (
    registryMu sync.RWMutex
    registry   = make(map[string]AdapterFactory)
)

func Register(name string, factory AdapterFactory) {
    registryMu.Lock()
    defer registryMu.Unlock()
    registry[name] = factory
}
```

### Context Propagation

Pass `context.Context` as the first parameter to all functions that perform I/O or long-running operations:

```go
func (a *Adapter) Apply(ctx context.Context, actions []Action) ([]Result, error)
```

## Struct Tags

Use consistent JSON struct tags:

```go
type Entry struct {
    FQN         string    `json:"fqn"`
    Hash        string    `json:"hash"`
    Status      Status    `json:"status"`
    LastApplied time.Time `json:"last_applied"`
    Error       string    `json:"error,omitempty"`
}
```

- Use `snake_case` for JSON field names.
- Use `omitempty` for optional fields.

## Comments

### Exported Symbols

All exported types, functions, and constants must have doc comments:

```go
// ComputePlan compares desired IR resources against current state
// and produces a set of actions (create/update/delete/noop).
func ComputePlan(desired []ir.Resource, current []state.Entry) *Plan
```

### Non-obvious Logic

Add comments for non-obvious logic, algorithms, or workarounds:

```go
// Sort actions deterministically: by kind extracted from FQN, then by FQN.
// This ensures consistent plan output regardless of map iteration order.
sortActions(actions)
```
