package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/szaher/designs/agentz/internal/expr"
)

// AgentInvoker is the interface for invoking an agent. This allows
// the validation retry logic to re-invoke the agent without depending
// on the specific runtime implementation.
type AgentInvoker interface {
	Invoke(ctx context.Context, agentName, input string) (string, error)
}

// RetryConfig configures the validation retry behavior.
type RetryConfig struct {
	// MaxRetries is the default max retries if not specified per rule.
	MaxRetries int
}

// RetryLoop runs the validation-retry cycle. It validates the output,
// and if any error-severity rules fail, re-invokes the agent with
// feedback about what failed. It continues until all rules pass
// or max retries are exhausted.
func RetryLoop(
	ctx context.Context,
	invoker AgentInvoker,
	validator *Validator,
	agentName string,
	originalInput string,
	initialOutput string,
	retryCfg RetryConfig,
) (string, *ValidationResult, error) {
	output := initialOutput
	retryCount := make(map[string]int) // track retries per rule

	for attempt := 0; ; attempt++ {
		// Build expression context with current output
		exprCtx := &expr.Context{
			Input:  originalInput,
			Output: output,
		}

		result := validator.Validate(exprCtx)
		if result.Passed {
			return output, result, nil
		}

		// Check which error-severity rules failed
		failedRules := validator.FailedErrorRules(result)
		if len(failedRules) == 0 {
			// Only warnings failed, that's OK
			return output, result, nil
		}

		// Check if any rule has retries remaining
		hasRetries := false
		var feedbackParts []string
		for _, rule := range failedRules {
			maxRetries := rule.MaxRetries
			if maxRetries <= 0 {
				maxRetries = retryCfg.MaxRetries
			}
			if maxRetries <= 0 {
				maxRetries = 3
			}

			count := retryCount[rule.Name]
			if count < maxRetries {
				hasRetries = true
				retryCount[rule.Name] = count + 1
				feedbackParts = append(feedbackParts,
					fmt.Sprintf("- %s: %s", rule.Name, rule.Message))
			}
		}

		if !hasRetries {
			return output, result, fmt.Errorf("validation failed after retries: %s",
				strings.Join(result.Errors, "; "))
		}

		// Build retry prompt with validation feedback
		retryInput := buildRetryPrompt(originalInput, output, feedbackParts)

		newOutput, err := invoker.Invoke(ctx, agentName, retryInput)
		if err != nil {
			return output, result, fmt.Errorf("retry invocation failed: %w", err)
		}
		output = newOutput
	}
}

func buildRetryPrompt(originalInput, previousOutput string, failures []string) string {
	var b strings.Builder
	b.WriteString("The previous response did not pass validation checks. ")
	b.WriteString("Please try again with the following feedback:\n\n")
	b.WriteString("Original request: ")
	b.WriteString(originalInput)
	b.WriteString("\n\nValidation failures:\n")
	for _, f := range failures {
		b.WriteString(f)
		b.WriteString("\n")
	}
	b.WriteString("\nPrevious response (rejected):\n")
	b.WriteString(previousOutput)
	b.WriteString("\n\nPlease provide a corrected response.")
	return b.String()
}
