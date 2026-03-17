package loop

import (
	"regexp"
	"strings"
)

// Violation records a guardrail trigger.
type Violation struct {
	Guardrail string // guardrail name
	Mode      string // "warn" or "block"
	Match     string // what triggered the violation
	Type      string // "keyword" or "pattern"
}

// GuardrailConfig defines a guardrail's matching rules.
type GuardrailConfig struct {
	Name        string
	Mode        string // "warn" or "block"
	Keywords    []string
	Patterns    []string
	FallbackMsg string
}

// GuardrailFilter checks agent output against guardrail rules.
type GuardrailFilter struct {
	guardrails []GuardrailConfig
}

// NewGuardrailFilter creates a filter from guardrail configs.
func NewGuardrailFilter(configs []GuardrailConfig) *GuardrailFilter {
	return &GuardrailFilter{guardrails: configs}
}

// Check applies all guardrails to the output.
// Returns the (possibly filtered) output and any violations.
func (f *GuardrailFilter) Check(output string) (string, []Violation) {
	var violations []Violation
	blocked := false
	fallbackMsg := ""

	for _, g := range f.guardrails {
		// Check keywords (case-insensitive)
		lower := strings.ToLower(output)
		for _, kw := range g.Keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				violations = append(violations, Violation{
					Guardrail: g.Name,
					Mode:      g.Mode,
					Match:     kw,
					Type:      "keyword",
				})
				if g.Mode == "block" {
					blocked = true
					fallbackMsg = g.FallbackMsg
				}
			}
		}

		// Check regex patterns
		for _, pat := range g.Patterns {
			re, err := regexp.Compile(pat)
			if err != nil {
				continue
			}
			if match := re.FindString(output); match != "" {
				violations = append(violations, Violation{
					Guardrail: g.Name,
					Mode:      g.Mode,
					Match:     match,
					Type:      "pattern",
				})
				if g.Mode == "block" {
					blocked = true
					fallbackMsg = g.FallbackMsg
				}
			}
		}
	}

	if blocked && fallbackMsg != "" {
		return fallbackMsg, violations
	}

	return output, violations
}
