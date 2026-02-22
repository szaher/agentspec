package integration_tests

import (
	"os"
	"path/filepath"
	"testing"

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
	for _, name := range []string{"__init__.py", "client.py", "types.py", "errors.py", "setup.py"} {
		path := filepath.Join(outDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s not found: %v", name, err)
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

	for _, name := range []string{"index.ts", "client.ts", "types.ts", "errors.ts", "package.json"} {
		path := filepath.Join(outDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s not found: %v", name, err)
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

	path := filepath.Join(outDir, "client.go")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file client.go not found: %v", err)
	}
}

func TestSDKGenerateDeterminism(t *testing.T) {
	dir1 := filepath.Join(t.TempDir(), "sdk1")
	dir2 := filepath.Join(t.TempDir(), "sdk2")

	for _, lang := range []generator.Language{generator.LangPython, generator.LangTypeScript, generator.LangGo} {
		out1 := filepath.Join(dir1, string(lang))
		out2 := filepath.Join(dir2, string(lang))

		generator.Generate(generator.Config{Language: lang, OutDir: out1})
		generator.Generate(generator.Config{Language: lang, OutDir: out2})

		// Compare all files
		files1, _ := os.ReadDir(out1)
		for _, f := range files1 {
			data1, _ := os.ReadFile(filepath.Join(out1, f.Name()))
			data2, _ := os.ReadFile(filepath.Join(out2, f.Name()))
			if string(data1) != string(data2) {
				t.Errorf("%s SDK file %s is not deterministic", lang, f.Name())
			}
		}
	}
}
