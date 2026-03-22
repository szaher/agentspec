// Package state defines the state backend interface and types
// for tracking applied resource lifecycle.
package state

import (
	"context"
	"time"
)

// Status represents the lifecycle status of a resource.
type Status string

const (
	StatusApplied  Status = "applied"
	StatusFailed   Status = "failed"
	StatusOrphaned Status = "orphaned"
)

// Entry records the state of a single resource after apply.
type Entry struct {
	FQN         string    `json:"fqn"`
	Hash        string    `json:"hash"`
	Status      Status    `json:"status"`
	LastApplied time.Time `json:"last_applied"`
	Adapter     string    `json:"adapter"`
	Error       string    `json:"error,omitempty"`
	OrphanedAt  time.Time `json:"orphaned_at,omitempty"`
}

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

// HealthChecker is an optional interface for backends that support health checks.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// Closer is an optional interface for backends that hold connections.
type Closer interface {
	Close() error
}

// Locker is an optional interface for backends that support distributed locking.
type Locker interface {
	Lock(ctx context.Context) error
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
