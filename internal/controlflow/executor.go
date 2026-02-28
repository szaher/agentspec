package controlflow

import (
	"context"
	"fmt"
)

// Action represents the result of executing a control flow statement.
type Action struct {
	Type       string            // "use_skill", "delegate", "respond"
	SkillName  string            // for use_skill
	AgentName  string            // for delegate
	Expression string            // for respond
	Params     map[string]string // for use_skill with params
	Result     interface{}       // evaluated result (for respond)
}

// SkillInvoker is called when a "use skill" statement is executed.
type SkillInvoker interface {
	InvokeSkill(ctx context.Context, skillName string, params map[string]string, input interface{}) (string, error)
}

// AgentDelegator is called when a "delegate to" statement is executed.
type AgentDelegator interface {
	DelegateToAgent(ctx context.Context, agentName string, input interface{}) (string, error)
}

// Executor runs control flow statements from an agent's on_input block.
type Executor struct {
	skillInvoker   SkillInvoker
	agentDelegator AgentDelegator
}

// NewExecutor creates a new control flow executor.
func NewExecutor(skillInvoker SkillInvoker, agentDelegator AgentDelegator) *Executor {
	return &Executor{
		skillInvoker:   skillInvoker,
		agentDelegator: agentDelegator,
	}
}

// ExecuteBlock executes an on_input block (a list of IR control flow instructions).
// Each instruction is a map with a "type" key and type-specific fields.
// Returns the list of actions taken and the final output.
func (e *Executor) ExecuteBlock(ctx context.Context, stmts []interface{}, rc *RuntimeContext) ([]Action, string, error) {
	var actions []Action
	var output string

	for _, stmt := range stmts {
		m, ok := stmt.(map[string]interface{})
		if !ok {
			continue
		}

		stmtType, _ := m["type"].(string)

		switch stmtType {
		case "use_skill":
			action, result, err := e.executeUseSkill(ctx, m, rc)
			if err != nil {
				return actions, output, err
			}
			actions = append(actions, action)
			output = result
			rc.RecordStep(action.SkillName, result)
			rc.SetOutput(result)

		case "delegate":
			action, result, err := e.executeDelegate(ctx, m, rc)
			if err != nil {
				return actions, output, err
			}
			actions = append(actions, action)
			output = result
			rc.SetOutput(result)

		case "respond":
			action, result, err := e.executeRespond(m, rc)
			if err != nil {
				return actions, output, err
			}
			actions = append(actions, action)
			output = result
			rc.SetOutput(result)

		case "if":
			ifActions, ifOutput, err := e.executeIf(ctx, m, rc)
			if err != nil {
				return actions, output, err
			}
			actions = append(actions, ifActions...)
			if ifOutput != "" {
				output = ifOutput
			}

		case "for_each":
			forActions, forOutput, err := e.executeForEach(ctx, m, rc)
			if err != nil {
				return actions, output, err
			}
			actions = append(actions, forActions...)
			if forOutput != "" {
				output = forOutput
			}

		default:
			return actions, output, fmt.Errorf("unknown control flow statement type: %q", stmtType)
		}
	}

	return actions, output, nil
}

func (e *Executor) executeUseSkill(ctx context.Context, m map[string]interface{}, rc *RuntimeContext) (Action, string, error) {
	skillName, _ := m["skill"].(string)
	if skillName == "" {
		return Action{}, "", fmt.Errorf("use_skill: missing skill name")
	}

	// Extract params
	var params map[string]string
	if p, ok := m["params"].(map[string]interface{}); ok {
		params = make(map[string]string)
		for k, v := range p {
			params[k], _ = v.(string)
		}
	}

	action := Action{
		Type:      "use_skill",
		SkillName: skillName,
		Params:    params,
	}

	if e.skillInvoker == nil {
		return action, "", fmt.Errorf("no skill invoker configured")
	}

	result, err := e.skillInvoker.InvokeSkill(ctx, skillName, params, rc.Input)
	if err != nil {
		return action, "", fmt.Errorf("invoking skill %q: %w", skillName, err)
	}

	action.Result = result
	return action, result, nil
}

func (e *Executor) executeDelegate(ctx context.Context, m map[string]interface{}, rc *RuntimeContext) (Action, string, error) {
	agentName, _ := m["agent"].(string)
	if agentName == "" {
		return Action{}, "", fmt.Errorf("delegate: missing agent name")
	}

	action := Action{
		Type:      "delegate",
		AgentName: agentName,
	}

	if e.agentDelegator == nil {
		return action, "", fmt.Errorf("no agent delegator configured")
	}

	result, err := e.agentDelegator.DelegateToAgent(ctx, agentName, rc.Input)
	if err != nil {
		return action, "", fmt.Errorf("delegating to agent %q: %w", agentName, err)
	}

	action.Result = result
	return action, result, nil
}

func (e *Executor) executeRespond(m map[string]interface{}, rc *RuntimeContext) (Action, string, error) {
	expression, _ := m["expression"].(string)
	if expression == "" {
		return Action{}, "", fmt.Errorf("respond: missing expression")
	}

	action := Action{
		Type:       "respond",
		Expression: expression,
	}

	result, err := rc.EvalExpr(expression)
	if err != nil {
		// If expression evaluation fails, treat it as a literal string.
		// This handles the case where the parser stripped quotes from a string literal
		// in a `respond "text"` statement.
		action.Result = expression
		return action, expression, nil
	}

	output := fmt.Sprintf("%v", result)
	action.Result = result
	return action, output, nil
}

func (e *Executor) executeIf(ctx context.Context, m map[string]interface{}, rc *RuntimeContext) ([]Action, string, error) {
	condition, _ := m["condition"].(string)
	if condition == "" {
		return nil, "", fmt.Errorf("if: missing condition")
	}

	// Evaluate the condition
	result, err := rc.EvalBool(condition)
	if err != nil {
		return nil, "", fmt.Errorf("if condition: %w", err)
	}

	if result {
		// Execute the if body
		body, _ := m["body"].([]interface{})
		return e.ExecuteBlock(ctx, body, rc)
	}

	// Check else-if clauses
	if elseIfs, ok := m["else_ifs"].([]interface{}); ok {
		for _, ei := range elseIfs {
			eiMap, ok := ei.(map[string]interface{})
			if !ok {
				continue
			}
			eiCond, _ := eiMap["condition"].(string)
			if eiCond == "" {
				continue
			}

			eiResult, err := rc.EvalBool(eiCond)
			if err != nil {
				return nil, "", fmt.Errorf("else-if condition: %w", err)
			}

			if eiResult {
				body, _ := eiMap["body"].([]interface{})
				return e.ExecuteBlock(ctx, body, rc)
			}
		}
	}

	// Execute else body if present
	if elseBody, ok := m["else_body"].([]interface{}); ok && len(elseBody) > 0 {
		return e.ExecuteBlock(ctx, elseBody, rc)
	}

	return nil, "", nil
}

func (e *Executor) executeForEach(ctx context.Context, m map[string]interface{}, rc *RuntimeContext) ([]Action, string, error) {
	variable, _ := m["variable"].(string)
	if variable == "" {
		return nil, "", fmt.Errorf("for_each: missing variable name")
	}

	collection, _ := m["collection"].(string)
	if collection == "" {
		return nil, "", fmt.Errorf("for_each: missing collection expression")
	}

	// Evaluate the collection expression
	collResult, err := rc.EvalExpr(collection)
	if err != nil {
		return nil, "", fmt.Errorf("for_each collection: %w", err)
	}

	// Convert to iterable
	items, err := toSlice(collResult)
	if err != nil {
		return nil, "", fmt.Errorf("for_each: collection is not iterable: %w", err)
	}

	body, _ := m["body"].([]interface{})
	var allActions []Action
	var lastOutput string

	for _, item := range items {
		// Set loop variable
		rc.SetVariable(variable, item)

		actions, output, err := e.ExecuteBlock(ctx, body, rc)
		if err != nil {
			rc.DeleteVariable(variable)
			return allActions, lastOutput, err
		}

		allActions = append(allActions, actions...)
		if output != "" {
			lastOutput = output
		}
	}

	// Clean up loop variable
	rc.DeleteVariable(variable)

	return allActions, lastOutput, nil
}

// toSlice converts an interface{} to a []interface{}.
func toSlice(v interface{}) ([]interface{}, error) {
	switch val := v.(type) {
	case []interface{}:
		return val, nil
	case []string:
		result := make([]interface{}, len(val))
		for i, s := range val {
			result[i] = s
		}
		return result, nil
	case []int:
		result := make([]interface{}, len(val))
		for i, n := range val {
			result[i] = n
		}
		return result, nil
	case []float64:
		result := make([]interface{}, len(val))
		for i, n := range val {
			result[i] = n
		}
		return result, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("cannot iterate over %T", v)
	}
}
