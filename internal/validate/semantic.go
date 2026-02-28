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
			if s.Fallback != "" {
				if !names["Agent"][s.Fallback] {
					hint := suggestName(s.Fallback, names["Agent"])
					errs = append(errs, posError(s.StartPos,
						fmt.Sprintf("fallback agent %q not found", s.Fallback), hint))
				}
				if s.Fallback == s.Name {
					errs = append(errs, posError(s.StartPos,
						fmt.Sprintf("agent %q cannot fallback to itself", s.Name), ""))
				}
			}
			for _, d := range s.Delegates {
				if !names["Agent"][d.AgentRef] {
					hint := suggestName(d.AgentRef, names["Agent"])
					errs = append(errs, posError(d.StartPos,
						fmt.Sprintf("delegate agent %q not found", d.AgentRef), hint))
				}
			}
			// IntentLang 3.0: check on-input block references
			if s.OnInput != nil {
				errs = append(errs, validateOnInputRefs(s.OnInput.Statements, names)...)
			}
		case *ast.Skill:
			if s.ToolConfig != nil && s.ToolConfig.Type == "mcp" && s.ToolConfig.ServerTool != "" {
				parts := strings.SplitN(s.ToolConfig.ServerTool, "/", 2)
				if len(parts) == 2 {
					serverName := parts[0]
					if !names["MCPServer"][serverName] {
						hint := suggestName(serverName, names["MCPServer"])
						errs = append(errs, posError(s.ToolConfig.StartPos,
							fmt.Sprintf("MCP server %q not found in tool reference %q", serverName, s.ToolConfig.ServerTool), hint))
					}
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
		case *ast.Pipeline:
			for _, step := range s.Steps {
				if step.Agent != "" && !names["Agent"][step.Agent] {
					hint := suggestName(step.Agent, names["Agent"])
					errs = append(errs, posError(step.StartPos,
						fmt.Sprintf("step %q references unknown agent %q", step.Name, step.Agent), hint))
				}
			}
		case *ast.Prompt:
			// Validate prompt variable references in content
			if len(s.Variables) > 0 {
				declaredVars := make(map[string]bool)
				for _, v := range s.Variables {
					declaredVars[v.Name] = true
				}
				// Check for {{var}} references in content
				content := s.Content
				for i := 0; i < len(content)-3; i++ {
					if content[i] == '{' && content[i+1] == '{' {
						end := strings.Index(content[i+2:], "}}")
						if end >= 0 {
							varName := strings.TrimSpace(content[i+2 : i+2+end])
							if !declaredVars[varName] {
								errs = append(errs, posError(s.StartPos,
									fmt.Sprintf("prompt %q references undeclared variable {{%s}}", s.Name, varName),
									"add variable declaration in the variables block"))
							}
						}
					}
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
	defaultBindings := 0
	for _, stmt := range f.Statements {
		if b, ok := stmt.(*ast.Binding); ok && b.Default {
			defaultBindings++
		}
	}
	if defaultBindings > 1 {
		errs = append(errs, &ValidationError{
			File:    f.Path,
			Line:    1,
			Column:  1,
			Message: "multiple bindings marked as default",
			Hint:    "only one binding may be the default",
		})
	}

	// Check for multiple default deploy targets
	defaultDeploys := 0
	for _, stmt := range f.Statements {
		if d, ok := stmt.(*ast.DeployTarget); ok && d.Default {
			defaultDeploys++
		}
	}
	if defaultDeploys > 1 {
		errs = append(errs, &ValidationError{
			File:    f.Path,
			Line:    1,
			Column:  1,
			Message: "multiple deploy targets marked as default",
			Hint:    "only one deploy target may be the default",
		})
	}

	return errs
}

// validateOnInputRefs checks skill and agent references in on-input statements.
func validateOnInputRefs(stmts []ast.OnInputStmt, names map[string]map[string]bool) []*ValidationError {
	var errs []*ValidationError
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.UseSkillStmt:
			if !names["Skill"][s.SkillName] {
				hint := suggestName(s.SkillName, names["Skill"])
				errs = append(errs, posError(s.StartPos,
					fmt.Sprintf("skill %q not found", s.SkillName), hint))
			}
		case *ast.DelegateToStmt:
			if !names["Agent"][s.AgentName] {
				hint := suggestName(s.AgentName, names["Agent"])
				errs = append(errs, posError(s.StartPos,
					fmt.Sprintf("delegate agent %q not found", s.AgentName), hint))
			}
		case *ast.IfBlock:
			errs = append(errs, validateOnInputRefs(s.Body, names)...)
			for _, elseIf := range s.ElseIfs {
				errs = append(errs, validateOnInputRefs(elseIf.Body, names)...)
			}
			if s.ElseBody != nil {
				errs = append(errs, validateOnInputRefs(s.ElseBody, names)...)
			}
		case *ast.ForEachBlock:
			errs = append(errs, validateOnInputRefs(s.Body, names)...)
		}
	}
	return errs
}

func collectNames(f *ast.File) map[string]map[string]bool {
	names := map[string]map[string]bool{
		"Agent":        {},
		"Prompt":       {},
		"Skill":        {},
		"MCPServer":    {},
		"MCPClient":    {},
		"Secret":       {},
		"Environment":  {},
		"Policy":       {},
		"Binding":      {},
		"DeployTarget": {},
		"Type":         {},
		"Pipeline":     {},
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
		case *ast.DeployTarget:
			names["DeployTarget"][s.Name] = true
		case *ast.TypeDef:
			names["Type"][s.Name] = true
		case *ast.Pipeline:
			names["Pipeline"][s.Name] = true
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
