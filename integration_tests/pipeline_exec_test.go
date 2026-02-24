package integration_tests

import (
	"context"
	"testing"

	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/loop"
	"github.com/szaher/designs/agentz/internal/pipeline"
)

func TestPipelineDAGBuild(t *testing.T) {
	steps := []pipeline.Step{
		{Name: "fetch", AgentRef: "data-analyst", Input: "raw data"},
		{Name: "analyze", AgentRef: "data-analyst", DependsOn: []string{"fetch"}},
		{Name: "report", AgentRef: "report-writer", DependsOn: []string{"analyze"}},
	}

	dag, err := pipeline.BuildDAG(steps)
	if err != nil {
		t.Fatalf("build DAG: %v", err)
	}

	if len(dag.Order) != 3 {
		t.Fatalf("expected 3 layers, got %d", len(dag.Order))
	}

	if dag.Order[0][0] != "fetch" {
		t.Errorf("expected first layer to be fetch, got %v", dag.Order[0])
	}
	if dag.Order[1][0] != "analyze" {
		t.Errorf("expected second layer to be analyze, got %v", dag.Order[1])
	}
	if dag.Order[2][0] != "report" {
		t.Errorf("expected third layer to be report, got %v", dag.Order[2])
	}
}

func TestPipelineDAGParallel(t *testing.T) {
	steps := []pipeline.Step{
		{Name: "a", AgentRef: "agent-a"},
		{Name: "b", AgentRef: "agent-b"},
		{Name: "c", AgentRef: "agent-c", DependsOn: []string{"a", "b"}},
	}

	dag, err := pipeline.BuildDAG(steps)
	if err != nil {
		t.Fatalf("build DAG: %v", err)
	}

	if len(dag.Order) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(dag.Order))
	}

	// First layer should have a and b (parallel)
	if len(dag.Order[0]) != 2 {
		t.Errorf("expected 2 steps in first layer, got %d", len(dag.Order[0]))
	}
	// Second layer should have c
	if len(dag.Order[1]) != 1 || dag.Order[1][0] != "c" {
		t.Errorf("expected [c] in second layer, got %v", dag.Order[1])
	}
}

func TestPipelineDAGCycleDetection(t *testing.T) {
	steps := []pipeline.Step{
		{Name: "a", AgentRef: "agent-a", DependsOn: []string{"c"}},
		{Name: "b", AgentRef: "agent-b", DependsOn: []string{"a"}},
		{Name: "c", AgentRef: "agent-c", DependsOn: []string{"b"}},
	}

	_, err := pipeline.BuildDAG(steps)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestPipelineDAGDuplicateStep(t *testing.T) {
	steps := []pipeline.Step{
		{Name: "a", AgentRef: "agent-a"},
		{Name: "a", AgentRef: "agent-b"},
	}

	_, err := pipeline.BuildDAG(steps)
	if err == nil {
		t.Fatal("expected duplicate step error")
	}
}

func TestPipelineDAGUnknownDependency(t *testing.T) {
	steps := []pipeline.Step{
		{Name: "a", AgentRef: "agent-a", DependsOn: []string{"nonexistent"}},
	}

	_, err := pipeline.BuildDAG(steps)
	if err == nil {
		t.Fatal("expected unknown dependency error")
	}
}

type mockInvoker struct {
	responses map[string]string
}

func (m *mockInvoker) Invoke(_ context.Context, agentName string, input string) (*loop.Response, error) {
	output := m.responses[agentName]
	if output == "" {
		output = "response from " + agentName
	}
	return &loop.Response{
		Output: output,
		Tokens: llm.TokenUsage{InputTokens: 50, OutputTokens: 50},
		Turns:  1,
	}, nil
}

func TestPipelineExecution(t *testing.T) {
	steps := []pipeline.Step{
		{Name: "fetch", AgentRef: "fetcher", Input: "get data"},
		{Name: "analyze", AgentRef: "analyzer", DependsOn: []string{"fetch"}},
		{Name: "report", AgentRef: "reporter", DependsOn: []string{"analyze"}},
	}

	dag, err := pipeline.BuildDAG(steps)
	if err != nil {
		t.Fatalf("build DAG: %v", err)
	}

	invoker := &mockInvoker{
		responses: map[string]string{
			"fetcher":  "fetched data",
			"analyzer": "analysis results",
			"reporter": "final report",
		},
	}

	executor := pipeline.NewExecutor(invoker)
	result, err := executor.Execute(context.Background(), "test-pipeline", dag, "trigger input")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("expected status completed, got %s", result.Status)
	}

	if len(result.Steps) != 3 {
		t.Errorf("expected 3 step results, got %d", len(result.Steps))
	}

	if result.Steps["report"].Output != "final report" {
		t.Errorf("expected final report output, got %s", result.Steps["report"].Output)
	}
}

func TestPipelineParallelExecution(t *testing.T) {
	steps := []pipeline.Step{
		{Name: "a", AgentRef: "agent-a"},
		{Name: "b", AgentRef: "agent-b"},
		{Name: "merge", AgentRef: "agent-merge", DependsOn: []string{"a", "b"}},
	}

	dag, err := pipeline.BuildDAG(steps)
	if err != nil {
		t.Fatalf("build DAG: %v", err)
	}

	invoker := &mockInvoker{responses: map[string]string{}}
	executor := pipeline.NewExecutor(invoker)
	result, err := executor.Execute(context.Background(), "parallel-pipeline", dag, "input")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("expected completed, got %s", result.Status)
	}

	if len(result.Steps) != 3 {
		t.Errorf("expected 3 results, got %d", len(result.Steps))
	}
}
