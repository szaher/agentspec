package plan

import (
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/state"
)

// DriftResult describes detected drift between state and desired.
type DriftResult struct {
	HasDrift bool
	Drifted  []DriftEntry
}

// DriftEntry describes a single resource with drift.
type DriftEntry struct {
	FQN       string
	Expected  string
	Actual    string
	Type      string // "missing", "hash_mismatch", "extra"
}

// DetectDrift compares current state against desired resources
// and reports any discrepancies.
func DetectDrift(desired []ir.Resource, current []state.Entry) *DriftResult {
	result := &DriftResult{}

	currentMap := make(map[string]state.Entry)
	for _, e := range current {
		currentMap[e.FQN] = e
	}

	desiredMap := make(map[string]ir.Resource)
	for _, r := range desired {
		desiredMap[r.FQN] = r
	}

	// Check desired resources against state
	for _, r := range desired {
		entry, exists := currentMap[r.FQN]
		if !exists {
			result.Drifted = append(result.Drifted, DriftEntry{
				FQN:      r.FQN,
				Expected: r.Hash,
				Actual:   "",
				Type:     "missing",
			})
			result.HasDrift = true
		} else if entry.Hash != r.Hash {
			result.Drifted = append(result.Drifted, DriftEntry{
				FQN:      r.FQN,
				Expected: r.Hash,
				Actual:   entry.Hash,
				Type:     "hash_mismatch",
			})
			result.HasDrift = true
		}
	}

	// Check for extra state entries
	for _, e := range current {
		if _, exists := desiredMap[e.FQN]; !exists {
			result.Drifted = append(result.Drifted, DriftEntry{
				FQN:    e.FQN,
				Actual: e.Hash,
				Type:   "extra",
			})
			result.HasDrift = true
		}
	}

	return result
}
