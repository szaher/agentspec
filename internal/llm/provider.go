package llm

import (
	"os"
	"strings"
)

// Provider identifies an LLM provider.
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOllama    Provider = "ollama"
	ProviderOpenAI    Provider = "openai"
)

// ParseModelString parses a model string into provider and model name.
//
// Supported formats:
//
//	"ollama/llama3.2"          → (ollama, "llama3.2")
//	"openai/gpt-4o"            → (openai, "gpt-4o")
//	"claude-sonnet-4-20250514" → (anthropic, "claude-sonnet-4-20250514")
//	"gpt-4o"                   → (openai, "gpt-4o")     if OPENAI_API_KEY set
//	"llama3.2"                 → (anthropic, "llama3.2") fallback
func ParseModelString(model string) (Provider, string) {
	if i := strings.Index(model, "/"); i > 0 {
		prefix := strings.ToLower(model[:i])
		name := model[i+1:]
		switch prefix {
		case "ollama":
			return ProviderOllama, name
		case "openai":
			return ProviderOpenAI, name
		case "anthropic":
			return ProviderAnthropic, name
		}
	}

	// No prefix — infer from model name patterns
	lower := strings.ToLower(model)
	if strings.HasPrefix(lower, "claude") {
		return ProviderAnthropic, model
	}
	if strings.HasPrefix(lower, "gpt-") || strings.HasPrefix(lower, "o1") || strings.HasPrefix(lower, "o3") || strings.HasPrefix(lower, "o4") {
		return ProviderOpenAI, model
	}

	// Check env vars as a last resort
	if os.Getenv("OLLAMA_HOST") != "" {
		return ProviderOllama, model
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		return ProviderOpenAI, model
	}

	// Default to Anthropic for backwards compatibility
	return ProviderAnthropic, model
}

// NewClientForModel creates the appropriate LLM client based on the model string.
//
// Environment variables used:
//
//	ANTHROPIC_API_KEY  — Anthropic API key (read by SDK automatically)
//	OPENAI_API_KEY     — OpenAI API key
//	OPENAI_BASE_URL    — Custom OpenAI-compatible base URL
//	OLLAMA_HOST        — Ollama server address (default: http://localhost:11434)
func NewClientForModel(model string) (Client, string) {
	provider, modelName := ParseModelString(model)

	switch provider {
	case ProviderOllama:
		host := os.Getenv("OLLAMA_HOST")
		return NewOllamaClient(host), modelName

	case ProviderOpenAI:
		apiKey := os.Getenv("OPENAI_API_KEY")
		baseURL := os.Getenv("OPENAI_BASE_URL")
		if baseURL != "" {
			return NewOpenAICompatibleClient(baseURL, apiKey), modelName
		}
		return NewOpenAIClient(apiKey), modelName

	default: // ProviderAnthropic
		return NewAnthropicClient(), modelName
	}
}
