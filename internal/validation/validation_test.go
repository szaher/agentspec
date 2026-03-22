package validation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/szaher/agentspec/internal/expr"
)

// mockInvoker implements AgentInvoker for testing.
type mockInvoker struct {
	responses []string
	callCount int
	err       error
}

func (m *mockInvoker) Invoke(ctx context.Context, agentName, input string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.callCount >= len(m.responses) {
		return "", errors.New("no more mock responses")
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return resp, nil
}

func TestNewValidator(t *testing.T) {
	rules := []RuleDef{
		{Name: "rule1", Expression: "true", Severity: "error"},
		{Name: "rule2", Expression: "false", Severity: "warning"},
	}
	v := NewValidator(rules)
	if v == nil {
		t.Fatal("NewValidator returned nil")
	}
	if len(v.rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(v.rules))
	}
}

func TestValidator_Validate_AllPass(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "length_check",
			Expression: "len(output) > 5",
			Severity:   "error",
			Message:    "output too short",
		},
		{
			Name:       "not_empty_check",
			Expression: `output != ""`,
			Severity:   "warning",
			Message:    "should not be empty",
		},
	}
	v := NewValidator(rules)
	ctx := &expr.Context{
		Output: "hello world",
	}

	result := v.Validate(ctx)
	if !result.Passed {
		t.Errorf("expected validation to pass, got errors: %v", result.Errors)
	}
	if result.RulesChecked != 2 {
		t.Errorf("expected 2 rules checked, got %d", result.RulesChecked)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
	for _, rr := range result.Results {
		if !rr.Passed {
			t.Errorf("rule %s failed unexpectedly", rr.RuleName)
		}
	}
}

func TestValidator_Validate_ErrorFails(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "length_check",
			Expression: "len(output) > 100",
			Severity:   "error",
			Message:    "output too short",
		},
	}
	v := NewValidator(rules)
	ctx := &expr.Context{
		Output: "short",
	}

	result := v.Validate(ctx)
	if result.Passed {
		t.Error("expected validation to fail")
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0] != "output too short" {
		t.Errorf("unexpected error message: %s", result.Errors[0])
	}
}

func TestValidator_Validate_WarningDoesNotFail(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "optional_check",
			Expression: "len(output) > 100",
			Severity:   "warning",
			Message:    "should be longer",
		},
	}
	v := NewValidator(rules)
	ctx := &expr.Context{
		Output: "short text",
	}

	result := v.Validate(ctx)
	if !result.Passed {
		t.Error("expected validation to pass despite warning")
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0] != "should be longer" {
		t.Errorf("unexpected warning message: %s", result.Warnings[0])
	}
}

func TestValidator_Validate_ExpressionError(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "invalid_expr",
			Expression: "nonexistent_var > 5",
			Severity:   "error",
			Message:    "should not be reached",
		},
	}
	v := NewValidator(rules)
	ctx := &expr.Context{
		Output: "test",
	}

	result := v.Validate(ctx)
	if result.Passed {
		t.Error("expected validation to fail due to expression error")
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	rr := result.Results[0]
	if rr.Passed {
		t.Error("expected rule result to be failed")
	}
	if rr.Error == "" {
		t.Error("expected Error field to be set")
	}
	if !strings.Contains(rr.Message, "expression evaluation failed") {
		t.Errorf("unexpected message: %s", rr.Message)
	}
}

func TestValidator_Validate_NonBooleanExpression(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "string_expr",
			Expression: "output",
			Severity:   "error",
			Message:    "should not be reached",
		},
	}
	v := NewValidator(rules)
	ctx := &expr.Context{
		Output: "test string",
	}

	result := v.Validate(ctx)
	if result.Passed {
		t.Error("expected validation to fail due to non-boolean result")
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	rr := result.Results[0]
	if rr.Passed {
		t.Error("expected rule result to be failed")
	}
	if !strings.Contains(rr.Error, "expected bool") {
		t.Errorf("unexpected error: %s", rr.Error)
	}
	if !strings.Contains(rr.Message, "did not return a boolean") {
		t.Errorf("unexpected message: %s", rr.Message)
	}
}

func TestValidator_Validate_EmptyMessage(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "no_message",
			Expression: "false",
			Severity:   "error",
			Message:    "", // Empty message
		},
	}
	v := NewValidator(rules)
	ctx := &expr.Context{
		Output: "test",
	}

	result := v.Validate(ctx)
	if result.Passed {
		t.Error("expected validation to fail")
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	rr := result.Results[0]
	if !strings.Contains(rr.Message, "Rule \"no_message\" failed") {
		t.Errorf("expected default message, got: %s", rr.Message)
	}
}

func TestValidator_FailedErrorRules(t *testing.T) {
	rules := []RuleDef{
		{Name: "error1", Expression: "false", Severity: "error", Message: "error 1"},
		{Name: "warning1", Expression: "false", Severity: "warning", Message: "warning 1"},
		{Name: "error2", Expression: "false", Severity: "error", Message: "error 2"},
		{Name: "passing", Expression: "true", Severity: "error", Message: "should pass"},
	}
	v := NewValidator(rules)
	ctx := &expr.Context{
		Output: "test",
	}

	result := v.Validate(ctx)
	failedErrors := v.FailedErrorRules(result)

	if len(failedErrors) != 2 {
		t.Errorf("expected 2 failed error rules, got %d", len(failedErrors))
	}
	names := make(map[string]bool)
	for _, r := range failedErrors {
		names[r.Name] = true
	}
	if !names["error1"] || !names["error2"] {
		t.Errorf("expected error1 and error2, got: %v", names)
	}
	if names["warning1"] || names["passing"] {
		t.Errorf("should not include warning1 or passing, got: %v", names)
	}
}

func TestExtractValidationRules_Valid(t *testing.T) {
	attrs := map[string]interface{}{
		"validation_rules": []interface{}{
			map[string]interface{}{
				"name":        "rule1",
				"expression":  "len(output) > 10",
				"severity":    "error",
				"message":     "too short",
				"max_retries": 5,
			},
			map[string]interface{}{
				"name":       "rule2",
				"expression": `output != ""`,
				"severity":   "warning",
				"message":    "missing test",
			},
		},
	}

	rules := ExtractValidationRules(attrs)
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	r1 := rules[0]
	if r1.Name != "rule1" {
		t.Errorf("expected name 'rule1', got %s", r1.Name)
	}
	if r1.Expression != "len(output) > 10" {
		t.Errorf("unexpected expression: %s", r1.Expression)
	}
	if r1.Severity != "error" {
		t.Errorf("expected severity 'error', got %s", r1.Severity)
	}
	if r1.Message != "too short" {
		t.Errorf("unexpected message: %s", r1.Message)
	}
	if r1.MaxRetries != 5 {
		t.Errorf("expected max_retries 5, got %d", r1.MaxRetries)
	}

	r2 := rules[1]
	if r2.Name != "rule2" {
		t.Errorf("expected name 'rule2', got %s", r2.Name)
	}
	if r2.Severity != "warning" {
		t.Errorf("expected severity 'warning', got %s", r2.Severity)
	}
	// Default max_retries should be 3
	if r2.MaxRetries != 3 {
		t.Errorf("expected default max_retries 3, got %d", r2.MaxRetries)
	}
}

func TestExtractValidationRules_MaxRetriesFloat(t *testing.T) {
	attrs := map[string]interface{}{
		"validation_rules": []interface{}{
			map[string]interface{}{
				"name":        "rule1",
				"expression":  "true",
				"max_retries": 7.0, // float64
			},
		},
	}

	rules := ExtractValidationRules(attrs)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].MaxRetries != 7 {
		t.Errorf("expected max_retries 7, got %d", rules[0].MaxRetries)
	}
}

func TestExtractValidationRules_NoRules(t *testing.T) {
	attrs := map[string]interface{}{}
	rules := ExtractValidationRules(attrs)
	if rules != nil {
		t.Errorf("expected nil, got %d rules", len(rules))
	}
}

func TestExtractValidationRules_WrongType(t *testing.T) {
	attrs := map[string]interface{}{
		"validation_rules": "not an array",
	}
	rules := ExtractValidationRules(attrs)
	if rules != nil {
		t.Errorf("expected nil, got %d rules", len(rules))
	}
}

func TestExtractValidationRules_InvalidItems(t *testing.T) {
	attrs := map[string]interface{}{
		"validation_rules": []interface{}{
			"string instead of map",
			123,
			map[string]interface{}{
				"name":       "valid",
				"expression": "true",
			},
		},
	}
	rules := ExtractValidationRules(attrs)
	// Should only extract the valid one
	if len(rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Name != "valid" {
		t.Errorf("expected name 'valid', got %s", rules[0].Name)
	}
}

func TestRetryLoop_FirstAttemptPasses(t *testing.T) {
	rules := []RuleDef{
		{Name: "check", Expression: "len(output) > 5", Severity: "error", Message: "too short"},
	}
	v := NewValidator(rules)
	invoker := &mockInvoker{responses: []string{"should not be called"}}

	ctx := context.Background()
	output, result, err := RetryLoop(ctx, invoker, v, "test-agent", "input", "valid output", RetryConfig{MaxRetries: 3})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if output != "valid output" {
		t.Errorf("expected original output, got: %s", output)
	}
	if !result.Passed {
		t.Error("expected validation to pass")
	}
	if invoker.callCount != 0 {
		t.Errorf("expected no retry invocations, got %d", invoker.callCount)
	}
}

func TestRetryLoop_WarningDoesNotRetry(t *testing.T) {
	rules := []RuleDef{
		{Name: "warning", Expression: "false", Severity: "warning", Message: "just a warning"},
	}
	v := NewValidator(rules)
	invoker := &mockInvoker{responses: []string{"should not be called"}}

	ctx := context.Background()
	output, result, err := RetryLoop(ctx, invoker, v, "test-agent", "input", "output", RetryConfig{MaxRetries: 3})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if output != "output" {
		t.Errorf("expected original output, got: %s", output)
	}
	if !result.Passed {
		t.Error("expected validation to pass (warnings only)")
	}
	if invoker.callCount != 0 {
		t.Errorf("expected no retry invocations, got %d", invoker.callCount)
	}
}

func TestRetryLoop_RetrySucceeds(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "length",
			Expression: "len(output) >= 10",
			Severity:   "error",
			Message:    "too short",
			MaxRetries: 2,
		},
	}
	v := NewValidator(rules)
	invoker := &mockInvoker{
		responses: []string{"valid output that is long enough"},
	}

	ctx := context.Background()
	output, result, err := RetryLoop(ctx, invoker, v, "test-agent", "input", "short", RetryConfig{MaxRetries: 3})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if output != "valid output that is long enough" {
		t.Errorf("unexpected output: %s", output)
	}
	if !result.Passed {
		t.Error("expected validation to pass after retry")
	}
	if invoker.callCount != 1 {
		t.Errorf("expected 1 retry invocation, got %d", invoker.callCount)
	}
}

func TestRetryLoop_MaxRetriesExhausted(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "length",
			Expression: "len(output) >= 100",
			Severity:   "error",
			Message:    "too short",
			MaxRetries: 2,
		},
	}
	v := NewValidator(rules)
	invoker := &mockInvoker{
		responses: []string{"short1", "short2"},
	}

	ctx := context.Background()
	output, result, err := RetryLoop(ctx, invoker, v, "test-agent", "input", "short0", RetryConfig{MaxRetries: 3})

	if err == nil {
		t.Error("expected error when retries exhausted")
	}
	if !strings.Contains(err.Error(), "validation failed after retries") {
		t.Errorf("unexpected error: %v", err)
	}
	// Should return last attempted output
	if output != "short2" {
		t.Errorf("expected last output 'short2', got: %s", output)
	}
	if result.Passed {
		t.Error("expected validation to fail")
	}
	if invoker.callCount != 2 {
		t.Errorf("expected 2 retry invocations, got %d", invoker.callCount)
	}
}

func TestRetryLoop_UsesDefaultMaxRetries(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "check",
			Expression: "false",
			Severity:   "error",
			Message:    "always fails",
			MaxRetries: 0, // Use default
		},
	}
	v := NewValidator(rules)
	invoker := &mockInvoker{
		responses: []string{"fail1", "fail2", "fail3"},
	}

	ctx := context.Background()
	_, _, err := RetryLoop(ctx, invoker, v, "test-agent", "input", "fail0", RetryConfig{MaxRetries: 2})

	if err == nil {
		t.Error("expected error")
	}
	// Should use config default of 2
	if invoker.callCount != 2 {
		t.Errorf("expected 2 retries, got %d", invoker.callCount)
	}
}

func TestRetryLoop_FallbackMaxRetries(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "check",
			Expression: "false",
			Severity:   "error",
			Message:    "always fails",
			MaxRetries: 0, // Use default
		},
	}
	v := NewValidator(rules)
	invoker := &mockInvoker{
		responses: []string{"fail1", "fail2", "fail3"},
	}

	ctx := context.Background()
	_, _, err := RetryLoop(ctx, invoker, v, "test-agent", "input", "fail0", RetryConfig{MaxRetries: 0})

	if err == nil {
		t.Error("expected error")
	}
	// Should use fallback of 3
	if invoker.callCount != 3 {
		t.Errorf("expected 3 retries (fallback), got %d", invoker.callCount)
	}
}

func TestRetryLoop_InvokerError(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "check",
			Expression: "false",
			Severity:   "error",
			Message:    "fails",
			MaxRetries: 3,
		},
	}
	v := NewValidator(rules)
	invoker := &mockInvoker{
		err: errors.New("invocation failed"),
	}

	ctx := context.Background()
	output, result, err := RetryLoop(ctx, invoker, v, "test-agent", "input", "initial", RetryConfig{MaxRetries: 3})

	if err == nil {
		t.Error("expected error from failed invocation")
	}
	if !strings.Contains(err.Error(), "retry invocation failed") {
		t.Errorf("unexpected error: %v", err)
	}
	if output != "initial" {
		t.Errorf("expected initial output, got: %s", output)
	}
	if result.Passed {
		t.Error("expected validation to fail")
	}
}

func TestRetryLoop_MultipleRules(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "length",
			Expression: "len(output) >= 10",
			Severity:   "error",
			Message:    "too short",
			MaxRetries: 2,
		},
		{
			Name:       "not_empty",
			Expression: `output != ""`,
			Severity:   "error",
			Message:    "must not be empty",
			MaxRetries: 2,
		},
	}
	v := NewValidator(rules)
	invoker := &mockInvoker{
		responses: []string{"hello world from agent"},
	}

	ctx := context.Background()
	output, result, err := RetryLoop(ctx, invoker, v, "test-agent", "input", "short", RetryConfig{MaxRetries: 3})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if output != "hello world from agent" {
		t.Errorf("unexpected output: %s", output)
	}
	if !result.Passed {
		t.Error("expected validation to pass")
	}
	if invoker.callCount != 1 {
		t.Errorf("expected 1 retry invocation, got %d", invoker.callCount)
	}
}

func TestBuildRetryPrompt(t *testing.T) {
	originalInput := "original request"
	previousOutput := "bad output"
	failures := []string{
		"- rule1: message 1",
		"- rule2: message 2",
	}

	prompt := buildRetryPrompt(originalInput, previousOutput, failures)

	if !strings.Contains(prompt, "previous response did not pass validation") {
		t.Error("expected validation failure message")
	}
	if !strings.Contains(prompt, "original request") {
		t.Error("expected original input in prompt")
	}
	if !strings.Contains(prompt, "bad output") {
		t.Error("expected previous output in prompt")
	}
	if !strings.Contains(prompt, "- rule1: message 1") {
		t.Error("expected failure 1 in prompt")
	}
	if !strings.Contains(prompt, "- rule2: message 2") {
		t.Error("expected failure 2 in prompt")
	}
	if !strings.Contains(prompt, "Please provide a corrected response") {
		t.Error("expected correction request in prompt")
	}
}

func TestValidator_Validate_WithInputAndSession(t *testing.T) {
	rules := []RuleDef{
		{
			Name:       "input_check",
			Expression: `input != ""`,
			Severity:   "error",
			Message:    "input must not be empty",
		},
		{
			Name:       "session_check",
			Expression: "session.user == 'admin'",
			Severity:   "error",
			Message:    "must be admin",
		},
	}
	v := NewValidator(rules)
	ctx := &expr.Context{
		Input:  "test input",
		Output: "any output",
		Session: map[string]interface{}{
			"user": "admin",
		},
	}

	result := v.Validate(ctx)
	if !result.Passed {
		t.Errorf("expected validation to pass, got errors: %v", result.Errors)
	}
}

func TestValidator_Validate_MultipleErrorsAndWarnings(t *testing.T) {
	rules := []RuleDef{
		{Name: "error1", Expression: "false", Severity: "error", Message: "error 1"},
		{Name: "warning1", Expression: "false", Severity: "warning", Message: "warning 1"},
		{Name: "error2", Expression: "false", Severity: "error", Message: "error 2"},
		{Name: "warning2", Expression: "false", Severity: "warning", Message: "warning 2"},
		{Name: "passing", Expression: "true", Severity: "error", Message: "should not appear"},
	}
	v := NewValidator(rules)
	ctx := &expr.Context{Output: "test"}

	result := v.Validate(ctx)
	if result.Passed {
		t.Error("expected validation to fail")
	}
	if result.RulesChecked != 5 {
		t.Errorf("expected 5 rules checked, got %d", result.RulesChecked)
	}
	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors))
	}
	if len(result.Warnings) != 2 {
		t.Errorf("expected 2 warnings, got %d", len(result.Warnings))
	}
}

func TestExtractValidationRules_DefaultValues(t *testing.T) {
	attrs := map[string]interface{}{
		"validation_rules": []interface{}{
			map[string]interface{}{
				"name":       "minimal",
				"expression": "true",
				// All other fields use defaults
			},
		},
	}

	rules := ExtractValidationRules(attrs)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	r := rules[0]
	if r.Name != "minimal" {
		t.Errorf("expected name 'minimal', got %s", r.Name)
	}
	if r.Severity != "error" {
		t.Errorf("expected default severity 'error', got %s", r.Severity)
	}
	if r.MaxRetries != 3 {
		t.Errorf("expected default max_retries 3, got %d", r.MaxRetries)
	}
	if r.Message != "" {
		t.Errorf("expected empty message, got %s", r.Message)
	}
}
