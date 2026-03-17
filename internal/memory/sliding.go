package memory

import (
	"context"
	"log/slog"
	"sync"

	"github.com/szaher/designs/agentz/internal/llm"
)

// SlidingWindowOption configures a SlidingWindow memory store.
type SlidingWindowOption func(*SlidingWindow)

// WithMaxSessions sets the maximum number of concurrent sessions tracked.
// When exceeded, the least-recently-used session is evicted.
// Default is 10000.
func WithMaxSessions(n int) SlidingWindowOption {
	return func(s *SlidingWindow) {
		if n > 0 {
			s.maxSessions = n
		}
	}
}

// SlidingWindow implements a fixed-size message history with FIFO eviction.
// It tracks session access order via an LRU and enforces a maximum session count.
type SlidingWindow struct {
	mu          sync.RWMutex
	maxMessages int
	maxSessions int
	sessions    map[string][]llm.Message
	lru         *LRU
	logger      *slog.Logger
}

// NewSlidingWindow creates a sliding window memory store.
// maxMessages is the maximum number of messages retained per session.
// Options can be provided to configure additional behaviour such as max sessions.
func NewSlidingWindow(maxMessages int, opts ...SlidingWindowOption) *SlidingWindow {
	if maxMessages <= 0 {
		maxMessages = 50
	}
	s := &SlidingWindow{
		maxMessages: maxMessages,
		maxSessions: 10000,
		sessions:    make(map[string][]llm.Message),
		lru:         NewLRU(),
		logger:      slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SetLogger sets the structured logger used for eviction events.
func (s *SlidingWindow) SetLogger(l *slog.Logger) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger = l
}

// Load retrieves the message history for a session.
// The session is promoted to the most-recently-used position.
func (s *SlidingWindow) Load(_ context.Context, sessionID string) ([]llm.Message, error) {
	s.mu.RLock()
	msgs := s.sessions[sessionID]
	result := make([]llm.Message, len(msgs))
	copy(result, msgs)
	s.mu.RUnlock()

	// Promote requires write access to the LRU; take a full lock.
	s.mu.Lock()
	s.lru.Promote(sessionID)
	s.mu.Unlock()

	return result, nil
}

// Save appends messages and evicts oldest when the window is exceeded.
// The session is promoted in the LRU tracker. If the total session count
// exceeds maxSessions, the least-recently-used sessions are evicted.
func (s *SlidingWindow) Save(_ context.Context, sessionID string, messages []llm.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing := s.sessions[sessionID]
	existing = append(existing, messages...)

	// Evict from the front if over per-session message limit.
	if len(existing) > s.maxMessages {
		existing = existing[len(existing)-s.maxMessages:]
	}

	s.sessions[sessionID] = existing
	s.lru.Promote(sessionID)

	// Evict least-recently-used sessions if over the session limit.
	evicted := 0
	for s.lru.Len() > s.maxSessions {
		victim := s.lru.Evict()
		if victim == "" {
			break
		}
		delete(s.sessions, victim)
		evicted++
	}
	if evicted > 0 {
		s.logger.Info("memory session eviction",
			"evicted", evicted,
			"remaining", len(s.sessions),
			"max", s.maxSessions,
		)
	}

	return nil
}

// Clear removes all messages for a session and removes it from the LRU tracker.
func (s *SlidingWindow) Clear(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	s.lru.Remove(sessionID)
	return nil
}
