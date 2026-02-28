package integration_tests

import (
	"context"
	"testing"

	"github.com/szaher/designs/agentz/internal/controlflow"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
)

// mockSkillInvoker returns canned responses for skill invocations.
type mockSkillInvoker struct {
	responses map[string]string
	calls     []string
}

func (m *mockSkillInvoker) InvokeSkill(_ context.Context, skillName string, _ map[string]string, _ interface{}) (string, error) {
	m.calls = append(m.calls, skillName)
	if resp, ok := m.responses[skillName]; ok {
		return resp, nil
	}
	return "default response", nil
}

// mockAgentDelegator returns canned responses for agent delegation.
type mockAgentDelegator struct {
	responses map[string]string
	calls     []string
}

func (m *mockAgentDelegator) DelegateToAgent(_ context.Context, agentName string, _ interface{}) (string, error) {
	m.calls = append(m.calls, agentName)
	if resp, ok := m.responses[agentName]; ok {
		return resp, nil
	}
	return "delegated response", nil
}

func TestControlFlowIfElse(t *testing.T) {
	invoker := &mockSkillInvoker{
		responses: map[string]string{
			"greet":    "Hello!",
			"search":   "Search results",
			"escalate": "Escalated",
		},
	}

	executor := controlflow.NewExecutor(invoker, nil)

	// Build an on_input block with if/else if/else
	stmts := []interface{}{
		map[string]interface{}{
			"type":      "if",
			"condition": `input == "hello"`,
			"body": []interface{}{
				map[string]interface{}{
					"type":  "use_skill",
					"skill": "greet",
				},
			},
			"else_ifs": []interface{}{
				map[string]interface{}{
					"condition": `input == "help"`,
					"body": []interface{}{
						map[string]interface{}{
							"type":  "use_skill",
							"skill": "search",
						},
					},
				},
			},
			"else_body": []interface{}{
				map[string]interface{}{
					"type":  "use_skill",
					"skill": "escalate",
				},
			},
		},
	}

	tests := []struct {
		name          string
		input         string
		expectedSkill string
		expectedOut   string
	}{
		{"if branch", "hello", "greet", "Hello!"},
		{"else-if branch", "help", "search", "Search results"},
		{"else branch", "something else", "escalate", "Escalated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoker.calls = nil
			rc := controlflow.NewRuntimeContext(tt.input, nil, nil)

			actions, output, err := executor.ExecuteBlock(context.Background(), stmts, rc)
			if err != nil {
				t.Fatalf("execution error: %v", err)
			}

			if len(actions) != 1 {
				t.Fatalf("expected 1 action, got %d", len(actions))
			}
			if actions[0].SkillName != tt.expectedSkill {
				t.Errorf("expected skill %q, got %q", tt.expectedSkill, actions[0].SkillName)
			}
			if output != tt.expectedOut {
				t.Errorf("expected output %q, got %q", tt.expectedOut, output)
			}
		})
	}
}

func TestControlFlowForEach(t *testing.T) {
	invoker := &mockSkillInvoker{
		responses: map[string]string{
			"process": "processed",
		},
	}

	executor := controlflow.NewExecutor(invoker, nil)

	// for each item in ["a", "b", "c"] { use skill "process" }
	stmts := []interface{}{
		map[string]interface{}{
			"type":       "for_each",
			"variable":   "item",
			"collection": `["a", "b", "c"]`,
			"body": []interface{}{
				map[string]interface{}{
					"type":  "use_skill",
					"skill": "process",
				},
			},
		},
	}

	rc := controlflow.NewRuntimeContext("test input", nil, nil)
	actions, _, err := executor.ExecuteBlock(context.Background(), stmts, rc)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if len(actions) != 3 {
		t.Fatalf("expected 3 actions (one per item), got %d", len(actions))
	}

	if len(invoker.calls) != 3 {
		t.Errorf("expected 3 skill calls, got %d", len(invoker.calls))
	}
}

func TestControlFlowRespond(t *testing.T) {
	executor := controlflow.NewExecutor(nil, nil)

	stmts := []interface{}{
		map[string]interface{}{
			"type":       "respond",
			"expression": `"Hello, " + input`,
		},
	}

	rc := controlflow.NewRuntimeContext("World", nil, nil)
	actions, output, err := executor.ExecuteBlock(context.Background(), stmts, rc)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Type != "respond" {
		t.Errorf("expected type 'respond', got %q", actions[0].Type)
	}
	if output != "Hello, World" {
		t.Errorf("expected 'Hello, World', got %q", output)
	}
}

func TestControlFlowDelegate(t *testing.T) {
	delegator := &mockAgentDelegator{
		responses: map[string]string{
			"specialist": "specialist response",
		},
	}

	executor := controlflow.NewExecutor(nil, delegator)

	stmts := []interface{}{
		map[string]interface{}{
			"type":  "delegate",
			"agent": "specialist",
		},
	}

	rc := controlflow.NewRuntimeContext("help me", nil, nil)
	actions, output, err := executor.ExecuteBlock(context.Background(), stmts, rc)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].AgentName != "specialist" {
		t.Errorf("expected agent 'specialist', got %q", actions[0].AgentName)
	}
	if output != "specialist response" {
		t.Errorf("expected 'specialist response', got %q", output)
	}
}

func TestControlFlowNestedIfInForEach(t *testing.T) {
	invoker := &mockSkillInvoker{
		responses: map[string]string{
			"process_a": "processed a",
			"process_b": "processed b",
			"default":   "processed other",
		},
	}

	executor := controlflow.NewExecutor(invoker, nil)

	// for each item in ["a", "b", "c"] {
	//   if item == "a" { use skill "process_a" }
	//   else if item == "b" { use skill "process_b" }
	//   else { use skill "default" }
	// }
	stmts := []interface{}{
		map[string]interface{}{
			"type":       "for_each",
			"variable":   "item",
			"collection": `["a", "b", "c"]`,
			"body": []interface{}{
				map[string]interface{}{
					"type":      "if",
					"condition": `item == "a"`,
					"body": []interface{}{
						map[string]interface{}{
							"type":  "use_skill",
							"skill": "process_a",
						},
					},
					"else_ifs": []interface{}{
						map[string]interface{}{
							"condition": `item == "b"`,
							"body": []interface{}{
								map[string]interface{}{
									"type":  "use_skill",
									"skill": "process_b",
								},
							},
						},
					},
					"else_body": []interface{}{
						map[string]interface{}{
							"type":  "use_skill",
							"skill": "default",
						},
					},
				},
			},
		},
	}

	rc := controlflow.NewRuntimeContext("test", nil, nil)
	actions, _, err := executor.ExecuteBlock(context.Background(), stmts, rc)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if len(actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(actions))
	}

	expected := []string{"process_a", "process_b", "default"}
	for i, exp := range expected {
		if actions[i].SkillName != exp {
			t.Errorf("action[%d]: expected skill %q, got %q", i, exp, actions[i].SkillName)
		}
	}
}

func TestControlFlowFromParsedIAS(t *testing.T) {
	// Test that an .ias file with on_input block can be parsed → lowered → executed
	content := `package "test" version "1.0.0" lang "3.0"

prompt "sys" {
  content "Test agent"
}

skill "greet" {
  description "Greet the user"

  input {
    name string required
  }

  output {
    greeting string required
  }

  tool command {
    binary "echo"
    args "hello"
  }
}

agent "router" {
  model "claude-sonnet-4-20250514"
  prompt "sys"
  strategy "react"
  max_turns 5

  uses skill "greet"

  on input {
    if input == "hi" {
      use skill "greet"
    } else {
      respond "Sorry, I cannot help with that"
    }
  }
}
`

	f, parseErrs := parser.Parse(content, "test.ias")
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("lower error: %v", err)
	}

	// Find agent resource
	var onInput []interface{}
	for _, r := range doc.Resources {
		if r.Kind == "Agent" && r.Name == "router" {
			if oi, ok := r.Attributes["on_input"].([]interface{}); ok {
				onInput = oi
			}
		}
	}

	if onInput == nil {
		t.Fatal("on_input not found in lowered IR")
	}

	// Execute the control flow
	invoker := &mockSkillInvoker{
		responses: map[string]string{"greet": "Hello!"},
	}
	executor := controlflow.NewExecutor(invoker, nil)

	// Test "hi" input → should invoke greet skill
	rc := controlflow.NewRuntimeContext("hi", nil, nil)
	actions, output, err := executor.ExecuteBlock(context.Background(), onInput, rc)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if len(actions) != 1 || actions[0].SkillName != "greet" {
		t.Errorf("expected greet skill, got %v", actions)
	}
	if output != "Hello!" {
		t.Errorf("expected 'Hello!', got %q", output)
	}

	// Test "bye" input → should respond with "Sorry, I cannot help with that"
	rc2 := controlflow.NewRuntimeContext("bye", nil, nil)
	actions2, output2, err := executor.ExecuteBlock(context.Background(), onInput, rc2)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if len(actions2) != 1 || actions2[0].Type != "respond" {
		t.Errorf("expected respond action, got %v", actions2)
	}
	if output2 != "Sorry, I cannot help with that" {
		t.Errorf("expected 'Sorry, I cannot help with that', got %q", output2)
	}
}

func TestRuntimeContextVariables(t *testing.T) {
	rc := controlflow.NewRuntimeContext("test input", nil, map[string]interface{}{
		"api_key": "sk-test",
	})

	// Test input evaluation
	result, err := rc.EvalBool(`input == "test input"`)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if !result {
		t.Error("expected true for input match")
	}

	// Test config access
	configResult, err := rc.EvalExpr(`config["api_key"]`)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if configResult != "sk-test" {
		t.Errorf("expected 'sk-test', got %v", configResult)
	}

	// Test step recording
	rc.RecordStep("step1", "step1 output")
	stepResult, err := rc.EvalExpr(`steps["step1"]`)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if stepResult != "step1 output" {
		t.Errorf("expected 'step1 output', got %v", stepResult)
	}

	// Test variable setting (loop variables)
	rc.SetVariable("item", "current_item")
	varResult, err := rc.EvalExpr(`item`)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if varResult != "current_item" {
		t.Errorf("expected 'current_item', got %v", varResult)
	}

	// Test variable cleanup
	rc.DeleteVariable("item")
}
