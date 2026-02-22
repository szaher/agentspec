// Package plan implements the desired-state diff engine for the
// Agentz toolchain.
package plan

import (
	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/state"
)

// Plan represents a computed set of changes.
type Plan struct {
	Actions   []adapters.Action
	TargetBinding string
	HasChanges    bool
}

// ComputePlan compares desired IR resources against current state
// and produces a set of actions (create/update/delete/noop).
func ComputePlan(desired []ir.Resource, current []state.Entry) *Plan {
	currentMap := make(map[string]state.Entry)
	for _, e := range current {
		currentMap[e.FQN] = e
	}

	desiredMap := make(map[string]ir.Resource)
	for _, r := range desired {
		desiredMap[r.FQN] = r
	}

	var actions []adapters.Action
	hasChanges := false

	// Check desired resources against current state
	for _, r := range desired {
		entry, exists := currentMap[r.FQN]
		if !exists {
			actions = append(actions, adapters.Action{
				FQN:      r.FQN,
				Type:     adapters.ActionCreate,
				Resource: copyResource(r),
				Reason:   "resource does not exist",
			})
			hasChanges = true
		} else if entry.Hash != r.Hash {
			actions = append(actions, adapters.Action{
				FQN:      r.FQN,
				Type:     adapters.ActionUpdate,
				Resource: copyResource(r),
				Reason:   "resource hash changed",
			})
			hasChanges = true
		} else if entry.Status == state.StatusFailed {
			// Retry failed resources
			actions = append(actions, adapters.Action{
				FQN:      r.FQN,
				Type:     adapters.ActionUpdate,
				Resource: copyResource(r),
				Reason:   "retrying previously failed resource",
			})
			hasChanges = true
		} else {
			actions = append(actions, adapters.Action{
				FQN:      r.FQN,
				Type:     adapters.ActionNoop,
				Resource: copyResource(r),
				Reason:   "no changes",
			})
		}
	}

	// Check for resources to delete (in state but not in desired)
	for _, e := range current {
		if _, exists := desiredMap[e.FQN]; !exists {
			actions = append(actions, adapters.Action{
				FQN:    e.FQN,
				Type:   adapters.ActionDelete,
				Reason: "resource no longer defined",
			})
			hasChanges = true
		}
	}

	// Sort actions deterministically: by kind extracted from FQN, then by FQN
	sortActions(actions)

	return &Plan{
		Actions:    actions,
		HasChanges: hasChanges,
	}
}

// ResolveBinding finds the target binding from IR, handling defaults.
// If targetName is empty, uses the default or sole binding.
func ResolveBinding(bindings []ir.Binding, targetName string) (*ir.Binding, error) {
	if len(bindings) == 0 {
		return nil, nil
	}

	if targetName != "" {
		for i := range bindings {
			if bindings[i].Name == targetName {
				return &bindings[i], nil
			}
		}
		return nil, nil
	}

	// Find explicit default
	for i := range bindings {
		if bindings[i].Default {
			return &bindings[i], nil
		}
	}

	// Sole binding is implicitly default
	if len(bindings) == 1 {
		return &bindings[0], nil
	}

	return nil, nil
}

func copyResource(r ir.Resource) *ir.Resource {
	copy := r
	return &copy
}

func sortActions(actions []adapters.Action) {
	// Use a simple insertion sort for determinism
	for i := 1; i < len(actions); i++ {
		j := i
		for j > 0 && actions[j].FQN < actions[j-1].FQN {
			actions[j], actions[j-1] = actions[j-1], actions[j]
			j--
		}
	}
}
