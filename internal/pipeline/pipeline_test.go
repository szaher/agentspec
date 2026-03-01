package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/loop"
)

// ---------- mock AgentInvoker ----------

type mockInvoker struct {
	results map[string]*loop.Response // agentRef → response
	err     map[string]error          // agentRef → error
	calls   int64                     // atomic call counter
	delay   time.Duration             // optional per-call delay
}

func newMockInvoker() *mockInvoker {
	return &mockInvoker{
		results: make(map[string]*loop.Response),
		err:     make(map[string]error),
	}
}

func (m *mockInvoker) addResult(agentRef string, output string, tokens llm.TokenUsage) {
	m.results[agentRef] = &loop.Response{
		Output: output,
		Tokens: tokens,
		Turns:  1,
	}
}

func (m *mockInvoker) addError(agentRef string, err error) {
	m.err[agentRef] = err
}

func (m *mockInvoker) Invoke(ctx context.Context, agentName string, input string) (*loop.Response, error) {
	atomic.AddInt64(&m.calls, 1)

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if err, ok := m.err[agentName]; ok {
		return nil, err
	}
	if resp, ok := m.results[agentName]; ok {
		return resp, nil
	}
	return &loop.Response{Output: "default output for " + agentName}, nil
}

// ---------- BuildDAG tests ----------

func TestBuildDAG(t *testing.T) {
	tests := []struct {
		name      string
		steps     []Step
		wantErr   string
		wantOrder [][]string // expected topological layers
	}{
		{
			name: "linear_pipeline_A_B_C",
			steps: []Step{
				{Name: "A", AgentRef: "agent-a"},
				{Name: "B", AgentRef: "agent-b", DependsOn: []string{"A"}},
				{Name: "C", AgentRef: "agent-c", DependsOn: []string{"B"}},
			},
			wantOrder: [][]string{{"A"}, {"B"}, {"C"}},
		},
		{
			name: "parallel_steps",
			steps: []Step{
				{Name: "A", AgentRef: "agent-a"},
				{Name: "B", AgentRef: "agent-b"},
				{Name: "C", AgentRef: "agent-c", DependsOn: []string{"A", "B"}},
			},
			wantOrder: [][]string{{"A", "B"}, {"C"}},
		},
		{
			name: "single_step",
			steps: []Step{
				{Name: "only", AgentRef: "agent-only"},
			},
			wantOrder: [][]string{{"only"}},
		},
		{
			name:    "empty_steps",
			steps:   []Step{},
			wantErr: "", // empty should succeed with empty order
		},
		{
			name: "self_dependency",
			steps: []Step{
				{Name: "A", AgentRef: "agent-a", DependsOn: []string{"A"}},
			},
			wantErr: "depends on itself",
		},
		{
			name: "unknown_dependency",
			steps: []Step{
				{Name: "A", AgentRef: "agent-a", DependsOn: []string{"Z"}},
			},
			wantErr: "unknown step",
		},
		{
			name: "duplicate_step_name",
			steps: []Step{
				{Name: "A", AgentRef: "agent-a"},
				{Name: "A", AgentRef: "agent-b"},
			},
			wantErr: "duplicate step",
		},
		{
			name: "diamond_dependency",
			steps: []Step{
				{Name: "A", AgentRef: "agent-a"},
				{Name: "B", AgentRef: "agent-b", DependsOn: []string{"A"}},
				{Name: "C", AgentRef: "agent-c", DependsOn: []string{"A"}},
				{Name: "D", AgentRef: "agent-d", DependsOn: []string{"B", "C"}},
			},
			wantOrder: [][]string{{"A"}, {"B", "C"}, {"D"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dag, err := BuildDAG(tc.steps)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantOrder != nil {
				if len(dag.Order) != len(tc.wantOrder) {
					t.Fatalf("Order layers = %d, want %d: %v", len(dag.Order), len(tc.wantOrder), dag.Order)
				}
				for i, layer := range tc.wantOrder {
					if len(dag.Order[i]) != len(layer) {
						t.Errorf("layer %d: got %v, want %v", i, dag.Order[i], layer)
						continue
					}
					for j, step := range layer {
						if dag.Order[i][j] != step {
							t.Errorf("layer %d step %d: got %q, want %q", i, j, dag.Order[i][j], step)
						}
					}
				}
			}
		})
	}
}

// ---------- Executor tests ----------

func TestExecutor_LinearExecution(t *testing.T) {
	invoker := newMockInvoker()
	invoker.addResult("agent-a", "output-a", llm.TokenUsage{InputTokens: 10, OutputTokens: 5})
	invoker.addResult("agent-b", "output-b", llm.TokenUsage{InputTokens: 20, OutputTokens: 10})
	invoker.addResult("agent-c", "output-c", llm.TokenUsage{InputTokens: 30, OutputTokens: 15})

	dag, err := BuildDAG([]Step{
		{Name: "A", AgentRef: "agent-a"},
		{Name: "B", AgentRef: "agent-b", DependsOn: []string{"A"}},
		{Name: "C", AgentRef: "agent-c", DependsOn: []string{"B"}},
	})
	if err != nil {
		t.Fatalf("BuildDAG: %v", err)
	}

	exec := NewExecutor(invoker)
	result, err := exec.Execute(context.Background(), "test-pipeline", dag, "trigger-input")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("Status = %q, want %q", result.Status, "completed")
	}
	if result.Name != "test-pipeline" {
		t.Errorf("Name = %q, want %q", result.Name, "test-pipeline")
	}
	if len(result.Steps) != 3 {
		t.Errorf("Steps count = %d, want 3", len(result.Steps))
	}

	for _, name := range []string{"A", "B", "C"} {
		sr, ok := result.Steps[name]
		if !ok {
			t.Errorf("step %q not found in results", name)
			continue
		}
		if sr.Status != "completed" {
			t.Errorf("step %q status = %q, want %q", name, sr.Status, "completed")
		}
	}

	// Verify token aggregation
	if result.TotalTokens.InputTokens != 60 {
		t.Errorf("TotalTokens.InputTokens = %d, want 60", result.TotalTokens.InputTokens)
	}
	if result.TotalTokens.OutputTokens != 30 {
		t.Errorf("TotalTokens.OutputTokens = %d, want 30", result.TotalTokens.OutputTokens)
	}
}

func TestExecutor_ParallelExecution(t *testing.T) {
	invoker := newMockInvoker()
	invoker.addResult("agent-a", "out-a", llm.TokenUsage{InputTokens: 10, OutputTokens: 5})
	invoker.addResult("agent-b", "out-b", llm.TokenUsage{InputTokens: 10, OutputTokens: 5})
	invoker.addResult("agent-c", "out-c", llm.TokenUsage{InputTokens: 10, OutputTokens: 5})

	dag, err := BuildDAG([]Step{
		{Name: "A", AgentRef: "agent-a"},
		{Name: "B", AgentRef: "agent-b"},
		{Name: "C", AgentRef: "agent-c", DependsOn: []string{"A", "B"}},
	})
	if err != nil {
		t.Fatalf("BuildDAG: %v", err)
	}

	exec := NewExecutor(invoker)
	result, err := exec.Execute(context.Background(), "parallel-test", dag, "input")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("Status = %q, want %q", result.Status, "completed")
	}
	if len(result.Steps) != 3 {
		t.Errorf("Steps count = %d, want 3", len(result.Steps))
	}
	// A and B were in the same layer (parallel)
	// C depended on both
	if result.Steps["C"].Status != "completed" {
		t.Errorf("step C status = %q, want %q", result.Steps["C"].Status, "completed")
	}
}

func TestExecutor_StepFailurePropagation(t *testing.T) {
	invoker := newMockInvoker()
	invoker.addResult("agent-a", "out-a", llm.TokenUsage{})
	invoker.addError("agent-b", fmt.Errorf("agent-b crashed"))
	invoker.addResult("agent-c", "out-c", llm.TokenUsage{})

	dag, err := BuildDAG([]Step{
		{Name: "A", AgentRef: "agent-a"},
		{Name: "B", AgentRef: "agent-b", DependsOn: []string{"A"}},
		{Name: "C", AgentRef: "agent-c", DependsOn: []string{"B"}},
	})
	if err != nil {
		t.Fatalf("BuildDAG: %v", err)
	}

	exec := NewExecutor(invoker)
	result, err := exec.Execute(context.Background(), "fail-test", dag, "input")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Status != "failed" {
		t.Errorf("Status = %q, want %q", result.Status, "failed")
	}
	if result.Steps["B"].Status != "failed" {
		t.Errorf("step B status = %q, want %q", result.Steps["B"].Status, "failed")
	}
	if result.Steps["B"].Error == "" {
		t.Error("step B error should not be empty")
	}
	// Step C should not be in results because B failed first and C depends on B
	if _, ok := result.Steps["C"]; ok {
		// C might not have been run because B's layer failed
		if result.Steps["C"].Status == "completed" {
			t.Error("step C should not have completed after B failed")
		}
	}
}

func TestExecutor_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	invoker := newMockInvoker()
	invoker.delay = 500 * time.Millisecond
	invoker.addResult("agent-a", "out-a", llm.TokenUsage{})
	invoker.addResult("agent-b", "out-b", llm.TokenUsage{})

	dag, err := BuildDAG([]Step{
		{Name: "A", AgentRef: "agent-a"},
		{Name: "B", AgentRef: "agent-b", DependsOn: []string{"A"}},
	})
	if err != nil {
		t.Fatalf("BuildDAG: %v", err)
	}

	// Cancel shortly after start
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	exec := NewExecutor(invoker)
	result, err := exec.Execute(ctx, "cancel-test", dag, "input")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Pipeline should either be cancelled or have a failed step
	if result.Status != "cancelled" && result.Status != "failed" {
		t.Errorf("Status = %q, want cancelled or failed", result.Status)
	}
}

// ---------- StepResult and PipelineResult ----------

func TestStepResult_Fields(t *testing.T) {
	sr := StepResult{
		StepName: "test-step",
		AgentRef: "agent-x",
		Output:   "hello",
		Status:   "completed",
		Tokens:   llm.TokenUsage{InputTokens: 100, OutputTokens: 50},
	}
	if sr.StepName != "test-step" {
		t.Errorf("StepName = %q, want %q", sr.StepName, "test-step")
	}
	if sr.Tokens.InputTokens+sr.Tokens.OutputTokens != 150 {
		t.Errorf("total tokens = %d, want 150", sr.Tokens.InputTokens+sr.Tokens.OutputTokens)
	}
}

func TestPipelineResult_TokenAggregation(t *testing.T) {
	invoker := newMockInvoker()
	invoker.addResult("agent-a", "out-a", llm.TokenUsage{InputTokens: 100, OutputTokens: 50, CacheRead: 10, CacheWrite: 5})
	invoker.addResult("agent-b", "out-b", llm.TokenUsage{InputTokens: 200, OutputTokens: 100, CacheRead: 20, CacheWrite: 10})

	dag, err := BuildDAG([]Step{
		{Name: "A", AgentRef: "agent-a"},
		{Name: "B", AgentRef: "agent-b", DependsOn: []string{"A"}},
	})
	if err != nil {
		t.Fatalf("BuildDAG: %v", err)
	}

	exec := NewExecutor(invoker)
	result, err := exec.Execute(context.Background(), "token-test", dag, "input")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.TotalTokens.InputTokens != 300 {
		t.Errorf("TotalTokens.InputTokens = %d, want 300", result.TotalTokens.InputTokens)
	}
	if result.TotalTokens.OutputTokens != 150 {
		t.Errorf("TotalTokens.OutputTokens = %d, want 150", result.TotalTokens.OutputTokens)
	}
	if result.TotalTokens.CacheRead != 30 {
		t.Errorf("TotalTokens.CacheRead = %d, want 30", result.TotalTokens.CacheRead)
	}
	if result.TotalTokens.CacheWrite != 15 {
		t.Errorf("TotalTokens.CacheWrite = %d, want 15", result.TotalTokens.CacheWrite)
	}
}

func TestExecutor_SingleStep(t *testing.T) {
	invoker := newMockInvoker()
	invoker.addResult("single-agent", "the output", llm.TokenUsage{InputTokens: 50, OutputTokens: 25})

	dag, err := BuildDAG([]Step{
		{Name: "only", AgentRef: "single-agent"},
	})
	if err != nil {
		t.Fatalf("BuildDAG: %v", err)
	}

	exec := NewExecutor(invoker)
	result, err := exec.Execute(context.Background(), "single-step-pipeline", dag, "hello")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("Status = %q, want %q", result.Status, "completed")
	}
	if result.Steps["only"].Output != "the output" {
		t.Errorf("output = %q, want %q", result.Steps["only"].Output, "the output")
	}
	if result.TotalDuration <= 0 {
		t.Error("TotalDuration should be positive")
	}
}
