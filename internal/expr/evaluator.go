package expr

import (
	"fmt"

	"github.com/expr-lang/expr"
)

// Context holds the runtime variables available to expressions.
type Context struct {
	Input   interface{}            `expr:"input"`
	Session interface{}            `expr:"session"`
	Steps   map[string]interface{} `expr:"steps"`
	Config  map[string]interface{} `expr:"config"`
	Output  interface{}            `expr:"output"`
}

// Eval evaluates a compiled expression against the given context.
func Eval(compiled *CompiledExpr, ctx *Context) (interface{}, error) {
	if compiled == nil || compiled.program == nil {
		return nil, fmt.Errorf("nil compiled expression")
	}

	env := map[string]interface{}{
		"input":   ctx.Input,
		"session": ctx.Session,
		"steps":   ctx.Steps,
		"config":  ctx.Config,
		"output":  ctx.Output,
	}

	result, err := expr.Run(compiled.program, env)
	if err != nil {
		return nil, fmt.Errorf("expression eval error for %q: %w", compiled.Source, err)
	}
	return result, nil
}

// EvalBool evaluates a compiled expression and returns a boolean result.
// Returns an error if the expression does not evaluate to a boolean.
func EvalBool(compiled *CompiledExpr, ctx *Context) (bool, error) {
	result, err := Eval(compiled, ctx)
	if err != nil {
		return false, err
	}

	b, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("expression %q returned %T, expected bool", compiled.Source, result)
	}
	return b, nil
}

// EvalString evaluates an expression source string directly against a context.
// This is a convenience function that compiles and evaluates in one step.
func EvalString(source string, ctx *Context) (interface{}, error) {
	env := map[string]interface{}{
		"input":   ctx.Input,
		"session": ctx.Session,
		"steps":   ctx.Steps,
		"config":  ctx.Config,
		"output":  ctx.Output,
	}

	result, err := expr.Eval(source, env)
	if err != nil {
		return nil, fmt.Errorf("expression eval error for %q: %w", source, err)
	}
	return result, nil
}

// EvalWithEnv evaluates an expression against a flat environment map.
// This supports custom variables like loop variables merged into the namespace.
func EvalWithEnv(source string, env map[string]interface{}) (interface{}, error) {
	result, err := expr.Eval(source, env)
	if err != nil {
		return nil, fmt.Errorf("expression eval error for %q: %w", source, err)
	}
	return result, nil
}
