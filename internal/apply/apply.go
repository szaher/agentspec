// Package apply implements the idempotent apply engine with
// partial failure handling for the Agentz toolchain.
package apply

import (
	"context"
	"time"

	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/events"
	"github.com/szaher/designs/agentz/internal/state"
)

// Result summarizes the outcome of an apply operation.
type Result struct {
	Created int
	Updated int
	Deleted int
	Failed  int
	Results []adapters.Result
}

// Apply executes a plan using the given adapter, emitting events and
// recording state. Uses mark-and-continue for partial failure handling.
func Apply(
	ctx context.Context,
	adapter adapters.Adapter,
	actions []adapters.Action,
	backend state.Backend,
	emitter events.Emitter,
	correlationID string,
) (*Result, error) {
	emitter.Emit(events.New(events.ApplyStarted, correlationID).
		WithData("adapter", adapter.Name()).
		WithData("action_count", len(actions)))

	// Filter out noop actions
	var toApply []adapters.Action
	for _, a := range actions {
		if a.Type != adapters.ActionNoop {
			toApply = append(toApply, a)
		}
	}

	if len(toApply) == 0 {
		emitter.Emit(events.New(events.ApplyCompleted, correlationID).
			WithData("message", "no changes"))
		return &Result{}, nil
	}

	// Apply through the adapter
	results, err := adapter.Apply(ctx, toApply)
	if err != nil {
		return nil, err
	}

	// Process results and update state
	existing, _ := backend.Load()
	stateMap := make(map[string]state.Entry)
	for _, e := range existing {
		stateMap[e.FQN] = e
	}

	result := &Result{Results: results}

	for i, r := range results {
		emitter.Emit(events.New(events.ApplyResource, correlationID).
			WithData("fqn", r.FQN).
			WithData("action", string(r.Action)).
			WithData("status", string(r.Status)))

		if r.Status == adapters.ResultSuccess {
			switch r.Action {
			case adapters.ActionCreate:
				result.Created++
			case adapters.ActionUpdate:
				result.Updated++
			case adapters.ActionDelete:
				result.Deleted++
				delete(stateMap, r.FQN)
				continue
			}

			// Find matching resource for hash
			hash := ""
			if i < len(toApply) && toApply[i].Resource != nil {
				hash = toApply[i].Resource.Hash
			}

			stateMap[r.FQN] = state.Entry{
				FQN:         r.FQN,
				Hash:        hash,
				Status:      state.StatusApplied,
				LastApplied: time.Now(),
				Adapter:     adapter.Name(),
			}
		} else {
			result.Failed++
			stateMap[r.FQN] = state.Entry{
				FQN:         r.FQN,
				Hash:        "",
				Status:      state.StatusFailed,
				LastApplied: time.Now(),
				Adapter:     adapter.Name(),
				Error:       r.Error,
			}
		}
	}

	// Save state
	var entries []state.Entry
	for _, e := range stateMap {
		entries = append(entries, e)
	}
	if err := backend.Save(entries); err != nil {
		return result, err
	}

	emitter.Emit(events.New(events.ApplyCompleted, correlationID).
		WithData("created", result.Created).
		WithData("updated", result.Updated).
		WithData("deleted", result.Deleted).
		WithData("failed", result.Failed))

	return result, nil
}
