package llm

import (
	"fmt"
	"sync"
)

// TokenTracker tracks cumulative token usage and enforces budgets.
type TokenTracker struct {
	mu     sync.Mutex
	budget int
	used   TokenUsage
}

// NewTokenTracker creates a tracker with the given budget.
// A budget of 0 means unlimited.
func NewTokenTracker(budget int) *TokenTracker {
	return &TokenTracker{budget: budget}
}

// Add records token usage from a single LLM call.
func (t *TokenTracker) Add(usage TokenUsage) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.used.InputTokens += usage.InputTokens
	t.used.OutputTokens += usage.OutputTokens
	t.used.CacheRead += usage.CacheRead
	t.used.CacheWrite += usage.CacheWrite
}

// CheckBudget returns an error if the budget would be exceeded by additional tokens.
func (t *TokenTracker) CheckBudget(additional int) error {
	if t.budget <= 0 {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	total := t.used.Total() + additional
	if total > t.budget {
		return fmt.Errorf("token budget exceeded: used %d + requested %d > budget %d",
			t.used.Total(), additional, t.budget)
	}
	return nil
}

// Usage returns the current cumulative usage.
func (t *TokenTracker) Usage() TokenUsage {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.used
}

// Remaining returns the number of tokens remaining in the budget.
// Returns -1 if the budget is unlimited.
func (t *TokenTracker) Remaining() int {
	if t.budget <= 0 {
		return -1
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	rem := t.budget - t.used.Total()
	if rem < 0 {
		return 0
	}
	return rem
}
