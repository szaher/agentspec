package controller

import (
	"context"
	"fmt"
	"sync"
)

// DAGStep represents a single node in a directed acyclic graph.
type DAGStep struct {
	Name      string
	AgentRef  string
	DependsOn []string
	Input     string
}

// DAGExecutor runs a set of DAG steps respecting dependency order.
type DAGExecutor struct {
	FailFast bool
}

// Run executes steps in topological order, running independent steps concurrently.
// Returns a map of step name to output, or an error if a cycle is detected or a step fails.
func (d *DAGExecutor) Run(ctx context.Context, steps []DAGStep) (map[string]string, error) {
	order, err := topoSort(steps)
	if err != nil {
		return nil, err
	}

	// Build lookup and in-degree tracking per topological level.
	stepMap := make(map[string]DAGStep, len(steps))
	for _, s := range steps {
		stepMap[s.Name] = s
	}

	results := make(map[string]string)
	var mu sync.Mutex
	completed := make(map[string]bool)

	// Process steps in topological waves.
	remaining := make([]string, len(order))
	copy(remaining, order)

	for len(remaining) > 0 {
		// Find steps whose dependencies are all completed.
		var ready, next []string
		for _, name := range remaining {
			s := stepMap[name]
			allMet := true
			for _, dep := range s.DependsOn {
				if !completed[dep] {
					allMet = false
					break
				}
			}
			if allMet {
				ready = append(ready, name)
			} else {
				next = append(next, name)
			}
		}

		// Execute ready steps concurrently.
		ctx2, cancel := context.WithCancel(ctx)
		var wg sync.WaitGroup
		var firstErr error

		for _, name := range ready {
			wg.Add(1)
			go func(n string) {
				defer wg.Done()
				select {
				case <-ctx2.Done():
					return
				default:
				}
				// Simulated execution — real implementation would invoke agent.
				output := fmt.Sprintf("output of %s", n)
				mu.Lock()
				results[n] = output
				completed[n] = true
				mu.Unlock()
			}(name)
		}
		wg.Wait()
		cancel()

		if d.FailFast && firstErr != nil {
			return results, firstErr
		}
		remaining = next
	}

	return results, nil
}

// topoSort performs Kahn's algorithm to produce a topological ordering.
func topoSort(steps []DAGStep) ([]string, error) {
	inDegree := make(map[string]int, len(steps))
	dependents := make(map[string][]string, len(steps))
	for _, s := range steps {
		if _, ok := inDegree[s.Name]; !ok {
			inDegree[s.Name] = 0
		}
		for _, dep := range s.DependsOn {
			inDegree[s.Name]++
			dependents[dep] = append(dependents[dep], s.Name)
		}
	}

	var queue []string
	for _, s := range steps {
		if inDegree[s.Name] == 0 {
			queue = append(queue, s.Name)
		}
	}

	var order []string
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		order = append(order, n)
		for _, dep := range dependents[n] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(order) != len(steps) {
		return nil, fmt.Errorf("cycle detected in DAG: sorted %d of %d steps", len(order), len(steps))
	}
	return order, nil
}
