// Package templates provides embedded project templates for agentspec init.
package templates

import "embed"

//go:embed files/*.ias
var FS embed.FS

// Template describes a project template.
type Template struct {
	Name        string
	Description string
	Filename    string
}

// All returns all available project templates.
func All() []Template {
	return []Template{
		{
			Name:        "customer-support",
			Description: "Customer support agent with order lookup and knowledge base tools",
			Filename:    "customer-support.ias",
		},
		{
			Name:        "rag-chatbot",
			Description: "RAG chatbot with document retrieval and semantic search",
			Filename:    "rag-chatbot.ias",
		},
		{
			Name:        "code-review-pipeline",
			Description: "Multi-agent pipeline for automated code review",
			Filename:    "code-review-pipeline.ias",
		},
		{
			Name:        "data-extraction",
			Description: "Data extraction agent with structured output parsing",
			Filename:    "data-extraction.ias",
		},
		{
			Name:        "research-assistant",
			Description: "Research assistant with web search and summarization",
			Filename:    "research-assistant.ias",
		},
		{
			Name:        "validated-agent",
			Description: "Agent with config, validation, eval, and control flow (IntentLang 3.0)",
			Filename:    "validated-agent.ias",
		},
	}
}

// Get returns a template by name, or nil if not found.
func Get(name string) *Template {
	for _, t := range All() {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

// Content returns the template file content.
func Content(t *Template) ([]byte, error) {
	return FS.ReadFile("files/" + t.Filename)
}
