package session

import (
	"context"
	"fmt"

	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/memory"
)

// Manager handles session lifecycle: create, send message, close, expire.
type Manager struct {
	store  Store
	memory memory.Store
}

// NewManager creates a session manager.
func NewManager(store Store, mem memory.Store) *Manager {
	return &Manager{store: store, memory: mem}
}

// Create creates a new session and returns its ID.
func (m *Manager) Create(ctx context.Context, agentName string, metadata map[string]string) (*Session, error) {
	return m.store.Create(ctx, agentName, metadata)
}

// LoadMessages retrieves conversation history for a session.
func (m *Manager) LoadMessages(ctx context.Context, sessionID string) ([]llm.Message, error) {
	// Verify session exists and is active
	sess, err := m.store.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	_ = sess

	return m.memory.Load(ctx, sessionID)
}

// SaveMessages appends messages to the session's conversation history.
func (m *Manager) SaveMessages(ctx context.Context, sessionID string, messages []llm.Message) error {
	if err := m.store.Touch(ctx, sessionID); err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return m.memory.Save(ctx, sessionID, messages)
}

// Close deletes a session and its conversation history.
func (m *Manager) Close(ctx context.Context, sessionID string) error {
	if err := m.memory.Clear(ctx, sessionID); err != nil {
		return fmt.Errorf("clear memory: %w", err)
	}
	return m.store.Delete(ctx, sessionID)
}

// Get retrieves a session by ID.
func (m *Manager) Get(ctx context.Context, sessionID string) (*Session, error) {
	return m.store.Get(ctx, sessionID)
}

// List returns all sessions for an agent.
func (m *Manager) List(ctx context.Context, agentName string) ([]*Session, error) {
	return m.store.List(ctx, agentName)
}
