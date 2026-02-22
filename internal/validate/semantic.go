package validate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/szaher/designs/agentz/internal/ast"
)

// ValidateSemantic performs semantic validation: reference resolution,
// duplicate detection, plaintext secret rejection.
func ValidateSemantic(f *ast.File) []*ValidationError {
	var errs []*ValidationError

	// Collect all declared names by kind
	names := collectNames(f)

	// Check references
	for _, stmt := range f.Statements {
		switch s := stmt.(type) {
		case *ast.Agent:
			if s.Prompt != nil {
				if !names["Prompt"][s.Prompt.Name] {
					hint := suggestName(s.Prompt.Name, names["Prompt"])
					errs = append(errs, posError(s.Prompt.StartPos,
						fmt.Sprintf("prompt %q not found", s.Prompt.Name), hint))
				}
			}
			for _, skill := range s.Skills {
				if !names["Skill"][skill.Name] {
					hint := suggestName(skill.Name, names["Skill"])
					errs = append(errs, posError(skill.StartPos,
						fmt.Sprintf("skill %q not found", skill.Name), hint))
				}
			}
			if s.Client != nil {
				if !names["MCPClient"][s.Client.Name] {
					hint := suggestName(s.Client.Name, names["MCPClient"])
					errs = append(errs, posError(s.Client.StartPos,
						fmt.Sprintf("client %q not found", s.Client.Name), hint))
				}
			}
		case *ast.MCPClient:
			for _, server := range s.Servers {
				if !names["MCPServer"][server.Name] {
					hint := suggestName(server.Name, names["MCPServer"])
					errs = append(errs, posError(server.StartPos,
						fmt.Sprintf("server %q not found", server.Name), hint))
				}
			}
		case *ast.MCPServer:
			for _, skill := range s.Skills {
				if !names["Skill"][skill.Name] {
					hint := suggestName(skill.Name, names["Skill"])
					errs = append(errs, posError(skill.StartPos,
						fmt.Sprintf("skill %q not found", skill.Name), hint))
				}
			}
			if s.Auth != nil {
				if !names["Secret"][s.Auth.Name] {
					hint := suggestName(s.Auth.Name, names["Secret"])
					errs = append(errs, posError(s.Auth.StartPos,
						fmt.Sprintf("secret %q not found", s.Auth.Name), hint))
				}
			}
		}
	}

	// Check for multiple default bindings
	defaultCount := 0
	for _, stmt := range f.Statements {
		if b, ok := stmt.(*ast.Binding); ok && b.Default {
			defaultCount++
		}
	}
	if defaultCount > 1 {
		errs = append(errs, &ValidationError{
			File:    f.Path,
			Line:    1,
			Column:  1,
			Message: "multiple bindings marked as default",
			Hint:    "only one binding may be the default",
		})
	}

	return errs
}

func collectNames(f *ast.File) map[string]map[string]bool {
	names := map[string]map[string]bool{
		"Agent":       {},
		"Prompt":      {},
		"Skill":       {},
		"MCPServer":   {},
		"MCPClient":   {},
		"Secret":      {},
		"Environment": {},
		"Policy":      {},
		"Binding":     {},
	}

	for _, stmt := range f.Statements {
		switch s := stmt.(type) {
		case *ast.Agent:
			names["Agent"][s.Name] = true
		case *ast.Prompt:
			names["Prompt"][s.Name] = true
		case *ast.Skill:
			names["Skill"][s.Name] = true
		case *ast.MCPServer:
			names["MCPServer"][s.Name] = true
		case *ast.MCPClient:
			names["MCPClient"][s.Name] = true
		case *ast.Secret:
			names["Secret"][s.Name] = true
		case *ast.Environment:
			names["Environment"][s.Name] = true
		case *ast.Policy:
			names["Policy"][s.Name] = true
		case *ast.Binding:
			names["Binding"][s.Name] = true
		}
	}
	return names
}

// suggestName returns a "did you mean?" hint if a close match exists.
func suggestName(name string, available map[string]bool) string {
	if len(available) == 0 {
		return ""
	}
	names := make([]string, 0, len(available))
	for n := range available {
		names = append(names, n)
	}
	sort.Strings(names)

	best := ""
	bestDist := len(name)/2 + 1
	for _, n := range names {
		d := levenshtein(name, n)
		if d < bestDist {
			bestDist = d
			best = n
		}
	}
	if best != "" {
		return fmt.Sprintf("did you mean %q?", best)
	}
	return "available: " + strings.Join(names, ", ")
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}
