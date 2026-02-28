package parser

import (
	"testing"

	"github.com/szaher/designs/agentz/internal/ast"
)

func TestParseIntentLang3Imports(t *testing.T) {
	input := `package "test" version "3.0" lang "3.0"

import "./skills/search.ias"
import "github.com/agentspec/web-tools" version "1.2.0" as web
`

	f, errs := Parse(input, "test.ias")
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("parse error: %s", e)
		}
		t.FailNow()
	}

	if len(f.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(f.Statements))
	}

	// First import: local file
	imp1, ok := f.Statements[0].(*ast.Import)
	if !ok {
		t.Fatalf("expected *ast.Import, got %T", f.Statements[0])
	}
	if imp1.Path != "./skills/search.ias" {
		t.Errorf("import path = %q, want %q", imp1.Path, "./skills/search.ias")
	}
	if imp1.Version != "" {
		t.Errorf("import version = %q, want empty", imp1.Version)
	}
	if imp1.Alias != "" {
		t.Errorf("import alias = %q, want empty", imp1.Alias)
	}

	// Second import: package with version and alias
	imp2, ok := f.Statements[1].(*ast.Import)
	if !ok {
		t.Fatalf("expected *ast.Import, got %T", f.Statements[1])
	}
	if imp2.Path != "github.com/agentspec/web-tools" {
		t.Errorf("import path = %q, want %q", imp2.Path, "github.com/agentspec/web-tools")
	}
	if imp2.Version != "1.2.0" {
		t.Errorf("import version = %q, want %q", imp2.Version, "1.2.0")
	}
	if imp2.Alias != "web" {
		t.Errorf("import alias = %q, want %q", imp2.Alias, "web")
	}
}

func TestParseAgentConfigBlock(t *testing.T) {
	input := `package "test" version "3.0" lang "3.0"

agent "support" {
  model "claude-sonnet-4-20250514"

  config {
    api_key string required secret
      "API key for LLM access"

    max_length int default 2000
      "Maximum response length"

    debug_mode bool default false
      "Enable debug logging"

    email string default "test@example.com"
      "Contact email"
  }
}
`

	f, errs := Parse(input, "test.ias")
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("parse error: %s", e)
		}
		t.FailNow()
	}

	agent, ok := f.Statements[0].(*ast.Agent)
	if !ok {
		t.Fatalf("expected *ast.Agent, got %T", f.Statements[0])
	}

	if len(agent.ConfigParams) != 4 {
		t.Fatalf("expected 4 config params, got %d", len(agent.ConfigParams))
	}

	// api_key: string required secret
	p0 := agent.ConfigParams[0]
	if p0.Name != "api_key" || p0.Type != "string" || !p0.Required || !p0.Secret {
		t.Errorf("config[0] = {name:%q type:%q required:%v secret:%v}, want api_key/string/true/true",
			p0.Name, p0.Type, p0.Required, p0.Secret)
	}
	if p0.Description != "API key for LLM access" {
		t.Errorf("config[0].Description = %q", p0.Description)
	}

	// max_length: int default 2000
	p1 := agent.ConfigParams[1]
	if p1.Name != "max_length" || p1.Type != "int" || !p1.HasDefault || p1.Default != "2000" {
		t.Errorf("config[1] = {name:%q type:%q hasDefault:%v default:%q}",
			p1.Name, p1.Type, p1.HasDefault, p1.Default)
	}

	// debug_mode: bool default false
	p2 := agent.ConfigParams[2]
	if p2.Name != "debug_mode" || p2.Type != "bool" || !p2.HasDefault || p2.Default != "false" {
		t.Errorf("config[2] = {name:%q type:%q hasDefault:%v default:%q}",
			p2.Name, p2.Type, p2.HasDefault, p2.Default)
	}

	// email: string default "test@example.com"
	p3 := agent.ConfigParams[3]
	if p3.Name != "email" || p3.Type != "string" || !p3.HasDefault || p3.Default != "test@example.com" {
		t.Errorf("config[3] = {name:%q type:%q hasDefault:%v default:%q}",
			p3.Name, p3.Type, p3.HasDefault, p3.Default)
	}
}

func TestParseAgentValidateBlock(t *testing.T) {
	input := `package "test" version "3.0" lang "3.0"

agent "support" {
  model "claude-sonnet-4-20250514"

  validate {
    rule no_pii error max_retries 3
      "Response must not contain PII"
      when not (output matches "\\b\\d{3}-\\d{2}-\\d{4}\\b")

    rule tone_check warning
      "Response should maintain professional tone"
      when output.sentiment != "negative"
  }
}
`

	f, errs := Parse(input, "test.ias")
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("parse error: %s", e)
		}
		t.FailNow()
	}

	agent, ok := f.Statements[0].(*ast.Agent)
	if !ok {
		t.Fatalf("expected *ast.Agent, got %T", f.Statements[0])
	}

	if len(agent.ValidationRules) != 2 {
		t.Fatalf("expected 2 validation rules, got %d", len(agent.ValidationRules))
	}

	r0 := agent.ValidationRules[0]
	if r0.Name != "no_pii" || r0.Severity != "error" || r0.MaxRetries != 3 {
		t.Errorf("rule[0] = {name:%q severity:%q maxRetries:%d}", r0.Name, r0.Severity, r0.MaxRetries)
	}
	if r0.Message != "Response must not contain PII" {
		t.Errorf("rule[0].Message = %q", r0.Message)
	}
	if r0.Expression == "" {
		t.Error("rule[0].Expression is empty")
	}

	r1 := agent.ValidationRules[1]
	if r1.Name != "tone_check" || r1.Severity != "warning" || r1.MaxRetries != 0 {
		t.Errorf("rule[1] = {name:%q severity:%q maxRetries:%d}", r1.Name, r1.Severity, r1.MaxRetries)
	}
	if r1.Expression == "" {
		t.Error("rule[1].Expression is empty")
	}
}

func TestParseAgentEvalBlock(t *testing.T) {
	input := `package "test" version "3.0" lang "3.0"

agent "support" {
  model "claude-sonnet-4-20250514"

  eval {
    case greeting_test
      input "Hello, I need help"
      expect "greeting response"
      scoring semantic threshold 0.8
      tags ["smoke", "greeting"]

    case refund_test
      input "I want a refund"
      expect "refund acknowledgment"
      scoring exact
  }
}
`

	f, errs := Parse(input, "test.ias")
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("parse error: %s", e)
		}
		t.FailNow()
	}

	agent, ok := f.Statements[0].(*ast.Agent)
	if !ok {
		t.Fatalf("expected *ast.Agent, got %T", f.Statements[0])
	}

	if len(agent.EvalCases) != 2 {
		t.Fatalf("expected 2 eval cases, got %d", len(agent.EvalCases))
	}

	c0 := agent.EvalCases[0]
	if c0.Name != "greeting_test" {
		t.Errorf("case[0].Name = %q", c0.Name)
	}
	if c0.Input != "Hello, I need help" {
		t.Errorf("case[0].Input = %q", c0.Input)
	}
	if c0.Expected != "greeting response" {
		t.Errorf("case[0].Expected = %q", c0.Expected)
	}
	if c0.Scoring != "semantic" {
		t.Errorf("case[0].Scoring = %q", c0.Scoring)
	}
	if c0.Threshold != 0.8 {
		t.Errorf("case[0].Threshold = %f", c0.Threshold)
	}
	if len(c0.Tags) != 2 || c0.Tags[0] != "smoke" || c0.Tags[1] != "greeting" {
		t.Errorf("case[0].Tags = %v", c0.Tags)
	}

	c1 := agent.EvalCases[1]
	if c1.Name != "refund_test" || c1.Scoring != "exact" {
		t.Errorf("case[1] = {name:%q scoring:%q}", c1.Name, c1.Scoring)
	}
	if c1.Threshold != 0.8 { // default
		t.Errorf("case[1].Threshold = %f, want 0.8 (default)", c1.Threshold)
	}
}

func TestParseAgentOnInputBlock(t *testing.T) {
	input := `package "test" version "3.0" lang "3.0"

agent "router" {
  model "claude-sonnet-4-20250514"

  on input {
    use skill classify with { text: input.content }

    if steps.classify.output.category == "billing" {
      use skill billing_handler
    } else if steps.classify.output.category == "technical" {
      delegate to tech_agent
    } else {
      use skill general_support
    }

    for each source in input.data_sources {
      use skill fetch_data with { url: source.url }
    }

    respond "Thank you for your question"
  }
}
`

	f, errs := Parse(input, "test.ias")
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("parse error: %s", e)
		}
		t.FailNow()
	}

	agent, ok := f.Statements[0].(*ast.Agent)
	if !ok {
		t.Fatalf("expected *ast.Agent, got %T", f.Statements[0])
	}

	if agent.OnInput == nil {
		t.Fatal("agent.OnInput is nil")
	}

	stmts := agent.OnInput.Statements
	if len(stmts) != 4 {
		t.Fatalf("expected 4 on-input statements, got %d", len(stmts))
	}

	// Statement 1: use skill classify with { text: input.content }
	useStmt, ok := stmts[0].(*ast.UseSkillStmt)
	if !ok {
		t.Fatalf("stmt[0]: expected *ast.UseSkillStmt, got %T", stmts[0])
	}
	if useStmt.SkillName != "classify" {
		t.Errorf("stmt[0].SkillName = %q", useStmt.SkillName)
	}
	if useStmt.Params == nil || useStmt.Params["text"] == "" {
		t.Errorf("stmt[0].Params = %v, expected 'text' param", useStmt.Params)
	}

	// Statement 2: if/else if/else block
	ifStmt, ok := stmts[1].(*ast.IfBlock)
	if !ok {
		t.Fatalf("stmt[1]: expected *ast.IfBlock, got %T", stmts[1])
	}
	if ifStmt.Condition == "" {
		t.Error("stmt[1].Condition is empty")
	}
	if len(ifStmt.Body) != 1 {
		t.Errorf("stmt[1].Body has %d statements, want 1", len(ifStmt.Body))
	}
	if len(ifStmt.ElseIfs) != 1 {
		t.Errorf("stmt[1].ElseIfs has %d clauses, want 1", len(ifStmt.ElseIfs))
	}
	if ifStmt.ElseBody == nil {
		t.Error("stmt[1].ElseBody is nil")
	}

	// Verify else if body has delegate statement
	if len(ifStmt.ElseIfs[0].Body) == 1 {
		del, ok := ifStmt.ElseIfs[0].Body[0].(*ast.DelegateToStmt)
		if !ok {
			t.Errorf("else-if body: expected *ast.DelegateToStmt, got %T", ifStmt.ElseIfs[0].Body[0])
		} else if del.AgentName != "tech_agent" {
			t.Errorf("delegate agent = %q, want %q", del.AgentName, "tech_agent")
		}
	}

	// Statement 3: for each loop
	forStmt, ok := stmts[2].(*ast.ForEachBlock)
	if !ok {
		t.Fatalf("stmt[2]: expected *ast.ForEachBlock, got %T", stmts[2])
	}
	if forStmt.Variable != "source" {
		t.Errorf("stmt[2].Variable = %q", forStmt.Variable)
	}
	if forStmt.Collection == "" {
		t.Error("stmt[2].Collection is empty")
	}
	if len(forStmt.Body) != 1 {
		t.Errorf("stmt[2].Body has %d statements, want 1", len(forStmt.Body))
	}

	// Statement 4: respond
	respStmt, ok := stmts[3].(*ast.RespondStmt)
	if !ok {
		t.Fatalf("stmt[3]: expected *ast.RespondStmt, got %T", stmts[3])
	}
	if respStmt.Expression != "Thank you for your question" {
		t.Errorf("stmt[3].Expression = %q", respStmt.Expression)
	}
}

func TestParseAgentPromptShorthand(t *testing.T) {
	input := `package "test" version "3.0" lang "3.0"

agent "support" {
  prompt system_prompt
  model "claude-sonnet-4-20250514"
}
`

	f, errs := Parse(input, "test.ias")
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("parse error: %s", e)
		}
		t.FailNow()
	}

	agent, ok := f.Statements[0].(*ast.Agent)
	if !ok {
		t.Fatalf("expected *ast.Agent, got %T", f.Statements[0])
	}

	if agent.Prompt == nil {
		t.Fatal("agent.Prompt is nil")
	}
	if agent.Prompt.Name != "system_prompt" {
		t.Errorf("agent.Prompt.Name = %q, want %q", agent.Prompt.Name, "system_prompt")
	}
}

func TestLexerOperators(t *testing.T) {
	input := `== != >= <= > < :`
	lexer := NewLexer(input, "test.ias")
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	expected := []TokenType{
		TokenEqual, TokenNotEqual, TokenGreaterEq, TokenLessEq,
		TokenGreater, TokenLess, TokenColon, TokenEOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type != exp {
			t.Errorf("token[%d] = %s, want %s", i, tokens[i].Type, exp)
		}
	}
}
