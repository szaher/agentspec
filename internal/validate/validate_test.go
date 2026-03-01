package validate

import (
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/ast"
)

// ---------------------------------------------------------------------------
// ValidationError.Error()
// ---------------------------------------------------------------------------

func TestValidationError_ErrorFormatting(t *testing.T) {
	tests := []struct {
		name string
		err  ValidationError
		want string
	}{
		{
			name: "with hint",
			err: ValidationError{
				File: "test.ias", Line: 10, Column: 5,
				Message: "something wrong", Hint: "try this",
			},
			want: "test.ias:10:5: error: something wrong\n  hint: try this",
		},
		{
			name: "without hint",
			err: ValidationError{
				File: "test.ias", Line: 1, Column: 1,
				Message: "missing field",
			},
			want: "test.ias:1:1: error: missing field",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.err.Error()
			if got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateStructural
// ---------------------------------------------------------------------------

func TestValidateStructural_ValidFile(t *testing.T) {
	f := validFile()
	errs := ValidateStructural(f)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d:", len(errs))
		for _, e := range errs {
			t.Logf("  %s", e.Error())
		}
	}
}

func TestValidateStructural_MissingPackage(t *testing.T) {
	f := &ast.File{Path: "test.ias"}
	errs := ValidateStructural(f)
	if len(errs) == 0 {
		t.Fatal("expected errors for missing package")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "missing package declaration") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'missing package declaration' error")
	}
}

func TestValidateStructural_AgentWithoutName(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{
				Name:   "",
				Model:  "gpt-4",
				Prompt: &ast.Ref{Name: "sys"},
			},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "agent name is required") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'agent name is required' error")
	}
}

func TestValidateStructural_AgentMissingPromptAndModel(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{
				Name: "myagent",
				// no prompt, no model
			},
		},
	}

	errs := ValidateStructural(f)
	var messages []string
	for _, e := range errs {
		messages = append(messages, e.Message)
	}

	if !containsSubstr(messages, "requires a prompt reference") {
		t.Error("expected prompt reference error")
	}
	if !containsSubstr(messages, "requires a model") {
		t.Error("expected model error")
	}
}

func TestValidateStructural_PromptWithoutContent(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Prompt{Name: "empty_prompt", Content: ""},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "requires content") {
			found = true
		}
	}
	if !found {
		t.Error("expected prompt content error")
	}
}

func TestValidateStructural_SkillMissingFields(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Skill{
				Name: "badskill",
				// no description, input, output, or execution/tool
			},
		},
	}

	errs := ValidateStructural(f)
	var messages []string
	for _, e := range errs {
		messages = append(messages, e.Message)
	}

	if !containsSubstr(messages, "requires a description") {
		t.Error("expected description error")
	}
	if !containsSubstr(messages, "requires an input schema") {
		t.Error("expected input schema error")
	}
	if !containsSubstr(messages, "requires an output schema") {
		t.Error("expected output schema error")
	}
	if !containsSubstr(messages, "requires an execution or tool block") {
		t.Error("expected execution/tool error")
	}
}

func TestValidateStructural_InvalidAgentStrategy(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{
				Name:     "myagent",
				Model:    "gpt-4",
				Prompt:   &ast.Ref{Name: "sys"},
				Strategy: "invalid-strategy",
			},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "invalid strategy") {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid strategy error")
	}
}

func TestValidateStructural_SecretMissingFields(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Secret{Name: "", Source: "", Key: ""},
		},
	}

	errs := ValidateStructural(f)
	var messages []string
	for _, e := range errs {
		messages = append(messages, e.Message)
	}

	if !containsSubstr(messages, "secret name is required") {
		t.Error("expected secret name error")
	}
	if !containsSubstr(messages, "requires a source") {
		t.Error("expected secret source error")
	}
	if !containsSubstr(messages, "requires a key") {
		t.Error("expected secret key error")
	}
}

func TestValidateStructural_PackageMissingVersion(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name:        "testpkg",
			Version:     "",
			LangVersion: "",
		},
	}

	errs := ValidateStructural(f)
	var messages []string
	for _, e := range errs {
		messages = append(messages, e.Message)
	}

	if !containsSubstr(messages, "package version is required") {
		t.Error("expected package version error")
	}
	if !containsSubstr(messages, "lang version is required") {
		t.Error("expected lang version error")
	}
}

func TestValidateStructural_DeployTargetValidation(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.DeployTarget{Name: "bad", Target: "invalid-target"},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "invalid target type") {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid target type error")
	}
}

func TestValidateStructural_PipelineDuplicateSteps(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Pipeline{
				Name: "mypipeline",
				Steps: []*ast.PipelineStep{
					{Name: "step1", Agent: "agent1"},
					{Name: "step1", Agent: "agent2"}, // duplicate
				},
			},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "duplicate step name") {
			found = true
		}
	}
	if !found {
		t.Error("expected duplicate step name error")
	}
}

// ---------------------------------------------------------------------------
// ValidateSemantic
// ---------------------------------------------------------------------------

func TestValidateSemantic_ValidFile(t *testing.T) {
	f := validFile()
	errs := ValidateSemantic(f)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d:", len(errs))
		for _, e := range errs {
			t.Logf("  %s", e.Error())
		}
	}
}

func TestValidateSemantic_BrokenPromptRef(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{
				Name:   "myagent",
				Model:  "gpt-4",
				Prompt: &ast.Ref{Name: "nonexistent"},
			},
		},
	}

	errs := ValidateSemantic(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "prompt \"nonexistent\" not found") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'prompt not found' error")
	}
}

func TestValidateSemantic_BrokenSkillRef(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Prompt{Name: "sys", Content: "hello"},
			&ast.Agent{
				Name:   "myagent",
				Model:  "gpt-4",
				Prompt: &ast.Ref{Name: "sys"},
				Skills: []*ast.Ref{
					{Name: "ghost_skill"},
				},
			},
		},
	}

	errs := ValidateSemantic(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "skill \"ghost_skill\" not found") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'skill not found' error")
	}
}

func TestValidateSemantic_DuplicateDefaultBindings(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Binding{Name: "b1", Adapter: "local", Default: true},
			&ast.Binding{Name: "b2", Adapter: "remote", Default: true},
		},
	}

	errs := ValidateSemantic(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "multiple bindings marked as default") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'multiple bindings marked as default' error")
	}
}

func TestValidateSemantic_FallbackAgentNotFound(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Prompt{Name: "sys", Content: "hello"},
			&ast.Agent{
				Name:     "myagent",
				Model:    "gpt-4",
				Prompt:   &ast.Ref{Name: "sys"},
				OnError:  "fallback",
				Fallback: "ghost_agent",
			},
		},
	}

	errs := ValidateSemantic(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "fallback agent \"ghost_agent\" not found") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'fallback agent not found' error")
	}
}

func TestValidateSemantic_PipelineUnknownAgent(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Pipeline{
				Name: "mypipeline",
				Steps: []*ast.PipelineStep{
					{Name: "step1", Agent: "nonexistent_agent"},
				},
			},
		},
	}

	errs := ValidateSemantic(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "unknown agent") {
			found = true
		}
	}
	if !found {
		t.Error("expected unknown agent error in pipeline step")
	}
}

func TestValidateSemantic_MCPClientBrokenServerRef(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.MCPClient{
				Name: "myclient",
				Servers: []*ast.Ref{
					{Name: "nonexistent_server"},
				},
			},
		},
	}

	errs := ValidateSemantic(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "server \"nonexistent_server\" not found") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'server not found' error")
	}
}

// ---------------------------------------------------------------------------
// ValidateEnvironments
// ---------------------------------------------------------------------------

func TestValidateEnvironments_Valid(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{Name: "myagent"},
			&ast.Environment{
				Name: "prod",
				Overrides: []*ast.Override{
					{Resource: "agent/myagent", Attribute: "model", Value: "gpt-4"},
				},
			},
		},
	}

	errs := ValidateEnvironments(f)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d:", len(errs))
		for _, e := range errs {
			t.Logf("  %s", e.Error())
		}
	}
}

func TestValidateEnvironments_UndefinedResource(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Environment{
				Name: "staging",
				Overrides: []*ast.Override{
					{Resource: "agent/ghost", Attribute: "model", Value: "gpt-4"},
				},
			},
		},
	}

	errs := ValidateEnvironments(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "not found") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'not found' error for undefined resource")
	}
}

func TestValidateEnvironments_ConflictingOverrides(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{Name: "myagent"},
			&ast.Environment{
				Name: "prod",
				Overrides: []*ast.Override{
					{Resource: "agent/myagent", Attribute: "model", Value: "gpt-4"},
					{Resource: "agent/myagent", Attribute: "model", Value: "gpt-3.5"}, // conflict
				},
			},
		},
	}

	errs := ValidateEnvironments(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "conflicting override") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'conflicting override' error")
	}
}

func TestValidateEnvironments_NoEnvironments(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{Name: "myagent"},
		},
	}

	errs := ValidateEnvironments(f)
	if len(errs) != 0 {
		t.Errorf("expected no errors for file without environments, got %d", len(errs))
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// validFile returns a well-formed ast.File that passes structural and semantic validation.
func validFile() *ast.File {
	return &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name:        "testpkg",
			Version:     "1.0.0",
			LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Prompt{
				Name:    "system_prompt",
				Content: "You are a helpful assistant.",
			},
			&ast.Secret{
				Name:   "api_key",
				Source: "env",
				Key:    "API_KEY",
			},
			&ast.Skill{
				Name:        "search",
				Description: "Search the web",
				Input: []*ast.Field{
					{Name: "query", Type: "string"},
				},
				Output: []*ast.Field{
					{Name: "results", Type: "string"},
				},
				Execution: &ast.Execution{
					Type:  "command",
					Value: "search-tool",
				},
			},
			&ast.Agent{
				Name:  "assistant",
				Model: "gpt-4",
				Prompt: &ast.Ref{
					Kind: "prompt",
					Name: "system_prompt",
				},
				Skills: []*ast.Ref{
					{Kind: "skill", Name: "search"},
				},
			},
		},
	}
}

func containsSubstr(strs []string, substr string) bool {
	for _, s := range strs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Additional structural tests for coverage
// ---------------------------------------------------------------------------

func TestValidateStructural_MCPServerValidation(t *testing.T) {
	tests := []struct {
		name    string
		server  *ast.MCPServer
		wantMsg string
	}{
		{
			name:    "missing name",
			server:  &ast.MCPServer{Name: "", Transport: "stdio", Command: "npx"},
			wantMsg: "server name is required",
		},
		{
			name:    "missing transport",
			server:  &ast.MCPServer{Name: "srv", Transport: ""},
			wantMsg: "requires a transport",
		},
		{
			name:    "stdio without command",
			server:  &ast.MCPServer{Name: "srv", Transport: "stdio", Command: ""},
			wantMsg: "requires a command",
		},
		{
			name:    "sse without url",
			server:  &ast.MCPServer{Name: "srv", Transport: "sse", URL: ""},
			wantMsg: "requires a url",
		},
		{
			name:    "streamable-http without url",
			server:  &ast.MCPServer{Name: "srv", Transport: "streamable-http", URL: ""},
			wantMsg: "requires a url",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &ast.File{
				Path: "test.ias",
				Package: &ast.Package{
					Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
				},
				Statements: []ast.Statement{tc.server},
			}
			errs := ValidateStructural(f)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, tc.wantMsg) {
					found = true
				}
			}
			if !found {
				msgs := make([]string, len(errs))
				for i, e := range errs {
					msgs[i] = e.Message
				}
				t.Errorf("expected error containing %q, got: %v", tc.wantMsg, msgs)
			}
		})
	}
}

func TestValidateStructural_MCPClientValidation(t *testing.T) {
	tests := []struct {
		name    string
		client  *ast.MCPClient
		wantMsg string
	}{
		{
			name:    "missing name",
			client:  &ast.MCPClient{Name: "", Servers: []*ast.Ref{{Name: "s"}}},
			wantMsg: "client name is required",
		},
		{
			name:    "no servers",
			client:  &ast.MCPClient{Name: "c", Servers: nil},
			wantMsg: "requires at least one server connection",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &ast.File{
				Path: "test.ias",
				Package: &ast.Package{
					Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
				},
				Statements: []ast.Statement{tc.client},
			}
			errs := ValidateStructural(f)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, tc.wantMsg) {
					found = true
				}
			}
			if !found {
				t.Errorf("expected error containing %q", tc.wantMsg)
			}
		})
	}
}

func TestValidateStructural_BindingValidation(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Binding{Name: "", Adapter: ""},
		},
	}

	errs := ValidateStructural(f)
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, e.Message)
	}
	if !containsSubstr(msgs, "binding name is required") {
		t.Error("expected binding name error")
	}
	if !containsSubstr(msgs, "requires an adapter") {
		t.Error("expected adapter error")
	}
}

func TestValidateStructural_TypeDefValidation(t *testing.T) {
	tests := []struct {
		name    string
		typedef *ast.TypeDef
		wantMsg string
	}{
		{
			name:    "missing name",
			typedef: &ast.TypeDef{Name: "", Fields: []*ast.TypeField{{Name: "f", Type: "string"}}},
			wantMsg: "type name is required",
		},
		{
			name:    "empty type def",
			typedef: &ast.TypeDef{Name: "Empty"},
			wantMsg: "must have fields, enum values, or list type",
		},
		{
			name: "field missing name",
			typedef: &ast.TypeDef{
				Name:   "WithBadField",
				Fields: []*ast.TypeField{{Name: "", Type: "string"}},
			},
			wantMsg: "type field name is required",
		},
		{
			name: "field missing type",
			typedef: &ast.TypeDef{
				Name:   "WithBadField",
				Fields: []*ast.TypeField{{Name: "f", Type: ""}},
			},
			wantMsg: "requires a type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &ast.File{
				Path: "test.ias",
				Package: &ast.Package{
					Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
				},
				Statements: []ast.Statement{tc.typedef},
			}
			errs := ValidateStructural(f)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, tc.wantMsg) {
					found = true
				}
			}
			if !found {
				msgs := make([]string, len(errs))
				for i, e := range errs {
					msgs[i] = e.Message
				}
				t.Errorf("expected error containing %q, got: %v", tc.wantMsg, msgs)
			}
		})
	}
}

func TestValidateStructural_ImportValidation(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Import{Path: ""},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "import path is required") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'import path is required' error")
	}
}

func TestValidateStructural_OnInputBlock(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "3.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{
				Name:   "router",
				Model:  "gpt-4",
				Prompt: &ast.Ref{Name: "sys"},
				OnInput: &ast.OnInputBlock{
					Statements: []ast.OnInputStmt{
						&ast.UseSkillStmt{SkillName: ""},
						&ast.DelegateToStmt{AgentName: ""},
						&ast.RespondStmt{Expression: ""},
						&ast.IfBlock{
							Condition: "",
							Body: []ast.OnInputStmt{
								&ast.RespondStmt{Expression: "valid"},
							},
						},
						&ast.ForEachBlock{
							Variable:   "",
							Collection: "",
							Body: []ast.OnInputStmt{
								&ast.UseSkillStmt{SkillName: "ok"},
							},
						},
					},
				},
			},
		},
	}

	errs := ValidateStructural(f)
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, e.Message)
	}

	if !containsSubstr(msgs, "use skill requires a skill name") {
		t.Error("expected skill name error")
	}
	if !containsSubstr(msgs, "delegate requires an agent name") {
		t.Error("expected agent name error")
	}
	if !containsSubstr(msgs, "respond requires an expression") {
		t.Error("expected expression error")
	}
	if !containsSubstr(msgs, "if block requires a condition") {
		t.Error("expected condition error")
	}
	if !containsSubstr(msgs, "for each requires a variable name") {
		t.Error("expected variable name error")
	}
	if !containsSubstr(msgs, "for each requires a collection expression") {
		t.Error("expected collection error")
	}
}

func TestValidateStructural_SkillBothExecutionAndTool(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Skill{
				Name:        "conflict",
				Description: "has both",
				Input:       []*ast.Field{{Name: "q", Type: "string"}},
				Output:      []*ast.Field{{Name: "r", Type: "string"}},
				Execution:   &ast.Execution{Type: "command", Value: "test"},
				ToolConfig:  &ast.ToolConfig{Type: "mcp", ServerTool: "s/t"},
			},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "both execution and tool blocks") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'both execution and tool blocks' error")
	}
}

func TestValidateStructural_SkillInvalidToolType(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Skill{
				Name:        "badtool",
				Description: "bad type",
				Input:       []*ast.Field{{Name: "q", Type: "string"}},
				Output:      []*ast.Field{{Name: "r", Type: "string"}},
				ToolConfig:  &ast.ToolConfig{Type: "invalid"},
			},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "invalid tool type") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'invalid tool type' error")
	}
}

func TestValidateStructural_AgentOnErrorFallbackMissing(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{
				Name:    "myagent",
				Model:   "gpt-4",
				Prompt:  &ast.Ref{Name: "sys"},
				OnError: "fallback",
				// Fallback is empty
			},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "no fallback agent specified") {
			found = true
		}
	}
	if !found {
		t.Error("expected fallback missing error")
	}
}

func TestValidateStructural_AgentTemperatureOutOfRange(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{
				Name:        "myagent",
				Model:       "gpt-4",
				Prompt:      &ast.Ref{Name: "sys"},
				Temperature: 3.0,
				HasTemp:     true,
			},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "temperature") && strings.Contains(e.Message, "out of range") {
			found = true
		}
	}
	if !found {
		t.Error("expected temperature out of range error")
	}
}

func TestValidateStructural_PipelineSelfDependency(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Pipeline{
				Name: "mypipeline",
				Steps: []*ast.PipelineStep{
					{Name: "step1", Agent: "a1", DependsOn: []string{"step1"}},
				},
			},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "cannot depend on itself") {
			found = true
		}
	}
	if !found {
		t.Error("expected self-dependency error")
	}
}

func TestValidateStructural_PipelineUnknownDependency(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Pipeline{
				Name: "mypipeline",
				Steps: []*ast.PipelineStep{
					{Name: "step1", Agent: "a1", DependsOn: []string{"nonexistent"}},
				},
			},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "depends_on unknown step") {
			found = true
		}
	}
	if !found {
		t.Error("expected unknown dependency error")
	}
}

// ---------------------------------------------------------------------------
// Additional semantic tests for coverage
// ---------------------------------------------------------------------------

func TestValidateSemantic_OnInputBrokenRefs(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "3.0",
		},
		Statements: []ast.Statement{
			&ast.Prompt{Name: "sys", Content: "hello"},
			&ast.Agent{
				Name:   "router",
				Model:  "gpt-4",
				Prompt: &ast.Ref{Name: "sys"},
				OnInput: &ast.OnInputBlock{
					Statements: []ast.OnInputStmt{
						&ast.UseSkillStmt{SkillName: "nonexistent_skill"},
						&ast.DelegateToStmt{AgentName: "nonexistent_agent"},
						&ast.IfBlock{
							Condition: "true",
							Body: []ast.OnInputStmt{
								&ast.UseSkillStmt{SkillName: "also_missing"},
							},
							ElseIfs: []*ast.ElseIfClause{
								{Condition: "false", Body: []ast.OnInputStmt{
									&ast.DelegateToStmt{AgentName: "missing_too"},
								}},
							},
							ElseBody: []ast.OnInputStmt{
								&ast.UseSkillStmt{SkillName: "still_missing"},
							},
						},
						&ast.ForEachBlock{
							Variable:   "item",
							Collection: "items",
							Body: []ast.OnInputStmt{
								&ast.UseSkillStmt{SkillName: "foreach_missing"},
							},
						},
					},
				},
			},
		},
	}

	errs := ValidateSemantic(f)
	// Should have multiple "not found" errors
	notFoundCount := 0
	for _, e := range errs {
		if strings.Contains(e.Message, "not found") {
			notFoundCount++
		}
	}
	if notFoundCount < 5 {
		t.Errorf("expected at least 5 'not found' errors, got %d", notFoundCount)
	}
}

func TestValidateSemantic_MCPServerBrokenSkillAndAuthRef(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.MCPServer{
				Name:      "srv",
				Transport: "stdio",
				Command:   "cmd",
				Skills:    []*ast.Ref{{Name: "ghost_skill"}},
				Auth:      &ast.Ref{Name: "ghost_secret"},
			},
		},
	}

	errs := ValidateSemantic(f)
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, e.Message)
	}
	if !containsSubstr(msgs, "skill \"ghost_skill\" not found") {
		t.Error("expected skill not found error")
	}
	if !containsSubstr(msgs, "secret \"ghost_secret\" not found") {
		t.Error("expected secret not found error")
	}
}

func TestValidateSemantic_SkillBrokenMCPServerRef(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Skill{
				Name:        "mcp_skill",
				Description: "test",
				Input:       []*ast.Field{{Name: "q", Type: "string"}},
				Output:      []*ast.Field{{Name: "r", Type: "string"}},
				ToolConfig:  &ast.ToolConfig{Type: "mcp", ServerTool: "ghost_server/search"},
			},
		},
	}

	errs := ValidateSemantic(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "MCP server \"ghost_server\" not found") {
			found = true
		}
	}
	if !found {
		t.Error("expected MCP server not found error")
	}
}

func TestValidateSemantic_AgentDelegateBrokenRef(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Prompt{Name: "sys", Content: "hello"},
			&ast.Agent{
				Name:   "main",
				Model:  "gpt-4",
				Prompt: &ast.Ref{Name: "sys"},
				Delegates: []*ast.Delegate{
					{AgentRef: "missing_delegate", Condition: "when complex"},
				},
			},
		},
	}

	errs := ValidateSemantic(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "delegate agent \"missing_delegate\" not found") {
			found = true
		}
	}
	if !found {
		t.Error("expected delegate agent not found error")
	}
}

func TestValidateSemantic_AgentFallbackToSelf(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Prompt{Name: "sys", Content: "hello"},
			&ast.Agent{
				Name:     "self_loop",
				Model:    "gpt-4",
				Prompt:   &ast.Ref{Name: "sys"},
				OnError:  "fallback",
				Fallback: "self_loop",
			},
		},
	}

	errs := ValidateSemantic(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "cannot fallback to itself") {
			found = true
		}
	}
	if !found {
		t.Error("expected self-fallback error")
	}
}

func TestValidateSemantic_AgentBrokenClientRef(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Prompt{Name: "sys", Content: "hello"},
			&ast.Agent{
				Name:   "main",
				Model:  "gpt-4",
				Prompt: &ast.Ref{Name: "sys"},
				Client: &ast.Ref{Name: "missing_client"},
			},
		},
	}

	errs := ValidateSemantic(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "client \"missing_client\" not found") {
			found = true
		}
	}
	if !found {
		t.Error("expected client not found error")
	}
}

func TestValidateSemantic_MultipleDefaultDeployTargets(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.DeployTarget{Name: "d1", Target: "docker", Default: true},
			&ast.DeployTarget{Name: "d2", Target: "kubernetes", Default: true},
		},
	}

	errs := ValidateSemantic(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "multiple deploy targets marked as default") {
			found = true
		}
	}
	if !found {
		t.Error("expected multiple default deploy targets error")
	}
}

func TestValidateSemantic_PromptUndeclaredVariable(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Prompt{
				Name:    "tmpl",
				Content: "Hello {{name}}, your role is {{role}}",
				Variables: []*ast.Variable{
					{Name: "name", Type: "string"},
					// "role" is not declared
				},
			},
		},
	}

	errs := ValidateSemantic(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "undeclared variable {{role}}") {
			found = true
		}
	}
	if !found {
		t.Error("expected undeclared variable error")
	}
}

func TestValidateStructural_DeployTargetMissingNameAndTarget(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.DeployTarget{Name: "", Target: ""},
		},
	}

	errs := ValidateStructural(f)
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, e.Message)
	}
	if !containsSubstr(msgs, "deploy target name is required") {
		t.Error("expected deploy target name error")
	}
	if !containsSubstr(msgs, "requires a target type") {
		t.Error("expected target type error")
	}
}

func TestValidateStructural_PipelineEmpty(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Pipeline{Name: "empty", Steps: nil},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "requires at least one step") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'requires at least one step' error")
	}
}

func TestValidateStructural_AgentInvalidOnError(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{
				Name:    "myagent",
				Model:   "gpt-4",
				Prompt:  &ast.Ref{Name: "sys"},
				OnError: "invalid_value",
			},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "invalid on_error") {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid on_error error")
	}
}

func TestValidateStructural_IfBlockWithElseIf(t *testing.T) {
	f := &ast.File{
		Path: "test.ias",
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "3.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{
				Name:   "agent",
				Model:  "gpt-4",
				Prompt: &ast.Ref{Name: "sys"},
				OnInput: &ast.OnInputBlock{
					Statements: []ast.OnInputStmt{
						&ast.IfBlock{
							Condition: "true",
							Body:      []ast.OnInputStmt{&ast.RespondStmt{Expression: "yes"}},
							ElseIfs: []*ast.ElseIfClause{
								{Condition: "", Body: []ast.OnInputStmt{&ast.RespondStmt{Expression: "maybe"}}},
							},
							ElseBody: []ast.OnInputStmt{&ast.RespondStmt{Expression: "no"}},
						},
					},
				},
			},
		},
	}

	errs := ValidateStructural(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "else if requires a condition") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'else if requires a condition' error")
	}
}
