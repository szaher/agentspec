package expr

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Compile
// ---------------------------------------------------------------------------

func TestCompile_ValidExpression(t *testing.T) {
	env := map[string]interface{}{
		"x": 0,
		"y": 0,
	}
	compiled, err := Compile("x + y", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compiled == nil {
		t.Fatal("compiled should not be nil")
	}
	if compiled.Source != "x + y" {
		t.Errorf("source: got %q, want %q", compiled.Source, "x + y")
	}
}

func TestCompile_EmptyExpression(t *testing.T) {
	_, err := Compile("", nil)
	if err == nil {
		t.Fatal("expected error for empty expression")
	}
}

func TestCompile_InvalidSyntax(t *testing.T) {
	env := map[string]interface{}{"x": 0}
	_, err := Compile("x ++ +", env)
	if err == nil {
		t.Fatal("expected error for invalid syntax")
	}
}

// ---------------------------------------------------------------------------
// CompileUnchecked
// ---------------------------------------------------------------------------

func TestCompileUnchecked_ValidExpression(t *testing.T) {
	compiled, err := CompileUnchecked("1 + 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compiled == nil {
		t.Fatal("compiled should not be nil")
	}
	if compiled.Source != "1 + 2" {
		t.Errorf("source: got %q, want %q", compiled.Source, "1 + 2")
	}
}

func TestCompileUnchecked_EmptyExpression(t *testing.T) {
	_, err := CompileUnchecked("")
	if err == nil {
		t.Fatal("expected error for empty expression")
	}
}

func TestCompileUnchecked_InvalidSyntax(t *testing.T) {
	_, err := CompileUnchecked(")(")
	if err == nil {
		t.Fatal("expected error for invalid syntax")
	}
}

// ---------------------------------------------------------------------------
// ValidateSyntax
// ---------------------------------------------------------------------------

func TestValidateSyntax(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{name: "valid arithmetic", source: "1 + 2", wantErr: false},
		{name: "valid comparison", source: "x > 10", wantErr: false},
		{name: "valid boolean", source: "true && false", wantErr: false},
		{name: "valid string", source: `"hello" + " " + "world"`, wantErr: false},
		{name: "empty", source: "", wantErr: true},
		{name: "invalid syntax", source: ")(", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSyntax(tc.source)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Eval
// ---------------------------------------------------------------------------

func TestEval_Arithmetic(t *testing.T) {
	compiled, err := CompileUnchecked("1 + 2")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	ctx := &Context{
		Steps:  map[string]interface{}{},
		Config: map[string]interface{}{},
	}
	result, err := Eval(compiled, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}

	// expr-lang may return int
	if result != 3 {
		t.Errorf("got %v (%T), want 3", result, result)
	}
}

func TestEval_StringConcatenation(t *testing.T) {
	compiled, err := CompileUnchecked(`"hello" + " " + "world"`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	ctx := &Context{
		Steps:  map[string]interface{}{},
		Config: map[string]interface{}{},
	}
	result, err := Eval(compiled, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if result != "hello world" {
		t.Errorf("got %v, want %q", result, "hello world")
	}
}

func TestEval_BooleanLogic(t *testing.T) {
	compiled, err := CompileUnchecked("true && !false")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	ctx := &Context{
		Steps:  map[string]interface{}{},
		Config: map[string]interface{}{},
	}
	result, err := Eval(compiled, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if result != true {
		t.Errorf("got %v, want true", result)
	}
}

func TestEval_NilCompiled(t *testing.T) {
	ctx := &Context{
		Steps:  map[string]interface{}{},
		Config: map[string]interface{}{},
	}
	_, err := Eval(nil, ctx)
	if err == nil {
		t.Fatal("expected error for nil compiled expression")
	}
}

// ---------------------------------------------------------------------------
// EvalBool
// ---------------------------------------------------------------------------

func TestEvalBool_TrueExpression(t *testing.T) {
	compiled, err := CompileUnchecked("10 > 5")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	ctx := &Context{
		Steps:  map[string]interface{}{},
		Config: map[string]interface{}{},
	}
	result, err := EvalBool(compiled, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if !result {
		t.Error("expected true")
	}
}

func TestEvalBool_FalseExpression(t *testing.T) {
	compiled, err := CompileUnchecked("1 > 100")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	ctx := &Context{
		Steps:  map[string]interface{}{},
		Config: map[string]interface{}{},
	}
	result, err := EvalBool(compiled, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if result {
		t.Error("expected false")
	}
}

func TestEvalBool_NonBoolResult(t *testing.T) {
	compiled, err := CompileUnchecked("1 + 2")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	ctx := &Context{
		Steps:  map[string]interface{}{},
		Config: map[string]interface{}{},
	}
	_, err = EvalBool(compiled, ctx)
	if err == nil {
		t.Fatal("expected error for non-boolean result")
	}
}

// ---------------------------------------------------------------------------
// EvalString
// ---------------------------------------------------------------------------

func TestEvalString(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   interface{}
	}{
		{name: "arithmetic", source: "2 * 3", want: 6},
		{name: "string", source: `"test"`, want: "test"},
		{name: "boolean", source: "true", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &Context{
				Steps:  map[string]interface{}{},
				Config: map[string]interface{}{},
			}
			result, err := EvalString(tc.source, ctx)
			if err != nil {
				t.Fatalf("eval: %v", err)
			}
			if result != tc.want {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tc.want, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EvalWithEnv
// ---------------------------------------------------------------------------

func TestEvalWithEnv_CustomVariables(t *testing.T) {
	env := map[string]interface{}{
		"x":    10,
		"y":    20,
		"name": "alice",
	}

	tests := []struct {
		name   string
		source string
		want   interface{}
	}{
		{name: "add variables", source: "x + y", want: 30},
		{name: "string variable", source: `name + " bob"`, want: "alice bob"},
		{name: "comparison", source: "x < y", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := EvalWithEnv(tc.source, env)
			if err != nil {
				t.Fatalf("eval: %v", err)
			}
			if result != tc.want {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tc.want, tc.want)
			}
		})
	}
}

func TestEvalWithEnv_UndefinedVariable(t *testing.T) {
	env := map[string]interface{}{}
	_, err := EvalWithEnv("nonexistent_var + 1", env)
	if err == nil {
		t.Fatal("expected error for undefined variable")
	}
}

// ---------------------------------------------------------------------------
// Context fields
// ---------------------------------------------------------------------------

func TestContext_InputField(t *testing.T) {
	compiled, err := CompileUnchecked("input")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	ctx := &Context{
		Input:  "hello world",
		Steps:  map[string]interface{}{},
		Config: map[string]interface{}{},
	}
	result, err := Eval(compiled, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if result != "hello world" {
		t.Errorf("got %v, want %q", result, "hello world")
	}
}

func TestContext_ConfigField(t *testing.T) {
	compiled, err := CompileUnchecked(`config["api_key"]`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	ctx := &Context{
		Steps:  map[string]interface{}{},
		Config: map[string]interface{}{"api_key": "sk-123"},
	}
	result, err := Eval(compiled, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if result != "sk-123" {
		t.Errorf("got %v, want %q", result, "sk-123")
	}
}

func TestContext_StepsField(t *testing.T) {
	compiled, err := CompileUnchecked(`steps["step1"]`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	ctx := &Context{
		Steps:  map[string]interface{}{"step1": "result-from-step1"},
		Config: map[string]interface{}{},
	}
	result, err := Eval(compiled, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if result != "result-from-step1" {
		t.Errorf("got %v, want %q", result, "result-from-step1")
	}
}

func TestContext_SessionField(t *testing.T) {
	compiled, err := CompileUnchecked("session")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	sessionData := map[string]interface{}{"user": "test-user"}
	ctx := &Context{
		Session: sessionData,
		Steps:   map[string]interface{}{},
		Config:  map[string]interface{}{},
	}
	result, err := Eval(compiled, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if resultMap["user"] != "test-user" {
		t.Errorf("got %v, want %q", resultMap["user"], "test-user")
	}
}
