package cost

import (
	"fmt"
	"sync"
	"time"
)

// BudgetEntry tracks spending for one agent in one period.
type BudgetEntry struct {
	AgentName    string  `json:"agent_name"`
	Period       string  `json:"period"` // "daily" or "monthly"
	LimitDollars float64 `json:"limit_dollars"`
	UsedDollars  float64 `json:"used_dollars"`
	ResetAt      string  `json:"reset_at"` // ISO 8601
	Paused       bool    `json:"paused"`
	WarnedAt     string  `json:"warned_at,omitempty"`
}

// CostTracker accumulates per-agent costs and enforces budgets.
type CostTracker struct {
	mu      sync.RWMutex
	costs   map[string]float64 // agent → total cost
	budgets map[string][]BudgetEntry
}

// New creates a CostTracker with the given budget entries.
func New(budgets []BudgetEntry) *CostTracker {
	ct := &CostTracker{
		costs:   make(map[string]float64),
		budgets: make(map[string][]BudgetEntry),
	}
	for _, b := range budgets {
		ct.budgets[b.AgentName] = append(ct.budgets[b.AgentName], b)
	}
	return ct
}

// RecordUsage calculates cost from token counts and accumulates it.
func (ct *CostTracker) RecordUsage(agent, model string, inputTokens, outputTokens int) float64 {
	inPrice, outPrice := LookupPrice(model)
	cost := (float64(inputTokens) / 1_000_000 * inPrice) + (float64(outputTokens) / 1_000_000 * outPrice)

	ct.mu.Lock()
	ct.costs[agent] += cost
	// Update budget entries
	for i, b := range ct.budgets[agent] {
		ct.budgets[agent][i].UsedDollars = b.UsedDollars + cost
	}
	ct.mu.Unlock()

	return cost
}

// GetAgentCost returns the total accumulated cost for an agent.
func (ct *CostTracker) GetAgentCost(agent string) float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.costs[agent]
}

// CheckBudget returns an error if any budget for the agent is exceeded.
// It also handles period resets and 80% warnings.
func (ct *CostTracker) CheckBudget(agent string) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	entries, ok := ct.budgets[agent]
	if !ok {
		return nil // no budget configured
	}

	now := time.Now().UTC()
	for i, b := range entries {
		// Check for period reset
		resetAt, err := time.Parse(time.RFC3339, b.ResetAt)
		if err == nil && now.After(resetAt) {
			ct.budgets[agent][i].UsedDollars = 0
			ct.budgets[agent][i].Paused = false
			ct.budgets[agent][i].WarnedAt = ""
			ct.budgets[agent][i].ResetAt = nextReset(now, b.Period).Format(time.RFC3339)
			continue
		}

		if b.Paused || b.UsedDollars >= b.LimitDollars {
			ct.budgets[agent][i].Paused = true
			return fmt.Errorf("agent '%s' %s budget of $%.2f exceeded (used: $%.2f), resets at %s",
				agent, b.Period, b.LimitDollars, b.UsedDollars, b.ResetAt)
		}
	}

	return nil
}

// CheckWarnings returns true and the budget entry if any budget is at 80%+ usage.
func (ct *CostTracker) CheckWarnings(agent string) (warn bool, entry BudgetEntry) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	for i, b := range ct.budgets[agent] {
		ratio := b.UsedDollars / b.LimitDollars
		if ratio >= 0.8 && b.WarnedAt == "" && !b.Paused {
			ct.budgets[agent][i].WarnedAt = time.Now().UTC().Format(time.RFC3339)
			return true, b
		}
	}
	return false, BudgetEntry{}
}

// GetBudgets returns all budget entries (for state persistence).
func (ct *CostTracker) GetBudgets() []BudgetEntry {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var all []BudgetEntry
	for _, entries := range ct.budgets {
		all = append(all, entries...)
	}
	return all
}

// BudgetUsageRatio returns the usage ratio for an agent's budget period (0.0–1.0+).
func (ct *CostTracker) BudgetUsageRatio(agent, period string) float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	for _, b := range ct.budgets[agent] {
		if b.Period == period {
			if b.LimitDollars == 0 {
				return 0
			}
			return b.UsedDollars / b.LimitDollars
		}
	}
	return 0
}

// Reset clears all accumulated costs.
func (ct *CostTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.costs = make(map[string]float64)
}

func nextReset(now time.Time, period string) time.Time {
	switch period {
	case "monthly":
		return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	default: // daily
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	}
}
