package integration_tests

import (
	"testing"

	"github.com/szaher/agentspec/internal/loop"
)

// TestGuardrailBlockMode verifies that block mode stops output containing forbidden keywords.
func TestGuardrailBlockMode(t *testing.T) {
	configs := []loop.GuardrailConfig{
		{
			Name:        "sensitive-data",
			Mode:        "block",
			Keywords:    []string{"password", "secret"},
			FallbackMsg: "I cannot share sensitive information.",
		},
	}

	filter := loop.NewGuardrailFilter(configs)

	// Test output containing forbidden keyword
	output := "Your password is 12345"
	filtered, violations := filter.Check(output)

	if filtered != "I cannot share sensitive information." {
		t.Errorf("expected fallback message, got: %q", filtered)
	}

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.Guardrail != "sensitive-data" {
		t.Errorf("expected guardrail 'sensitive-data', got %q", v.Guardrail)
	}
	if v.Mode != "block" {
		t.Errorf("expected mode 'block', got %q", v.Mode)
	}
	if v.Type != "keyword" {
		t.Errorf("expected type 'keyword', got %q", v.Type)
	}
	if v.Match != "password" {
		t.Errorf("expected match 'password', got %q", v.Match)
	}
}

// TestGuardrailWarnMode verifies that warn mode passes through output but reports violations.
func TestGuardrailWarnMode(t *testing.T) {
	configs := []loop.GuardrailConfig{
		{
			Name:     "warn-pii",
			Mode:     "warn",
			Keywords: []string{"email", "phone"},
		},
	}

	filter := loop.NewGuardrailFilter(configs)

	output := "Contact me at email@example.com"
	filtered, violations := filter.Check(output)

	// Output should pass through unchanged
	if filtered != output {
		t.Errorf("expected output to pass through in warn mode, got: %q", filtered)
	}

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.Guardrail != "warn-pii" {
		t.Errorf("expected guardrail 'warn-pii', got %q", v.Guardrail)
	}
	if v.Mode != "warn" {
		t.Errorf("expected mode 'warn', got %q", v.Mode)
	}
	if v.Match != "email" {
		t.Errorf("expected match 'email', got %q", v.Match)
	}
}

// TestGuardrailRegexPattern verifies that regex patterns are matched correctly.
func TestGuardrailRegexPattern(t *testing.T) {
	configs := []loop.GuardrailConfig{
		{
			Name:        "ssn-blocker",
			Mode:        "block",
			Patterns:    []string{`\b\d{3}-\d{2}-\d{4}\b`}, // SSN pattern
			FallbackMsg: "SSN detected and blocked.",
		},
	}

	filter := loop.NewGuardrailFilter(configs)

	// Test with SSN
	output := "My SSN is 123-45-6789 and I live in CA."
	filtered, violations := filter.Check(output)

	if filtered != "SSN detected and blocked." {
		t.Errorf("expected fallback message for SSN, got: %q", filtered)
	}

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.Type != "pattern" {
		t.Errorf("expected type 'pattern', got %q", v.Type)
	}
	if v.Match != "123-45-6789" {
		t.Errorf("expected match '123-45-6789', got %q", v.Match)
	}
}

// TestGuardrailNoViolations verifies that clean output passes through.
func TestGuardrailNoViolations(t *testing.T) {
	configs := []loop.GuardrailConfig{
		{
			Name:     "test-filter",
			Mode:     "block",
			Keywords: []string{"forbidden"},
		},
	}

	filter := loop.NewGuardrailFilter(configs)

	output := "This is a clean message."
	filtered, violations := filter.Check(output)

	if filtered != output {
		t.Errorf("expected output unchanged, got: %q", filtered)
	}

	if len(violations) != 0 {
		t.Errorf("expected no violations, got %d", len(violations))
	}
}

// TestGuardrailCaseInsensitive verifies that keyword matching is case-insensitive.
func TestGuardrailCaseInsensitive(t *testing.T) {
	configs := []loop.GuardrailConfig{
		{
			Name:        "case-test",
			Mode:        "block",
			Keywords:    []string{"password"},
			FallbackMsg: "Blocked.",
		},
	}

	filter := loop.NewGuardrailFilter(configs)

	// Test with different cases
	testCases := []string{
		"Your PASSWORD is here",
		"Your Password is here",
		"Your password is here",
		"Your pAsSwOrD is here",
	}

	for _, output := range testCases {
		filtered, violations := filter.Check(output)
		if filtered != "Blocked." {
			t.Errorf("case-insensitive match failed for %q, got: %q", output, filtered)
		}
		if len(violations) != 1 {
			t.Errorf("expected 1 violation for %q, got %d", output, len(violations))
		}
	}
}

// TestGuardrailMultiplePatterns verifies handling of multiple patterns.
func TestGuardrailMultiplePatterns(t *testing.T) {
	configs := []loop.GuardrailConfig{
		{
			Name:     "multi-pattern",
			Mode:     "warn",
			Patterns: []string{`\b\d{3}-\d{2}-\d{4}\b`, `\b[A-Z]{2}\d{6}\b`}, // SSN and passport
		},
	}

	filter := loop.NewGuardrailFilter(configs)

	output := "SSN: 123-45-6789, Passport: AB123456"
	_, violations := filter.Check(output)

	// Should have 2 violations (one for each pattern match)
	if len(violations) != 2 {
		t.Errorf("expected 2 violations, got %d", len(violations))
	}
}
