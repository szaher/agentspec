package integration_tests

import (
	"testing"

	"github.com/szaher/designs/agentz/internal/expr"
	"github.com/szaher/designs/agentz/internal/validation"
)

func TestValidationRuleExecution(t *testing.T) {
	rules := []validation.RuleDef{
		{
			Name:       "output_not_empty",
			Expression: `output != ""`,
			Severity:   "error",
			Message:    "Output must not be empty",
			MaxRetries: 2,
		},
		{
			Name:       "tone_check",
			Expression: `output != ""`,
			Severity:   "warning",
			Message:    "Output should maintain professional tone",
		},
	}

	v := validation.NewValidator(rules)

	t.Run("all rules pass", func(t *testing.T) {
		ctx := &expr.Context{
			Input:  "Hello",
			Output: "Hi there! How can I help you?",
		}
		result := v.Validate(ctx)
		if !result.Passed {
			t.Errorf("expected validation to pass, got errors: %v", result.Errors)
		}
		if result.RulesChecked != 2 {
			t.Errorf("expected 2 rules checked, got %d", result.RulesChecked)
		}
	})

	t.Run("error rule fails", func(t *testing.T) {
		ctx := &expr.Context{
			Input:  "Hello",
			Output: "",
		}
		result := v.Validate(ctx)
		if result.Passed {
			t.Error("expected validation to fail for empty output")
		}
		if len(result.Errors) == 0 {
			t.Error("expected error messages")
		}
	})

	t.Run("warning rule fails gracefully", func(t *testing.T) {
		warningRules := []validation.RuleDef{
			{
				Name:       "has_greeting",
				Expression: `output != ""`,
				Severity:   "warning",
				Message:    "Should include greeting",
			},
		}
		wv := validation.NewValidator(warningRules)
		ctx := &expr.Context{
			Output: "",
		}
		result := wv.Validate(ctx)
		// Warnings don't cause overall failure
		if len(result.Warnings) == 0 {
			t.Error("expected warning messages")
		}
	})
}

func TestValidationRuleExtraction(t *testing.T) {
	attrs := map[string]interface{}{
		"validation_rules": []interface{}{
			map[string]interface{}{
				"name":        "rule1",
				"expression":  `output != ""`,
				"severity":    "error",
				"message":     "Output must not be empty",
				"max_retries": 3,
			},
			map[string]interface{}{
				"name":       "rule2",
				"expression": `output != ""`,
				"severity":   "warning",
				"message":    "Should be non-empty",
			},
		},
	}

	rules := validation.ExtractValidationRules(attrs)
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	if rules[0].Name != "rule1" {
		t.Errorf("expected rule name 'rule1', got %q", rules[0].Name)
	}
	if rules[0].Severity != "error" {
		t.Errorf("expected severity 'error', got %q", rules[0].Severity)
	}
	if rules[0].MaxRetries != 3 {
		t.Errorf("expected max_retries 3, got %d", rules[0].MaxRetries)
	}
	if rules[1].Severity != "warning" {
		t.Errorf("expected severity 'warning', got %q", rules[1].Severity)
	}
}

func TestExpressionEvalForValidation(t *testing.T) {
	tests := []struct {
		name   string
		expr   string
		ctx    *expr.Context
		expect bool
	}{
		{
			name:   "output not empty - passes",
			expr:   `output != ""`,
			ctx:    &expr.Context{Output: "Hello"},
			expect: true,
		},
		{
			name:   "output not empty - fails",
			expr:   `output != ""`,
			ctx:    &expr.Context{Output: ""},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expr.EvalString(tt.expr, tt.ctx)
			if err != nil {
				t.Fatalf("eval error: %v", err)
			}
			b, ok := result.(bool)
			if !ok {
				t.Fatalf("expected bool result, got %T", result)
			}
			if b != tt.expect {
				t.Errorf("expected %v, got %v", tt.expect, b)
			}
		})
	}
}
