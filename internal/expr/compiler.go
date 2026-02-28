// Package expr provides compile-time validation and runtime evaluation
// of expressions used in IntentLang 3.0 control flow and validation rules.
package expr

import (
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// CompiledExpr represents a compiled expression ready for evaluation.
type CompiledExpr struct {
	Source  string
	program *vm.Program
}

// Compile validates and compiles an expression string for later evaluation.
// The env parameter defines the available variables and their types.
func Compile(source string, env map[string]interface{}) (*CompiledExpr, error) {
	if source == "" {
		return nil, fmt.Errorf("empty expression")
	}

	program, err := expr.Compile(source, expr.Env(env))
	if err != nil {
		return nil, fmt.Errorf("expression compile error: %w", err)
	}

	return &CompiledExpr{
		Source:  source,
		program: program,
	}, nil
}

// CompileUnchecked compiles an expression without type checking.
// Use this when the runtime environment shape is not known at compile time.
func CompileUnchecked(source string) (*CompiledExpr, error) {
	if source == "" {
		return nil, fmt.Errorf("empty expression")
	}

	program, err := expr.Compile(source)
	if err != nil {
		return nil, fmt.Errorf("expression compile error: %w", err)
	}

	return &CompiledExpr{
		Source:  source,
		program: program,
	}, nil
}

// ValidateSyntax checks if an expression is syntactically valid without
// compiling it against a specific environment.
func ValidateSyntax(source string) error {
	if source == "" {
		return fmt.Errorf("empty expression")
	}
	_, err := expr.Compile(source)
	if err != nil {
		return fmt.Errorf("invalid expression syntax: %w", err)
	}
	return nil
}
