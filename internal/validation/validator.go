// Package validation implements output validation rules for compiled agents.
// It evaluates declared validation rules against agent responses using
// the expression engine, supporting both error (reject + retry) and
// warning (log only) severity levels.
package validation

import (
	"fmt"

	"github.com/szaher/designs/agentz/internal/expr"
)

// RuleDef describes a declared validation rule from the IR.
type RuleDef struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
	Severity   string `json:"severity"`    // "error" or "warning"
	Message    string `json:"message"`     // Human-readable failure message
	MaxRetries int    `json:"max_retries"` // Only for "error" severity
}

// RuleResult is the outcome of evaluating a single validation rule.
type RuleResult struct {
	RuleName   string `json:"rule_name"`
	Passed     bool   `json:"passed"`
	Severity   string `json:"severity"`
	Message    string `json:"message,omitempty"`
	Expression string `json:"expression"`
	Error      string `json:"error,omitempty"` // Set if expression eval failed
}

// ValidationResult is the outcome of evaluating all validation rules.
type ValidationResult struct {
	Passed       bool         `json:"passed"`
	RulesChecked int          `json:"rules_checked"`
	Results      []RuleResult `json:"results"`
	Warnings     []string     `json:"warnings,omitempty"`
	Errors       []string     `json:"errors,omitempty"`
}

// Validator evaluates validation rules against agent output.
type Validator struct {
	rules []RuleDef
}

// NewValidator creates a Validator from rule definitions.
func NewValidator(rules []RuleDef) *Validator {
	return &Validator{rules: rules}
}

// Validate runs all rules against the given context.
// The context should contain the agent output, input, session, etc.
func (v *Validator) Validate(ctx *expr.Context) *ValidationResult {
	result := &ValidationResult{
		Passed:       true,
		RulesChecked: len(v.rules),
	}

	for _, rule := range v.rules {
		rr := v.evaluateRule(rule, ctx)
		result.Results = append(result.Results, rr)

		if !rr.Passed {
			switch rule.Severity {
			case "error":
				result.Passed = false
				result.Errors = append(result.Errors, rr.Message)
			case "warning":
				result.Warnings = append(result.Warnings, rr.Message)
			}
		}
	}

	return result
}

// FailedErrorRules returns rule definitions that failed with "error" severity.
func (v *Validator) FailedErrorRules(result *ValidationResult) []RuleDef {
	var failed []RuleDef
	for _, rr := range result.Results {
		if !rr.Passed && rr.Severity == "error" {
			for _, rule := range v.rules {
				if rule.Name == rr.RuleName {
					failed = append(failed, rule)
					break
				}
			}
		}
	}
	return failed
}

func (v *Validator) evaluateRule(rule RuleDef, ctx *expr.Context) RuleResult {
	rr := RuleResult{
		RuleName:   rule.Name,
		Severity:   rule.Severity,
		Expression: rule.Expression,
	}

	result, err := expr.EvalString(rule.Expression, ctx)
	if err != nil {
		rr.Passed = false
		rr.Error = err.Error()
		rr.Message = fmt.Sprintf("Rule %q: expression evaluation failed: %v", rule.Name, err)
		return rr
	}

	passed, ok := result.(bool)
	if !ok {
		rr.Passed = false
		rr.Error = fmt.Sprintf("expression returned %T, expected bool", result)
		rr.Message = fmt.Sprintf("Rule %q: expression did not return a boolean", rule.Name)
		return rr
	}

	rr.Passed = passed
	if !passed {
		rr.Message = rule.Message
		if rr.Message == "" {
			rr.Message = fmt.Sprintf("Rule %q failed", rule.Name)
		}
	}

	return rr
}

// ExtractValidationRules extracts RuleDef from IR agent attributes.
func ExtractValidationRules(attrs map[string]interface{}) []RuleDef {
	raw, ok := attrs["validation_rules"].([]interface{})
	if !ok {
		return nil
	}

	var rules []RuleDef
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		r := RuleDef{
			Severity:   "error",
			MaxRetries: 3,
		}
		if n, ok := m["name"].(string); ok {
			r.Name = n
		}
		if e, ok := m["expression"].(string); ok {
			r.Expression = e
		}
		if s, ok := m["severity"].(string); ok {
			r.Severity = s
		}
		if msg, ok := m["message"].(string); ok {
			r.Message = msg
		}
		if mr, ok := m["max_retries"]; ok {
			switch n := mr.(type) {
			case int:
				r.MaxRetries = n
			case float64:
				r.MaxRetries = int(n)
			}
		}
		rules = append(rules, r)
	}
	return rules
}
