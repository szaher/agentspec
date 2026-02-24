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
		case *ast.DeployTarget:
			errs = append(errs, validateDeployTarget(s)...)
		case *ast.TypeDef:
			errs = append(errs, validateTypeDef(s)...)
		case *ast.Pipeline:
			errs = append(errs, validatePipeline(s)...)
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
	if a.Strategy != "" {
		validStrategies := map[string]bool{
			"react": true, "plan-and-execute": true, "reflexion": true,
			"router": true, "map-reduce": true,
		}
		if !validStrategies[a.Strategy] {
			errs = append(errs, posError(a.StartPos,
				fmt.Sprintf("agent %q has invalid strategy %q", a.Name, a.Strategy),
				"valid strategies: react, plan-and-execute, reflexion, router, map-reduce"))
		}
	}
	if a.OnError != "" {
		validOnError := map[string]bool{"retry": true, "fail": true, "fallback": true}
		if !validOnError[a.OnError] {
			errs = append(errs, posError(a.StartPos,
				fmt.Sprintf("agent %q has invalid on_error %q", a.Name, a.OnError),
				"valid values: retry, fail, fallback"))
		}
	}
	if a.OnError == "fallback" && a.Fallback == "" {
		errs = append(errs, posError(a.StartPos,
			fmt.Sprintf("agent %q uses on_error \"fallback\" but no fallback agent specified", a.Name),
			"add 'fallback \"agent-name\"'"))
	}
	if a.HasTemp && (a.Temperature < 0 || a.Temperature > 2) {
		errs = append(errs, posError(a.StartPos,
			fmt.Sprintf("agent %q has temperature %.1f out of range [0, 2]", a.Name, a.Temperature),
			""))
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
	if s.Execution == nil && s.ToolConfig == nil {
		errs = append(errs, posError(s.StartPos,
			fmt.Sprintf("skill %q requires an execution or tool block", s.Name),
			"add 'tool mcp \"server/tool\"' or 'execution command \"...\"'"))
	}
	if s.Execution != nil && s.ToolConfig != nil {
		errs = append(errs, posError(s.StartPos,
			fmt.Sprintf("skill %q has both execution and tool blocks", s.Name),
			"use only one: 'tool' (2.0) or 'execution' (1.0)"))
	}
	if s.ToolConfig != nil {
		validTypes := map[string]bool{"mcp": true, "http": true, "command": true, "inline": true}
		if !validTypes[s.ToolConfig.Type] {
			errs = append(errs, posError(s.ToolConfig.StartPos,
				fmt.Sprintf("skill %q has invalid tool type %q", s.Name, s.ToolConfig.Type),
				"valid types: mcp, http, command, inline"))
		}
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

func validateDeployTarget(d *ast.DeployTarget) []*ValidationError {
	var errs []*ValidationError
	if d.Name == "" {
		errs = append(errs, posError(d.StartPos, "deploy target name is required", ""))
	}
	if d.Target == "" {
		errs = append(errs, posError(d.StartPos,
			fmt.Sprintf("deploy %q requires a target type", d.Name),
			"add 'target \"process\"', 'target \"docker\"', or 'target \"kubernetes\"'"))
	}
	validTargets := map[string]bool{"process": true, "docker": true, "docker-compose": true, "kubernetes": true}
	if d.Target != "" && !validTargets[d.Target] {
		errs = append(errs, posError(d.StartPos,
			fmt.Sprintf("deploy %q has invalid target type %q", d.Name, d.Target),
			"valid targets: process, docker, docker-compose, kubernetes"))
	}
	return errs
}

func validateTypeDef(t *ast.TypeDef) []*ValidationError {
	var errs []*ValidationError
	if t.Name == "" {
		errs = append(errs, posError(t.StartPos, "type name is required", ""))
	}
	if len(t.Fields) == 0 && len(t.EnumVals) == 0 && t.ListOf == "" {
		errs = append(errs, posError(t.StartPos,
			fmt.Sprintf("type %q must have fields, enum values, or list type", t.Name),
			"add fields in a block, enum [...], or list <type>"))
	}
	for _, f := range t.Fields {
		if f.Name == "" {
			errs = append(errs, posError(f.StartPos, "type field name is required", ""))
		}
		if f.Type == "" {
			errs = append(errs, posError(f.StartPos,
				fmt.Sprintf("type field %q requires a type", f.Name), ""))
		}
	}
	return errs
}

func validatePipeline(p *ast.Pipeline) []*ValidationError {
	var errs []*ValidationError
	if p.Name == "" {
		errs = append(errs, posError(p.StartPos, "pipeline name is required", ""))
	}
	if len(p.Steps) == 0 {
		errs = append(errs, posError(p.StartPos,
			fmt.Sprintf("pipeline %q requires at least one step", p.Name),
			"add 'step \"name\" { agent \"...\" }'"))
	}

	// Check for duplicate step names and validate step references
	stepNames := make(map[string]bool)
	for _, step := range p.Steps {
		if step.Name == "" {
			errs = append(errs, posError(step.StartPos, "step name is required", ""))
			continue
		}
		if stepNames[step.Name] {
			errs = append(errs, posError(step.StartPos,
				fmt.Sprintf("duplicate step name %q in pipeline %q", step.Name, p.Name), ""))
		}
		stepNames[step.Name] = true

		if step.Agent == "" {
			errs = append(errs, posError(step.StartPos,
				fmt.Sprintf("step %q requires an agent", step.Name),
				"add 'agent \"agent-name\"'"))
		}
	}

	// Validate depends_on references point to valid step names
	for _, step := range p.Steps {
		for _, dep := range step.DependsOn {
			if !stepNames[dep] {
				errs = append(errs, posError(step.StartPos,
					fmt.Sprintf("step %q depends_on unknown step %q", step.Name, dep),
					""))
			}
			if dep == step.Name {
				errs = append(errs, posError(step.StartPos,
					fmt.Sprintf("step %q cannot depend on itself", step.Name), ""))
			}
		}
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
