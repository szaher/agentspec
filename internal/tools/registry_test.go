package tools

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/szaher/designs/agentz/internal/llm"
)

// mockExecutor is a simple Executor that returns a fixed result or error.
type mockExecutor struct {
	result string
	err    error
}

func (m *mockExecutor) Execute(_ context.Context, _ map[string]interface{}) (string, error) {
	return m.result, m.err
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	defs := r.Definitions()
	if defs == nil {
		t.Fatal("Definitions() returned nil, expected empty slice")
	}
	if len(defs) != 0 {
		t.Fatalf("Definitions() returned %d items, expected 0", len(defs))
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	def := llm.ToolDefinition{
		Name:        "greet",
		Description: "Says hello",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
			},
		},
	}
	r.Register("greet", def, &mockExecutor{result: "hello"})

	defs := r.Definitions()
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "greet" {
		t.Errorf("expected tool name %q, got %q", "greet", defs[0].Name)
	}
	if defs[0].Description != "Says hello" {
		t.Errorf("expected description %q, got %q", "Says hello", defs[0].Description)
	}
}

func TestRegistry_Execute(t *testing.T) {
	r := NewRegistry()
	def := llm.ToolDefinition{Name: "echo", Description: "echoes input"}
	r.Register("echo", def, &mockExecutor{result: "echoed"})

	call := llm.ToolCall{
		ID:    "call-1",
		Name:  "echo",
		Input: map[string]interface{}{"text": "hi"},
	}
	result, err := r.Execute(context.Background(), call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "echoed" {
		t.Errorf("expected result %q, got %q", "echoed", result)
	}
}

func TestRegistry_ExecuteUnregistered(t *testing.T) {
	r := NewRegistry()
	call := llm.ToolCall{
		ID:   "call-1",
		Name: "nonexistent",
	}
	_, err := r.Execute(context.Background(), call)
	if err == nil {
		t.Fatal("expected error for unregistered tool, got nil")
	}
	expected := `tool "nonexistent" not registered`
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestRegistry_ExecuteConcurrent(t *testing.T) {
	r := NewRegistry()
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("tool-%d", i)
		def := llm.ToolDefinition{Name: name, Description: name}
		r.Register(name, def, &mockExecutor{result: fmt.Sprintf("result-%d", i)})
	}

	calls := []llm.ToolCall{
		{ID: "id-0", Name: "tool-0", Input: nil},
		{ID: "id-1", Name: "tool-1", Input: nil},
		{ID: "id-2", Name: "tool-2", Input: nil},
	}

	results := r.ExecuteConcurrent(context.Background(), calls)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for i, res := range results {
		expectedID := fmt.Sprintf("id-%d", i)
		expectedContent := fmt.Sprintf("result-%d", i)
		if res.ToolUseID != expectedID {
			t.Errorf("result[%d]: expected ToolUseID %q, got %q", i, expectedID, res.ToolUseID)
		}
		if res.Content != expectedContent {
			t.Errorf("result[%d]: expected Content %q, got %q", i, expectedContent, res.Content)
		}
		if res.IsError {
			t.Errorf("result[%d]: expected IsError false, got true", i)
		}
	}
}

func TestRegistry_ExecuteConcurrentWithError(t *testing.T) {
	r := NewRegistry()

	r.Register("pass", llm.ToolDefinition{Name: "pass"}, &mockExecutor{result: "ok"})
	r.Register("fail", llm.ToolDefinition{Name: "fail"}, &mockExecutor{err: errors.New("something went wrong")})

	calls := []llm.ToolCall{
		{ID: "id-pass", Name: "pass", Input: nil},
		{ID: "id-fail", Name: "fail", Input: nil},
	}

	results := r.ExecuteConcurrent(context.Background(), calls)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First result should succeed.
	if results[0].ToolUseID != "id-pass" {
		t.Errorf("expected ToolUseID %q, got %q", "id-pass", results[0].ToolUseID)
	}
	if results[0].Content != "ok" {
		t.Errorf("expected Content %q, got %q", "ok", results[0].Content)
	}
	if results[0].IsError {
		t.Error("expected IsError false for passing tool")
	}

	// Second result should be an error.
	if results[1].ToolUseID != "id-fail" {
		t.Errorf("expected ToolUseID %q, got %q", "id-fail", results[1].ToolUseID)
	}
	if results[1].Content != "something went wrong" {
		t.Errorf("expected Content %q, got %q", "something went wrong", results[1].Content)
	}
	if !results[1].IsError {
		t.Error("expected IsError true for failing tool")
	}
}

func TestRegistry_Definitions(t *testing.T) {
	r := NewRegistry()

	def1 := llm.ToolDefinition{Name: "alpha", Description: "first tool"}
	def2 := llm.ToolDefinition{Name: "beta", Description: "second tool"}

	r.Register("alpha", def1, &mockExecutor{result: "a"})
	r.Register("beta", def2, &mockExecutor{result: "b"})

	defs := r.Definitions()
	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}

	// Since map iteration order is not deterministic, collect by name.
	found := map[string]llm.ToolDefinition{}
	for _, d := range defs {
		found[d.Name] = d
	}

	if _, ok := found["alpha"]; !ok {
		t.Error("expected definition for 'alpha' to be present")
	}
	if _, ok := found["beta"]; !ok {
		t.Error("expected definition for 'beta' to be present")
	}
	if found["alpha"].Description != "first tool" {
		t.Errorf("expected description %q, got %q", "first tool", found["alpha"].Description)
	}
	if found["beta"].Description != "second tool" {
		t.Errorf("expected description %q, got %q", "second tool", found["beta"].Description)
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup

	// Launch 10 goroutines that register tools.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := fmt.Sprintf("concurrent-tool-%d", idx)
			def := llm.ToolDefinition{Name: name, Description: name}
			r.Register(name, def, &mockExecutor{result: fmt.Sprintf("result-%d", idx)})
		}(i)
	}

	// Launch 10 goroutines that execute tool calls (some may hit unregistered tools).
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			call := llm.ToolCall{
				ID:   fmt.Sprintf("concurrent-call-%d", idx),
				Name: fmt.Sprintf("concurrent-tool-%d", idx),
			}
			// We don't check the result because the tool may or may not be
			// registered yet due to goroutine scheduling. The important thing
			// is that there are no data races or panics.
			_, _ = r.Execute(context.Background(), call)
		}(i)
	}

	// Also read definitions concurrently.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Definitions()
		}()
	}

	wg.Wait()

	// After all goroutines complete, all 10 tools should be registered.
	defs := r.Definitions()
	if len(defs) != 10 {
		t.Errorf("expected 10 definitions after concurrent registration, got %d", len(defs))
	}
}
