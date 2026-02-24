package integration_tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/tools"
)

func TestToolRegistryDispatch(t *testing.T) {
	registry := tools.NewRegistry()

	registry.Register("echo", llm.ToolDefinition{
		Name:        "echo",
		Description: "Echo back input",
	}, &mockToolExecutor{output: "echoed"})

	result, err := registry.Execute(context.Background(), llm.ToolCall{
		ID:    "tc_1",
		Name:  "echo",
		Input: map[string]interface{}{"text": "hello"},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result != "echoed" {
		t.Errorf("expected 'echoed', got %q", result)
	}
}

func TestToolRegistryUnknownTool(t *testing.T) {
	registry := tools.NewRegistry()
	_, err := registry.Execute(context.Background(), llm.ToolCall{
		ID:   "tc_1",
		Name: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestToolRegistryConcurrentExecution(t *testing.T) {
	registry := tools.NewRegistry()

	registry.Register("tool_a", llm.ToolDefinition{Name: "tool_a"}, &mockToolExecutor{output: "a_result"})
	registry.Register("tool_b", llm.ToolDefinition{Name: "tool_b"}, &mockToolExecutor{output: "b_result"})

	calls := []llm.ToolCall{
		{ID: "tc_1", Name: "tool_a", Input: map[string]interface{}{}},
		{ID: "tc_2", Name: "tool_b", Input: map[string]interface{}{}},
	}

	results := registry.ExecuteConcurrent(context.Background(), calls)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Content != "a_result" {
		t.Errorf("expected 'a_result', got %q", results[0].Content)
	}
	if results[0].ToolUseID != "tc_1" {
		t.Errorf("expected tool_use_id 'tc_1', got %q", results[0].ToolUseID)
	}
	if results[1].Content != "b_result" {
		t.Errorf("expected 'b_result', got %q", results[1].Content)
	}
	if results[1].IsError {
		t.Error("expected no error for tool_b")
	}
}

func TestHTTPToolExecutor(t *testing.T) {
	// Start a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		query, _ := body["query"].(string)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"result": "success",
			"query":  query,
		})
	}))
	defer ts.Close()

	executor := tools.NewHTTPExecutor(tools.HTTPConfig{
		Method: "POST",
		URL:    ts.URL,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})

	result, err := executor.Execute(context.Background(), map[string]interface{}{
		"query": "test-query",
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	var resp map[string]string
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp["result"] != "success" {
		t.Errorf("expected 'success', got %q", resp["result"])
	}
	if resp["query"] != "test-query" {
		t.Errorf("expected 'test-query', got %q", resp["query"])
	}
}

func TestCommandToolExecutor(t *testing.T) {
	executor := tools.NewCommandExecutor(tools.CommandConfig{
		Binary: "echo",
		Args:   []string{"hello world"},
	}, nil)

	result, err := executor.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if result != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got %q", result)
	}
}

func TestToolDefinitionsReturned(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register("tool1", llm.ToolDefinition{
		Name:        "tool1",
		Description: "First tool",
	}, &mockToolExecutor{output: ""})
	registry.Register("tool2", llm.ToolDefinition{
		Name:        "tool2",
		Description: "Second tool",
	}, &mockToolExecutor{output: ""})

	defs := registry.Definitions()
	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}

	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Name] = true
	}
	if !names["tool1"] || !names["tool2"] {
		t.Errorf("expected tool1 and tool2 in definitions, got %v", names)
	}
}
