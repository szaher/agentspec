package session

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryStore is an in-memory session store with expiry support.
type MemoryStore struct {
	mu       sync.Mutex
	sessions map[string]*Session
	expiry   time.Duration
}

// NewMemoryStore creates an in-memory session store.
// expiry defines session idle timeout; 0 means no expiry.
func NewMemoryStore(expiry time.Duration) *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*Session),
		expiry:   expiry,
	}
}

// Create creates a new session for the given agent.
func (s *MemoryStore) Create(_ context.Context, agentName string, metadata map[string]string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := generateSecureID()
	now := time.Now()
	sess := &Session{
		ID:         id,
		AgentName:  agentName,
		CreatedAt:  now,
		LastActive: now,
		Metadata:   metadata,
	}
	s.sessions[id] = sess
	return sess, nil
}

// Get retrieves a session by ID.
func (s *MemoryStore) Get(_ context.Context, id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %q not found", id)
	}

	if s.expiry > 0 && time.Since(sess.LastActive) > s.expiry {
		delete(s.sessions, id)
		return nil, fmt.Errorf("session %q expired", id)
	}

	return sess, nil
}

// Delete removes a session by ID.
func (s *MemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

// List returns all sessions, optionally filtered by agent name.
func (s *MemoryStore) List(_ context.Context, agentName string) ([]*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []*Session
	for _, sess := range s.sessions {
		if agentName != "" && sess.AgentName != agentName {
			continue
		}
		if s.expiry > 0 && time.Since(sess.LastActive) > s.expiry {
			continue
		}
		result = append(result, sess)
	}
	return result, nil
}

// Touch updates the last active timestamp.
func (s *MemoryStore) Touch(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %q not found", id)
	}
	sess.LastActive = time.Now()
	return nil
}
