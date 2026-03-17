package memory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/szaher/designs/agentz/internal/llm"
)

// SummaryOption configures a Summary memory store.
type SummaryOption func(*Summary)

// WithSummaryMaxSessions sets the maximum number of sessions to keep in memory.
// When exceeded, the least-recently-used session is evicted.
func WithSummaryMaxSessions(n int) SummaryOption {
	return func(s *Summary) {
		if n > 0 {
			s.maxSessions = n
		}
	}
}

// Summary implements conversation memory with LLM-based summarization.
// When message count exceeds the threshold, older messages are summarized
// into a single summary message.
type Summary struct {
	mu          sync.RWMutex
	threshold   int
	maxSessions int
	sessions    map[string][]llm.Message
	llmClient   llm.Client
	model       string
	lru         *LRU
	logger      *slog.Logger
}

// NewSummary creates a summarization memory store.
func NewSummary(threshold int, llmClient llm.Client, model string, opts ...SummaryOption) *Summary {
	if threshold <= 0 {
		threshold = 20
	}
	s := &Summary{
		threshold:   threshold,
		maxSessions: 10000,
		sessions:    make(map[string][]llm.Message),
		llmClient:   llmClient,
		model:       model,
		lru:         NewLRU(),
		logger:      slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SetLogger sets the logger for the summary store.
func (s *Summary) SetLogger(logger *slog.Logger) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger = logger
}

// Load retrieves the message history for a session.
func (s *Summary) Load(_ context.Context, sessionID string) ([]llm.Message, error) {
	s.mu.RLock()
	msgs := s.sessions[sessionID]
	result := make([]llm.Message, len(msgs))
	copy(result, msgs)
	s.mu.RUnlock()

	s.mu.Lock()
	s.lru.Promote(sessionID)
	s.mu.Unlock()

	return result, nil
}

// Save appends messages and triggers summarization when threshold is exceeded.
func (s *Summary) Save(ctx context.Context, sessionID string, messages []llm.Message) error {
	s.mu.Lock()
	existing := s.sessions[sessionID]
	existing = append(existing, messages...)
	s.sessions[sessionID] = existing
	needsSummary := len(existing) > s.threshold

	s.lru.Promote(sessionID)

	// Evict least-recently-used sessions if we exceed maxSessions.
	for s.lru.Len() > s.maxSessions {
		evicted := s.lru.Evict()
		if evicted == "" {
			break
		}
		delete(s.sessions, evicted)
		s.logger.Info("memory session eviction",
			"evicted", evicted,
			"remaining", s.lru.Len(),
			"max", s.maxSessions,
		)
	}
	s.mu.Unlock()

	if needsSummary {
		return s.summarize(ctx, sessionID)
	}
	return nil
}

// Clear removes all messages for a session.
func (s *Summary) Clear(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	s.lru.Remove(sessionID)
	return nil
}

func (s *Summary) summarize(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	msgs := s.sessions[sessionID]
	if len(msgs) <= s.threshold {
		s.mu.Unlock()
		return nil
	}

	// Keep recent messages, summarize older ones
	keepCount := s.threshold / 2
	toSummarize := msgs[:len(msgs)-keepCount]
	toKeep := msgs[len(msgs)-keepCount:]
	s.mu.Unlock()

	// Build summary prompt
	var summaryContent string
	for _, m := range toSummarize {
		summaryContent += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
	}

	resp, err := s.llmClient.Chat(ctx, llm.ChatRequest{
		Model: s.model,
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "Summarize this conversation concisely, preserving key facts and decisions:\n\n" + summaryContent},
		},
		MaxTokens: 500,
	})
	if err != nil {
		return fmt.Errorf("summarize memory: %w", err)
	}

	// Replace old messages with summary + recent messages
	summarized := []llm.Message{
		{Role: llm.RoleAssistant, Content: "[Previous conversation summary: " + resp.Content + "]"},
	}
	summarized = append(summarized, toKeep...)

	s.mu.Lock()
	s.sessions[sessionID] = summarized
	s.mu.Unlock()

	return nil
}
