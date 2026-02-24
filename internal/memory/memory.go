// Package memory defines the conversation memory abstraction for the AgentSpec runtime.
package memory

import (
	"context"

	"github.com/szaher/designs/agentz/internal/llm"
)

// Strategy identifies a memory management strategy.
type Strategy string

const (
	StrategySlidingWindow Strategy = "sliding_window"
	StrategySummary       Strategy = "summary"
)

// Store manages conversation message history for a session.
type Store interface {
	// Load retrieves the message history for a session.
	Load(ctx context.Context, sessionID string) ([]llm.Message, error)

	// Save appends messages to the session history, applying the
	// configured retention strategy (e.g., sliding window eviction).
	Save(ctx context.Context, sessionID string, messages []llm.Message) error

	// Clear removes all messages for a session.
	Clear(ctx context.Context, sessionID string) error
}
