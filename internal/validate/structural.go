// Package validate implements structural and semantic validation
// for parsed .ias definitions.
package validate

import (
	"fmt"

	"github.com/szaher/designs/agentz/internal/ast"
)

// ValidationError represents a validation error with position and hint.
type ValidationError struct {
	File    string
	Line    int
	Column  int
	Message string
	Hint    string
}

func (e *ValidationError) Error() string {
	s := fmt.Sprintf("%s:%d:%d: error: %s", e.File, e.Line, e.Column, e.Message)
	if e.Hint != "" {
		s += "\n  hint: " + e.Hint
	}
	return s
}

// ValidateStructural performs structural validation: required fields,
// type checks, and schema conformance.
func ValidateStructural(f *ast.File) []*ValidationError {
	var errs []*ValidationError

	if f.Package == nil {
		errs = append(errs, &ValidationError{
			File: f.Path, Line: 1, Column: 1,
			Message: "missing package declaration",
			Hint:    "add 'package \"name\" version \"x.y.z\" lang \"1.0\"' at the start",
		})
		return errs
	}

	pkg := f.Package
	if pkg.Name == "" {
		errs = append(errs, posError(pkg.StartPos, "package name is required", ""))
	}
	if pkg.Version == "" {
		errs = append(errs, posError(pkg.StartPos, "package version is required", "add version \"x.y.z\""))
	}
	if pkg.LangVersion == "" {
		errs = append(errs, posError(pkg.StartPos, "lang version is required", "add lang \"1.0\""))
	}

	for _, stmt := range f.Statements {
		switch s := stmt.(type) {
		case *ast.Agent:
			errs = append(errs, validateAgent(s)...)
		case *ast.Prompt:
			errs = append(errs, validatePrompt(s)...)
		case *ast.Skill:
			errs = append(errs, validateSkill(s)...)
		case *ast.MCPServer:
			errs = append(errs, validateMCPServer(s)...)
		case *ast.MCPClient:
			errs = append(errs, validateMCPClient(s)...)
		case *ast.Secret:
			errs = append(errs, validateSecret(s)...)
		case *ast.Binding:
			errs = append(errs, validateBinding(s)...)
		}
	}
	return errs
}

func validateAgent(a *ast.Agent) []*ValidationError {
	var errs []*ValidationError
	if a.Name == "" {
		errs = append(errs, posError(a.StartPos, "agent name is required", ""))
	}
	if a.Prompt == nil {
		errs = append(errs, posError(a.StartPos,
			fmt.Sprintf("agent %q requires a prompt reference", a.Name),
			"add 'uses prompt \"name\"'"))
	}
	if a.Model == "" {
		errs = append(errs, posError(a.StartPos,
			fmt.Sprintf("agent %q requires a model", a.Name),
			"add 'model \"model-name\"'"))
	}
	return errs
}

func validatePrompt(p *ast.Prompt) []*ValidationError {
	var errs []*ValidationError
	if p.Name == "" {
		errs = append(errs, posError(p.StartPos, "prompt name is required", ""))
	}
	if p.Content == "" {
		errs = append(errs, posError(p.StartPos,
			fmt.Sprintf("prompt %q requires content", p.Name),
			"add 'content \"...\"'"))
	}
	return errs
}

func validateSkill(s *ast.Skill) []*ValidationError {
	var errs []*ValidationError
	if s.Name == "" {
		errs = append(errs, posError(s.StartPos, "skill name is required", ""))
	}
	if s.Description == "" {
		errs = append(errs, posError(s.StartPos,
			fmt.Sprintf("skill %q requires a description", s.Name),
			"add 'description \"...\"'"))
	}
	if len(s.Input) == 0 {
		errs = append(errs, posError(s.StartPos,
			fmt.Sprintf("skill %q requires an input schema", s.Name),
			"add 'input { name type }'"))
	}
	if len(s.Output) == 0 {
		errs = append(errs, posError(s.StartPos,
			fmt.Sprintf("skill %q requires an output schema", s.Name),
			"add 'output { name type }'"))
	}
	if s.Execution == nil {
		errs = append(errs, posError(s.StartPos,
			fmt.Sprintf("skill %q requires an execution block", s.Name),
			"add 'execution command \"...\"'"))
	}
	return errs
}

func validateMCPServer(s *ast.MCPServer) []*ValidationError {
	var errs []*ValidationError
	if s.Name == "" {
		errs = append(errs, posError(s.StartPos, "server name is required", ""))
	}
	if s.Transport == "" {
		errs = append(errs, posError(s.StartPos,
			fmt.Sprintf("server %q requires a transport", s.Name),
			"add 'transport \"stdio\"' or 'transport \"sse\"'"))
	}
	if s.Transport == "stdio" && s.Command == "" {
		errs = append(errs, posError(s.StartPos,
			fmt.Sprintf("server %q with stdio transport requires a command", s.Name),
			"add 'command \"...\"'"))
	}
	if (s.Transport == "sse" || s.Transport == "streamable-http") && s.URL == "" {
		errs = append(errs, posError(s.StartPos,
			fmt.Sprintf("server %q with %s transport requires a url", s.Name, s.Transport),
			"add 'url \"...\"'"))
	}
	return errs
}

func validateMCPClient(c *ast.MCPClient) []*ValidationError {
	var errs []*ValidationError
	if c.Name == "" {
		errs = append(errs, posError(c.StartPos, "client name is required", ""))
	}
	if len(c.Servers) == 0 {
		errs = append(errs, posError(c.StartPos,
			fmt.Sprintf("client %q requires at least one server connection", c.Name),
			"add 'connects to server \"name\"'"))
	}
	return errs
}

func validateSecret(s *ast.Secret) []*ValidationError {
	var errs []*ValidationError
	if s.Name == "" {
		errs = append(errs, posError(s.StartPos, "secret name is required", ""))
	}
	if s.Source == "" {
		errs = append(errs, posError(s.StartPos,
			fmt.Sprintf("secret %q requires a source (env or store)", s.Name),
			"add 'env(KEY)' or 'store(path)'"))
	}
	if s.Key == "" {
		errs = append(errs, posError(s.StartPos,
			fmt.Sprintf("secret %q requires a key", s.Name),
			""))
	}
	return errs
}

func validateBinding(b *ast.Binding) []*ValidationError {
	var errs []*ValidationError
	if b.Name == "" {
		errs = append(errs, posError(b.StartPos, "binding name is required", ""))
	}
	if b.Adapter == "" {
		errs = append(errs, posError(b.StartPos,
			fmt.Sprintf("binding %q requires an adapter", b.Name),
			"add 'adapter \"local-mcp\"'"))
	}
	return errs
}

func posError(pos ast.Pos, msg, hint string) *ValidationError {
	return &ValidationError{
		File:    pos.File,
		Line:    pos.Line,
		Column:  pos.Column,
		Message: msg,
		Hint:    hint,
	}
}
