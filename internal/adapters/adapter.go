// Package adapters defines the adapter interface and registry
// for the AgentSpec toolchain.
package adapters

import (
	"context"
	"fmt"
	"sync"

	"github.com/szaher/designs/agentz/internal/ir"
)

// ActionType represents the type of change for a resource.
type ActionType string

const (
	ActionCreate ActionType = "create"
	ActionUpdate ActionType = "update"
	ActionDelete ActionType = "delete"
	ActionNoop   ActionType = "noop"
)

// Action describes a planned change for a single resource.
type Action struct {
	FQN      string
	Type     ActionType
	Resource *ir.Resource
	Reason   string
}

// ResultStatus represents the outcome of applying an action.
type ResultStatus string

const (
	ResultSuccess ResultStatus = "success"
	ResultFailed  ResultStatus = "failed"
)

// Result describes the outcome of applying a single action.
type Result struct {
	FQN      string
	Action   ActionType
	Status   ResultStatus
	Error    string
	Artifact string
}

// Adapter translates IR resources into platform-specific
// artifacts and applies them.
type Adapter interface {
	// Name returns the adapter identifier.
	Name() string

	// Validate checks whether the IR resources are compatible.
	Validate(ctx context.Context, resources []ir.Resource) error

	// Apply executes the planned actions.
	Apply(ctx context.Context, actions []Action) ([]Result, error)

	// Export generates platform-specific artifacts without applying.
	Export(ctx context.Context, resources []ir.Resource, outDir string) error
}

// AdapterFactory is a function that creates a new adapter instance.
type AdapterFactory func() Adapter

var (
	registryMu sync.RWMutex
	registry   = make(map[string]AdapterFactory)
)

// Register adds an adapter factory to the global registry.
func Register(name string, factory AdapterFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// Get retrieves an adapter factory by name.
func Get(name string) (AdapterFactory, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("adapter %q not registered", name)
	}
	return factory, nil
}

// List returns the names of all registered adapters.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
