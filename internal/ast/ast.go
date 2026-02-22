// Package ast defines the abstract syntax tree node types for the
// Agentz DSL (.az files).
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

// File represents a parsed .az file.
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
	StartPos Pos
	EndPos   Pos
}

func (i *Import) Pos() Pos { return i.StartPos }
func (i *Import) End() Pos { return i.EndPos }

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
	StartPos   Pos
	EndPos     Pos
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
	Execution   *Execution
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
