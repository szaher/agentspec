package memory

import (
	"context"
	"sync"

	"github.com/szaher/designs/agentz/internal/llm"
)

// SlidingWindow implements a fixed-size message history with FIFO eviction.
type SlidingWindow struct {
	mu          sync.Mutex
	maxMessages int
	sessions    map[string][]llm.Message
}

// NewSlidingWindow creates a sliding window memory store.
// maxMessages is the maximum number of messages retained per session.
func NewSlidingWindow(maxMessages int) *SlidingWindow {
	if maxMessages <= 0 {
		maxMessages = 50
	}
	return &SlidingWindow{
		maxMessages: maxMessages,
		sessions:    make(map[string][]llm.Message),
	}
}

// Load retrieves the message history for a session.
func (s *SlidingWindow) Load(_ context.Context, sessionID string) ([]llm.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs := s.sessions[sessionID]
	result := make([]llm.Message, len(msgs))
	copy(result, msgs)
	return result, nil
}

// Save appends messages and evicts oldest when the window is exceeded.
func (s *SlidingWindow) Save(_ context.Context, sessionID string, messages []llm.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing := s.sessions[sessionID]
	existing = append(existing, messages...)

	// Evict from the front if over limit
	if len(existing) > s.maxMessages {
		existing = existing[len(existing)-s.maxMessages:]
	}

	s.sessions[sessionID] = existing
	return nil
}

// Clear removes all messages for a session.
func (s *SlidingWindow) Clear(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	return nil
}
