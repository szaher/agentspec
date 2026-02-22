// Package state defines the state backend interface and types
// for tracking applied resource lifecycle.
package state

import "time"

// Status represents the lifecycle status of a resource.
type Status string

const (
	StatusApplied Status = "applied"
	StatusFailed  Status = "failed"
)

// Entry records the state of a single resource after apply.
type Entry struct {
	FQN         string    `json:"fqn"`
	Hash        string    `json:"hash"`
	Status      Status    `json:"status"`
	LastApplied time.Time `json:"last_applied"`
	Adapter     string    `json:"adapter"`
	Error       string    `json:"error,omitempty"`
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
