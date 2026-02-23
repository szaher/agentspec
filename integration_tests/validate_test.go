package integration_tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/ast"
	"github.com/szaher/designs/agentz/internal/formatter"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/migrate"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/validate"
)

func TestParseValidateFormatRoundTrip(t *testing.T) {
	input := readTestFile(t, "testdata/valid.ias")

	// Parse
	f, parseErrs := parser.Parse(input, "valid.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}
	if f == nil {
		t.Fatal("parsed file is nil")
	}
	if f.Package == nil {
		t.Fatal("parsed package is nil")
	}
	if f.Package.Name != "demo" {
		t.Errorf("expected package name 'demo', got %q", f.Package.Name)
	}

	// Structural validation
	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("unexpected structural validation errors: %v", structErrs)
	}

	// Semantic validation
	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) > 0 {
		t.Fatalf("unexpected semantic validation errors: %v", semErrs)
	}

	// Format and verify idempotency
	formatted1 := formatter.Format(f)
	f2, parseErrs2 := parser.Parse(formatted1, "valid.ias")
	if parseErrs2 != nil {
		t.Fatalf("re-parse after format failed: %v", parseErrs2)
	}
	formatted2 := formatter.Format(f2)

	if formatted1 != formatted2 {
		t.Errorf("formatter is not idempotent:\nfirst:\n%s\nsecond:\n%s", formatted1, formatted2)
	}
}

func TestParseValidateInvalidRef(t *testing.T) {
	input := readTestFile(t, "testdata/invalid_ref.ias")

	// Parse should succeed
	f, parseErrs := parser.Parse(input, "invalid_ref.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	// Structural validation should pass
	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("unexpected structural errors: %v", structErrs)
	}

	// Semantic validation should find the bad reference
	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) == 0 {
		t.Fatal("expected semantic validation errors for bad skill reference")
	}

	// Check for "did you mean" suggestion
	found := false
	for _, e := range semErrs {
		if strings.Contains(e.Message, "web-serch") &&
			strings.Contains(e.Hint, "web-search") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'did you mean' hint for 'web-serch' -> 'web-search', got errors: %v", semErrs)
	}
}

func TestIRLowering(t *testing.T) {
	input := readTestFile(t, "testdata/valid.ias")

	f, parseErrs := parser.Parse(input, "valid.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering failed: %v", err)
	}

	if doc.Package.Name != "demo" {
		t.Errorf("expected package name 'demo', got %q", doc.Package.Name)
	}
	if doc.IRVersion != "1.0" {
		t.Errorf("expected IR version '1.0', got %q", doc.IRVersion)
	}

	// Check resources are sorted by kind then name
	for i := 1; i < len(doc.Resources); i++ {
		prev := doc.Resources[i-1]
		curr := doc.Resources[i]
		if prev.Kind > curr.Kind || (prev.Kind == curr.Kind && prev.Name > curr.Name) {
			t.Errorf("resources not sorted: %s/%s before %s/%s",
				prev.Kind, prev.Name, curr.Kind, curr.Name)
		}
	}

	// Verify hashes are computed
	for _, r := range doc.Resources {
		if r.Hash == "" {
			t.Errorf("resource %s has empty hash", r.FQN)
		}
		if !strings.HasPrefix(r.Hash, "sha256:") {
			t.Errorf("resource %s hash has wrong format: %s", r.FQN, r.Hash)
		}
	}

	// Verify deterministic IR output
	data1, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("first marshal failed: %v", err)
	}
	data2, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("second marshal failed: %v", err)
	}
	if string(data1) != string(data2) {
		t.Error("IR serialization is not deterministic")
	}

	// Verify deploy targets are present
	if len(doc.DeployTargets) != 1 {
		t.Errorf("expected 1 deploy target, got %d", len(doc.DeployTargets))
	}
}

// IntentLang 2.0 integration tests

func TestV2BasicAgentParsing(t *testing.T) {
	input := readV2TestFile(t, "basic-agent.ias")

	f, parseErrs := parser.Parse(input, "basic-agent.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	// Structural validation
	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("unexpected structural errors: %v", structErrs)
	}

	// Semantic validation
	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", semErrs)
	}

	// IR lowering
	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering failed: %v", err)
	}

	// Verify agent has runtime config
	for _, r := range doc.Resources {
		if r.Kind == "Agent" && r.Name == "assistant" {
			if r.Attributes["strategy"] != "react" {
				t.Errorf("expected strategy 'react', got %v", r.Attributes["strategy"])
			}
			if r.Attributes["max_turns"] != 5 {
				t.Errorf("expected max_turns 5, got %v", r.Attributes["max_turns"])
			}
			if r.Attributes["timeout"] != "60s" {
				t.Errorf("expected timeout '60s', got %v", r.Attributes["timeout"])
			}
			if r.Attributes["token_budget"] != 50000 {
				t.Errorf("expected token_budget 50000, got %v", r.Attributes["token_budget"])
			}
			if r.Attributes["temperature"] != 0.7 {
				t.Errorf("expected temperature 0.7, got %v", r.Attributes["temperature"])
			}
			if r.Attributes["stream"] != true {
				t.Errorf("expected stream true, got %v", r.Attributes["stream"])
			}
		}
	}

	// Verify deploy target
	if len(doc.DeployTargets) != 1 {
		t.Fatalf("expected 1 deploy target, got %d", len(doc.DeployTargets))
	}
	dt := doc.DeployTargets[0]
	if dt.Name != "local" {
		t.Errorf("expected deploy name 'local', got %q", dt.Name)
	}
	if dt.Target != "process" {
		t.Errorf("expected deploy target 'process', got %q", dt.Target)
	}

	// Format round-trip
	formatted := formatter.Format(f)
	f2, errs2 := parser.Parse(formatted, "basic-agent-formatted.ias")
	if errs2 != nil {
		t.Fatalf("re-parse after format failed: %v", errs2)
	}
	formatted2 := formatter.Format(f2)
	if formatted != formatted2 {
		t.Errorf("formatter not idempotent for v2 basic-agent")
	}
}

func TestV2ToolVariants(t *testing.T) {
	input := readV2TestFile(t, "tool-variants.ias")

	f, parseErrs := parser.Parse(input, "tool-variants.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("unexpected structural errors: %v", structErrs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering failed: %v", err)
	}

	// Check 4 skills with different tool types
	toolTypes := map[string]string{}
	for _, r := range doc.Resources {
		if r.Kind == "Skill" {
			if tool, ok := r.Attributes["tool"].(map[string]interface{}); ok {
				toolType, _ := tool["type"].(string)
				toolTypes[r.Name] = toolType
			}
		}
	}

	expected := map[string]string{
		"mcp-search":  "mcp",
		"http-api":    "http",
		"run-cmd":     "command",
		"inline-code": "inline",
	}
	for name, expectedType := range expected {
		if toolTypes[name] != expectedType {
			t.Errorf("skill %q: expected tool type %q, got %q", name, expectedType, toolTypes[name])
		}
	}
}

func TestV2DeployTargets(t *testing.T) {
	input := readV2TestFile(t, "deploy-targets.ias")

	f, parseErrs := parser.Parse(input, "deploy-targets.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("unexpected structural errors: %v", structErrs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering failed: %v", err)
	}

	if len(doc.DeployTargets) != 3 {
		t.Fatalf("expected 3 deploy targets, got %d", len(doc.DeployTargets))
	}

	targets := map[string]string{}
	for _, dt := range doc.DeployTargets {
		targets[dt.Name] = dt.Target
	}

	expectedTargets := map[string]string{
		"local":      "process",
		"staging":    "docker",
		"production": "kubernetes",
	}
	for name, expectedTarget := range expectedTargets {
		if targets[name] != expectedTarget {
			t.Errorf("deploy %q: expected target %q, got %q", name, expectedTarget, targets[name])
		}
	}
}

func TestV2ErrorHandling(t *testing.T) {
	input := readV2TestFile(t, "error-handling.ias")

	f, parseErrs := parser.Parse(input, "error-handling.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("unexpected structural errors: %v", structErrs)
	}

	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", semErrs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering failed: %v", err)
	}

	// Verify primary agent has fallback config
	for _, r := range doc.Resources {
		if r.Kind == "Agent" && r.Name == "primary" {
			if r.Attributes["on_error"] != "fallback" {
				t.Errorf("expected on_error 'fallback', got %v", r.Attributes["on_error"])
			}
			if r.Attributes["fallback"] != "backup" {
				t.Errorf("expected fallback 'backup', got %v", r.Attributes["fallback"])
			}
			// Verify fallback creates a reference
			foundRef := false
			for _, ref := range r.References {
				if strings.Contains(ref, "Agent/backup") {
					foundRef = true
				}
			}
			if !foundRef {
				t.Errorf("expected reference to Agent/backup, got refs: %v", r.References)
			}
		}
	}
}

func TestV2InvalidDeployTarget(t *testing.T) {
	input := `package "test" version "1.0.0" lang "2.0"
prompt "sys" { content "test" }
agent "a" { uses prompt "sys" model "m" }
deploy "bad" target "invalid" { port 8080 }`

	f, parseErrs := parser.Parse(input, "invalid-deploy.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	structErrs := validate.ValidateStructural(f)
	found := false
	for _, e := range structErrs {
		if strings.Contains(e.Message, "invalid target type") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected validation error for invalid target type, got: %v", structErrs)
	}
}

func TestV2FallbackSelfReference(t *testing.T) {
	input := `package "test" version "1.0.0" lang "2.0"
prompt "sys" { content "test" }
agent "loop" {
  uses prompt "sys"
  model "m"
  on_error "fallback"
  fallback "loop"
}`

	f, parseErrs := parser.Parse(input, "self-fallback.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	semErrs := validate.ValidateSemantic(f)
	found := false
	for _, e := range semErrs {
		if strings.Contains(e.Message, "cannot fallback to itself") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error for self-referencing fallback, got: %v", semErrs)
	}
}

func TestV2PromptVariables(t *testing.T) {
	input := readV2TestFile(t, "prompt-variables.ias")

	f, parseErrs := parser.Parse(input, "prompt-variables.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	// Structural validation
	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("unexpected structural errors: %v", structErrs)
	}

	// Semantic validation
	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", semErrs)
	}

	// Verify prompt has variables
	for _, stmt := range f.Statements {
		if p, ok := stmt.(*ast.Prompt); ok && p.Name == "greeting" {
			if len(p.Variables) != 2 {
				t.Fatalf("expected 2 variables, got %d", len(p.Variables))
			}
			if p.Variables[0].Name != "name" || p.Variables[0].Type != "string" || !p.Variables[0].Required {
				t.Errorf("first variable mismatch: got %+v", p.Variables[0])
			}
			if p.Variables[1].Name != "role" || p.Variables[1].Default != "assistant" {
				t.Errorf("second variable mismatch: got %+v", p.Variables[1])
			}
		}
	}

	// IR lowering
	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering failed: %v", err)
	}

	// Verify prompt resource has variables in attributes
	for _, r := range doc.Resources {
		if r.Kind == "Prompt" && r.Name == "greeting" {
			vars, ok := r.Attributes["variables"]
			if !ok {
				t.Error("expected 'variables' in prompt attributes")
			}
			varList, ok := vars.([]interface{})
			if !ok || len(varList) != 2 {
				t.Errorf("expected 2 variables in IR, got %v", vars)
			}
		}
	}

	// Format round-trip
	formatted := formatter.Format(f)
	f2, errs2 := parser.Parse(formatted, "prompt-variables-formatted.ias")
	if errs2 != nil {
		t.Fatalf("re-parse after format failed: %v", errs2)
	}
	formatted2 := formatter.Format(f2)
	if formatted != formatted2 {
		t.Errorf("formatter not idempotent for prompt-variables")
	}
}

func TestV2TypeDefinitions(t *testing.T) {
	input := readV2TestFile(t, "type-definitions.ias")

	f, parseErrs := parser.Parse(input, "type-definitions.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("unexpected structural errors: %v", structErrs)
	}

	// Count type definitions
	types := map[string]*ast.TypeDef{}
	for _, stmt := range f.Statements {
		if td, ok := stmt.(*ast.TypeDef); ok {
			types[td.Name] = td
		}
	}

	if len(types) != 3 {
		t.Fatalf("expected 3 type definitions, got %d", len(types))
	}

	// Struct type
	user := types["user"]
	if user == nil || len(user.Fields) != 4 {
		t.Fatalf("expected 'user' type with 4 fields, got %+v", user)
	}
	if !user.Fields[0].Required || user.Fields[0].Name != "name" {
		t.Errorf("first field mismatch: %+v", user.Fields[0])
	}

	// Enum type
	status := types["status"]
	if status == nil || len(status.EnumVals) != 3 {
		t.Fatalf("expected 'status' enum with 3 values, got %+v", status)
	}

	// List type
	tags := types["tags"]
	if tags == nil || tags.ListOf != "string" {
		t.Fatalf("expected 'tags' list of string, got %+v", tags)
	}

	// IR lowering
	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering failed: %v", err)
	}

	// Verify type resources exist
	typeResources := 0
	for _, r := range doc.Resources {
		if r.Kind == "Type" {
			typeResources++
		}
	}
	if typeResources != 3 {
		t.Errorf("expected 3 Type resources in IR, got %d", typeResources)
	}

	// Format round-trip
	formatted := formatter.Format(f)
	f2, errs2 := parser.Parse(formatted, "type-definitions-formatted.ias")
	if errs2 != nil {
		t.Fatalf("re-parse after format failed: %v", errs2)
	}
	formatted2 := formatter.Format(f2)
	if formatted != formatted2 {
		t.Errorf("formatter not idempotent for type-definitions")
	}
}

func TestV2Pipeline(t *testing.T) {
	input := readV2TestFile(t, "pipeline.ias")

	f, parseErrs := parser.Parse(input, "pipeline.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("unexpected structural errors: %v", structErrs)
	}

	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", semErrs)
	}

	// Verify pipeline structure
	var pipeline *ast.Pipeline
	for _, stmt := range f.Statements {
		if p, ok := stmt.(*ast.Pipeline); ok {
			pipeline = p
		}
	}

	if pipeline == nil {
		t.Fatal("expected a pipeline statement")
	}
	if pipeline.Name != "data-report" {
		t.Errorf("expected pipeline name 'data-report', got %q", pipeline.Name)
	}
	if len(pipeline.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(pipeline.Steps))
	}

	// Check step dependencies
	analyzeStep := pipeline.Steps[1]
	if analyzeStep.Name != "analyze" {
		t.Errorf("expected second step 'analyze', got %q", analyzeStep.Name)
	}
	if len(analyzeStep.DependsOn) != 1 || analyzeStep.DependsOn[0] != "fetch" {
		t.Errorf("expected analyze depends_on [fetch], got %v", analyzeStep.DependsOn)
	}

	reportStep := pipeline.Steps[2]
	if len(reportStep.DependsOn) != 1 || reportStep.DependsOn[0] != "analyze" {
		t.Errorf("expected report depends_on [analyze], got %v", reportStep.DependsOn)
	}

	// IR lowering
	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering failed: %v", err)
	}

	// Verify pipeline resource exists
	found := false
	for _, r := range doc.Resources {
		if r.Kind == "Pipeline" && r.Name == "data-report" {
			found = true
			steps, ok := r.Attributes["steps"].([]interface{})
			if !ok || len(steps) != 3 {
				t.Errorf("expected 3 steps in IR, got %v", r.Attributes["steps"])
			}
		}
	}
	if !found {
		t.Error("expected Pipeline resource in IR")
	}

	// Format round-trip
	formatted := formatter.Format(f)
	f2, errs2 := parser.Parse(formatted, "pipeline-formatted.ias")
	if errs2 != nil {
		t.Fatalf("re-parse after format failed: %v", errs2)
	}
	formatted2 := formatter.Format(f2)
	if formatted != formatted2 {
		t.Errorf("formatter not idempotent for pipeline")
	}
}

func TestV2Delegation(t *testing.T) {
	input := readV2TestFile(t, "delegation.ias")

	f, parseErrs := parser.Parse(input, "delegation.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("unexpected structural errors: %v", structErrs)
	}

	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) > 0 {
		t.Fatalf("unexpected semantic errors: %v", semErrs)
	}

	// Verify router agent has delegates
	for _, stmt := range f.Statements {
		if a, ok := stmt.(*ast.Agent); ok && a.Name == "router" {
			if len(a.Delegates) != 2 {
				t.Fatalf("expected 2 delegates, got %d", len(a.Delegates))
			}
			if a.Delegates[0].AgentRef != "searcher" {
				t.Errorf("expected first delegate to 'searcher', got %q", a.Delegates[0].AgentRef)
			}
			if a.Delegates[0].Condition != "user asks for information" {
				t.Errorf("expected first delegate condition, got %q", a.Delegates[0].Condition)
			}
			if a.Delegates[1].AgentRef != "calculator" {
				t.Errorf("expected second delegate to 'calculator', got %q", a.Delegates[1].AgentRef)
			}
		}
	}

	// IR lowering
	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering failed: %v", err)
	}

	// Verify router agent references delegates
	for _, r := range doc.Resources {
		if r.Kind == "Agent" && r.Name == "router" {
			delegates, ok := r.Attributes["delegates"]
			if !ok {
				t.Error("expected 'delegates' in router agent attributes")
				continue
			}
			delegateList, ok := delegates.([]interface{})
			if !ok || len(delegateList) != 2 {
				t.Errorf("expected 2 delegates in IR, got %v", delegates)
			}
		}
	}

	// Format round-trip
	formatted := formatter.Format(f)
	f2, errs2 := parser.Parse(formatted, "delegation-formatted.ias")
	if errs2 != nil {
		t.Fatalf("re-parse after format failed: %v", errs2)
	}
	formatted2 := formatter.Format(f2)
	if formatted != formatted2 {
		t.Errorf("formatter not idempotent for delegation")
	}
}

func TestV2MigrateCommand(t *testing.T) {
	// Test that a 1.0-style file (if we simulate it in-memory) can be migrated
	// We can't parse 1.0 anymore (it's rejected), so we test the migration
	// at the AST level directly
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name:        "test",
			Version:     "1.0.0",
			LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Skill{
				Name:        "cmd-skill",
				Description: "test skill",
				Input: []*ast.Field{
					{Name: "arg", Type: "string", Required: true},
				},
				Output: []*ast.Field{
					{Name: "result", Type: "string"},
				},
				Execution: &ast.Execution{
					Type:  "command",
					Value: "my-binary",
				},
			},
			&ast.Binding{
				Name:    "local",
				Adapter: "local-mcp",
				Default: true,
			},
		},
	}

	migrated := migrate.ToV2(f)

	if migrated.Package.LangVersion != "2.0" {
		t.Errorf("expected lang version '2.0', got %q", migrated.Package.LangVersion)
	}

	// Check skill migration: execution → tool
	for _, stmt := range migrated.Statements {
		if s, ok := stmt.(*ast.Skill); ok {
			if s.Execution != nil {
				t.Error("expected execution to be nil after migration")
			}
			if s.ToolConfig == nil {
				t.Fatal("expected tool config after migration")
			}
			if s.ToolConfig.Type != "command" {
				t.Errorf("expected tool type 'command', got %q", s.ToolConfig.Type)
			}
			if s.ToolConfig.Binary != "my-binary" {
				t.Errorf("expected binary 'my-binary', got %q", s.ToolConfig.Binary)
			}
		}
	}

	// Check binding migration: binding → deploy target
	foundDeploy := false
	for _, stmt := range migrated.Statements {
		if dt, ok := stmt.(*ast.DeployTarget); ok {
			foundDeploy = true
			if dt.Name != "local" {
				t.Errorf("expected deploy name 'local', got %q", dt.Name)
			}
			if dt.Target != "process" {
				t.Errorf("expected deploy target 'process', got %q", dt.Target)
			}
			if !dt.Default {
				t.Error("expected deploy default to be true")
			}
		}
	}
	if !foundDeploy {
		t.Error("expected binding to be migrated to deploy target")
	}
}

func TestV2PromptVariableUndeclared(t *testing.T) {
	input := `package "test" version "1.0.0" lang "2.0"
prompt "bad" {
  content "Hello {{unknown_var}}"
  variables {
    name string required
  }
}
skill "s" { description "d" input { a string required } output { b string } tool command { binary "x" } }
agent "a" { uses prompt "bad" model "m" uses skill "s" }`

	f, parseErrs := parser.Parse(input, "bad-var.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	semErrs := validate.ValidateSemantic(f)
	found := false
	for _, e := range semErrs {
		if strings.Contains(e.Message, "undeclared variable") && strings.Contains(e.Message, "unknown_var") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error for undeclared variable, got: %v", semErrs)
	}
}

func TestV2PipelineDuplicateStep(t *testing.T) {
	input := `package "test" version "1.0.0" lang "2.0"
prompt "sys" { content "test" }
skill "s" { description "d" input { a string required } output { b string } tool command { binary "x" } }
agent "a" { uses prompt "sys" model "m" uses skill "s" }
pipeline "bad" {
  step "dup" { agent "a" }
  step "dup" { agent "a" }
}`

	f, parseErrs := parser.Parse(input, "dup-step.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	structErrs := validate.ValidateStructural(f)
	found := false
	for _, e := range structErrs {
		if strings.Contains(e.Message, "duplicate step name") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error for duplicate step name, got: %v", structErrs)
	}
}

func TestV2PipelineStepSelfDependency(t *testing.T) {
	input := `package "test" version "1.0.0" lang "2.0"
prompt "sys" { content "test" }
skill "s" { description "d" input { a string required } output { b string } tool command { binary "x" } }
agent "a" { uses prompt "sys" model "m" uses skill "s" }
pipeline "bad" {
  step "loop" { agent "a" depends_on ["loop"] }
}`

	f, parseErrs := parser.Parse(input, "self-dep.ias")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	structErrs := validate.ValidateStructural(f)
	found := false
	for _, e := range structErrs {
		if strings.Contains(e.Message, "cannot depend on itself") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error for self-dependency, got: %v", structErrs)
	}
}

func readV2TestFile(t *testing.T, name string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "testdata", "v2", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read v2 test file %s: %v", name, err)
	}
	return string(data)
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	absPath := filepath.Join(getTestdataDir(t), filepath.Base(path))
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("failed to read test file %s: %v", path, err)
	}
	return string(data)
}

func getTestdataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "testdata")
}
