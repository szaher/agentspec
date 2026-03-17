package integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/szaher/agentspec/internal/pipeline"
)

// TestDAGSortPerformance (T031A) builds a 100-step linear chain where
// step_N depends on step_N-1 and verifies that BuildDAG completes
// within 10ms. The resulting DAG must have 100 layers (one per step).
func TestDAGSortPerformance(t *testing.T) {
	steps := make([]pipeline.Step, 100)
	steps[0] = pipeline.Step{Name: "step_0", AgentRef: "agent"}
	for i := 1; i < 100; i++ {
		steps[i] = pipeline.Step{
			Name:      fmt.Sprintf("step_%d", i),
			AgentRef:  "agent",
			DependsOn: []string{fmt.Sprintf("step_%d", i-1)},
		}
	}

	start := time.Now()
	dag, err := pipeline.BuildDAG(steps)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	if elapsed > 10*time.Millisecond {
		t.Errorf("BuildDAG took %v, expected under 10ms", elapsed)
	}

	if len(dag.Order) != 100 {
		t.Fatalf("expected 100 layers (one per chain step), got %d", len(dag.Order))
	}

	// Each layer should contain exactly one step in chain order.
	for i := 0; i < 100; i++ {
		expected := fmt.Sprintf("step_%d", i)
		if len(dag.Order[i]) != 1 {
			t.Errorf("layer %d: expected 1 step, got %d", i, len(dag.Order[i]))
		} else if dag.Order[i][0] != expected {
			t.Errorf("layer %d: expected %s, got %s", i, expected, dag.Order[i][0])
		}
	}
}

// TestDAGSortWideParallel creates 100 independent steps plus one final
// step that depends on all 100. The resulting DAG must have exactly 2
// layers and BuildDAG should complete quickly.
func TestDAGSortWideParallel(t *testing.T) {
	steps := make([]pipeline.Step, 0, 101)
	depNames := make([]string, 100)

	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("worker_%d", i)
		depNames[i] = name
		steps = append(steps, pipeline.Step{
			Name:     name,
			AgentRef: "agent",
		})
	}

	steps = append(steps, pipeline.Step{
		Name:      "merge",
		AgentRef:  "agent",
		DependsOn: depNames,
	})

	start := time.Now()
	dag, err := pipeline.BuildDAG(steps)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	if elapsed > 10*time.Millisecond {
		t.Errorf("BuildDAG took %v, expected under 10ms", elapsed)
	}

	if len(dag.Order) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(dag.Order))
	}

	// First layer: all 100 independent workers.
	if len(dag.Order[0]) != 100 {
		t.Errorf("first layer: expected 100 steps, got %d", len(dag.Order[0]))
	}

	// Second layer: the single merge step.
	if len(dag.Order[1]) != 1 || dag.Order[1][0] != "merge" {
		t.Errorf("second layer: expected [merge], got %v", dag.Order[1])
	}
}

// BenchmarkDAGBuildChain benchmarks BuildDAG on a 100-step linear chain.
func BenchmarkDAGBuildChain(b *testing.B) {
	steps := make([]pipeline.Step, 100)
	steps[0] = pipeline.Step{Name: "step_0", AgentRef: "agent"}
	for i := 1; i < 100; i++ {
		steps[i] = pipeline.Step{
			Name:      fmt.Sprintf("step_%d", i),
			AgentRef:  "agent",
			DependsOn: []string{fmt.Sprintf("step_%d", i-1)},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pipeline.BuildDAG(steps)
		if err != nil {
			b.Fatalf("BuildDAG failed: %v", err)
		}
	}
}

// BenchmarkDAGBuildWide benchmarks BuildDAG on 100 independent steps
// plus one merge step depending on all of them.
func BenchmarkDAGBuildWide(b *testing.B) {
	steps := make([]pipeline.Step, 0, 101)
	depNames := make([]string, 100)

	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("worker_%d", i)
		depNames[i] = name
		steps = append(steps, pipeline.Step{
			Name:     name,
			AgentRef: "agent",
		})
	}

	steps = append(steps, pipeline.Step{
		Name:      "merge",
		AgentRef:  "agent",
		DependsOn: depNames,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pipeline.BuildDAG(steps)
		if err != nil {
			b.Fatalf("BuildDAG failed: %v", err)
		}
	}
}
