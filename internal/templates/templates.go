// Package templates provides embedded project templates for agentspec init.
package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed files/*
var FS embed.FS

// Template describes a project template.
type Template struct {
	Name        string
	Description string
	Category    string // beginner, intermediate, advanced
	Dir         string // directory name under files/
	Filename    string // primary .ias filename
}

// All returns all available project templates ordered by category.
func All() []Template {
	return []Template{
		// Beginner
		{
			Name:        "basic-chatbot",
			Description: "Simple conversational agent",
			Category:    "beginner",
			Dir:         "basic-chatbot",
			Filename:    "basic-chatbot.ias",
		},
		{
			Name:        "support-bot",
			Description: "Customer support with tools",
			Category:    "beginner",
			Dir:         "support-bot",
			Filename:    "support-bot.ias",
		},
		// Intermediate
		{
			Name:        "rag-assistant",
			Description: "RAG with document retrieval",
			Category:    "intermediate",
			Dir:         "rag-assistant",
			Filename:    "rag-assistant.ias",
		},
		{
			Name:        "incident-response",
			Description: "Incident triage and response",
			Category:    "intermediate",
			Dir:         "incident-response",
			Filename:    "incident-response.ias",
		},
		// Advanced
		{
			Name:        "research-swarm",
			Description: "Multi-agent research coordination",
			Category:    "advanced",
			Dir:         "research-swarm",
			Filename:    "research-swarm.ias",
		},
		{
			Name:        "multi-agent-router",
			Description: "Request routing across agents",
			Category:    "advanced",
			Dir:         "multi-agent-router",
			Filename:    "multi-agent-router.ias",
		},
		// Additional templates (not in the required 6 but kept for existing users)
		{
			Name:        "code-review-pipeline",
			Description: "Multi-agent pipeline for automated code review",
			Category:    "advanced",
			Dir:         "code-review-pipeline",
			Filename:    "code-review-pipeline.ias",
		},
		{
			Name:        "data-extraction",
			Description: "Data extraction with structured output parsing",
			Category:    "intermediate",
			Dir:         "data-extraction",
			Filename:    "data-extraction.ias",
		},
		{
			Name:        "research-assistant",
			Description: "Research assistant with web search and summarization",
			Category:    "intermediate",
			Dir:         "research-assistant",
			Filename:    "research-assistant.ias",
		},
		{
			Name:        "validated-agent",
			Description: "Agent with config, validation, eval, and control flow",
			Category:    "advanced",
			Dir:         "validated-agent",
			Filename:    "validated-agent.ias",
		},
	}
}

// Required returns the 6 required starter templates.
func Required() []Template {
	all := All()
	if len(all) >= 6 {
		return all[:6]
	}
	return all
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

// Content returns the primary .ias file content for a template.
func Content(t *Template) ([]byte, error) {
	return FS.ReadFile("files/" + t.Dir + "/" + t.Filename)
}

// ScaffoldDir copies all files from a template directory to the target path,
// replacing {{.PackageName}} in .ias files.
func ScaffoldDir(t *Template, targetDir, packageName string) ([]string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	templateRoot := "files/" + t.Dir
	var created []string

	err := fs.WalkDir(FS, templateRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(templateRoot, path)
		destPath := filepath.Join(targetDir, relPath)

		data, readErr := FS.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}

		// Replace template variables in .ias files
		if strings.HasSuffix(relPath, ".ias") {
			data = []byte(strings.ReplaceAll(string(data), "{{.PackageName}}", packageName))
		}

		if writeErr := os.WriteFile(destPath, data, 0644); writeErr != nil {
			return fmt.Errorf("write %s: %w", destPath, writeErr)
		}

		created = append(created, relPath)
		return nil
	})

	return created, err
}
