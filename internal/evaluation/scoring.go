// Package evaluation implements the batch evaluation framework for
// testing agent quality against declared eval cases.
package evaluation

import (
	"fmt"
	"strings"

	"github.com/szaher/designs/agentz/internal/expr"
)

// Scorer evaluates how well an actual output matches the expected output.
type Scorer interface {
	Score(actual, expected string) (float64, error)
}

// NewScorer creates a Scorer for the given scoring method.
func NewScorer(method string) (Scorer, error) {
	switch method {
	case "exact":
		return &exactScorer{}, nil
	case "contains":
		return &containsScorer{}, nil
	case "semantic":
		return &semanticScorer{}, nil
	case "custom":
		return &customScorer{}, nil
	default:
		return nil, fmt.Errorf("unknown scoring method: %q", method)
	}
}

// exactScorer performs exact string match (case-insensitive trim).
type exactScorer struct{}

func (s *exactScorer) Score(actual, expected string) (float64, error) {
	if strings.TrimSpace(strings.ToLower(actual)) == strings.TrimSpace(strings.ToLower(expected)) {
		return 1.0, nil
	}
	return 0.0, nil
}

// containsScorer checks if expected is a substring of actual.
type containsScorer struct{}

func (s *containsScorer) Score(actual, expected string) (float64, error) {
	lower := strings.ToLower(actual)
	expectedLower := strings.ToLower(expected)
	if strings.Contains(lower, expectedLower) {
		return 1.0, nil
	}
	// Partial matching: count words from expected found in actual
	expectedWords := strings.Fields(expectedLower)
	if len(expectedWords) == 0 {
		return 1.0, nil
	}
	found := 0
	for _, w := range expectedWords {
		if strings.Contains(lower, w) {
			found++
		}
	}
	return float64(found) / float64(len(expectedWords)), nil
}

// semanticScorer performs semantic similarity scoring.
// For MVP, this uses word overlap (Jaccard similarity) as a proxy.
// In production, this would use embedding similarity.
type semanticScorer struct{}

func (s *semanticScorer) Score(actual, expected string) (float64, error) {
	actualWords := toWordSet(actual)
	expectedWords := toWordSet(expected)

	if len(expectedWords) == 0 && len(actualWords) == 0 {
		return 1.0, nil
	}
	if len(expectedWords) == 0 || len(actualWords) == 0 {
		return 0.0, nil
	}

	// Jaccard similarity
	intersection := 0
	for w := range expectedWords {
		if actualWords[w] {
			intersection++
		}
	}

	union := len(actualWords)
	for w := range expectedWords {
		if !actualWords[w] {
			union++
		}
	}

	if union == 0 {
		return 1.0, nil
	}

	return float64(intersection) / float64(union), nil
}

func toWordSet(s string) map[string]bool {
	words := strings.Fields(strings.ToLower(s))
	set := make(map[string]bool, len(words))
	for _, w := range words {
		// Strip punctuation
		w = strings.Trim(w, ".,!?;:\"'()[]{}")
		if w != "" {
			set[w] = true
		}
	}
	return set
}

// customScorer uses an expr expression for scoring.
type customScorer struct{}

func (s *customScorer) Score(actual, expected string) (float64, error) {
	// The expected field contains the expression for custom scoring
	ctx := &expr.Context{
		Input:  expected,
		Output: actual,
	}

	result, err := expr.EvalString(expected, ctx)
	if err != nil {
		return 0.0, fmt.Errorf("custom score expression error: %w", err)
	}

	switch v := result.(type) {
	case float64:
		return v, nil
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	case int:
		return float64(v), nil
	default:
		return 0.0, fmt.Errorf("custom score returned %T, expected float64 or bool", result)
	}
}
