package session

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// defaultEvictionInterval is the default interval between background cleanup cycles.
const defaultEvictionInterval = 5 * time.Minute

// MemoryStore is an in-memory session store with expiry support and
// background eviction of expired sessions.
type MemoryStore struct {
	mu               sync.RWMutex
	sessions         map[string]*Session
	expiry           time.Duration
	evictionInterval time.Duration
	logger           *slog.Logger
}

// NewMemoryStore creates an in-memory session store.
// expiry defines session idle timeout; 0 means no expiry.
// evictionInterval controls how often the background cleanup runs;
// 0 uses the default of 5 minutes.
func NewMemoryStore(expiry time.Duration, evictionInterval time.Duration) *MemoryStore {
	if evictionInterval <= 0 {
		evictionInterval = defaultEvictionInterval
	}
	return &MemoryStore{
		sessions:         make(map[string]*Session),
		expiry:           expiry,
		evictionInterval: evictionInterval,
	}
}

// SetLogger sets the structured logger used for cleanup diagnostics.
func (s *MemoryStore) SetLogger(logger *slog.Logger) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger = logger
}

// Start launches a background goroutine that periodically evicts expired
// sessions. The goroutine stops when ctx is cancelled.
func (s *MemoryStore) Start(ctx context.Context) {
	go s.evictionLoop(ctx)
}

// evictionLoop runs the periodic cleanup cycle.
func (s *MemoryStore) evictionLoop(ctx context.Context) {
	ticker := time.NewTicker(s.evictionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.evict()
		}
	}
}

// evict removes all expired sessions under a write lock and emits a
// structured log line with eviction statistics.
func (s *MemoryStore) evict() {
	s.mu.Lock()
	var evicted int
	for id, sess := range s.sessions {
		if s.expiry > 0 && time.Since(sess.LastActive) > s.expiry {
			delete(s.sessions, id)
			evicted++
		}
	}
	active := len(s.sessions)
	logger := s.logger
	s.mu.Unlock()

	if logger != nil {
		logger.Info("session cleanup",
			slog.Int("evicted", evicted),
			slog.Int("active", active),
		)
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
//
// The happy path (session exists and is not expired) uses a read lock only.
// If the session is found but expired, the read lock is released, a write lock
// is acquired, the expiry condition is re-checked, and the session is deleted.
func (s *MemoryStore) Get(_ context.Context, id string) (*Session, error) {
	s.mu.RLock()
	sess, ok := s.sessions[id]
	if !ok {
		s.mu.RUnlock()
		return nil, fmt.Errorf("session %q not found", id)
	}

	// Happy path: session exists and is not expired.
	if s.expiry <= 0 || time.Since(sess.LastActive) <= s.expiry {
		s.mu.RUnlock()
		return sess, nil
	}
	s.mu.RUnlock()

	// Session appears expired — promote to write lock and re-check.
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok = s.sessions[id]
	if !ok {
		// Already removed by another goroutine.
		return nil, fmt.Errorf("session %q not found", id)
	}
	if s.expiry > 0 && time.Since(sess.LastActive) > s.expiry {
		delete(s.sessions, id)
		return nil, fmt.Errorf("session %q expired", id)
	}
	// Session was touched between the two lock acquisitions; it is now valid.
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
// Expired sessions are excluded from the result and lazily deleted:
// the scan runs under a read lock, then any expired IDs are removed
// under a write lock.
func (s *MemoryStore) List(_ context.Context, agentName string) ([]*Session, error) {
	s.mu.RLock()
	var result []*Session
	var expiredIDs []string
	for id, sess := range s.sessions {
		if s.expiry > 0 && time.Since(sess.LastActive) > s.expiry {
			expiredIDs = append(expiredIDs, id)
			continue
		}
		if agentName != "" && sess.AgentName != agentName {
			continue
		}
		result = append(result, sess)
	}
	s.mu.RUnlock()

	// Lazily delete expired sessions under a write lock.
	if len(expiredIDs) > 0 {
		s.mu.Lock()
		for _, id := range expiredIDs {
			// Re-check expiry under write lock in case the session was touched.
			if sess, ok := s.sessions[id]; ok {
				if s.expiry > 0 && time.Since(sess.LastActive) > s.expiry {
					delete(s.sessions, id)
				}
			}
		}
		s.mu.Unlock()
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
