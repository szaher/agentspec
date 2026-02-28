package evaluation

import (
	"context"
	"fmt"
	"time"

	"github.com/szaher/designs/agentz/internal/runtime"
)

// CaseResult is the outcome of evaluating a single test case.
type CaseResult struct {
	Name      string  `json:"name"`
	Input     string  `json:"input"`
	Expected  string  `json:"expected"`
	Actual    string  `json:"actual"`
	Score     float64 `json:"score"`
	Threshold float64 `json:"threshold"`
	Scoring   string  `json:"scoring"`
	Passed    bool    `json:"passed"`
	Error     string  `json:"error,omitempty"`
	Duration  time.Duration
}

// RunResult is the outcome of an entire evaluation run.
type RunResult struct {
	AgentName    string       `json:"agent_name"`
	Cases        []CaseResult `json:"cases"`
	TotalCases   int          `json:"total_cases"`
	PassedCases  int          `json:"passed_cases"`
	FailedCases  int          `json:"failed_cases"`
	OverallScore float64      `json:"overall_score"`
	Duration     time.Duration
	Timestamp    time.Time `json:"timestamp"`
}

// AgentInvoker invokes an agent with input and returns output.
type AgentInvoker interface {
	Invoke(ctx context.Context, agentName, input string) (string, error)
}

// RunnerOptions configures the evaluation runner.
type RunnerOptions struct {
	AgentName string   // specific agent to evaluate
	Tags      []string // filter cases by tags
}

// Runner executes eval cases against agents.
type Runner struct {
	invoker AgentInvoker
}

// NewRunner creates an evaluation runner.
func NewRunner(invoker AgentInvoker) *Runner {
	return &Runner{invoker: invoker}
}

// Run executes all eval cases for an agent.
func (r *Runner) Run(ctx context.Context, agentName string, cases []runtime.EvalCaseDef, tags []string) (*RunResult, error) {
	startTime := time.Now()

	// Filter by tags if specified
	filtered := filterByTags(cases, tags)

	result := &RunResult{
		AgentName:  agentName,
		TotalCases: len(filtered),
		Timestamp:  startTime,
	}

	var totalScore float64

	for _, evalCase := range filtered {
		cr := r.runCase(ctx, agentName, evalCase)
		result.Cases = append(result.Cases, cr)

		totalScore += cr.Score
		if cr.Passed {
			result.PassedCases++
		} else {
			result.FailedCases++
		}
	}

	if result.TotalCases > 0 {
		result.OverallScore = totalScore / float64(result.TotalCases)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (r *Runner) runCase(ctx context.Context, agentName string, evalCase runtime.EvalCaseDef) CaseResult {
	start := time.Now()

	cr := CaseResult{
		Name:      evalCase.Name,
		Input:     evalCase.Input,
		Expected:  evalCase.Expected,
		Scoring:   evalCase.Scoring,
		Threshold: evalCase.Threshold,
	}

	if cr.Threshold == 0 {
		cr.Threshold = 0.8
	}

	if cr.Scoring == "" {
		cr.Scoring = "semantic"
	}

	// Invoke the agent
	output, err := r.invoker.Invoke(ctx, agentName, evalCase.Input)
	if err != nil {
		cr.Error = fmt.Sprintf("invocation failed: %v", err)
		cr.Duration = time.Since(start)
		return cr
	}
	cr.Actual = output

	// Score the output
	scorer, err := NewScorer(cr.Scoring)
	if err != nil {
		cr.Error = fmt.Sprintf("scorer error: %v", err)
		cr.Duration = time.Since(start)
		return cr
	}

	score, err := scorer.Score(output, evalCase.Expected)
	if err != nil {
		cr.Error = fmt.Sprintf("scoring error: %v", err)
		cr.Duration = time.Since(start)
		return cr
	}

	cr.Score = score
	cr.Passed = score >= cr.Threshold
	cr.Duration = time.Since(start)
	return cr
}

func filterByTags(cases []runtime.EvalCaseDef, tags []string) []runtime.EvalCaseDef {
	if len(tags) == 0 {
		return cases
	}

	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}

	var filtered []runtime.EvalCaseDef
	for _, c := range cases {
		if len(c.Tags) == 0 {
			continue // skip cases with no tags when filtering
		}
		for _, t := range c.Tags {
			if tagSet[t] {
				filtered = append(filtered, c)
				break
			}
		}
	}
	return filtered
}
