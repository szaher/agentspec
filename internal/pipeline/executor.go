package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/loop"
)

// StepResult holds the result of executing a single pipeline step.
type StepResult struct {
	StepName string         `json:"step_name"`
	AgentRef string         `json:"agent_ref"`
	Output   string         `json:"output"`
	Duration time.Duration  `json:"duration_ms"`
	Status   string         `json:"status"` // completed, failed, cancelled
	Error    string         `json:"error,omitempty"`
	Tokens   llm.TokenUsage `json:"tokens"`
}

// PipelineResult holds the result of executing a complete pipeline.
type PipelineResult struct {
	Name          string                `json:"pipeline"`
	Steps         map[string]StepResult `json:"steps"`
	Status        string                `json:"status"`
	TotalDuration time.Duration         `json:"total_duration_ms"`
	TotalTokens   llm.TokenUsage        `json:"tokens"`
}

// AgentInvoker is the interface for invoking an agent within a pipeline.
type AgentInvoker interface {
	Invoke(ctx context.Context, agentName string, input string) (*loop.Response, error)
}

// Executor runs a pipeline according to its DAG ordering.
type Executor struct {
	invoker AgentInvoker
}

// NewExecutor creates a new pipeline executor.
func NewExecutor(invoker AgentInvoker) *Executor {
	return &Executor{invoker: invoker}
}

// Execute runs the pipeline, executing steps layer by layer.
// Steps within the same layer run concurrently. If any step fails,
// remaining steps are cancelled (fail-fast).
func (e *Executor) Execute(ctx context.Context, name string, dag *DAG, triggerInput string) (*PipelineResult, error) {
	start := time.Now()
	result := &PipelineResult{
		Name:   name,
		Steps:  make(map[string]StepResult),
		Status: "completed",
	}

	// Track outputs from completed steps for input chaining
	stepOutputs := make(map[string]string)
	stepOutputs["trigger"] = triggerInput

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, layer := range dag.Order {
		if err := ctx.Err(); err != nil {
			result.Status = "cancelled"
			break
		}

		if len(layer) == 1 {
			// Single step, run directly
			stepName := layer[0]
			step := dag.Steps[stepName]
			sr := e.executeStep(ctx, step, stepOutputs)
			result.Steps[stepName] = sr
			result.TotalTokens = addTokens(result.TotalTokens, sr.Tokens)

			if sr.Status == "failed" {
				result.Status = "failed"
				cancel()
				break
			}
			stepOutputs[stepName] = sr.Output
		} else {
			// Multiple steps, run concurrently
			var mu sync.Mutex
			var wg sync.WaitGroup
			failed := false

			for _, stepName := range layer {
				wg.Add(1)
				go func(sn string) {
					defer wg.Done()
					step := dag.Steps[sn]
					sr := e.executeStep(ctx, step, stepOutputs)

					mu.Lock()
					result.Steps[sn] = sr
					result.TotalTokens = addTokens(result.TotalTokens, sr.Tokens)
					if sr.Status == "failed" {
						failed = true
						cancel()
					} else {
						stepOutputs[sn] = sr.Output
					}
					mu.Unlock()
				}(stepName)
			}
			wg.Wait()

			if failed {
				result.Status = "failed"
				break
			}
		}
	}

	result.TotalDuration = time.Since(start)
	return result, nil
}

func (e *Executor) executeStep(ctx context.Context, step *Step, outputs map[string]string) StepResult {
	start := time.Now()

	// Determine input: use step's input field, or output from dependency
	input := step.Input
	if input == "" && len(step.DependsOn) > 0 {
		// Use output from the first dependency
		input = outputs[step.DependsOn[0]]
	}
	if input == "" {
		input = outputs["trigger"]
	}

	resp, err := e.invoker.Invoke(ctx, step.AgentRef, input)
	if err != nil {
		return StepResult{
			StepName: step.Name,
			AgentRef: step.AgentRef,
			Duration: time.Since(start),
			Status:   "failed",
			Error:    fmt.Sprintf("invoke %s: %v", step.AgentRef, err),
		}
	}

	return StepResult{
		StepName: step.Name,
		AgentRef: step.AgentRef,
		Output:   resp.Output,
		Duration: time.Since(start),
		Status:   "completed",
		Tokens:   resp.Tokens,
	}
}

func addTokens(a, b llm.TokenUsage) llm.TokenUsage {
	return llm.TokenUsage{
		InputTokens:  a.InputTokens + b.InputTokens,
		OutputTokens: a.OutputTokens + b.OutputTokens,
		CacheRead:    a.CacheRead + b.CacheRead,
		CacheWrite:   a.CacheWrite + b.CacheWrite,
	}
}
