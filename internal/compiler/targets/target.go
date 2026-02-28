// Package targets provides built-in compilation targets that generate
// framework-specific source code from AgentSpec IR documents.
package targets

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/plugins"
)

// Target is the interface for a built-in compilation target.
type Target interface {
	// Name returns the target identifier (e.g., "crewai", "langgraph").
	Name() string

	// FeatureSupport returns the feature support map for this target.
	FeatureSupport() plugins.FeatureMap

	// Compile generates source code files from an IR document.
	Compile(doc *ir.Document, name string) (*Result, error)
}

// Result is the output of a built-in compilation target.
type Result struct {
	Files    []plugins.GeneratedFile
	Warnings []plugins.CompileWarning
	Metadata plugins.CompileMetadata
}

// registry holds all built-in compilation targets.
var registry = map[string]Target{}

// Register registers a built-in compilation target.
func Register(t Target) {
	registry[t.Name()] = t
}

// Get returns a built-in target by name.
func Get(name string) (Target, bool) {
	t, ok := registry[name]
	return t, ok
}

// List returns all registered built-in target names.
func List() []string {
	var names []string
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// WriteFiles writes generated files to the output directory, creating
// subdirectories as needed.
func WriteFiles(outputDir string, files []plugins.GeneratedFile) error {
	for _, f := range files {
		fullPath := filepath.Join(outputDir, f.Path)

		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}

		mode := os.FileMode(0644)
		if f.Mode == "0755" {
			mode = 0755
		}

		if err := os.WriteFile(fullPath, []byte(f.Content), mode); err != nil {
			return fmt.Errorf("writing %s: %w", fullPath, err)
		}
	}

	return nil
}

// extractAgents returns all Agent resources from an IR document.
func extractAgents(doc *ir.Document) []ir.Resource {
	var agents []ir.Resource
	for _, r := range doc.Resources {
		if r.Kind == "Agent" {
			agents = append(agents, r)
		}
	}
	return agents
}

// extractSkills returns all Skill resources from an IR document.
func extractSkills(doc *ir.Document) []ir.Resource {
	var skills []ir.Resource
	for _, r := range doc.Resources {
		if r.Kind == "Skill" {
			skills = append(skills, r)
		}
	}
	return skills
}

// extractPrompts returns all Prompt resources from an IR document.
func extractPrompts(doc *ir.Document) []ir.Resource {
	var prompts []ir.Resource
	for _, r := range doc.Resources {
		if r.Kind == "Prompt" {
			prompts = append(prompts, r)
		}
	}
	return prompts
}

// extractPipelines returns all Pipeline resources from an IR document.
func extractPipelines(doc *ir.Document) []ir.Resource {
	var pipelines []ir.Resource
	for _, r := range doc.Resources {
		if r.Kind == "Pipeline" {
			pipelines = append(pipelines, r)
		}
	}
	return pipelines
}

// getPromptContent looks up a prompt name in the document and returns its content.
func getPromptContent(doc *ir.Document, promptName string) string {
	for _, r := range doc.Resources {
		if r.Kind == "Prompt" && r.Name == promptName {
			if content, ok := r.Attributes["content"].(string); ok {
				return content
			}
		}
	}
	return ""
}

// getStringAttr safely gets a string attribute from a resource.
func getStringAttr(r ir.Resource, key string) string {
	if v, ok := r.Attributes[key].(string); ok {
		return v
	}
	return ""
}

// getIntAttr safely gets an int attribute from a resource.
func getIntAttr(r ir.Resource, key string) int {
	switch v := r.Attributes[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

// getStringSliceAttr safely gets a []string attribute from a resource.
func getStringSliceAttr(r ir.Resource, key string) []string {
	switch v := r.Attributes[key].(type) {
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return v
	}
	return nil
}

// pythonSafe converts a name to a Python-safe identifier.
func pythonSafe(name string) string {
	result := make([]byte, 0, len(name))
	for i, c := range name {
		if c == '-' || c == '.' || c == ' ' {
			result = append(result, '_')
		} else if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || (i > 0 && c >= '0' && c <= '9') {
			result = append(result, byte(c))
		}
	}
	if len(result) == 0 {
		return "unnamed"
	}
	return string(result)
}
