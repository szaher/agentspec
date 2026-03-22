package controlflow

import (
	"context"
	"errors"
	"testing"
)

// Mock SkillInvoker for testing
type mockSkillInvoker struct {
	invocations []skillInvocation
	returnValue string
	returnError error
}

type skillInvocation struct {
	skillName string
	params    map[string]string
	input     interface{}
}

func (m *mockSkillInvoker) InvokeSkill(ctx context.Context, skillName string, params map[string]string, input interface{}) (string, error) {
	m.invocations = append(m.invocations, skillInvocation{
		skillName: skillName,
		params:    params,
		input:     input,
	})
	return m.returnValue, m.returnError
}

// Mock AgentDelegator for testing
type mockAgentDelegator struct {
	invocations []agentInvocation
	returnValue string
	returnError error
}

type agentInvocation struct {
	agentName string
	input     interface{}
}

func (m *mockAgentDelegator) DelegateToAgent(ctx context.Context, agentName string, input interface{}) (string, error) {
	m.invocations = append(m.invocations, agentInvocation{
		agentName: agentName,
		input:     input,
	})
	return m.returnValue, m.returnError
}

// TestNewRuntimeContext tests the creation of a new runtime context
func TestNewRuntimeContext(t *testing.T) {
	t.Run("with all parameters", func(t *testing.T) {
		input := "test input"
		session := map[string]interface{}{"key": "value"}
		config := map[string]interface{}{"param": "config"}

		rc := NewRuntimeContext(input, session, config)

		if rc.Input != input {
			t.Errorf("expected Input %v, got %v", input, rc.Input)
		}
		if rc.Session["key"] != "value" {
			t.Errorf("expected Session['key'] = 'value', got %v", rc.Session["key"])
		}
		if rc.Config["param"] != "config" {
			t.Errorf("expected Config['param'] = 'config', got %v", rc.Config["param"])
		}
		if rc.Steps == nil {
			t.Error("expected Steps to be initialized")
		}
		if rc.Variables == nil {
			t.Error("expected Variables to be initialized")
		}
	})

	t.Run("with nil session and config", func(t *testing.T) {
		rc := NewRuntimeContext("input", nil, nil)

		if rc.Session == nil {
			t.Error("expected Session to be initialized when nil")
		}
		if rc.Config == nil {
			t.Error("expected Config to be initialized when nil")
		}
	})
}

// TestRuntimeContextSetOutput tests setting output
func TestRuntimeContextSetOutput(t *testing.T) {
	rc := NewRuntimeContext("input", nil, nil)
	output := "test output"

	rc.SetOutput(output)

	if rc.Output != output {
		t.Errorf("expected Output %v, got %v", output, rc.Output)
	}
}

// TestRuntimeContextRecordStep tests recording steps
func TestRuntimeContextRecordStep(t *testing.T) {
	rc := NewRuntimeContext("input", nil, nil)

	rc.RecordStep("step1", "result1")
	rc.RecordStep("step2", "result2")

	if rc.Steps["step1"] != "result1" {
		t.Errorf("expected Steps['step1'] = 'result1', got %v", rc.Steps["step1"])
	}
	if rc.Steps["step2"] != "result2" {
		t.Errorf("expected Steps['step2'] = 'result2', got %v", rc.Steps["step2"])
	}
}

// TestRuntimeContextVariables tests variable operations
func TestRuntimeContextVariables(t *testing.T) {
	rc := NewRuntimeContext("input", nil, nil)

	rc.SetVariable("var1", "value1")
	rc.SetVariable("var2", 42)

	if rc.Variables["var1"] != "value1" {
		t.Errorf("expected Variables['var1'] = 'value1', got %v", rc.Variables["var1"])
	}
	if rc.Variables["var2"] != 42 {
		t.Errorf("expected Variables['var2'] = 42, got %v", rc.Variables["var2"])
	}

	rc.DeleteVariable("var1")

	if _, exists := rc.Variables["var1"]; exists {
		t.Error("expected var1 to be deleted")
	}
	if rc.Variables["var2"] != 42 {
		t.Error("expected var2 to still exist")
	}
}

// TestRuntimeContextToEnv tests environment map generation
func TestRuntimeContextToEnv(t *testing.T) {
	session := map[string]interface{}{"session_key": "session_value"}
	config := map[string]interface{}{"config_key": "config_value"}
	rc := NewRuntimeContext("test input", session, config)
	rc.SetOutput("test output")
	rc.RecordStep("step1", "result1")
	rc.SetVariable("loopVar", "loopValue")

	env := rc.ToEnv()

	if env["input"] != "test input" {
		t.Errorf("expected env['input'] = 'test input', got %v", env["input"])
	}
	if env["output"] != "test output" {
		t.Errorf("expected env['output'] = 'test output', got %v", env["output"])
	}
	if env["loopVar"] != "loopValue" {
		t.Errorf("expected env['loopVar'] = 'loopValue', got %v", env["loopVar"])
	}

	sessionMap, ok := env["session"].(map[string]interface{})
	if !ok || sessionMap["session_key"] != "session_value" {
		t.Error("expected session to be in env")
	}

	stepsMap, ok := env["steps"].(map[string]interface{})
	if !ok || stepsMap["step1"] != "result1" {
		t.Error("expected steps to be in env")
	}
}

// TestNewExecutor tests executor creation
func TestNewExecutor(t *testing.T) {
	skillInvoker := &mockSkillInvoker{}
	agentDelegator := &mockAgentDelegator{}

	exec := NewExecutor(skillInvoker, agentDelegator)

	if exec.skillInvoker != skillInvoker {
		t.Error("expected skillInvoker to be set")
	}
	if exec.agentDelegator != agentDelegator {
		t.Error("expected agentDelegator to be set")
	}
}

// TestExecuteUseSkill tests executing a use_skill statement
func TestExecuteUseSkill(t *testing.T) {
	t.Run("successful skill invocation", func(t *testing.T) {
		skillInvoker := &mockSkillInvoker{returnValue: "skill result"}
		exec := NewExecutor(skillInvoker, nil)
		rc := NewRuntimeContext("user input", nil, nil)

		stmt := map[string]interface{}{
			"type":  "use_skill",
			"skill": "test_skill",
			"params": map[string]interface{}{
				"param1": "value1",
				"param2": "value2",
			},
		}

		action, result, err := exec.executeUseSkill(context.Background(), stmt, rc)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if action.Type != "use_skill" {
			t.Errorf("expected action type 'use_skill', got %v", action.Type)
		}
		if action.SkillName != "test_skill" {
			t.Errorf("expected skill name 'test_skill', got %v", action.SkillName)
		}
		if action.Params["param1"] != "value1" {
			t.Errorf("expected param1 = 'value1', got %v", action.Params["param1"])
		}
		if result != "skill result" {
			t.Errorf("expected result 'skill result', got %v", result)
		}
		if len(skillInvoker.invocations) != 1 {
			t.Fatalf("expected 1 invocation, got %d", len(skillInvoker.invocations))
		}
		if skillInvoker.invocations[0].skillName != "test_skill" {
			t.Errorf("expected skill name 'test_skill', got %v", skillInvoker.invocations[0].skillName)
		}
	})

	t.Run("missing skill name", func(t *testing.T) {
		exec := NewExecutor(&mockSkillInvoker{}, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type": "use_skill",
		}

		_, _, err := exec.executeUseSkill(context.Background(), stmt, rc)

		if err == nil || err.Error() != "use_skill: missing skill name" {
			t.Errorf("expected missing skill name error, got %v", err)
		}
	})

	t.Run("no skill invoker configured", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type":  "use_skill",
			"skill": "test_skill",
		}

		_, _, err := exec.executeUseSkill(context.Background(), stmt, rc)

		if err == nil || err.Error() != "no skill invoker configured" {
			t.Errorf("expected no skill invoker error, got %v", err)
		}
	})

	t.Run("skill invocation error", func(t *testing.T) {
		skillInvoker := &mockSkillInvoker{returnError: errors.New("skill error")}
		exec := NewExecutor(skillInvoker, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type":  "use_skill",
			"skill": "test_skill",
		}

		_, _, err := exec.executeUseSkill(context.Background(), stmt, rc)

		if err == nil {
			t.Error("expected error from skill invocation")
		}
	})
}

// TestExecuteDelegate tests executing a delegate statement
func TestExecuteDelegate(t *testing.T) {
	t.Run("successful delegation", func(t *testing.T) {
		delegator := &mockAgentDelegator{returnValue: "agent result"}
		exec := NewExecutor(nil, delegator)
		rc := NewRuntimeContext("user input", nil, nil)

		stmt := map[string]interface{}{
			"type":  "delegate",
			"agent": "test_agent",
		}

		action, result, err := exec.executeDelegate(context.Background(), stmt, rc)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if action.Type != "delegate" {
			t.Errorf("expected action type 'delegate', got %v", action.Type)
		}
		if action.AgentName != "test_agent" {
			t.Errorf("expected agent name 'test_agent', got %v", action.AgentName)
		}
		if result != "agent result" {
			t.Errorf("expected result 'agent result', got %v", result)
		}
		if len(delegator.invocations) != 1 {
			t.Fatalf("expected 1 invocation, got %d", len(delegator.invocations))
		}
	})

	t.Run("missing agent name", func(t *testing.T) {
		exec := NewExecutor(nil, &mockAgentDelegator{})
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type": "delegate",
		}

		_, _, err := exec.executeDelegate(context.Background(), stmt, rc)

		if err == nil || err.Error() != "delegate: missing agent name" {
			t.Errorf("expected missing agent name error, got %v", err)
		}
	})

	t.Run("no agent delegator configured", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type":  "delegate",
			"agent": "test_agent",
		}

		_, _, err := exec.executeDelegate(context.Background(), stmt, rc)

		if err == nil || err.Error() != "no agent delegator configured" {
			t.Errorf("expected no agent delegator error, got %v", err)
		}
	})
}

// TestExecuteRespond tests executing a respond statement
func TestExecuteRespond(t *testing.T) {
	t.Run("literal string expression", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type":       "respond",
			"expression": "Hello, world!",
		}

		action, result, err := exec.executeRespond(stmt, rc)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if action.Type != "respond" {
			t.Errorf("expected action type 'respond', got %v", action.Type)
		}
		if result != "Hello, world!" {
			t.Errorf("expected result 'Hello, world!', got %v", result)
		}
	})

	t.Run("expression with input variable", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("test value", nil, nil)

		stmt := map[string]interface{}{
			"type":       "respond",
			"expression": "input",
		}

		_, result, err := exec.executeRespond(stmt, rc)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "test value" {
			t.Errorf("expected result 'test value', got %v", result)
		}
	})

	t.Run("missing expression", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type": "respond",
		}

		_, _, err := exec.executeRespond(stmt, rc)

		if err == nil || err.Error() != "respond: missing expression" {
			t.Errorf("expected missing expression error, got %v", err)
		}
	})
}

// TestExecuteIf tests executing if/else-if/else statements
func TestExecuteIf(t *testing.T) {
	t.Run("if condition true", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type":      "if",
			"condition": "1 == 1",
			"body": []interface{}{
				map[string]interface{}{
					"type":       "respond",
					"expression": "condition was true",
				},
			},
		}

		_, output, err := exec.executeIf(context.Background(), stmt, rc)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output != "condition was true" {
			t.Errorf("expected output 'condition was true', got %v", output)
		}
	})

	t.Run("if condition false, else-if true", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type":      "if",
			"condition": "1 == 2",
			"body": []interface{}{
				map[string]interface{}{
					"type":       "respond",
					"expression": "if body",
				},
			},
			"else_ifs": []interface{}{
				map[string]interface{}{
					"condition": "2 == 2",
					"body": []interface{}{
						map[string]interface{}{
							"type":       "respond",
							"expression": "else-if body",
						},
					},
				},
			},
		}

		_, output, err := exec.executeIf(context.Background(), stmt, rc)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output != "else-if body" {
			t.Errorf("expected output 'else-if body', got %v", output)
		}
	})

	t.Run("all conditions false, else body", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type":      "if",
			"condition": "false",
			"body": []interface{}{
				map[string]interface{}{
					"type":       "respond",
					"expression": "if body",
				},
			},
			"else_body": []interface{}{
				map[string]interface{}{
					"type":       "respond",
					"expression": "else body",
				},
			},
		}

		_, output, err := exec.executeIf(context.Background(), stmt, rc)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output != "else body" {
			t.Errorf("expected output 'else body', got %v", output)
		}
	})

	t.Run("missing condition", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type": "if",
			"body": []interface{}{},
		}

		_, _, err := exec.executeIf(context.Background(), stmt, rc)

		if err == nil || err.Error() != "if: missing condition" {
			t.Errorf("expected missing condition error, got %v", err)
		}
	})
}

// TestExecuteForEach tests executing for_each loops
func TestExecuteForEach(t *testing.T) {
	t.Run("iterate over slice", func(t *testing.T) {
		skillInvoker := &mockSkillInvoker{returnValue: "processed"}
		exec := NewExecutor(skillInvoker, nil)
		rc := NewRuntimeContext("input", nil, nil)
		rc.SetVariable("items", []interface{}{"a", "b", "c"})

		stmt := map[string]interface{}{
			"type":       "for_each",
			"variable":   "item",
			"collection": "items",
			"body": []interface{}{
				map[string]interface{}{
					"type":  "use_skill",
					"skill": "process",
				},
			},
		}

		actions, _, err := exec.executeForEach(context.Background(), stmt, rc)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(actions) != 3 {
			t.Errorf("expected 3 actions, got %d", len(actions))
		}
		if len(skillInvoker.invocations) != 3 {
			t.Errorf("expected 3 skill invocations, got %d", len(skillInvoker.invocations))
		}
	})

	t.Run("loop variable is cleaned up", func(t *testing.T) {
		exec := NewExecutor(&mockSkillInvoker{returnValue: "ok"}, nil)
		rc := NewRuntimeContext("input", nil, nil)
		rc.SetVariable("items", []interface{}{"a"})

		stmt := map[string]interface{}{
			"type":       "for_each",
			"variable":   "item",
			"collection": "items",
			"body": []interface{}{
				map[string]interface{}{
					"type":       "respond",
					"expression": "item",
				},
			},
		}

		_, _, err := exec.executeForEach(context.Background(), stmt, rc)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, exists := rc.Variables["item"]; exists {
			t.Error("expected loop variable to be cleaned up")
		}
	})

	t.Run("missing variable name", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type":       "for_each",
			"collection": "items",
		}

		_, _, err := exec.executeForEach(context.Background(), stmt, rc)

		if err == nil || err.Error() != "for_each: missing variable name" {
			t.Errorf("expected missing variable error, got %v", err)
		}
	})

	t.Run("missing collection", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmt := map[string]interface{}{
			"type":     "for_each",
			"variable": "item",
		}

		_, _, err := exec.executeForEach(context.Background(), stmt, rc)

		if err == nil || err.Error() != "for_each: missing collection expression" {
			t.Errorf("expected missing collection error, got %v", err)
		}
	})
}

// TestExecuteBlock tests executing a complete block of statements
func TestExecuteBlock(t *testing.T) {
	t.Run("empty block", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("input", nil, nil)

		actions, output, err := exec.ExecuteBlock(context.Background(), []interface{}{}, rc)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(actions) != 0 {
			t.Errorf("expected 0 actions, got %d", len(actions))
		}
		if output != "" {
			t.Errorf("expected empty output, got %v", output)
		}
	})

	t.Run("multiple statements", func(t *testing.T) {
		skillInvoker := &mockSkillInvoker{returnValue: "skill result"}
		exec := NewExecutor(skillInvoker, nil)
		rc := NewRuntimeContext("test input", nil, nil)

		stmts := []interface{}{
			map[string]interface{}{
				"type":  "use_skill",
				"skill": "skill1",
			},
			map[string]interface{}{
				"type":       "respond",
				"expression": "input",
			},
		}

		actions, output, err := exec.ExecuteBlock(context.Background(), stmts, rc)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(actions) != 2 {
			t.Errorf("expected 2 actions, got %d", len(actions))
		}
		if output != "test input" {
			t.Errorf("expected output 'test input', got %v", output)
		}
	})

	t.Run("unknown statement type", func(t *testing.T) {
		exec := NewExecutor(nil, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmts := []interface{}{
			map[string]interface{}{
				"type": "unknown_type",
			},
		}

		_, _, err := exec.ExecuteBlock(context.Background(), stmts, rc)

		if err == nil {
			t.Error("expected error for unknown statement type")
		}
	})

	t.Run("records steps and updates output", func(t *testing.T) {
		skillInvoker := &mockSkillInvoker{returnValue: "step result"}
		exec := NewExecutor(skillInvoker, nil)
		rc := NewRuntimeContext("input", nil, nil)

		stmts := []interface{}{
			map[string]interface{}{
				"type":  "use_skill",
				"skill": "myskill",
			},
		}

		_, _, err := exec.ExecuteBlock(context.Background(), stmts, rc)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rc.Steps["myskill"] != "step result" {
			t.Errorf("expected step 'myskill' to be recorded")
		}
		if rc.Output != "step result" {
			t.Errorf("expected output to be updated")
		}
	})
}

// TestToSlice tests the toSlice conversion function
func TestToSlice(t *testing.T) {
	t.Run("slice of interfaces", func(t *testing.T) {
		input := []interface{}{"a", "b", "c"}
		result, err := toSlice(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected length 3, got %d", len(result))
		}
	})

	t.Run("slice of strings", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result, err := toSlice(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected length 3, got %d", len(result))
		}
		if result[0] != "a" {
			t.Errorf("expected result[0] = 'a', got %v", result[0])
		}
	})

	t.Run("slice of ints", func(t *testing.T) {
		input := []int{1, 2, 3}
		result, err := toSlice(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected length 3, got %d", len(result))
		}
		if result[0] != 1 {
			t.Errorf("expected result[0] = 1, got %v", result[0])
		}
	})

	t.Run("slice of floats", func(t *testing.T) {
		input := []float64{1.5, 2.5, 3.5}
		result, err := toSlice(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected length 3, got %d", len(result))
		}
		if result[0] != 1.5 {
			t.Errorf("expected result[0] = 1.5, got %v", result[0])
		}
	})

	t.Run("nil value", func(t *testing.T) {
		result, err := toSlice(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result, got %v", result)
		}
	})

	t.Run("non-iterable type", func(t *testing.T) {
		input := 42
		_, err := toSlice(input)
		if err == nil {
			t.Error("expected error for non-iterable type")
		}
	})
}

// TestActionTypes tests the Action struct
func TestActionTypes(t *testing.T) {
	t.Run("use_skill action", func(t *testing.T) {
		a := Action{
			Type:      "use_skill",
			SkillName: "test_skill",
			Params:    map[string]string{"key": "value"},
			Result:    "result",
		}

		if a.Type != "use_skill" {
			t.Errorf("expected Type 'use_skill', got %v", a.Type)
		}
		if a.SkillName != "test_skill" {
			t.Errorf("expected SkillName 'test_skill', got %v", a.SkillName)
		}
	})

	t.Run("delegate action", func(t *testing.T) {
		a := Action{
			Type:      "delegate",
			AgentName: "test_agent",
			Result:    "result",
		}

		if a.Type != "delegate" {
			t.Errorf("expected Type 'delegate', got %v", a.Type)
		}
		if a.AgentName != "test_agent" {
			t.Errorf("expected AgentName 'test_agent', got %v", a.AgentName)
		}
	})

	t.Run("respond action", func(t *testing.T) {
		a := Action{
			Type:       "respond",
			Expression: "output",
			Result:     "result",
		}

		if a.Type != "respond" {
			t.Errorf("expected Type 'respond', got %v", a.Type)
		}
		if a.Expression != "output" {
			t.Errorf("expected Expression 'output', got %v", a.Expression)
		}
	})
}
