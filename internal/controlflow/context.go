// Package controlflow provides runtime execution of IntentLang 3.0
// control flow constructs (if/else, for each) within agent on_input blocks.
package controlflow

import (
	"fmt"

	"github.com/szaher/designs/agentz/internal/expr"
)

// RuntimeContext holds the runtime variables available to control flow expressions.
// It wraps the expr.Context and provides helper methods for managing state
// during on_input block execution.
type RuntimeContext struct {
	// Input is the raw user input string or structured data.
	Input interface{}
	// Session holds session state across turns.
	Session map[string]interface{}
	// Steps records outputs from each executed step (keyed by step index or skill name).
	Steps map[string]interface{}
	// Config holds resolved configuration parameters.
	Config map[string]interface{}
	// Output accumulates the response being built.
	Output interface{}
	// Variables holds loop variables and other temporary state.
	Variables map[string]interface{}
}

// NewRuntimeContext creates a new runtime context with the given input.
func NewRuntimeContext(input interface{}, session, config map[string]interface{}) *RuntimeContext {
	if session == nil {
		session = make(map[string]interface{})
	}
	if config == nil {
		config = make(map[string]interface{})
	}
	return &RuntimeContext{
		Input:     input,
		Session:   session,
		Steps:     make(map[string]interface{}),
		Config:    config,
		Variables: make(map[string]interface{}),
	}
}

// SetOutput sets the current output value.
func (rc *RuntimeContext) SetOutput(output interface{}) {
	rc.Output = output
}

// RecordStep records the output of a named step.
func (rc *RuntimeContext) RecordStep(name string, output interface{}) {
	rc.Steps[name] = output
}

// SetVariable sets a temporary variable (e.g., loop variable).
func (rc *RuntimeContext) SetVariable(name string, value interface{}) {
	rc.Variables[name] = value
}

// DeleteVariable removes a temporary variable.
func (rc *RuntimeContext) DeleteVariable(name string) {
	delete(rc.Variables, name)
}

// ToExprContext converts the runtime context to an expr.Context for expression evaluation.
func (rc *RuntimeContext) ToExprContext() *expr.Context {
	return &expr.Context{
		Input:   rc.Input,
		Session: rc.Session,
		Steps:   rc.Steps,
		Config:  rc.Config,
		Output:  rc.Output,
	}
}

// ToEnv converts the runtime context to a flat environment map for expression evaluation.
// This merges variables into the top-level namespace so loop variables are accessible.
func (rc *RuntimeContext) ToEnv() map[string]interface{} {
	env := map[string]interface{}{
		"input":   rc.Input,
		"session": rc.Session,
		"steps":   rc.Steps,
		"config":  rc.Config,
		"output":  rc.Output,
	}

	// Merge variables into the environment (for loop variables, etc.)
	for k, v := range rc.Variables {
		env[k] = v
	}

	return env
}

// EvalBool evaluates a boolean expression against this context.
func (rc *RuntimeContext) EvalBool(expression string) (bool, error) {
	env := rc.ToEnv()

	result, err := expr.EvalWithEnv(expression, env)
	if err != nil {
		return false, fmt.Errorf("evaluating condition %q: %w", expression, err)
	}

	b, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("condition %q returned %T, expected bool", expression, result)
	}
	return b, nil
}

// EvalExpr evaluates an expression against this context and returns the result.
func (rc *RuntimeContext) EvalExpr(expression string) (interface{}, error) {
	env := rc.ToEnv()
	return expr.EvalWithEnv(expression, env)
}
