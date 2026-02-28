package integration_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/runtime"
)

// TestBenchmarkCompilationTime measures compilation time for a large .ias file.
// Target: <10s for 500-line .ias (SC-002).
func TestBenchmarkCompilationTime(t *testing.T) {
	// Generate a ~500 line .ias file
	var sb strings.Builder
	sb.WriteString(`package "benchmark" version "1.0.0" lang "3.0"` + "\n\n")

	// Generate 50 prompts (~100 lines)
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&sb, "prompt \"prompt-%d\" {\n", i)
		fmt.Fprintf(&sb, "  content \"This is benchmark prompt number %d with some content.\"\n", i)
		sb.WriteString("}\n\n")
	}

	// Generate 10 agents with config and validation (~300 lines)
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&sb, "agent \"agent-%d\" {\n", i)
		fmt.Fprintf(&sb, "  model \"claude-sonnet-4-20250514\"\n")
		fmt.Fprintf(&sb, "  prompt \"prompt-%d\"\n", i)
		sb.WriteString("  strategy \"react\"\n")
		sb.WriteString("  max_turns 5\n")
		sb.WriteString("\n  config {\n")
		for j := 0; j < 3; j++ {
			fmt.Fprintf(&sb, "    param_%d string default \"val%d\" \"Param %d\"\n", j, j, j)
		}
		sb.WriteString("  }\n")
		sb.WriteString("\n  validate {\n")
		fmt.Fprintf(&sb, "    rule check_%d warning \"Check %d\" when output != \"\"\n", i, i)
		sb.WriteString("  }\n")
		sb.WriteString("}\n\n")
	}

	// Generate 10 skills (~100 lines)
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&sb, "skill \"skill-%d\" {\n", i)
		fmt.Fprintf(&sb, "  description \"Benchmark skill %d\"\n", i)
		sb.WriteString("  tool http {\n")
		sb.WriteString("    method \"POST\"\n")
		fmt.Fprintf(&sb, "    url \"https://api.example.com/skill/%d\"\n", i)
		sb.WriteString("  }\n")
		sb.WriteString("}\n\n")
	}

	content := sb.String()
	lines := strings.Count(content, "\n")
	t.Logf("Generated %d lines of .ias content", lines)

	// Measure full compilation pipeline
	start := time.Now()

	f, errs := parser.Parse(content, "benchmark.ias")
	if errs != nil {
		t.Fatalf("parse errors: %v", errs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("lower error: %v", err)
	}

	_, err = runtime.FromIR(doc)
	if err != nil {
		t.Fatalf("config error: %v", err)
	}

	elapsed := time.Since(start)
	t.Logf("Compilation time: %v", elapsed)

	if elapsed > 10*time.Second {
		t.Errorf("compilation took %v, exceeds 10s target (SC-002)", elapsed)
	}
}

// TestBenchmarkAgentStartup measures agent server startup time.
// Target: <3s (SC-004).
func TestBenchmarkAgentStartup(t *testing.T) {
	iasContent := `package "startup-test" version "1.0.0" lang "3.0"

prompt "sys" {
  content "You are a test agent."
}

agent "test-agent" {
  model "claude-sonnet-4-20250514"
  prompt "sys"
  strategy "react"
  max_turns 5
}
`
	f, errs := parser.Parse(iasContent, "startup.ias")
	if errs != nil {
		t.Fatalf("parse errors: %v", errs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("lower error: %v", err)
	}

	config, err := runtime.FromIR(doc)
	if err != nil {
		t.Fatalf("config error: %v", err)
	}

	start := time.Now()

	rt, err := runtime.New(config, runtime.Options{
		Port:     0, // auto-assign port
		EnableUI: true,
	})
	if err != nil {
		t.Fatalf("runtime creation error: %v", err)
	}

	elapsed := time.Since(start)
	t.Logf("Agent startup time: %v", elapsed)

	if elapsed > 3*time.Second {
		t.Errorf("startup took %v, exceeds 3s target (SC-004)", elapsed)
	}

	_ = rt // ensure rt is used
}

// TestBenchmarkPackageResolution measures local package resolution time.
// Target: <5s (SC-007).
func TestBenchmarkPackageResolution(t *testing.T) {
	start := time.Now()

	// Test parsing 20 .ias files to simulate package resolution
	for i := 0; i < 20; i++ {
		content := fmt.Sprintf(`package "pkg%d" version "1.0.0" lang "3.0"

prompt "pkg-prompt-%d" {
  content "Package prompt number %d"
}`, i, i, i)
		_, errs := parser.Parse(content, fmt.Sprintf("pkg%d.ias", i))
		if errs != nil {
			t.Fatalf("parse error: %v", errs)
		}
	}

	elapsed := time.Since(start)
	t.Logf("Package resolution time (20 files): %v", elapsed)

	if elapsed > 5*time.Second {
		t.Errorf("resolution took %v, exceeds 5s target (SC-007)", elapsed)
	}
}
