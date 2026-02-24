package integration_tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/runtime"
	"github.com/szaher/designs/agentz/internal/sdk/generator"
)

func TestSDKGeneratePython(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "python-sdk")
	cfg := generator.Config{
		Language: generator.LangPython,
		OutDir:   outDir,
	}

	if err := generator.Generate(cfg); err != nil {
		t.Fatalf("Python SDK generation failed: %v", err)
	}

	// Verify expected files
	for _, name := range []string{"__init__.py", "client.py", "pyproject.toml"} {
		path := filepath.Join(outDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s not found: %v", name, err)
		}
	}

	// Verify client.py contains runtime API methods
	clientBytes, err := os.ReadFile(filepath.Join(outDir, "client.py"))
	if err != nil {
		t.Fatalf("read client.py: %v", err)
	}
	client := string(clientBytes)
	for _, method := range []string{"invoke", "stream", "create_session", "run_pipeline", "list_agents"} {
		if !strings.Contains(client, method) {
			t.Errorf("client.py missing method: %s", method)
		}
	}
}

func TestSDKGenerateTypeScript(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "ts-sdk")
	cfg := generator.Config{
		Language: generator.LangTypeScript,
		OutDir:   outDir,
	}

	if err := generator.Generate(cfg); err != nil {
		t.Fatalf("TypeScript SDK generation failed: %v", err)
	}

	for _, name := range []string{"index.ts", "package.json", "tsconfig.json"} {
		path := filepath.Join(outDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s not found: %v", name, err)
		}
	}

	// Verify index.ts contains runtime API methods
	indexBytes, err := os.ReadFile(filepath.Join(outDir, "index.ts"))
	if err != nil {
		t.Fatalf("read index.ts: %v", err)
	}
	index := string(indexBytes)
	for _, method := range []string{"invoke", "listAgents", "createSession", "runPipeline"} {
		if !strings.Contains(index, method) {
			t.Errorf("index.ts missing method: %s", method)
		}
	}
}

func TestSDKGenerateGo(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "go-sdk")
	cfg := generator.Config{
		Language: generator.LangGo,
		OutDir:   outDir,
	}

	if err := generator.Generate(cfg); err != nil {
		t.Fatalf("Go SDK generation failed: %v", err)
	}

	// Go generator creates agentspec/ subdirectory
	path := filepath.Join(outDir, "agentspec", "client.go")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file agentspec/client.go not found: %v", err)
	}

	// Verify go.mod exists
	modPath := filepath.Join(outDir, "go.mod")
	if _, err := os.Stat(modPath); err != nil {
		t.Errorf("expected file go.mod not found: %v", err)
	}

	// Verify client.go contains runtime API methods
	clientBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read client.go: %v", err)
	}
	client := string(clientBytes)
	for _, method := range []string{"Invoke", "Stream", "ListAgents", "CreateSession", "RunPipeline"} {
		if !strings.Contains(client, method) {
			t.Errorf("client.go missing method: %s", method)
		}
	}
}

func TestSDKGenerateWithRuntimeConfig(t *testing.T) {
	rc := &runtime.RuntimeConfig{
		PackageName: "test-package",
		Agents: []runtime.AgentConfig{
			{Name: "support-bot", Model: "claude-sonnet-4-20250514", Strategy: "react", Skills: []string{"search"}},
			{Name: "code-reviewer", Model: "claude-sonnet-4-20250514", Strategy: "plan_execute"},
		},
		Pipelines: []runtime.PipelineConfig{
			{Name: "code-review", Steps: []runtime.PipelineStepConfig{{Name: "analyze"}}},
		},
	}

	outDir := filepath.Join(t.TempDir(), "typed-sdk")
	cfg := generator.Config{
		Language:      generator.LangPython,
		OutDir:        outDir,
		RuntimeConfig: rc,
	}

	if err := generator.Generate(cfg); err != nil {
		t.Fatalf("typed Python SDK generation failed: %v", err)
	}

	clientBytes, err := os.ReadFile(filepath.Join(outDir, "client.py"))
	if err != nil {
		t.Fatalf("read client.py: %v", err)
	}
	client := string(clientBytes)

	// Verify agent constants are generated
	if !strings.Contains(client, "AGENT_SUPPORT_BOT") {
		t.Error("expected AGENT_SUPPORT_BOT constant in generated client")
	}
	if !strings.Contains(client, "AGENT_CODE_REVIEWER") {
		t.Error("expected AGENT_CODE_REVIEWER constant in generated client")
	}
	if !strings.Contains(client, "PIPELINE_CODE_REVIEW") {
		t.Error("expected PIPELINE_CODE_REVIEW constant in generated client")
	}
}

func TestSDKGenerateDeterminism(t *testing.T) {
	dir1 := filepath.Join(t.TempDir(), "sdk1")
	dir2 := filepath.Join(t.TempDir(), "sdk2")

	for _, lang := range []generator.Language{generator.LangPython, generator.LangTypeScript, generator.LangGo} {
		out1 := filepath.Join(dir1, string(lang))
		out2 := filepath.Join(dir2, string(lang))

		if err := generator.Generate(generator.Config{Language: lang, OutDir: out1}); err != nil {
			t.Fatalf("generate %s to out1 failed: %v", lang, err)
		}
		if err := generator.Generate(generator.Config{Language: lang, OutDir: out2}); err != nil {
			t.Fatalf("generate %s to out2 failed: %v", lang, err)
		}

		// Compare all files recursively
		compareDir(t, string(lang), out1, out2)
	}
}

func TestSDKGenerateAll(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "all-sdks")
	rc := &runtime.RuntimeConfig{
		PackageName: "test-pkg",
		Agents:      []runtime.AgentConfig{{Name: "test-agent"}},
	}

	if err := generator.GenerateAll(baseDir, rc); err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	// Verify all language directories exist
	for _, lang := range []string{"python", "typescript", "go"} {
		if _, err := os.Stat(filepath.Join(baseDir, lang)); err != nil {
			t.Errorf("expected %s directory not found", lang)
		}
	}
}

func compareDir(t *testing.T, lang, dir1, dir2 string) {
	t.Helper()
	err := filepath.Walk(dir1, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir1, path)
		data1, _ := os.ReadFile(path)
		data2, _ := os.ReadFile(filepath.Join(dir2, rel))
		if string(data1) != string(data2) {
			t.Errorf("%s SDK file %s is not deterministic", lang, rel)
		}
		return nil
	})
	if err != nil {
		t.Errorf("walk %s dir1: %v", lang, err)
	}
}
