package integration_tests

import (
	"os"
	"testing"

	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/runtime"
)

func TestParseModelString(t *testing.T) {
	tests := []struct {
		input        string
		wantProvider llm.Provider
		wantModel    string
	}{
		// Explicit provider prefix
		{"ollama/llama3.1", llm.ProviderOllama, "llama3.1"},
		{"ollama/mistral", llm.ProviderOllama, "mistral"},
		{"ollama/codellama:7b", llm.ProviderOllama, "codellama:7b"},
		{"openai/gpt-4o", llm.ProviderOpenAI, "gpt-4o"},
		{"openai/gpt-4o-mini", llm.ProviderOpenAI, "gpt-4o-mini"},
		{"anthropic/claude-sonnet-4-20250514", llm.ProviderAnthropic, "claude-sonnet-4-20250514"},

		// Inferred from model name patterns
		{"claude-sonnet-4-20250514", llm.ProviderAnthropic, "claude-sonnet-4-20250514"},
		{"claude-haiku-3-5-20241022", llm.ProviderAnthropic, "claude-haiku-3-5-20241022"},
		{"gpt-4o", llm.ProviderOpenAI, "gpt-4o"},
		{"gpt-4o-mini", llm.ProviderOpenAI, "gpt-4o-mini"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			provider, model := llm.ParseModelString(tt.input)
			if provider != tt.wantProvider {
				t.Errorf("ParseModelString(%q) provider = %q, want %q", tt.input, provider, tt.wantProvider)
			}
			if model != tt.wantModel {
				t.Errorf("ParseModelString(%q) model = %q, want %q", tt.input, model, tt.wantModel)
			}
		})
	}
}

func TestNewClientForModel(t *testing.T) {
	tests := []struct {
		model     string
		wantType  string
		wantModel string
	}{
		{"ollama/llama3.1", "*llm.OpenAIClient", "llama3.1"},
		{"openai/gpt-4o", "*llm.OpenAIClient", "gpt-4o"},
		{"claude-sonnet-4-20250514", "*llm.AnthropicClient", "claude-sonnet-4-20250514"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			client, resolvedModel := llm.NewClientForModel(tt.model)
			if client == nil {
				t.Fatal("NewClientForModel returned nil client")
			}
			if resolvedModel != tt.wantModel {
				t.Errorf("NewClientForModel(%q) model = %q, want %q", tt.model, resolvedModel, tt.wantModel)
			}

			// Verify the client type
			switch tt.wantType {
			case "*llm.OpenAIClient":
				if _, ok := client.(*llm.OpenAIClient); !ok {
					t.Errorf("NewClientForModel(%q) returned %T, want %s", tt.model, client, tt.wantType)
				}
			case "*llm.AnthropicClient":
				if _, ok := client.(*llm.AnthropicClient); !ok {
					t.Errorf("NewClientForModel(%q) returned %T, want %s", tt.model, client, tt.wantType)
				}
			}
		})
	}
}

func TestOllamaExampleParses(t *testing.T) {
	data, err := os.ReadFile("../examples/ollama-agent/ollama-agent.ias")
	if err != nil {
		t.Fatalf("read example: %v", err)
	}

	f, errs := parser.Parse(string(data), "ollama-agent.ias")
	if errs != nil {
		t.Fatalf("parse errors: %v", errs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("lower error: %v", err)
	}

	if doc.Package.Name != "ollama-agent" {
		t.Errorf("package name = %q, want %q", doc.Package.Name, "ollama-agent")
	}

	// Convert to RuntimeConfig to get typed agent configs
	config, err := runtime.FromIR(doc)
	if err != nil {
		t.Fatalf("config error: %v", err)
	}

	// Expect two agents: coder and reviewer
	if len(config.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(config.Agents))
	}

	// Find coder agent
	var coder *runtime.AgentConfig
	var reviewer *runtime.AgentConfig
	for i := range config.Agents {
		switch config.Agents[i].Name {
		case "coder":
			coder = &config.Agents[i]
		case "reviewer":
			reviewer = &config.Agents[i]
		}
	}

	if coder == nil {
		t.Fatal("coder agent not found")
	}
	if reviewer == nil {
		t.Fatal("reviewer agent not found")
	}

	if coder.Model != "ollama/llama3.1" {
		t.Errorf("coder model = %q, want %q", coder.Model, "ollama/llama3.1")
	}
	if reviewer.Model != "ollama/llama3.2" {
		t.Errorf("reviewer model = %q, want %q", reviewer.Model, "ollama/llama3.2")
	}

	// Verify coder has 4 skills
	if len(coder.Skills) != 4 {
		t.Errorf("coder skills = %d, want 4", len(coder.Skills))
	}

	// Verify provider detection strips prefix
	provider, modelName := llm.ParseModelString(coder.Model)
	if provider != llm.ProviderOllama {
		t.Errorf("provider = %q, want %q", provider, llm.ProviderOllama)
	}
	if modelName != "llama3.1" {
		t.Errorf("modelName = %q, want %q", modelName, "llama3.1")
	}
}
