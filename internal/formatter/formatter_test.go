package formatter

import (
	"testing"

	"github.com/szaher/agentspec/internal/ast"
)

func TestFormat_EmptyFile(t *testing.T) {
	f := &ast.File{}
	result := Format(f)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestFormat_PackageOnly(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name:        "myagent",
			Version:     "1.0.0",
			LangVersion: "3.0",
		},
	}
	result := Format(f)
	expected := `package "myagent" version "1.0.0" lang "3.0"` + "\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_PackageNameOnly(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name: "simple",
		},
	}
	result := Format(f)
	expected := `package "simple"` + "\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_Prompt(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Prompt{
				Name:    "test-prompt",
				Content: "You are a helpful assistant.",
				Version: "1.0",
				Variables: []*ast.Variable{
					{Name: "topic", Type: "string", Required: true},
					{Name: "format", Type: "string", Default: "json"},
				},
			},
		},
	}
	result := Format(f)
	expected := `
prompt "test-prompt" {
  content "You are a helpful assistant."
  version "1.0"
  variables {
    topic string required
    format string default "json"
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_Skill(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Skill{
				Name:        "search",
				Description: "Search the web",
				Input: []*ast.Field{
					{Name: "query", Type: "string", Required: true},
				},
				Output: []*ast.Field{
					{Name: "results", Type: "string"},
				},
				Execution: &ast.Execution{
					Type:  "wasm",
					Value: "search.wasm",
				},
			},
		},
	}
	result := Format(f)
	expected := `
skill "search" {
  description "Search the web"
  input {
    query string required
  }
  output {
    results string
  }
  execution wasm "search.wasm"
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_Agent(t *testing.T) {
	stream := true
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Agent{
				Name:   "assistant",
				Prompt: &ast.Ref{Name: "main-prompt"},
				Skills: []*ast.Ref{
					{Name: "search"},
					{Name: "calculator"},
				},
				Model:         "claude-3-5-sonnet-20241022",
				BudgetDaily:   10.0,
				BudgetMonthly: 100.0,
				GuardrailRefs: []string{"content-filter"},
				Strategy:      "react",
				MaxTurns:      10,
				Timeout:       "30s",
				TokenBudget:   100000,
				Temperature:   0.7,
				HasTemp:       true,
				Stream:        &stream,
				OnError:       "retry",
				MaxRetries:    3,
				Fallback:      "backup-agent",
				MemoryCfg: &ast.MemoryConfig{
					Strategy:    "sliding-window",
					MaxMessages: 20,
				},
			},
		},
	}
	result := Format(f)
	expected := `
agent "assistant" {
  uses prompt "main-prompt"
  uses skill "search"
  uses skill "calculator"
  model "claude-3-5-sonnet-20241022"
  budget daily 10
  budget monthly 100
  uses guardrail "content-filter"
  strategy "react"
  max_turns 10
  timeout "30s"
  token_budget 100000
  temperature 0.7
  stream true
  on_error "retry"
  max_retries 3
  fallback "backup-agent"
  memory {
    strategy "sliding-window"
    max_messages 20
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_AgentWithModels(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Agent{
				Name:   "multi-model",
				Models: []string{"gpt-4", "claude-3"},
			},
		},
	}
	result := Format(f)
	expected := `
agent "multi-model" {
  models ["gpt-4", "claude-3"]
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_AgentWithDelegates(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Agent{
				Name: "router",
				Delegates: []*ast.Delegate{
					{AgentRef: "specialist1", Condition: "topic == 'math'"},
					{AgentRef: "specialist2", Condition: "topic == 'science'"},
				},
			},
		},
	}
	result := Format(f)
	expected := `
agent "router" {
  delegate to agent "specialist1" when "topic == 'math'"
  delegate to agent "specialist2" when "topic == 'science'"
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_Binding(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Binding{
				Name:    "anthropic",
				Adapter: "anthropic",
				Default: true,
				Config: map[string]string{
					"api_key": "secret(ANTHROPIC_KEY)",
					"region":  "us-west-2",
				},
			},
		},
	}
	result := Format(f)
	// Keys are sorted alphabetically
	expected := `
binding "anthropic" adapter "anthropic" {
  default true
  api_key "secret(ANTHROPIC_KEY)"
  region "us-west-2"
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_Secret(t *testing.T) {
	tests := []struct {
		name     string
		secret   *ast.Secret
		expected string
	}{
		{
			name: "env source",
			secret: &ast.Secret{
				Name:   "api-key",
				Source: "env",
				Key:    "API_KEY",
			},
			expected: `
secret "api-key" {
  env(API_KEY)
}
`,
		},
		{
			name: "store source",
			secret: &ast.Secret{
				Name:   "db-password",
				Source: "store",
				Key:    "db-password",
			},
			expected: `
secret "db-password" {
  store(db-password)
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &ast.File{
				Statements: []ast.Statement{tt.secret},
			}
			result := Format(f)
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestFormat_Environment(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Environment{
				Name: "production",
				Overrides: []*ast.Override{
					{Resource: "agent/assistant", Attribute: "model", Value: "gpt-4"},
					{Resource: "agent/assistant", Attribute: "temperature", Value: "0.5"},
					{Resource: "binding/anthropic", Attribute: "region", Value: "us-east-1"},
				},
			},
		},
	}
	result := Format(f)
	expected := `
environment "production" {
  agent "assistant" {
    model "gpt-4"
    temperature "0.5"
  }
  binding "anthropic" {
    region "us-east-1"
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_Policy(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Policy{
				Name: "admin-policy",
				Rules: []*ast.Rule{
					{Action: "allow", Resource: "agent/*", Subject: "admin"},
					{Action: "deny", Resource: "secret/*"},
				},
			},
		},
	}
	result := Format(f)
	expected := `
policy "admin-policy" {
  allow agent/* admin
  deny secret/*
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_Plugin(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Plugin{
				Name:    "logger",
				Version: "1.0.0",
			},
		},
	}
	result := Format(f)
	expected := `
plugin "logger" version "1.0.0"
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_MCPServer(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.MCPServer{
				Name:      "filesystem",
				Transport: "stdio",
				Command:   "/usr/bin/mcp-server",
				Args:      []string{"--config", "config.json"},
				URL:       "http://localhost:8080",
				Auth:      &ast.Ref{Name: "mcp-auth"},
				Skills:    []*ast.Ref{{Name: "read-file"}, {Name: "write-file"}},
				Env: map[string]string{
					"LOG_LEVEL": "info",
					"DATA_DIR":  "/var/data",
				},
				Metadata: map[string]string{
					"owner": "team-platform",
				},
			},
		},
	}
	result := Format(f)
	expected := `
server "filesystem" {
  transport "stdio"
  command "/usr/bin/mcp-server"
  args ["--config", "config.json"]
  url "http://localhost:8080"
  auth "mcp-auth"
  exposes skill "read-file"
  exposes skill "write-file"
  env {
    DATA_DIR "/var/data"
    LOG_LEVEL "info"
  }
  metadata {
    owner "team-platform"
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_MCPClient(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.MCPClient{
				Name: "main-client",
				Servers: []*ast.Ref{
					{Name: "server1"},
					{Name: "server2"},
				},
				Metadata: map[string]string{
					"version": "1.0",
				},
			},
		},
	}
	result := Format(f)
	expected := `
client "main-client" {
  connects to server "server1"
  connects to server "server2"
  metadata {
    version "1.0"
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_PluginRef(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.PluginRef{
				Name:    "validator",
				Version: "2.0.0",
			},
		},
	}
	result := Format(f)
	expected := `
plugin "validator" version "2.0.0"
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_TypeDefEnum(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.TypeDef{
				Name:     "Status",
				EnumVals: []string{"pending", "approved", "rejected"},
			},
		},
	}
	result := Format(f)
	expected := `
type "Status" enum ["pending", "approved", "rejected"]
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_TypeDefList(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.TypeDef{
				Name:   "StringList",
				ListOf: "string",
			},
		},
	}
	result := Format(f)
	expected := `
type "StringList" list string
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_TypeDefStruct(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.TypeDef{
				Name: "Person",
				Fields: []*ast.TypeField{
					{Name: "name", Type: "string", Required: true},
					{Name: "age", Type: "int"},
					{Name: "email", Type: "string", Default: "none@example.com"},
				},
			},
		},
	}
	result := Format(f)
	expected := `
type "Person" {
  name string required
  age int
  email string default "none@example.com"
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_Pipeline(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Pipeline{
				Name: "process-data",
				Steps: []*ast.PipelineStep{
					{
						Name:   "fetch",
						Agent:  "fetcher",
						Input:  "url",
						Output: "raw_data",
					},
					{
						Name:      "transform",
						Agent:     "transformer",
						Input:     "raw_data",
						Output:    "clean_data",
						DependsOn: []string{"fetch"},
						Parallel:  true,
						When:      "raw_data != null",
					},
				},
			},
		},
	}
	result := Format(f)
	expected := `
pipeline "process-data" {
  step "fetch" {
    agent "fetcher"
    input "url"
    output "raw_data"
  }
  step "transform" {
    agent "transformer"
    input "raw_data"
    output "clean_data"
    depends_on ["fetch"]
    parallel true
    when "raw_data != null"
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_Import(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Import{
				Path:    "github.com/example/agents",
				Version: "1.2.3",
				Alias:   "ext",
			},
		},
	}
	result := Format(f)
	expected := `
import "github.com/example/agents" version "1.2.3" as ext
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_User(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.User{
				Name:   "alice",
				KeyRef: "alice-key",
				Agents: []string{"agent1", "agent2"},
				Role:   "admin",
			},
		},
	}
	result := Format(f)
	expected := `
user "alice" {
  key secret("alice-key")
  agents ["agent1", "agent2"]
  role "admin"
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_Guardrail(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Guardrail{
				Name:        "content-filter",
				Mode:        "block",
				Keywords:    []string{"offensive", "spam"},
				Patterns:    []string{`\d{3}-\d{2}-\d{4}`},
				FallbackMsg: "Content blocked",
			},
		},
	}
	result := Format(f)
	expected := `
guardrail "content-filter" {
  mode "block"
  keywords ["offensive", "spam"]
  patterns ["\\d{3}-\\d{2}-\\d{4}"]
  fallback "Content blocked"
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_ToolConfigMCP(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Skill{
				Name: "test-skill",
				ToolConfig: &ast.ToolConfig{
					Type:       "mcp",
					ServerTool: "server/tool",
				},
			},
		},
	}
	result := Format(f)
	expected := `
skill "test-skill" {
  tool mcp "server/tool"
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_ToolConfigHTTP(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Skill{
				Name: "api-skill",
				ToolConfig: &ast.ToolConfig{
					Type:   "http",
					Method: "POST",
					URL:    "https://api.example.com",
					Headers: map[string]string{
						"Authorization": "Bearer token",
						"Content-Type":  "application/json",
					},
					BodyTemplate: `{"query": "{{.input}}"}`,
					Timeout:      "10s",
				},
			},
		},
	}
	result := Format(f)
	expected := `
skill "api-skill" {
  tool http {
    method "POST"
    url "https://api.example.com"
    headers {
      Authorization "Bearer token"
      Content-Type "application/json"
    }
    body_template "{\"query\": \"{{.input}}\"}"
    timeout "10s"
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_ToolConfigCommand(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Skill{
				Name: "cmd-skill",
				ToolConfig: &ast.ToolConfig{
					Type:    "command",
					Binary:  "/bin/sh",
					Args:    []string{"-c", "echo hello"},
					Timeout: "5s",
					Env: map[string]string{
						"PATH": "/usr/bin",
					},
					Secrets: map[string]string{
						"API_KEY": "my-secret",
					},
				},
			},
		},
	}
	result := Format(f)
	expected := `
skill "cmd-skill" {
  tool command {
    binary "/bin/sh"
    args ["-c", "echo hello"]
    timeout "5s"
    env {
      PATH "/usr/bin"
    }
    secrets {
      API_KEY "my-secret"
    }
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_ToolConfigInline(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Skill{
				Name: "inline-skill",
				ToolConfig: &ast.ToolConfig{
					Type:        "inline",
					Language:    "python",
					Code:        "print('hello')",
					Timeout:     "3s",
					MemoryLimit: "100MB",
				},
			},
		},
	}
	result := Format(f)
	expected := `
skill "inline-skill" {
  tool inline {
    language "python"
    code "print('hello')"
    timeout "3s"
    memory "100MB"
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_DeployTarget(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.DeployTarget{
				Name:      "prod",
				Target:    "kubernetes",
				Port:      8080,
				Default:   true,
				Namespace: "production",
				Replicas:  3,
				Image:     "myapp:v1.0",
				Resources: &ast.ResourceLimits{
					CPU:    "500m",
					Memory: "1Gi",
				},
				Health: &ast.HealthConfig{
					Path:     "/health",
					Interval: "10s",
					Timeout:  "5s",
				},
				Autoscale: &ast.AutoscaleConfig{
					MinReplicas: 2,
					MaxReplicas: 10,
					Metric:      "cpu",
					Target:      80,
				},
				Env: map[string]string{
					"ENV": "production",
				},
				Secrets: map[string]string{
					"DB_PASSWORD": "db-secret",
				},
			},
		},
	}
	result := Format(f)
	expected := `
deploy "prod" target "kubernetes" {
  port 8080
  default true
  namespace "production"
  replicas 3
  image "myapp:v1.0"
  resources {
    cpu "500m"
    memory "1Gi"
  }
  health {
    path "/health"
    interval "10s"
    timeout "5s"
  }
  autoscale {
    min 2
    max 10
    metric "cpu"
    target 80
  }
  env {
    ENV "production"
  }
  secrets {
    DB_PASSWORD "db-secret"
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_AgentWithConfig(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Agent{
				Name: "configurable",
				ConfigParams: []*ast.ConfigParam{
					{Name: "api_key", Type: "string", Required: true, Secret: true},
					{Name: "timeout", Type: "int", HasDefault: true, Default: "30", Description: "Request timeout in seconds"},
				},
			},
		},
	}
	result := Format(f)
	expected := `
agent "configurable" {
  config {
    api_key string required secret
    timeout int default "30"
      "Request timeout in seconds"
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_AgentWithValidation(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Agent{
				Name: "validated",
				ValidationRules: []*ast.ValidationRule{
					{Name: "check-length", Severity: "error", MaxRetries: 3, Message: "Response too short", Expression: "len(output) > 10"},
				},
			},
		},
	}
	result := Format(f)
	expected := `
agent "validated" {
  validate {
    rule check-length error max_retries 3
      "Response too short"
      when len(output) > 10
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_AgentWithEval(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Agent{
				Name: "tested",
				EvalCases: []*ast.EvalCase{
					{
						Name:      "test1",
						Input:     "hello",
						Expected:  "Hello!",
						Scoring:   "exact",
						Threshold: 0.9,
						Tags:      []string{"greeting", "basic"},
					},
				},
			},
		},
	}
	result := Format(f)
	expected := `
agent "tested" {
  eval {
    case test1
      input "hello"
      expect "Hello!"
      scoring exact threshold 0.9
      tags ["greeting", "basic"]
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_AgentWithOnInput(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Agent{
				Name: "reactive",
				OnInput: &ast.OnInputBlock{
					Statements: []ast.OnInputStmt{
						&ast.UseSkillStmt{
							SkillName: "search",
							Params: map[string]string{
								"query": "input.text",
								"limit": "10",
							},
						},
						&ast.DelegateToStmt{
							AgentName: "specialist",
						},
						&ast.RespondStmt{
							Expression: "result.summary",
						},
						&ast.IfBlock{
							Condition: "input.urgent",
							Body: []ast.OnInputStmt{
								&ast.RespondStmt{Expression: "emergency_response"},
							},
							ElseIfs: []*ast.ElseIfClause{
								{
									Condition: "input.priority == 'high'",
									Body: []ast.OnInputStmt{
										&ast.RespondStmt{Expression: "high_priority_response"},
									},
								},
							},
							ElseBody: []ast.OnInputStmt{
								&ast.RespondStmt{Expression: "normal_response"},
							},
						},
						&ast.ForEachBlock{
							Variable:   "item",
							Collection: "input.items",
							Body: []ast.OnInputStmt{
								&ast.UseSkillStmt{SkillName: "process", Params: map[string]string{"data": "item"}},
							},
						},
					},
				},
			},
		},
	}
	result := Format(f)
	expected := `
agent "reactive" {
  on input {
    use skill search with { limit: 10, query: input.text }
    delegate to specialist
    respond "result.summary"
    if input.urgent {
      respond "emergency_response"
    } else if input.priority == 'high' {
      respond "high_priority_response"
    } else {
      respond "normal_response"
    }
    for each item in input.items {
      use skill process with { data: item }
    }
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_ComplexFile(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name:        "complex",
			Version:     "1.0.0",
			LangVersion: "3.0",
		},
		Statements: []ast.Statement{
			&ast.Import{
				Path:    "github.com/example/lib",
				Version: "1.0.0",
			},
			&ast.Prompt{
				Name:    "main",
				Content: "You are helpful.",
			},
			&ast.Skill{
				Name:        "tool",
				Description: "A tool",
			},
			&ast.Agent{
				Name:   "agent1",
				Prompt: &ast.Ref{Name: "main"},
				Skills: []*ast.Ref{{Name: "tool"}},
			},
		},
	}
	result := Format(f)
	expected := `package "complex" version "1.0.0" lang "3.0"

import "github.com/example/lib" version "1.0.0"

prompt "main" {
  content "You are helpful."
}

skill "tool" {
  description "A tool"
}

agent "agent1" {
  uses prompt "main"
  uses skill "tool"
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_SortedKeys(t *testing.T) {
	// Test that map keys are sorted for deterministic output
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Binding{
				Name: "test",
				Config: map[string]string{
					"zebra": "z",
					"alpha": "a",
					"beta":  "b",
				},
			},
		},
	}
	result := Format(f)
	expected := `
binding "test" {
  alpha "a"
  beta "b"
  zebra "z"
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_SkillWithMetadata(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Skill{
				Name: "meta-skill",
				Metadata: map[string]string{
					"author":  "John Doe",
					"version": "1.0",
				},
			},
		},
	}
	result := Format(f)
	expected := `
skill "meta-skill" {
  metadata {
    author "John Doe"
    version "1.0"
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_StreamFalse(t *testing.T) {
	stream := false
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Agent{
				Name:   "no-stream",
				Stream: &stream,
			},
		},
	}
	result := Format(f)
	expected := `
agent "no-stream" {
  stream false
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_AgentConnectsToClient(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Agent{
				Name:   "mcp-agent",
				Client: &ast.Ref{Name: "main-client"},
			},
		},
	}
	result := Format(f)
	expected := `
agent "mcp-agent" {
  connects to client "main-client"
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormat_EvalCaseDefaultThreshold(t *testing.T) {
	// When threshold is 0.8 (default), it should not be included
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Agent{
				Name: "default-threshold",
				EvalCases: []*ast.EvalCase{
					{
						Name:      "test",
						Input:     "hi",
						Expected:  "hello",
						Scoring:   "semantic",
						Threshold: 0.8, // default value
					},
				},
			},
		},
	}
	result := Format(f)
	expected := `
agent "default-threshold" {
  eval {
    case test
      input "hi"
      expect "hello"
      scoring semantic
  }
}
`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}
