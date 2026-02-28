package integration_tests

import (
	"context"
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/evaluation"
	"github.com/szaher/designs/agentz/internal/runtime"
)

// testInvoker returns canned responses for eval testing.
type testInvoker struct {
	responses map[string]string
}

func (t *testInvoker) Invoke(_ context.Context, agentName, input string) (string, error) {
	if resp, ok := t.responses[input]; ok {
		return resp, nil
	}
	return "Default response", nil
}

func TestEvalRunner(t *testing.T) {
	invoker := &testInvoker{
		responses: map[string]string{
			"Hello, my name is Alice": "Hello Alice! Nice to meet you!",
			"hi":                      "Hello! How can I help you today?",
		},
	}

	runner := evaluation.NewRunner(invoker)

	cases := []runtime.EvalCaseDef{
		{
			Name:      "greeting_test",
			Input:     "Hello, my name is Alice",
			Expected:  "Hello Alice",
			Scoring:   "contains",
			Threshold: 0.7,
			Tags:      []string{"greeting"},
		},
		{
			Name:      "simple_hi",
			Input:     "hi",
			Expected:  "Hello! How can I help you today?",
			Scoring:   "exact",
			Threshold: 1.0,
		},
	}

	result, err := runner.Run(context.Background(), "test-agent", cases, nil)
	if err != nil {
		t.Fatalf("eval run failed: %v", err)
	}

	if result.TotalCases != 2 {
		t.Errorf("expected 2 total cases, got %d", result.TotalCases)
	}
	if result.AgentName != "test-agent" {
		t.Errorf("expected agent name 'test-agent', got %q", result.AgentName)
	}

	// greeting_test should pass (contains "Hello Alice")
	if !result.Cases[0].Passed {
		t.Errorf("greeting_test should pass, score: %.2f", result.Cases[0].Score)
	}

	// simple_hi should pass (exact match)
	if !result.Cases[1].Passed {
		t.Errorf("simple_hi should pass, score: %.2f", result.Cases[1].Score)
	}
}

func TestEvalRunnerWithTags(t *testing.T) {
	invoker := &testInvoker{
		responses: map[string]string{
			"test1": "response1",
			"test2": "response2",
		},
	}

	runner := evaluation.NewRunner(invoker)

	cases := []runtime.EvalCaseDef{
		{Name: "case1", Input: "test1", Expected: "response1", Scoring: "exact", Threshold: 1.0, Tags: []string{"smoke"}},
		{Name: "case2", Input: "test2", Expected: "response2", Scoring: "exact", Threshold: 1.0, Tags: []string{"regression"}},
	}

	// Filter by tag "smoke"
	result, err := runner.Run(context.Background(), "test-agent", cases, []string{"smoke"})
	if err != nil {
		t.Fatalf("eval run failed: %v", err)
	}

	if result.TotalCases != 1 {
		t.Errorf("expected 1 filtered case, got %d", result.TotalCases)
	}
	if result.Cases[0].Name != "case1" {
		t.Errorf("expected case1, got %q", result.Cases[0].Name)
	}
}

func TestEvalScoring(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		actual    string
		expected  string
		wantScore float64
		wantPass  bool
		threshold float64
	}{
		{
			name:      "exact match pass",
			method:    "exact",
			actual:    "Hello World",
			expected:  "hello world",
			wantScore: 1.0,
			wantPass:  true,
			threshold: 1.0,
		},
		{
			name:      "exact match fail",
			method:    "exact",
			actual:    "Hello World",
			expected:  "Goodbye",
			wantScore: 0.0,
			wantPass:  false,
			threshold: 1.0,
		},
		{
			name:      "contains match pass",
			method:    "contains",
			actual:    "Hello Alice, how are you?",
			expected:  "hello alice",
			wantScore: 1.0,
			wantPass:  true,
			threshold: 0.8,
		},
		{
			name:      "semantic similarity",
			method:    "semantic",
			actual:    "The weather is sunny and warm today",
			expected:  "Today the weather is warm and sunny",
			wantScore: 0.5, // will have some overlap
			wantPass:  true,
			threshold: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer, err := evaluation.NewScorer(tt.method)
			if err != nil {
				t.Fatalf("creating scorer: %v", err)
			}

			score, err := scorer.Score(tt.actual, tt.expected)
			if err != nil {
				t.Fatalf("scoring error: %v", err)
			}

			if tt.wantPass && score < tt.threshold {
				t.Errorf("expected pass (score >= %.2f), got %.2f", tt.threshold, score)
			}
			if !tt.wantPass && score >= tt.threshold {
				t.Errorf("expected fail (score < %.2f), got %.2f", tt.threshold, score)
			}
		})
	}
}

func TestEvalReportFormats(t *testing.T) {
	result := &evaluation.RunResult{
		AgentName:    "test-agent",
		TotalCases:   2,
		PassedCases:  1,
		FailedCases:  1,
		OverallScore: 0.75,
		Cases: []evaluation.CaseResult{
			{Name: "test1", Score: 1.0, Threshold: 0.8, Passed: true, Scoring: "exact"},
			{Name: "test2", Score: 0.5, Threshold: 0.8, Passed: false, Scoring: "semantic", Input: "hi", Expected: "hello", Actual: "hey"},
		},
	}

	t.Run("table format", func(t *testing.T) {
		output, err := evaluation.FormatReport(result, "table")
		if err != nil {
			t.Fatalf("format error: %v", err)
		}
		if !strings.Contains(output, "test-agent") {
			t.Error("table should contain agent name")
		}
		if !strings.Contains(output, "1/2 passed") {
			t.Error("table should contain pass count")
		}
	})

	t.Run("json format", func(t *testing.T) {
		output, err := evaluation.FormatReport(result, "json")
		if err != nil {
			t.Fatalf("format error: %v", err)
		}
		if !strings.Contains(output, `"agent_name"`) {
			t.Error("json should contain agent_name field")
		}
	})

	t.Run("markdown format", func(t *testing.T) {
		output, err := evaluation.FormatReport(result, "markdown")
		if err != nil {
			t.Fatalf("format error: %v", err)
		}
		if !strings.Contains(output, "# Evaluation Report") {
			t.Error("markdown should contain report header")
		}
		if !strings.Contains(output, "FAIL") {
			t.Error("markdown should show failed cases")
		}
	})
}

func TestEvalComparison(t *testing.T) {
	current := &evaluation.RunResult{
		OverallScore: 0.85,
		Cases: []evaluation.CaseResult{
			{Name: "test1", Score: 0.9},
			{Name: "test2", Score: 0.8},
		},
	}
	previous := &evaluation.RunResult{
		OverallScore: 0.80,
		Cases: []evaluation.CaseResult{
			{Name: "test1", Score: 0.85},
			{Name: "test2", Score: 0.75},
		},
	}

	output := evaluation.CompareResults(current, previous)

	if !strings.Contains(output, "Compared to previous run") {
		t.Error("comparison should contain header")
	}
	if !strings.Contains(output, "Improvements: 2") {
		t.Error("comparison should show improvements")
	}
}
