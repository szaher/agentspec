// Package session defines the session management abstraction for the AgentSpec runtime.
package session

import (
	"context"
	"time"
)

// Session represents a stateful conversation with an agent.
type Session struct {
	ID         string            `json:"id"`
	AgentName  string            `json:"agent_name"`
	CreatedAt  time.Time         `json:"created_at"`
	LastActive time.Time         `json:"last_active"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Store manages session lifecycle.
type Store interface {
	// Create creates a new session for the given agent.
	Create(ctx context.Context, agentName string, metadata map[string]string) (*Session, error)

	// Get retrieves a session by ID.
	Get(ctx context.Context, id string) (*Session, error)

	// Delete removes a session by ID.
	Delete(ctx context.Context, id string) error

	// List returns all sessions, optionally filtered by agent name.
	List(ctx context.Context, agentName string) ([]*Session, error)

	// Touch updates the last active timestamp.
	Touch(ctx context.Context, id string) error
}
