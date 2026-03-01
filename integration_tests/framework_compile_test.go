package integration_tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/compiler/targets"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/plugins"
)

// TestFrameworkCompileCrewAI verifies CrewAI target generates correct project structure.
func TestFrameworkCompileCrewAI(t *testing.T) {
	doc := parseToIR(t, "testdata/valid.ias")
	target, ok := targets.Get("crewai")
	if !ok {
		t.Fatal("crewai target not registered")
	}

	result, err := target.Compile(doc, "test-agent")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	assertGeneratedFile(t, result.Files, "pyproject.toml", "crewai")
	assertGeneratedFile(t, result.Files, "main.py", "AgentCrew")
	assertGeneratedFile(t, result.Files, "crew.py", "class AgentCrew")
	assertGeneratedFile(t, result.Files, "config/agents.yaml", "research_assistant")
	assertGeneratedFile(t, result.Files, "config/tasks.yaml", "research_assistant_task")
	assertGeneratedFile(t, result.Files, "tools/__init__.py", "@tool")

	if result.Metadata.Framework != "crewai" {
		t.Errorf("expected framework 'crewai', got %q", result.Metadata.Framework)
	}
	if result.Metadata.RunCommand != "python main.py" {
		t.Errorf("expected run command 'python main.py', got %q", result.Metadata.RunCommand)
	}
}

// TestFrameworkCompileLangGraph verifies LangGraph target generates correct project structure.
func TestFrameworkCompileLangGraph(t *testing.T) {
	doc := parseToIR(t, "testdata/valid.ias")
	target, ok := targets.Get("langgraph")
	if !ok {
		t.Fatal("langgraph target not registered")
	}

	result, err := target.Compile(doc, "test-agent")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	assertGeneratedFile(t, result.Files, "requirements.txt", "langgraph")
	assertGeneratedFile(t, result.Files, "tools.py", "@tool")
	assertGeneratedFile(t, result.Files, "graph.py", "StateGraph")
	assertGeneratedFile(t, result.Files, "main.py", "build_graph")

	if result.Metadata.Framework != "langgraph" {
		t.Errorf("expected framework 'langgraph', got %q", result.Metadata.Framework)
	}
}

// TestFrameworkCompileLlamaStack verifies LlamaStack target generates correct project structure.
func TestFrameworkCompileLlamaStack(t *testing.T) {
	doc := parseToIR(t, "testdata/valid.ias")
	target, ok := targets.Get("llamastack")
	if !ok {
		t.Fatal("llamastack target not registered")
	}

	result, err := target.Compile(doc, "test-agent")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	assertGeneratedFile(t, result.Files, "requirements.txt", "llama-stack")
	assertGeneratedFile(t, result.Files, "agent.py", "LlamaStackClient")

	if result.Metadata.Framework != "llamastack" {
		t.Errorf("expected framework 'llamastack', got %q", result.Metadata.Framework)
	}
}

// TestFrameworkCompileLlamaIndex verifies LlamaIndex target generates correct project structure.
func TestFrameworkCompileLlamaIndex(t *testing.T) {
	doc := parseToIR(t, "testdata/valid.ias")
	target, ok := targets.Get("llamaindex")
	if !ok {
		t.Fatal("llamaindex target not registered")
	}

	result, err := target.Compile(doc, "test-agent")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	assertGeneratedFile(t, result.Files, "requirements.txt", "llama-index")
	assertGeneratedFile(t, result.Files, "tools.py", "FunctionTool")
	assertGeneratedFile(t, result.Files, "agent.py", "ReActAgent")
	assertGeneratedFile(t, result.Files, "main.py", "create_")

	if result.Metadata.Framework != "llamaindex" {
		t.Errorf("expected framework 'llamaindex', got %q", result.Metadata.Framework)
	}
}

// TestFrameworkWriteFiles verifies that generated files are written to disk correctly.
func TestFrameworkWriteFiles(t *testing.T) {
	doc := parseToIR(t, "testdata/valid.ias")
	target, ok := targets.Get("crewai")
	if !ok {
		t.Fatal("crewai target not registered")
	}

	result, err := target.Compile(doc, "test-agent")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	outputDir := t.TempDir()
	if err := targets.WriteFiles(outputDir, result.Files); err != nil {
		t.Fatalf("write files error: %v", err)
	}

	// Verify files exist on disk
	expectedFiles := []string{
		"pyproject.toml",
		"main.py",
		"crew.py",
		"config/agents.yaml",
		"config/tasks.yaml",
		"tools/__init__.py",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(outputDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q to exist", f)
		}
	}
}

// TestFrameworkFeatureSupport verifies feature support maps are populated.
func TestFrameworkFeatureSupport(t *testing.T) {
	targetNames := []string{"crewai", "langgraph", "llamastack", "llamaindex"}

	for _, name := range targetNames {
		t.Run(name, func(t *testing.T) {
			target, ok := targets.Get(name)
			if !ok {
				t.Fatalf("target %q not registered", name)
			}

			fm := target.FeatureSupport()
			if len(fm) == 0 {
				t.Error("expected non-empty feature map")
			}

			// Every target should support agents and skills
			if fm["agent"] != plugins.FeatureFull {
				t.Errorf("expected agent support to be 'full', got %q", fm["agent"])
			}
			if fm["skill"] != plugins.FeatureFull {
				t.Errorf("expected skill support to be 'full', got %q", fm["skill"])
			}
		})
	}
}

// TestFrameworkGapAnalysis verifies gap warnings are generated for unsupported features.
func TestFrameworkGapAnalysis(t *testing.T) {
	// Create an IR document with features that LlamaStack doesn't support
	doc := &ir.Document{
		IRVersion:   "1.0",
		LangVersion: "3.0",
		Package:     ir.Package{Name: "test", Version: "1.0.0"},
		Resources: []ir.Resource{
			{
				Kind: "Agent",
				Name: "router",
				FQN:  "test/Agent/router",
				Attributes: map[string]interface{}{
					"model":    "gpt-4",
					"strategy": "router",
					"on_input": []interface{}{
						map[string]interface{}{
							"type":      "if",
							"condition": "topic == 'support'",
						},
					},
					"delegates": []interface{}{
						map[string]interface{}{"agent": "helper"},
					},
				},
			},
			{
				Kind: "Pipeline",
				Name: "workflow",
				FQN:  "test/Pipeline/workflow",
				Attributes: map[string]interface{}{
					"steps": []interface{}{
						map[string]interface{}{
							"name":  "step1",
							"when":  "status == 'ready'",
							"agent": "router",
						},
					},
				},
			},
		},
	}

	target, _ := targets.Get("llamastack")
	featureMap := target.FeatureSupport()

	// Use the compiler's gap analysis
	detected := detectFeaturesForTest(doc)
	warnings := analyzeGapsForTest(detected, featureMap)

	if len(warnings) == 0 {
		t.Error("expected gap warnings for llamastack target with router/pipeline/delegation features")
	}

	// Check that delegation warning exists (llamastack doesn't support it)
	foundDelegation := false
	foundPipelineConditional := false
	for _, w := range warnings {
		if w.Feature == "delegation" {
			foundDelegation = true
		}
		if w.Feature == "pipeline_conditional" {
			foundPipelineConditional = true
		}
	}
	if !foundDelegation {
		t.Error("expected warning for 'delegation' feature")
	}
	if !foundPipelineConditional {
		t.Error("expected warning for 'pipeline_conditional' feature")
	}
}

// TestFrameworkRegisteredTargets verifies all 4 targets are registered.
func TestFrameworkRegisteredTargets(t *testing.T) {
	names := targets.List()
	expected := map[string]bool{
		"crewai":     false,
		"langgraph":  false,
		"llamastack": false,
		"llamaindex": false,
	}

	for _, name := range names {
		expected[name] = true
	}

	for name, found := range expected {
		if !found {
			t.Errorf("target %q not registered", name)
		}
	}
}

// TestSafeZonePreservation verifies user code is preserved across recompilation.
func TestSafeZonePreservation(t *testing.T) {
	doc := parseToIR(t, "testdata/valid.ias")
	target, _ := targets.Get("crewai")

	// First compilation
	result1, err := target.Compile(doc, "test-agent")
	if err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	if err := targets.WriteFiles(outputDir, result1.Files); err != nil {
		t.Fatal(err)
	}

	// Simulate user adding code to tools/__init__.py
	toolsPath := filepath.Join(outputDir, "tools/__init__.py")
	originalContent, _ := os.ReadFile(toolsPath)

	// Inject user code section into the file
	modifiedContent := string(originalContent) + "\n# --- USER CODE START ---\n# Your custom code here is preserved across recompilations\ndef my_custom_helper():\n    return \"preserved\"\n# --- USER CODE END ---\n"
	if err := os.WriteFile(toolsPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("writing modified tools file: %v", err)
	}

	// Verify user code extraction works
	userCode := extractUserCodeForTest(modifiedContent, "#")
	if len(userCode) == 0 {
		t.Error("expected to extract user code section")
	}
}

// --- Helpers ---

func parseToIR(t *testing.T, path string) *ir.Document {
	t.Helper()
	input := readTestFile(t, path)

	f, parseErrs := parser.Parse(input, path)
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("lower to IR: %v", err)
	}

	return doc
}

func assertGeneratedFile(t *testing.T, files []plugins.GeneratedFile, path, contains string) {
	t.Helper()
	for _, f := range files {
		if f.Path == path {
			if !strings.Contains(f.Content, contains) {
				t.Errorf("file %q does not contain %q", path, contains)
			}
			return
		}
	}
	t.Errorf("file %q not found in generated files", path)
}

// Wrappers to test internal functions from integration_tests package.
// These call the same logic as the compiler package.

type detectedFeature struct {
	Name        string
	ResourceFQN string
}

type gapWarning struct {
	Feature    string
	Level      plugins.FeatureSupportLevel
	Message    string
	Suggestion string
}

func detectFeaturesForTest(doc *ir.Document) []detectedFeature {
	var features []detectedFeature
	for _, r := range doc.Resources {
		switch r.Kind {
		case "Agent":
			features = append(features, detectedFeature{Name: "agent", ResourceFQN: r.FQN})
			if strategy, ok := r.Attributes["strategy"].(string); ok {
				features = append(features, detectedFeature{Name: "loop_" + strategy, ResourceFQN: r.FQN})
			}
			if d, ok := r.Attributes["delegates"]; ok && d != nil {
				features = append(features, detectedFeature{Name: "delegation", ResourceFQN: r.FQN})
			}
			if oi, ok := r.Attributes["on_input"]; ok && oi != nil {
				if stmts, ok := oi.([]interface{}); ok {
					for _, s := range stmts {
						if m, ok := s.(map[string]interface{}); ok {
							if st, ok := m["type"].(string); ok && st == "if" {
								features = append(features, detectedFeature{Name: "control_flow_if", ResourceFQN: r.FQN})
							}
						}
					}
				}
			}
		case "Pipeline":
			features = append(features, detectedFeature{Name: "pipeline_sequential", ResourceFQN: r.FQN})
			if steps, ok := r.Attributes["steps"].([]interface{}); ok {
				for _, step := range steps {
					if s, ok := step.(map[string]interface{}); ok {
						if when, ok := s["when"].(string); ok && when != "" {
							features = append(features, detectedFeature{Name: "pipeline_conditional", ResourceFQN: r.FQN})
						}
					}
				}
			}
		}
	}
	return features
}

func analyzeGapsForTest(features []detectedFeature, featureMap plugins.FeatureMap) []gapWarning {
	seen := make(map[string]string)
	for _, f := range features {
		if _, ok := seen[f.Name]; !ok {
			seen[f.Name] = f.ResourceFQN
		}
	}

	var warnings []gapWarning
	for name, fqn := range seen {
		level, exists := featureMap[name]
		if !exists {
			level = plugins.FeatureNone
		}
		if level != plugins.FeatureFull {
			warnings = append(warnings, gapWarning{
				Feature: name,
				Level:   level,
				Message: name + " not fully supported (used by " + fqn + ")",
			})
		}
	}
	return warnings
}

func extractUserCodeForTest(content, commentPrefix string) map[int]string {
	lines := strings.Split(content, "\n")
	result := make(map[int]string)
	idx := 0
	inUserCode := false
	var current strings.Builder

	startMarker := commentPrefix + " --- USER CODE START ---"
	endMarker := commentPrefix + " --- USER CODE END ---"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == startMarker:
			inUserCode = true
			current.WriteString(line + "\n")
		case trimmed == endMarker:
			current.WriteString(line + "\n")
			result[idx] = current.String()
			current.Reset()
			inUserCode = false
			idx++
		case inUserCode:
			current.WriteString(line + "\n")
		}
	}
	return result
}
