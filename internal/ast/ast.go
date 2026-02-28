// Package ast defines the abstract syntax tree node types for the
// IntentLang DSL (.ias/.az files).
package ast

// Pos represents a position in source code.
type Pos struct {
	File   string
	Line   int
	Column int
}

// Node is the interface implemented by all AST nodes.
type Node interface {
	Pos() Pos
	End() Pos
}

// File represents a parsed IntentLang AgentSpec (.ias/.az) file.
type File struct {
	Path       string
	Package    *Package
	Statements []Statement
	StartPos   Pos
	EndPos     Pos
}

func (f *File) Pos() Pos { return f.StartPos }
func (f *File) End() Pos { return f.EndPos }

// Statement is the interface for top-level statements.
type Statement interface {
	Node
	stmtNode()
}

// Package declares the package header.
type Package struct {
	Name        string
	Version     string
	LangVersion string
	Description string
	Imports     []*Import
	Plugins     []*PluginRef
	StartPos    Pos
	EndPos      Pos
}

func (p *Package) Pos() Pos  { return p.StartPos }
func (p *Package) End() Pos  { return p.EndPos }
func (p *Package) stmtNode() {}

// Import declares an external package dependency.
type Import struct {
	Path     string
	Version  string
	SHA      string
	Alias    string // IntentLang 3.0: optional import alias (import "..." as alias)
	StartPos Pos
	EndPos   Pos
}

func (i *Import) Pos() Pos  { return i.StartPos }
func (i *Import) End() Pos  { return i.EndPos }
func (i *Import) stmtNode() {}

// PluginRef declares a plugin dependency.
type PluginRef struct {
	Name       string
	Version    string
	HooksOrder []string
	StartPos   Pos
	EndPos     Pos
}

func (p *PluginRef) Pos() Pos  { return p.StartPos }
func (p *PluginRef) End() Pos  { return p.EndPos }
func (p *PluginRef) stmtNode() {}

// Agent defines an agent resource.
type Agent struct {
	Name       string
	Prompt     *Ref
	Skills     []*Ref
	Model      string
	Parameters map[string]string
	Client     *Ref
	Metadata   map[string]string

	// IntentLang 2.0 runtime config
	Strategy    string // "react", "plan-and-execute", "reflexion", "router", "map-reduce"
	MaxTurns    int
	Timeout     string
	TokenBudget int
	Temperature float64
	HasTemp     bool // distinguish zero value from unset
	Stream      *bool
	OnError     string // "retry", "fail", "fallback"
	MaxRetries  int
	Fallback    string // fallback agent name
	MemoryCfg   *MemoryConfig
	Delegates   []*Delegate

	// IntentLang 3.0: agent compilation extensions
	ConfigParams    []*ConfigParam
	ValidationRules []*ValidationRule
	EvalCases       []*EvalCase
	OnInput         *OnInputBlock

	StartPos Pos
	EndPos   Pos
}

func (a *Agent) Pos() Pos  { return a.StartPos }
func (a *Agent) End() Pos  { return a.EndPos }
func (a *Agent) stmtNode() {}

// Prompt defines a prompt resource.
type Prompt struct {
	Name      string
	Content   string
	Variables []*Variable
	Version   string
	Metadata  map[string]string
	StartPos  Pos
	EndPos    Pos
}

func (p *Prompt) Pos() Pos  { return p.StartPos }
func (p *Prompt) End() Pos  { return p.EndPos }
func (p *Prompt) stmtNode() {}

// Variable declares a template variable in a prompt.
type Variable struct {
	Name     string
	Type     string
	Default  string
	Required bool
	StartPos Pos
	EndPos   Pos
}

func (v *Variable) Pos() Pos { return v.StartPos }
func (v *Variable) End() Pos { return v.EndPos }

// Skill defines a skill resource.
type Skill struct {
	Name        string
	Description string
	Input       []*Field
	Output      []*Field
	Execution   *Execution  // IntentLang 1.0 execution
	ToolConfig  *ToolConfig // IntentLang 2.0 tool block
	Metadata    map[string]string
	StartPos    Pos
	EndPos      Pos
}

func (s *Skill) Pos() Pos  { return s.StartPos }
func (s *Skill) End() Pos  { return s.EndPos }
func (s *Skill) stmtNode() {}

// Field represents a typed field in input/output schemas.
type Field struct {
	Name     string
	Type     string
	Required bool
	StartPos Pos
	EndPos   Pos
}

func (f *Field) Pos() Pos { return f.StartPos }
func (f *Field) End() Pos { return f.EndPos }

// Execution specifies how a skill runs.
type Execution struct {
	Type     string // "command", "http", etc.
	Value    string
	Args     []string
	StartPos Pos
	EndPos   Pos
}

func (e *Execution) Pos() Pos { return e.StartPos }
func (e *Execution) End() Pos { return e.EndPos }

// MCPServer defines an MCP server resource.
type MCPServer struct {
	Name      string
	Transport string // "stdio", "sse", "streamable-http"
	Command   string
	Args      []string
	URL       string
	Auth      *Ref
	Skills    []*Ref
	Env       map[string]string
	Metadata  map[string]string
	StartPos  Pos
	EndPos    Pos
}

func (m *MCPServer) Pos() Pos  { return m.StartPos }
func (m *MCPServer) End() Pos  { return m.EndPos }
func (m *MCPServer) stmtNode() {}

// MCPClient defines an MCP client resource.
type MCPClient struct {
	Name     string
	Servers  []*Ref
	Metadata map[string]string
	StartPos Pos
	EndPos   Pos
}

func (m *MCPClient) Pos() Pos  { return m.StartPos }
func (m *MCPClient) End() Pos  { return m.EndPos }
func (m *MCPClient) stmtNode() {}

// Environment defines an environment overlay.
type Environment struct {
	Name      string
	Overrides []*Override
	StartPos  Pos
	EndPos    Pos
}

func (e *Environment) Pos() Pos  { return e.StartPos }
func (e *Environment) End() Pos  { return e.EndPos }
func (e *Environment) stmtNode() {}

// Override specifies an attribute override within an environment.
type Override struct {
	Resource  string
	Attribute string
	Value     string
	StartPos  Pos
	EndPos    Pos
}

func (o *Override) Pos() Pos { return o.StartPos }
func (o *Override) End() Pos { return o.EndPos }

// Secret defines a secret reference.
type Secret struct {
	Name     string
	Source   string // "env" or "store"
	Key      string
	StartPos Pos
	EndPos   Pos
}

func (s *Secret) Pos() Pos  { return s.StartPos }
func (s *Secret) End() Pos  { return s.EndPos }
func (s *Secret) stmtNode() {}

// Policy defines security constraint rules.
type Policy struct {
	Name     string
	Rules    []*Rule
	StartPos Pos
	EndPos   Pos
}

func (p *Policy) Pos() Pos  { return p.StartPos }
func (p *Policy) End() Pos  { return p.EndPos }
func (p *Policy) stmtNode() {}

// Rule defines a single policy rule.
type Rule struct {
	Action   string // "allow", "deny", "require"
	Resource string // resource type pattern
	Subject  string // what is constrained
	StartPos Pos
	EndPos   Pos
}

func (r *Rule) Pos() Pos { return r.StartPos }
func (r *Rule) End() Pos { return r.EndPos }

// Binding defines a target adapter binding.
type Binding struct {
	Name     string
	Adapter  string
	Default  bool
	Config   map[string]string
	StartPos Pos
	EndPos   Pos
}

func (b *Binding) Pos() Pos  { return b.StartPos }
func (b *Binding) End() Pos  { return b.EndPos }
func (b *Binding) stmtNode() {}

// Plugin declares a custom resource from a plugin.
type Plugin struct {
	Name     string
	Version  string
	StartPos Pos
	EndPos   Pos
}

func (p *Plugin) Pos() Pos  { return p.StartPos }
func (p *Plugin) End() Pos  { return p.EndPos }
func (p *Plugin) stmtNode() {}

// Ref is a reference to another resource.
type Ref struct {
	Kind     string // "prompt", "skill", "server", etc.
	Name     string
	Package  string // empty if local
	StartPos Pos
	EndPos   Pos
}

func (r *Ref) Pos() Pos { return r.StartPos }
func (r *Ref) End() Pos { return r.EndPos }

// ToolConfig defines how a skill executes (IntentLang 2.0, replaces Execution).
type ToolConfig struct {
	Type string // "mcp", "http", "command", "inline"

	// MCP variant
	ServerTool string // "server-name/tool-name"

	// HTTP variant
	Method       string
	URL          string
	Headers      map[string]string
	BodyTemplate string

	// Command variant
	Binary  string
	Args    []string
	Timeout string
	Env     map[string]string
	Secrets map[string]string

	// Inline variant
	Language    string
	Code        string
	MemoryLimit string

	StartPos Pos
	EndPos   Pos
}

func (t *ToolConfig) Pos() Pos { return t.StartPos }
func (t *ToolConfig) End() Pos { return t.EndPos }

// DeployTarget defines a deployment configuration (IntentLang 2.0, replaces Binding).
type DeployTarget struct {
	Name      string
	Target    string // "process", "docker", "docker-compose", "kubernetes"
	Default   bool
	Port      int
	Namespace string
	Replicas  int
	Image     string
	Resources *ResourceLimits
	Health    *HealthConfig
	Autoscale *AutoscaleConfig
	Env       map[string]string
	Secrets   map[string]string
	StartPos  Pos
	EndPos    Pos
}

func (d *DeployTarget) Pos() Pos  { return d.StartPos }
func (d *DeployTarget) End() Pos  { return d.EndPos }
func (d *DeployTarget) stmtNode() {}

// MemoryConfig specifies conversation memory settings.
type MemoryConfig struct {
	Strategy    string // "sliding_window", "summary"
	MaxMessages int
	StartPos    Pos
	EndPos      Pos
}

func (m *MemoryConfig) Pos() Pos { return m.StartPos }
func (m *MemoryConfig) End() Pos { return m.EndPos }

// HealthConfig specifies health check settings.
type HealthConfig struct {
	Path     string
	Interval string
	Timeout  string
	StartPos Pos
	EndPos   Pos
}

func (h *HealthConfig) Pos() Pos { return h.StartPos }
func (h *HealthConfig) End() Pos { return h.EndPos }

// AutoscaleConfig specifies horizontal scaling rules.
type AutoscaleConfig struct {
	MinReplicas int
	MaxReplicas int
	Metric      string
	Target      int
	StartPos    Pos
	EndPos      Pos
}

func (a *AutoscaleConfig) Pos() Pos { return a.StartPos }
func (a *AutoscaleConfig) End() Pos { return a.EndPos }

// ResourceLimits specifies CPU/memory constraints.
type ResourceLimits struct {
	CPU      string
	Memory   string
	StartPos Pos
	EndPos   Pos
}

func (r *ResourceLimits) Pos() Pos { return r.StartPos }
func (r *ResourceLimits) End() Pos { return r.EndPos }

// Delegate defines an agent delegation rule.
type Delegate struct {
	AgentRef  string // target agent name
	Condition string // natural language condition
	StartPos  Pos
	EndPos    Pos
}

func (d *Delegate) Pos() Pos { return d.StartPos }
func (d *Delegate) End() Pos { return d.EndPos }

// TypeDef defines a named type with fields, enums, or lists.
type TypeDef struct {
	Name     string
	Fields   []*TypeField
	EnumVals []string // non-empty for enum types
	ListOf   string   // non-empty for list types
	StartPos Pos
	EndPos   Pos
}

func (t *TypeDef) Pos() Pos  { return t.StartPos }
func (t *TypeDef) End() Pos  { return t.EndPos }
func (t *TypeDef) stmtNode() {}

// TypeField defines a field within a type definition.
type TypeField struct {
	Name     string
	Type     string
	Required bool
	Default  string
	StartPos Pos
	EndPos   Pos
}

func (f *TypeField) Pos() Pos { return f.StartPos }
func (f *TypeField) End() Pos { return f.EndPos }

// Pipeline defines a multi-step agent workflow.
type Pipeline struct {
	Name     string
	Steps    []*PipelineStep
	StartPos Pos
	EndPos   Pos
}

func (p *Pipeline) Pos() Pos  { return p.StartPos }
func (p *Pipeline) End() Pos  { return p.EndPos }
func (p *Pipeline) stmtNode() {}

// PipelineStep defines a single step in a pipeline.
type PipelineStep struct {
	Name      string
	Agent     string   // agent to invoke
	Input     string   // input expression or reference
	Output    string   // output variable name
	DependsOn []string // step names this step depends on
	Parallel  bool     // can run in parallel
	When      string   // conditional execution expression
	StartPos  Pos
	EndPos    Pos
}

func (s *PipelineStep) Pos() Pos { return s.StartPos }
func (s *PipelineStep) End() Pos { return s.EndPos }

// ---------------------------------------------------------------------------
// IntentLang 3.0: Agent compilation extensions
// ---------------------------------------------------------------------------

// ConfigParam declares a runtime configuration parameter within an agent.
type ConfigParam struct {
	Name        string
	Type        string // "string", "int", "float", "bool"
	Description string
	Required    bool
	Secret      bool
	Default     string
	HasDefault  bool // distinguish empty string default from unset
	StartPos    Pos
	EndPos      Pos
}

func (c *ConfigParam) Pos() Pos { return c.StartPos }
func (c *ConfigParam) End() Pos { return c.EndPos }

// ValidationRule declares an output validation rule within an agent.
type ValidationRule struct {
	Name       string
	Severity   string // "error" or "warning"
	MaxRetries int
	Message    string
	Expression string // "when" expression (expr syntax)
	StartPos   Pos
	EndPos     Pos
}

func (v *ValidationRule) Pos() Pos { return v.StartPos }
func (v *ValidationRule) End() Pos { return v.EndPos }

// EvalCase declares an evaluation test case within an agent.
type EvalCase struct {
	Name      string
	Input     string
	Expected  string
	Scoring   string  // "exact", "contains", "semantic", "custom"
	Threshold float64 // similarity threshold (default 0.8)
	Tags      []string
	StartPos  Pos
	EndPos    Pos
}

func (e *EvalCase) Pos() Pos { return e.StartPos }
func (e *EvalCase) End() Pos { return e.EndPos }

// OnInputBlock defines the agent's request processing flow.
type OnInputBlock struct {
	Statements []OnInputStmt
	StartPos   Pos
	EndPos     Pos
}

func (o *OnInputBlock) Pos() Pos { return o.StartPos }
func (o *OnInputBlock) End() Pos { return o.EndPos }

// OnInputStmt is the interface for statements within an on input block.
type OnInputStmt interface {
	Node
	onInputStmtNode()
}

// UseSkillStmt invokes a skill with optional parameters.
type UseSkillStmt struct {
	SkillName string
	Params    map[string]string // key-value pairs for "with { ... }"
	StartPos  Pos
	EndPos    Pos
}

func (u *UseSkillStmt) Pos() Pos        { return u.StartPos }
func (u *UseSkillStmt) End() Pos        { return u.EndPos }
func (u *UseSkillStmt) onInputStmtNode() {}

// DelegateToStmt hands off processing to another agent.
type DelegateToStmt struct {
	AgentName string
	StartPos  Pos
	EndPos    Pos
}

func (d *DelegateToStmt) Pos() Pos        { return d.StartPos }
func (d *DelegateToStmt) End() Pos        { return d.EndPos }
func (d *DelegateToStmt) onInputStmtNode() {}

// RespondStmt returns a response directly.
type RespondStmt struct {
	Expression string
	StartPos   Pos
	EndPos     Pos
}

func (r *RespondStmt) Pos() Pos        { return r.StartPos }
func (r *RespondStmt) End() Pos        { return r.EndPos }
func (r *RespondStmt) onInputStmtNode() {}

// IfBlock represents an if/else if/else conditional block.
type IfBlock struct {
	Condition  string // expr expression
	Body       []OnInputStmt
	ElseIfs    []*ElseIfClause
	ElseBody   []OnInputStmt // nil if no else block
	StartPos   Pos
	EndPos     Pos
}

func (i *IfBlock) Pos() Pos        { return i.StartPos }
func (i *IfBlock) End() Pos        { return i.EndPos }
func (i *IfBlock) onInputStmtNode() {}

// ElseIfClause represents an else if clause.
type ElseIfClause struct {
	Condition string
	Body      []OnInputStmt
	StartPos  Pos
	EndPos    Pos
}

func (e *ElseIfClause) Pos() Pos { return e.StartPos }
func (e *ElseIfClause) End() Pos { return e.EndPos }

// ForEachBlock represents a for each iteration loop.
type ForEachBlock struct {
	Variable   string // loop variable name
	Collection string // expr expression for the collection
	Body       []OnInputStmt
	StartPos   Pos
	EndPos     Pos
}

func (f *ForEachBlock) Pos() Pos        { return f.StartPos }
func (f *ForEachBlock) End() Pos        { return f.EndPos }
func (f *ForEachBlock) onInputStmtNode() {}
