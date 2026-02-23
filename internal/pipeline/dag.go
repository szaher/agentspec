// Package pipeline implements the multi-agent pipeline executor for AgentSpec.
package pipeline

import "fmt"

// Step represents a single pipeline step with its dependencies.
type Step struct {
	Name      string
	AgentRef  string
	Input     string
	Output    string
	Parallel  bool
	DependsOn []string
}

// DAG represents a directed acyclic graph of pipeline steps.
type DAG struct {
	Steps    map[string]*Step
	Order    [][]string // Topological layers (steps in same layer can run in parallel)
	Incoming map[string]int
}

// BuildDAG constructs a DAG from a list of pipeline steps.
// Returns an error if the graph contains cycles or references unknown steps.
func BuildDAG(steps []Step) (*DAG, error) {
	dag := &DAG{
		Steps:    make(map[string]*Step),
		Incoming: make(map[string]int),
	}

	// Index steps
	for i := range steps {
		s := &steps[i]
		if _, exists := dag.Steps[s.Name]; exists {
			return nil, fmt.Errorf("duplicate step %q", s.Name)
		}
		dag.Steps[s.Name] = s
		dag.Incoming[s.Name] = 0
	}

	// Validate dependencies and count incoming edges
	for _, s := range dag.Steps {
		for _, dep := range s.DependsOn {
			if _, exists := dag.Steps[dep]; !exists {
				return nil, fmt.Errorf("step %q depends on unknown step %q", s.Name, dep)
			}
			if dep == s.Name {
				return nil, fmt.Errorf("step %q depends on itself", s.Name)
			}
			dag.Incoming[s.Name]++
		}
	}

	// Kahn's algorithm for topological sort with layer detection
	order, err := topologicalSort(dag)
	if err != nil {
		return nil, err
	}
	dag.Order = order

	return dag, nil
}

// topologicalSort performs a layered topological sort using Kahn's algorithm.
// Each layer contains steps that can execute concurrently.
func topologicalSort(dag *DAG) ([][]string, error) {
	incoming := make(map[string]int)
	for k, v := range dag.Incoming {
		incoming[k] = v
	}

	var layers [][]string
	processed := 0
	total := len(dag.Steps)

	for processed < total {
		// Find all steps with no remaining dependencies
		var layer []string
		for name, count := range incoming {
			if count == 0 {
				layer = append(layer, name)
			}
		}

		if len(layer) == 0 {
			return nil, fmt.Errorf("cycle detected in pipeline dependencies")
		}

		// Sort layer for determinism
		sortStrings(layer)
		layers = append(layers, layer)

		// Remove processed steps and update incoming counts
		for _, name := range layer {
			delete(incoming, name)
			processed++

			// Reduce count for dependents
			for depName, depStep := range dag.Steps {
				for _, dep := range depStep.DependsOn {
					if dep == name {
						incoming[depName]--
					}
				}
			}
		}
	}

	return layers, nil
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		j := i
		for j > 0 && s[j] < s[j-1] {
			s[j], s[j-1] = s[j-1], s[j]
			j--
		}
	}
}
