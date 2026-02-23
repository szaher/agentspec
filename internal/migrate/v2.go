// Package migrate provides IntentLang 1.0 to 2.0 migration.
package migrate

import (
	"github.com/szaher/designs/agentz/internal/ast"
)

// ToV2 rewrites an IntentLang 1.0 AST to 2.0:
// - Sets lang version to "2.0"
// - Replaces Execution blocks with ToolConfig equivalents
// - Replaces Binding with DeployTarget
func ToV2(f *ast.File) *ast.File {
	if f.Package != nil {
		f.Package.LangVersion = "2.0"
	}

	var newStmts []ast.Statement
	for _, stmt := range f.Statements {
		switch s := stmt.(type) {
		case *ast.Skill:
			migrateSkill(s)
			newStmts = append(newStmts, s)
		case *ast.Binding:
			dt := migrateBinding(s)
			newStmts = append(newStmts, dt)
		default:
			newStmts = append(newStmts, stmt)
		}
	}
	f.Statements = newStmts
	return f
}

// migrateSkill converts execution blocks to tool blocks.
func migrateSkill(s *ast.Skill) {
	if s.Execution == nil || s.ToolConfig != nil {
		return
	}

	tc := &ast.ToolConfig{
		StartPos: s.Execution.StartPos,
		EndPos:   s.Execution.EndPos,
	}

	switch s.Execution.Type {
	case "command":
		tc.Type = "command"
		tc.Binary = s.Execution.Value
		tc.Args = s.Execution.Args
	case "http":
		tc.Type = "http"
		tc.URL = s.Execution.Value
	default:
		tc.Type = "command"
		tc.Binary = s.Execution.Value
	}

	s.ToolConfig = tc
	s.Execution = nil
}

// migrateBinding converts a Binding to a DeployTarget.
func migrateBinding(b *ast.Binding) *ast.DeployTarget {
	target := "process" // default target
	if b.Adapter == "docker" || b.Adapter == "docker-compose" || b.Adapter == "kubernetes" {
		target = b.Adapter
	}

	dt := &ast.DeployTarget{
		Name:     b.Name,
		Target:   target,
		Default:  b.Default,
		StartPos: b.StartPos,
		EndPos:   b.EndPos,
	}

	return dt
}
