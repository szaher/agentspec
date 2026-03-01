// Package formatter implements canonical formatting of IntentLang (.ias/.az) files
// from AST back to source, with deterministic output.
package formatter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/szaher/designs/agentz/internal/ast"
)

// Format formats an AST File back to canonical IntentLang (.ias/.az) source.
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
	case *ast.DeployTarget:
		formatDeployTarget(sb, s)
	case *ast.PluginRef:
		formatPluginRef(sb, s)
	case *ast.TypeDef:
		formatTypeDef(sb, s)
	case *ast.Pipeline:
		formatPipeline(sb, s)
	case *ast.Import:
		formatImport(sb, s)
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
	if len(p.Variables) > 0 {
		sb.WriteString("  variables {\n")
		for _, v := range p.Variables {
			fmt.Fprintf(sb, "    %s %s", v.Name, v.Type)
			if v.Required {
				sb.WriteString(" required")
			}
			if v.Default != "" {
				fmt.Fprintf(sb, " default %q", v.Default)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("  }\n")
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
	if s.ToolConfig != nil {
		formatToolConfig(sb, s.ToolConfig)
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
	if a.Strategy != "" {
		fmt.Fprintf(sb, "  strategy %q\n", a.Strategy)
	}
	if a.MaxTurns > 0 {
		fmt.Fprintf(sb, "  max_turns %d\n", a.MaxTurns)
	}
	if a.Timeout != "" {
		fmt.Fprintf(sb, "  timeout %q\n", a.Timeout)
	}
	if a.TokenBudget > 0 {
		fmt.Fprintf(sb, "  token_budget %d\n", a.TokenBudget)
	}
	if a.HasTemp {
		fmt.Fprintf(sb, "  temperature %g\n", a.Temperature)
	}
	if a.Stream != nil {
		fmt.Fprintf(sb, "  stream %t\n", *a.Stream)
	}
	if a.OnError != "" {
		fmt.Fprintf(sb, "  on_error %q\n", a.OnError)
	}
	if a.MaxRetries > 0 {
		fmt.Fprintf(sb, "  max_retries %d\n", a.MaxRetries)
	}
	if a.Fallback != "" {
		fmt.Fprintf(sb, "  fallback %q\n", a.Fallback)
	}
	if a.MemoryCfg != nil {
		sb.WriteString("  memory {\n")
		if a.MemoryCfg.Strategy != "" {
			fmt.Fprintf(sb, "    strategy %q\n", a.MemoryCfg.Strategy)
		}
		if a.MemoryCfg.MaxMessages > 0 {
			fmt.Fprintf(sb, "    max_messages %d\n", a.MemoryCfg.MaxMessages)
		}
		sb.WriteString("  }\n")
	}
	for _, d := range a.Delegates {
		fmt.Fprintf(sb, "  delegate to agent %q when %q\n", d.AgentRef, d.Condition)
	}
	// IntentLang 3.0: config, validate, eval, on input
	if len(a.ConfigParams) > 0 {
		formatConfigBlock(sb, a.ConfigParams, "  ")
	}
	if len(a.ValidationRules) > 0 {
		formatValidateBlock(sb, a.ValidationRules, "  ")
	}
	if len(a.EvalCases) > 0 {
		formatEvalBlock(sb, a.EvalCases, "  ")
	}
	if a.OnInput != nil {
		formatOnInputBlock(sb, a.OnInput, "  ")
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

func formatToolConfig(sb *strings.Builder, tc *ast.ToolConfig) {
	switch tc.Type {
	case "mcp":
		fmt.Fprintf(sb, "  tool mcp %q\n", tc.ServerTool)
	case "http":
		sb.WriteString("  tool http {\n")
		if tc.Method != "" {
			fmt.Fprintf(sb, "    method %q\n", tc.Method)
		}
		if tc.URL != "" {
			fmt.Fprintf(sb, "    url %q\n", tc.URL)
		}
		if len(tc.Headers) > 0 {
			sb.WriteString("    headers {\n")
			keys := sortedKeys(tc.Headers)
			for _, k := range keys {
				fmt.Fprintf(sb, "      %s %q\n", k, tc.Headers[k])
			}
			sb.WriteString("    }\n")
		}
		if tc.BodyTemplate != "" {
			fmt.Fprintf(sb, "    body_template %q\n", tc.BodyTemplate)
		}
		if tc.Timeout != "" {
			fmt.Fprintf(sb, "    timeout %q\n", tc.Timeout)
		}
		sb.WriteString("  }\n")
	case "command":
		sb.WriteString("  tool command {\n")
		if tc.Binary != "" {
			fmt.Fprintf(sb, "    binary %q\n", tc.Binary)
		}
		if len(tc.Args) > 0 {
			sb.WriteString("    args [")
			for i, arg := range tc.Args {
				if i > 0 {
					sb.WriteString(", ")
				}
				fmt.Fprintf(sb, "%q", arg)
			}
			sb.WriteString("]\n")
		}
		if tc.Timeout != "" {
			fmt.Fprintf(sb, "    timeout %q\n", tc.Timeout)
		}
		if len(tc.Env) > 0 {
			sb.WriteString("    env {\n")
			keys := sortedKeys(tc.Env)
			for _, k := range keys {
				fmt.Fprintf(sb, "      %s %q\n", k, tc.Env[k])
			}
			sb.WriteString("    }\n")
		}
		if len(tc.Secrets) > 0 {
			sb.WriteString("    secrets {\n")
			keys := sortedKeys(tc.Secrets)
			for _, k := range keys {
				fmt.Fprintf(sb, "      %s %q\n", k, tc.Secrets[k])
			}
			sb.WriteString("    }\n")
		}
		sb.WriteString("  }\n")
	case "inline":
		sb.WriteString("  tool inline {\n")
		if tc.Language != "" {
			fmt.Fprintf(sb, "    language %q\n", tc.Language)
		}
		if tc.Code != "" {
			fmt.Fprintf(sb, "    code %q\n", tc.Code)
		}
		if tc.Timeout != "" {
			fmt.Fprintf(sb, "    timeout %q\n", tc.Timeout)
		}
		if tc.MemoryLimit != "" {
			fmt.Fprintf(sb, "    memory %q\n", tc.MemoryLimit)
		}
		sb.WriteString("  }\n")
	}
}

func formatDeployTarget(sb *strings.Builder, d *ast.DeployTarget) {
	fmt.Fprintf(sb, "deploy %q target %q {\n", d.Name, d.Target)
	if d.Port > 0 {
		fmt.Fprintf(sb, "  port %d\n", d.Port)
	}
	if d.Default {
		sb.WriteString("  default true\n")
	}
	if d.Namespace != "" {
		fmt.Fprintf(sb, "  namespace %q\n", d.Namespace)
	}
	if d.Replicas > 0 {
		fmt.Fprintf(sb, "  replicas %d\n", d.Replicas)
	}
	if d.Image != "" {
		fmt.Fprintf(sb, "  image %q\n", d.Image)
	}
	if d.Resources != nil {
		sb.WriteString("  resources {\n")
		if d.Resources.CPU != "" {
			fmt.Fprintf(sb, "    cpu %q\n", d.Resources.CPU)
		}
		if d.Resources.Memory != "" {
			fmt.Fprintf(sb, "    memory %q\n", d.Resources.Memory)
		}
		sb.WriteString("  }\n")
	}
	if d.Health != nil {
		sb.WriteString("  health {\n")
		if d.Health.Path != "" {
			fmt.Fprintf(sb, "    path %q\n", d.Health.Path)
		}
		if d.Health.Interval != "" {
			fmt.Fprintf(sb, "    interval %q\n", d.Health.Interval)
		}
		if d.Health.Timeout != "" {
			fmt.Fprintf(sb, "    timeout %q\n", d.Health.Timeout)
		}
		sb.WriteString("  }\n")
	}
	if d.Autoscale != nil {
		sb.WriteString("  autoscale {\n")
		if d.Autoscale.MinReplicas > 0 {
			fmt.Fprintf(sb, "    min %d\n", d.Autoscale.MinReplicas)
		}
		if d.Autoscale.MaxReplicas > 0 {
			fmt.Fprintf(sb, "    max %d\n", d.Autoscale.MaxReplicas)
		}
		if d.Autoscale.Metric != "" {
			fmt.Fprintf(sb, "    metric %q\n", d.Autoscale.Metric)
		}
		if d.Autoscale.Target > 0 {
			fmt.Fprintf(sb, "    target %d\n", d.Autoscale.Target)
		}
		sb.WriteString("  }\n")
	}
	if len(d.Env) > 0 {
		sb.WriteString("  env {\n")
		keys := sortedKeys(d.Env)
		for _, k := range keys {
			fmt.Fprintf(sb, "    %s %q\n", k, d.Env[k])
		}
		sb.WriteString("  }\n")
	}
	if len(d.Secrets) > 0 {
		sb.WriteString("  secrets {\n")
		keys := sortedKeys(d.Secrets)
		for _, k := range keys {
			fmt.Fprintf(sb, "    %s %q\n", k, d.Secrets[k])
		}
		sb.WriteString("  }\n")
	}
	sb.WriteString("}\n")
}

func formatTypeDef(sb *strings.Builder, t *ast.TypeDef) {
	switch {
	case len(t.EnumVals) > 0:
		fmt.Fprintf(sb, "type %q enum [", t.Name)
		for i, v := range t.EnumVals {
			if i > 0 {
				sb.WriteString(", ")
			}
			fmt.Fprintf(sb, "%q", v)
		}
		sb.WriteString("]\n")
	case t.ListOf != "":
		fmt.Fprintf(sb, "type %q list %s\n", t.Name, t.ListOf)
	default:
		fmt.Fprintf(sb, "type %q {\n", t.Name)
		for _, f := range t.Fields {
			fmt.Fprintf(sb, "  %s %s", f.Name, f.Type)
			if f.Required {
				sb.WriteString(" required")
			}
			if f.Default != "" {
				fmt.Fprintf(sb, " default %q", f.Default)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("}\n")
	}
}

func formatPipeline(sb *strings.Builder, p *ast.Pipeline) {
	fmt.Fprintf(sb, "pipeline %q {\n", p.Name)
	for _, step := range p.Steps {
		fmt.Fprintf(sb, "  step %q {\n", step.Name)
		if step.Agent != "" {
			fmt.Fprintf(sb, "    agent %q\n", step.Agent)
		}
		if step.Input != "" {
			fmt.Fprintf(sb, "    input %q\n", step.Input)
		}
		if step.Output != "" {
			fmt.Fprintf(sb, "    output %q\n", step.Output)
		}
		if len(step.DependsOn) > 0 {
			sb.WriteString("    depends_on [")
			for i, dep := range step.DependsOn {
				if i > 0 {
					sb.WriteString(", ")
				}
				fmt.Fprintf(sb, "%q", dep)
			}
			sb.WriteString("]\n")
		}
		if step.Parallel {
			sb.WriteString("    parallel true\n")
		}
		if step.When != "" {
			fmt.Fprintf(sb, "    when %q\n", step.When)
		}
		sb.WriteString("  }\n")
	}
	sb.WriteString("}\n")
}

// ---------------------------------------------------------------------------
// IntentLang 3.0: Import, Config, Validate, Eval, On Input formatters
// ---------------------------------------------------------------------------

func formatImport(sb *strings.Builder, imp *ast.Import) {
	fmt.Fprintf(sb, "import %q", imp.Path)
	if imp.Version != "" {
		fmt.Fprintf(sb, " version %q", imp.Version)
	}
	if imp.Alias != "" {
		fmt.Fprintf(sb, " as %s", imp.Alias)
	}
	sb.WriteString("\n")
}

func formatConfigBlock(sb *strings.Builder, params []*ast.ConfigParam, indent string) {
	sb.WriteString(indent + "config {\n")
	for _, p := range params {
		fmt.Fprintf(sb, "%s  %s %s", indent, p.Name, p.Type)
		if p.Required {
			sb.WriteString(" required")
		}
		if p.Secret {
			sb.WriteString(" secret")
		}
		if p.HasDefault {
			fmt.Fprintf(sb, " default %q", p.Default)
		}
		sb.WriteString("\n")
		if p.Description != "" {
			fmt.Fprintf(sb, "%s    %q\n", indent, p.Description)
		}
	}
	sb.WriteString(indent + "}\n")
}

func formatValidateBlock(sb *strings.Builder, rules []*ast.ValidationRule, indent string) {
	sb.WriteString(indent + "validate {\n")
	for _, r := range rules {
		fmt.Fprintf(sb, "%s  rule %s %s", indent, r.Name, r.Severity)
		if r.MaxRetries > 0 {
			fmt.Fprintf(sb, " max_retries %d", r.MaxRetries)
		}
		sb.WriteString("\n")
		if r.Message != "" {
			fmt.Fprintf(sb, "%s    %q\n", indent, r.Message)
		}
		if r.Expression != "" {
			fmt.Fprintf(sb, "%s    when %s\n", indent, r.Expression)
		}
	}
	sb.WriteString(indent + "}\n")
}

func formatEvalBlock(sb *strings.Builder, cases []*ast.EvalCase, indent string) {
	sb.WriteString(indent + "eval {\n")
	for _, c := range cases {
		fmt.Fprintf(sb, "%s  case %s\n", indent, c.Name)
		fmt.Fprintf(sb, "%s    input %q\n", indent, c.Input)
		fmt.Fprintf(sb, "%s    expect %q\n", indent, c.Expected)
		if c.Scoring != "" {
			fmt.Fprintf(sb, "%s    scoring %s", indent, c.Scoring)
			if c.Threshold != 0.8 {
				fmt.Fprintf(sb, " threshold %g", c.Threshold)
			}
			sb.WriteString("\n")
		}
		if len(c.Tags) > 0 {
			fmt.Fprintf(sb, "%s    tags [", indent)
			for i, tag := range c.Tags {
				if i > 0 {
					sb.WriteString(", ")
				}
				fmt.Fprintf(sb, "%q", tag)
			}
			sb.WriteString("]\n")
		}
	}
	sb.WriteString(indent + "}\n")
}

func formatOnInputBlock(sb *strings.Builder, block *ast.OnInputBlock, indent string) {
	sb.WriteString(indent + "on input {\n")
	for _, stmt := range block.Statements {
		formatOnInputStmt(sb, stmt, indent+"  ")
	}
	sb.WriteString(indent + "}\n")
}

func formatOnInputStmt(sb *strings.Builder, stmt ast.OnInputStmt, indent string) {
	switch s := stmt.(type) {
	case *ast.UseSkillStmt:
		fmt.Fprintf(sb, "%suse skill %s", indent, s.SkillName)
		if len(s.Params) > 0 {
			sb.WriteString(" with { ")
			keys := make([]string, 0, len(s.Params))
			for k := range s.Params {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for i, k := range keys {
				if i > 0 {
					sb.WriteString(", ")
				}
				fmt.Fprintf(sb, "%s: %s", k, s.Params[k])
			}
			sb.WriteString(" }")
		}
		sb.WriteString("\n")
	case *ast.DelegateToStmt:
		fmt.Fprintf(sb, "%sdelegate to %s\n", indent, s.AgentName)
	case *ast.RespondStmt:
		fmt.Fprintf(sb, "%srespond %q\n", indent, s.Expression)
	case *ast.IfBlock:
		formatIfBlock(sb, s, indent)
	case *ast.ForEachBlock:
		formatForEachBlock(sb, s, indent)
	}
}

func formatIfBlock(sb *strings.Builder, block *ast.IfBlock, indent string) {
	fmt.Fprintf(sb, "%sif %s {\n", indent, block.Condition)
	for _, stmt := range block.Body {
		formatOnInputStmt(sb, stmt, indent+"  ")
	}
	sb.WriteString(indent + "}")
	for _, elseIf := range block.ElseIfs {
		fmt.Fprintf(sb, " else if %s {\n", elseIf.Condition)
		for _, stmt := range elseIf.Body {
			formatOnInputStmt(sb, stmt, indent+"  ")
		}
		sb.WriteString(indent + "}")
	}
	if block.ElseBody != nil {
		sb.WriteString(" else {\n")
		for _, stmt := range block.ElseBody {
			formatOnInputStmt(sb, stmt, indent+"  ")
		}
		sb.WriteString(indent + "}")
	}
	sb.WriteString("\n")
}

func formatForEachBlock(sb *strings.Builder, block *ast.ForEachBlock, indent string) {
	fmt.Fprintf(sb, "%sfor each %s in %s {\n", indent, block.Variable, block.Collection)
	for _, stmt := range block.Body {
		formatOnInputStmt(sb, stmt, indent+"  ")
	}
	sb.WriteString(indent + "}\n")
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
