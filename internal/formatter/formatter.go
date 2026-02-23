// Package formatter implements canonical formatting of .az files
// from AST back to source, with deterministic output.
package formatter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/szaher/designs/agentz/internal/ast"
)

// Format formats an AST File back to canonical .az source.
func Format(f *ast.File) string {
	var sb strings.Builder

	if f.Package != nil {
		formatPackage(&sb, f.Package)
	}

	for _, stmt := range f.Statements {
		sb.WriteString("\n")
		formatStatement(&sb, stmt)
	}

	return sb.String()
}

func formatPackage(sb *strings.Builder, pkg *ast.Package) {
	fmt.Fprintf(sb, "package %q", pkg.Name)
	if pkg.Version != "" {
		fmt.Fprintf(sb, " version %q", pkg.Version)
	}
	if pkg.LangVersion != "" {
		fmt.Fprintf(sb, " lang %q", pkg.LangVersion)
	}
	sb.WriteString("\n")
}

func formatStatement(sb *strings.Builder, stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.Prompt:
		formatPrompt(sb, s)
	case *ast.Skill:
		formatSkill(sb, s)
	case *ast.Agent:
		formatAgent(sb, s)
	case *ast.Binding:
		formatBinding(sb, s)
	case *ast.Secret:
		formatSecret(sb, s)
	case *ast.Environment:
		formatEnvironment(sb, s)
	case *ast.Policy:
		formatPolicy(sb, s)
	case *ast.Plugin:
		formatPlugin(sb, s)
	case *ast.MCPServer:
		formatMCPServer(sb, s)
	case *ast.MCPClient:
		formatMCPClient(sb, s)
	case *ast.PluginRef:
		formatPluginRef(sb, s)
	}
}

func formatPrompt(sb *strings.Builder, p *ast.Prompt) {
	fmt.Fprintf(sb, "prompt %q {\n", p.Name)
	if p.Content != "" {
		fmt.Fprintf(sb, "  content %q\n", p.Content)
	}
	if p.Version != "" {
		fmt.Fprintf(sb, "  version %q\n", p.Version)
	}
	formatMetadata(sb, p.Metadata)
	sb.WriteString("}\n")
}

func formatSkill(sb *strings.Builder, s *ast.Skill) {
	fmt.Fprintf(sb, "skill %q {\n", s.Name)
	if s.Description != "" {
		fmt.Fprintf(sb, "  description %q\n", s.Description)
	}
	if len(s.Input) > 0 {
		sb.WriteString("  input {\n")
		for _, f := range s.Input {
			fmt.Fprintf(sb, "    %s %s", f.Name, f.Type)
			if f.Required {
				sb.WriteString(" required")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("  }\n")
	}
	if len(s.Output) > 0 {
		sb.WriteString("  output {\n")
		for _, f := range s.Output {
			fmt.Fprintf(sb, "    %s %s", f.Name, f.Type)
			if f.Required {
				sb.WriteString(" required")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("  }\n")
	}
	if s.Execution != nil {
		fmt.Fprintf(sb, "  execution %s %q\n", s.Execution.Type, s.Execution.Value)
	}
	formatMetadata(sb, s.Metadata)
	sb.WriteString("}\n")
}

func formatAgent(sb *strings.Builder, a *ast.Agent) {
	fmt.Fprintf(sb, "agent %q {\n", a.Name)
	if a.Prompt != nil {
		fmt.Fprintf(sb, "  uses prompt %q\n", a.Prompt.Name)
	}
	for _, skill := range a.Skills {
		fmt.Fprintf(sb, "  uses skill %q\n", skill.Name)
	}
	if a.Model != "" {
		fmt.Fprintf(sb, "  model %q\n", a.Model)
	}
	if a.Client != nil {
		fmt.Fprintf(sb, "  connects to client %q\n", a.Client.Name)
	}
	formatMetadata(sb, a.Metadata)
	sb.WriteString("}\n")
}

func formatBinding(sb *strings.Builder, b *ast.Binding) {
	fmt.Fprintf(sb, "binding %q", b.Name)
	if b.Adapter != "" {
		fmt.Fprintf(sb, " adapter %q", b.Adapter)
	}
	sb.WriteString(" {\n")
	if b.Default {
		sb.WriteString("  default true\n")
	}
	keys := sortedKeys(b.Config)
	for _, k := range keys {
		fmt.Fprintf(sb, "  %s %q\n", k, b.Config[k])
	}
	sb.WriteString("}\n")
}

func formatSecret(sb *strings.Builder, s *ast.Secret) {
	fmt.Fprintf(sb, "secret %q {\n", s.Name)
	switch s.Source {
	case "env":
		fmt.Fprintf(sb, "  env(%s)\n", s.Key)
	case "store":
		fmt.Fprintf(sb, "  store(%s)\n", s.Key)
	}
	sb.WriteString("}\n")
}

func formatEnvironment(sb *strings.Builder, e *ast.Environment) {
	fmt.Fprintf(sb, "environment %q {\n", e.Name)
	// Group overrides by resource
	groups := make(map[string][]*ast.Override)
	var order []string
	for _, o := range e.Overrides {
		if _, seen := groups[o.Resource]; !seen {
			order = append(order, o.Resource)
		}
		groups[o.Resource] = append(groups[o.Resource], o)
	}
	for _, res := range order {
		parts := strings.SplitN(res, "/", 2)
		if len(parts) == 2 {
			fmt.Fprintf(sb, "  %s %q {\n", parts[0], parts[1])
		} else {
			fmt.Fprintf(sb, "  %s {\n", res)
		}
		for _, o := range groups[res] {
			fmt.Fprintf(sb, "    %s %q\n", o.Attribute, o.Value)
		}
		sb.WriteString("  }\n")
	}
	sb.WriteString("}\n")
}

func formatPolicy(sb *strings.Builder, p *ast.Policy) {
	fmt.Fprintf(sb, "policy %q {\n", p.Name)
	for _, r := range p.Rules {
		fmt.Fprintf(sb, "  %s %s", r.Action, r.Resource)
		if r.Subject != "" {
			sb.WriteString(" " + r.Subject)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("}\n")
}

func formatPlugin(sb *strings.Builder, p *ast.Plugin) {
	fmt.Fprintf(sb, "plugin %q", p.Name)
	if p.Version != "" {
		fmt.Fprintf(sb, " version %q", p.Version)
	}
	sb.WriteString("\n")
}

func formatMCPServer(sb *strings.Builder, s *ast.MCPServer) {
	fmt.Fprintf(sb, "server %q {\n", s.Name)
	if s.Transport != "" {
		fmt.Fprintf(sb, "  transport %q\n", s.Transport)
	}
	if s.Command != "" {
		fmt.Fprintf(sb, "  command %q\n", s.Command)
	}
	if len(s.Args) > 0 {
		sb.WriteString("  args [")
		for i, arg := range s.Args {
			if i > 0 {
				sb.WriteString(", ")
			}
			fmt.Fprintf(sb, "%q", arg)
		}
		sb.WriteString("]\n")
	}
	if s.URL != "" {
		fmt.Fprintf(sb, "  url %q\n", s.URL)
	}
	if s.Auth != nil {
		fmt.Fprintf(sb, "  auth %q\n", s.Auth.Name)
	}
	for _, skill := range s.Skills {
		fmt.Fprintf(sb, "  exposes skill %q\n", skill.Name)
	}
	if len(s.Env) > 0 {
		sb.WriteString("  env {\n")
		keys := sortedKeys(s.Env)
		for _, k := range keys {
			fmt.Fprintf(sb, "    %s %q\n", k, s.Env[k])
		}
		sb.WriteString("  }\n")
	}
	formatMetadata(sb, s.Metadata)
	sb.WriteString("}\n")
}

func formatMCPClient(sb *strings.Builder, c *ast.MCPClient) {
	fmt.Fprintf(sb, "client %q {\n", c.Name)
	for _, server := range c.Servers {
		fmt.Fprintf(sb, "  connects to server %q\n", server.Name)
	}
	formatMetadata(sb, c.Metadata)
	sb.WriteString("}\n")
}

func formatPluginRef(sb *strings.Builder, p *ast.PluginRef) {
	fmt.Fprintf(sb, "plugin %q", p.Name)
	if p.Version != "" {
		fmt.Fprintf(sb, " version %q", p.Version)
	}
	sb.WriteString("\n")
}

func formatMetadata(sb *strings.Builder, m map[string]string) {
	if len(m) == 0 {
		return
	}
	sb.WriteString("  metadata {\n")
	keys := sortedKeys(m)
	for _, k := range keys {
		fmt.Fprintf(sb, "    %s %q\n", k, m[k])
	}
	sb.WriteString("  }\n")
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
