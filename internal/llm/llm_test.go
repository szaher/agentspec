package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- ParseModelString Tests (table-driven) ---

func TestParseModelString(t *testing.T) {
	// Unset env vars that could influence provider detection
	t.Setenv("OLLAMA_HOST", "")
	t.Setenv("OPENAI_API_KEY", "")

	tests := []struct {
		name         string
		input        string
		wantProvider Provider
		wantModel    string
	}{
		{
			name:         "anthropic prefix",
			input:        "anthropic/claude-3",
			wantProvider: ProviderAnthropic,
			wantModel:    "claude-3",
		},
		{
			name:         "openai prefix",
			input:        "openai/gpt-4",
			wantProvider: ProviderOpenAI,
			wantModel:    "gpt-4",
		},
		{
			name:         "ollama prefix",
			input:        "ollama/llama2",
			wantProvider: ProviderOllama,
			wantModel:    "llama2",
		},
		{
			name:         "claude model name inferred as anthropic",
			input:        "claude-sonnet-4-20250514",
			wantProvider: ProviderAnthropic,
			wantModel:    "claude-sonnet-4-20250514",
		},
		{
			name:         "gpt model name inferred as openai",
			input:        "gpt-4o",
			wantProvider: ProviderOpenAI,
			wantModel:    "gpt-4o",
		},
		{
			name:         "o1 model name inferred as openai",
			input:        "o1-preview",
			wantProvider: ProviderOpenAI,
			wantModel:    "o1-preview",
		},
		{
			name:         "o3 model name inferred as openai",
			input:        "o3-mini",
			wantProvider: ProviderOpenAI,
			wantModel:    "o3-mini",
		},
		{
			name:         "unknown model defaults to anthropic",
			input:        "llama3.2",
			wantProvider: ProviderAnthropic,
			wantModel:    "llama3.2",
		},
		{
			name:         "case-insensitive prefix",
			input:        "Anthropic/claude-3.5",
			wantProvider: ProviderAnthropic,
			wantModel:    "claude-3.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProvider, gotModel := ParseModelString(tt.input)
			if gotProvider != tt.wantProvider {
				t.Errorf("ParseModelString(%q) provider = %q, want %q", tt.input, gotProvider, tt.wantProvider)
			}
			if gotModel != tt.wantModel {
				t.Errorf("ParseModelString(%q) model = %q, want %q", tt.input, gotModel, tt.wantModel)
			}
		})
	}
}

func TestParseModelStringWithOllamaEnv(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "http://localhost:11434")
	t.Setenv("OPENAI_API_KEY", "")

	provider, model := ParseModelString("llama3.2")
	if provider != ProviderOllama {
		t.Errorf("expected ProviderOllama when OLLAMA_HOST is set, got %q", provider)
	}
	if model != "llama3.2" {
		t.Errorf("expected model 'llama3.2', got %q", model)
	}
}

func TestParseModelStringWithOpenAIEnv(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "")
	t.Setenv("OPENAI_API_KEY", "sk-test-key")

	provider, model := ParseModelString("some-unknown-model")
	if provider != ProviderOpenAI {
		t.Errorf("expected ProviderOpenAI when OPENAI_API_KEY is set, got %q", provider)
	}
	if model != "some-unknown-model" {
		t.Errorf("expected model 'some-unknown-model', got %q", model)
	}
}

// --- TokenTracker Tests ---

func TestNewTokenTracker(t *testing.T) {
	tracker := NewTokenTracker(1000)
	if tracker == nil {
		t.Fatal("expected non-nil TokenTracker")
	}
	if tracker.budget != 1000 {
		t.Errorf("expected budget=1000, got %d", tracker.budget)
	}
}

func TestTokenTrackerAdd(t *testing.T) {
	tracker := NewTokenTracker(10000)

	tracker.Add(TokenUsage{InputTokens: 100, OutputTokens: 50, CacheRead: 10, CacheWrite: 5})
	tracker.Add(TokenUsage{InputTokens: 200, OutputTokens: 100, CacheRead: 20, CacheWrite: 10})

	usage := tracker.Usage()
	if usage.InputTokens != 300 {
		t.Errorf("expected InputTokens=300, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 150 {
		t.Errorf("expected OutputTokens=150, got %d", usage.OutputTokens)
	}
	if usage.CacheRead != 30 {
		t.Errorf("expected CacheRead=30, got %d", usage.CacheRead)
	}
	if usage.CacheWrite != 15 {
		t.Errorf("expected CacheWrite=15, got %d", usage.CacheWrite)
	}
}

func TestTokenTrackerCheckBudget(t *testing.T) {
	tracker := NewTokenTracker(500)

	// Add 300 total tokens (200 input + 100 output)
	tracker.Add(TokenUsage{InputTokens: 200, OutputTokens: 100})

	// Check: 300 used + 100 additional = 400 <= 500 budget -- should pass
	if err := tracker.CheckBudget(100); err != nil {
		t.Errorf("expected no error for within-budget check, got: %v", err)
	}

	// Check: 300 used + 250 additional = 550 > 500 budget -- should fail
	if err := tracker.CheckBudget(250); err == nil {
		t.Error("expected error for over-budget check, got nil")
	}
}

func TestTokenTrackerCheckBudgetUnlimited(t *testing.T) {
	tracker := NewTokenTracker(0) // unlimited

	tracker.Add(TokenUsage{InputTokens: 999999, OutputTokens: 999999})

	// Should never fail with unlimited budget
	if err := tracker.CheckBudget(999999); err != nil {
		t.Errorf("expected no error for unlimited budget, got: %v", err)
	}
}

func TestTokenTrackerRemaining(t *testing.T) {
	t.Run("with budget", func(t *testing.T) {
		tracker := NewTokenTracker(1000)
		tracker.Add(TokenUsage{InputTokens: 300, OutputTokens: 200})

		remaining := tracker.Remaining()
		if remaining != 500 {
			t.Errorf("expected remaining=500, got %d", remaining)
		}
	})

	t.Run("unlimited budget", func(t *testing.T) {
		tracker := NewTokenTracker(0)
		remaining := tracker.Remaining()
		if remaining != -1 {
			t.Errorf("expected remaining=-1 for unlimited, got %d", remaining)
		}
	})

	t.Run("overused budget returns 0", func(t *testing.T) {
		tracker := NewTokenTracker(100)
		tracker.Add(TokenUsage{InputTokens: 80, OutputTokens: 80}) // 160 total > 100 budget

		remaining := tracker.Remaining()
		if remaining != 0 {
			t.Errorf("expected remaining=0 for overused budget, got %d", remaining)
		}
	})
}

// --- MockClient Tests ---

func TestMockClientChat(t *testing.T) {
	mock := NewMockClient(
		MockResponse{Content: "first response", StopReason: StopEndTurn},
		MockResponse{Content: "second response", StopReason: StopEndTurn},
	)

	ctx := context.Background()

	// First call
	resp1, err := mock.Chat(ctx, ChatRequest{Model: "test", Messages: []Message{{Role: RoleUser, Content: "q1"}}})
	if err != nil {
		t.Fatalf("first Chat error: %v", err)
	}
	if resp1.Content != "first response" {
		t.Errorf("expected 'first response', got %q", resp1.Content)
	}

	// Second call
	resp2, err := mock.Chat(ctx, ChatRequest{Model: "test", Messages: []Message{{Role: RoleUser, Content: "q2"}}})
	if err != nil {
		t.Fatalf("second Chat error: %v", err)
	}
	if resp2.Content != "second response" {
		t.Errorf("expected 'second response', got %q", resp2.Content)
	}

	// Third call: should repeat last response
	resp3, err := mock.Chat(ctx, ChatRequest{Model: "test", Messages: []Message{{Role: RoleUser, Content: "q3"}}})
	if err != nil {
		t.Fatalf("third Chat error: %v", err)
	}
	if resp3.Content != "second response" {
		t.Errorf("expected 'second response' (repeated), got %q", resp3.Content)
	}
}

func TestMockClientCalls(t *testing.T) {
	mock := NewMockClient(MockResponse{Content: "ok"})
	ctx := context.Background()

	req1 := ChatRequest{Model: "m1", Messages: []Message{{Role: RoleUser, Content: "q1"}}}
	req2 := ChatRequest{Model: "m2", Messages: []Message{{Role: RoleUser, Content: "q2"}}}

	_, _ = mock.Chat(ctx, req1)
	_, _ = mock.Chat(ctx, req2)

	calls := mock.Calls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls recorded, got %d", len(calls))
	}
	if calls[0].Model != "m1" {
		t.Errorf("expected first call model='m1', got %q", calls[0].Model)
	}
	if calls[1].Model != "m2" {
		t.Errorf("expected second call model='m2', got %q", calls[1].Model)
	}
}

func TestMockClientReset(t *testing.T) {
	mock := NewMockClient(
		MockResponse{Content: "first"},
		MockResponse{Content: "second"},
	)
	ctx := context.Background()

	_, _ = mock.Chat(ctx, ChatRequest{Model: "test"})
	mock.Reset()

	if len(mock.Calls()) != 0 {
		t.Error("expected 0 calls after Reset")
	}

	// After reset, should start from first response again
	resp, _ := mock.Chat(ctx, ChatRequest{Model: "test"})
	if resp.Content != "first" {
		t.Errorf("expected 'first' after reset, got %q", resp.Content)
	}
}

func TestMockClientChatError(t *testing.T) {
	mock := NewMockClient(MockResponse{Error: fmt.Errorf("api error")})
	ctx := context.Background()

	_, err := mock.Chat(ctx, ChatRequest{Model: "test"})
	if err == nil {
		t.Fatal("expected error from mock, got nil")
	}
	if err.Error() != "api error" {
		t.Errorf("expected 'api error', got %q", err.Error())
	}
}

func TestMockClientNoResponses(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	_, err := mock.Chat(ctx, ChatRequest{Model: "test"})
	if err == nil {
		t.Fatal("expected error when no responses configured, got nil")
	}
}

func TestMockClientChatStream(t *testing.T) {
	mock := NewMockClient(MockResponse{
		Content:    "streamed text",
		StopReason: StopEndTurn,
		Usage:      TokenUsage{InputTokens: 10, OutputTokens: 5},
	})

	ctx := context.Background()
	ch, err := mock.ChatStream(ctx, ChatRequest{Model: "test"})
	if err != nil {
		t.Fatalf("ChatStream error: %v", err)
	}

	var events []StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) < 2 {
		t.Fatalf("expected at least 2 events (text + done), got %d", len(events))
	}

	// First event should be text
	if events[0].Type != "text" {
		t.Errorf("expected first event type='text', got %q", events[0].Type)
	}
	if events[0].Text != "streamed text" {
		t.Errorf("expected text='streamed text', got %q", events[0].Text)
	}

	// Last event should be done
	last := events[len(events)-1]
	if last.Type != "done" {
		t.Errorf("expected last event type='done', got %q", last.Type)
	}
	if last.Response == nil {
		t.Fatal("expected done event to have Response")
	}
}

// --- TokenUsage Tests ---

func TestTokenUsageTotal(t *testing.T) {
	usage := TokenUsage{InputTokens: 100, OutputTokens: 50, CacheRead: 10, CacheWrite: 5}
	total := usage.Total()
	// Total() returns InputTokens + OutputTokens (not cache tokens)
	if total != 150 {
		t.Errorf("expected Total()=150, got %d", total)
	}
}

func TestTokenUsageTotalZero(t *testing.T) {
	usage := TokenUsage{}
	if usage.Total() != 0 {
		t.Errorf("expected Total()=0 for zero usage, got %d", usage.Total())
	}
}

// --- Message and Type Tests ---

func TestRoleConstants(t *testing.T) {
	if RoleUser != "user" {
		t.Errorf("expected RoleUser='user', got %q", RoleUser)
	}
	if RoleAssistant != "assistant" {
		t.Errorf("expected RoleAssistant='assistant', got %q", RoleAssistant)
	}
}

func TestStopReasonConstants(t *testing.T) {
	tests := []struct {
		name  string
		value StopReason
		want  string
	}{
		{"StopEndTurn", StopEndTurn, "end_turn"},
		{"StopMaxTokens", StopMaxTokens, "max_tokens"},
		{"StopToolUse", StopToolUse, "tool_use"},
		{"StopStopSequence", StopStopSequence, "stop_sequence"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.want {
				t.Errorf("expected %s=%q, got %q", tt.name, tt.want, tt.value)
			}
		})
	}
}

func TestMessageConstruction(t *testing.T) {
	msg := Message{
		Role:    RoleAssistant,
		Content: "response text",
		ToolCalls: []ToolCall{
			{
				ID:    "tc-1",
				Name:  "search",
				Input: map[string]interface{}{"query": "test"},
			},
		},
	}

	if msg.Role != RoleAssistant {
		t.Errorf("expected role=assistant, got %q", msg.Role)
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].Name != "search" {
		t.Errorf("expected tool call name='search', got %q", msg.ToolCalls[0].Name)
	}
}

func TestToolResultConstruction(t *testing.T) {
	msg := Message{
		Role: RoleUser,
		ToolResult: &ToolResult{
			ToolUseID: "tc-1",
			Content:   "result data",
			IsError:   false,
		},
	}

	if msg.ToolResult == nil {
		t.Fatal("expected non-nil ToolResult")
	}
	if msg.ToolResult.ToolUseID != "tc-1" {
		t.Errorf("expected ToolUseID='tc-1', got %q", msg.ToolResult.ToolUseID)
	}
	if msg.ToolResult.Content != "result data" {
		t.Errorf("expected Content='result data', got %q", msg.ToolResult.Content)
	}
	if msg.ToolResult.IsError {
		t.Error("expected IsError=false")
	}
}

func TestProviderConstants(t *testing.T) {
	if ProviderAnthropic != "anthropic" {
		t.Errorf("expected ProviderAnthropic='anthropic', got %q", ProviderAnthropic)
	}
	if ProviderOpenAI != "openai" {
		t.Errorf("expected ProviderOpenAI='openai', got %q", ProviderOpenAI)
	}
	if ProviderOllama != "ollama" {
		t.Errorf("expected ProviderOllama='ollama', got %q", ProviderOllama)
	}
}

func TestToolDefinitionConstruction(t *testing.T) {
	td := ToolDefinition{
		Name:        "calculator",
		Description: "Performs arithmetic",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "math expression",
				},
			},
		},
	}

	if td.Name != "calculator" {
		t.Errorf("expected name='calculator', got %q", td.Name)
	}
	if td.Description != "Performs arithmetic" {
		t.Errorf("expected description='Performs arithmetic', got %q", td.Description)
	}
	if td.InputSchema == nil {
		t.Fatal("expected non-nil InputSchema")
	}
	if td.InputSchema["type"] != "object" {
		t.Errorf("expected schema type='object', got %v", td.InputSchema["type"])
	}
}

// --- OpenAI Client Tests (using httptest) ---

func TestOpenAIClientChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("expected /chat/completions path, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization 'Bearer test-key', got %q", r.Header.Get("Authorization"))
		}

		// Decode the request to verify it was constructed correctly
		var req oaiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		resp := oaiResponse{
			Choices: []oaiChoice{
				{
					Message:      oaiMessage{Role: "assistant", Content: "Hello from OpenAI!"},
					FinishReason: "stop",
				},
			},
			Usage: oaiUsage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(server.URL+"/v1", "test-key")
	ctx := context.Background()

	resp, err := client.Chat(ctx, ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: RoleUser, Content: "Hi"},
		},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if resp.Content != "Hello from OpenAI!" {
		t.Errorf("expected content 'Hello from OpenAI!', got %q", resp.Content)
	}
	if resp.StopReason != StopEndTurn {
		t.Errorf("expected StopEndTurn, got %q", resp.StopReason)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("expected InputTokens=10, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 20 {
		t.Errorf("expected OutputTokens=20, got %d", resp.Usage.OutputTokens)
	}
}

func TestOpenAIClientChatWithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req oaiRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		// Verify tools were sent
		if len(req.Tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(req.Tools))
		}

		resp := oaiResponse{
			Choices: []oaiChoice{
				{
					Message: oaiMessage{
						Role: "assistant",
						ToolCalls: []oaiToolCall{
							{
								ID:   "call-1",
								Type: "function",
								Function: oaiToolCallFunc{
									Name:      "search",
									Arguments: `{"query":"test"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
			Usage: oaiUsage{PromptTokens: 15, CompletionTokens: 10},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(server.URL+"/v1", "key")
	ctx := context.Background()

	resp, err := client.Chat(ctx, ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: RoleUser, Content: "search for something"},
		},
		Tools: []ToolDefinition{
			{
				Name:        "search",
				Description: "Search the web",
				InputSchema: map[string]interface{}{"type": "object"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if resp.StopReason != StopToolUse {
		t.Errorf("expected StopToolUse, got %q", resp.StopReason)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "search" {
		t.Errorf("expected tool name 'search', got %q", resp.ToolCalls[0].Name)
	}
	if resp.ToolCalls[0].Input["query"] != "test" {
		t.Errorf("expected tool input query='test', got %v", resp.ToolCalls[0].Input["query"])
	}
}

func TestOpenAIClientChatAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(oaiResponse{
			Error: &oaiError{Type: "authentication_error", Message: "invalid api key"},
		})
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(server.URL+"/v1", "bad-key")
	ctx := context.Background()

	_, err := client.Chat(ctx, ChatRequest{Model: "gpt-4", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	if !strings.Contains(err.Error(), "authentication_error") {
		t.Errorf("expected error to contain 'authentication_error', got %q", err.Error())
	}
}

func TestOpenAIClientChatHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(server.URL+"/v1", "key")
	ctx := context.Background()

	_, err := client.Chat(ctx, ChatRequest{Model: "gpt-4", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain '500', got %q", err.Error())
	}
}

func TestOpenAIClientChatWithSystemAndTemperature(t *testing.T) {
	var capturedReq oaiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedReq)
		json.NewEncoder(w).Encode(oaiResponse{
			Choices: []oaiChoice{{Message: oaiMessage{Role: "assistant", Content: "ok"}, FinishReason: "stop"}},
		})
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(server.URL+"/v1", "key")
	temp := 0.7

	_, err := client.Chat(context.Background(), ChatRequest{
		Model:       "gpt-4",
		System:      "You are a helpful assistant",
		Messages:    []Message{{Role: RoleUser, Content: "hi"}},
		Temperature: &temp,
		MaxTokens:   200,
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	// Verify system message was added
	if len(capturedReq.Messages) < 2 {
		t.Fatalf("expected at least 2 messages (system + user), got %d", len(capturedReq.Messages))
	}
	if capturedReq.Messages[0].Role != "system" {
		t.Errorf("expected first message role='system', got %q", capturedReq.Messages[0].Role)
	}
	if capturedReq.Messages[0].Content != "You are a helpful assistant" {
		t.Errorf("expected system content, got %q", capturedReq.Messages[0].Content)
	}
	if capturedReq.Temperature == nil || *capturedReq.Temperature != 0.7 {
		t.Errorf("expected temperature=0.7, got %v", capturedReq.Temperature)
	}
	if capturedReq.MaxTokens != 200 {
		t.Errorf("expected MaxTokens=200, got %d", capturedReq.MaxTokens)
	}
}

func TestOpenAIClientChatWithToolResultMessage(t *testing.T) {
	var capturedReq oaiRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedReq)
		json.NewEncoder(w).Encode(oaiResponse{
			Choices: []oaiChoice{{Message: oaiMessage{Role: "assistant", Content: "result"}, FinishReason: "stop"}},
		})
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(server.URL+"/v1", "key")

	_, err := client.Chat(context.Background(), ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: RoleUser, Content: "use tool"},
			{Role: RoleAssistant, Content: "", ToolCalls: []ToolCall{{ID: "tc-1", Name: "calc", Input: map[string]interface{}{"x": float64(1)}}}},
			{Role: RoleUser, ToolResult: &ToolResult{ToolUseID: "tc-1", Content: "42", IsError: false}},
		},
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	// Verify tool result message was converted to "tool" role
	if len(capturedReq.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(capturedReq.Messages))
	}
	// Third message should be role "tool"
	if capturedReq.Messages[2].Role != "tool" {
		t.Errorf("expected tool role for tool result, got %q", capturedReq.Messages[2].Role)
	}
	if capturedReq.Messages[2].ToolCallID != "tc-1" {
		t.Errorf("expected ToolCallID='tc-1', got %q", capturedReq.Messages[2].ToolCallID)
	}

	// Second message should have tool calls
	if len(capturedReq.Messages[1].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call in assistant message, got %d", len(capturedReq.Messages[1].ToolCalls))
	}
}

func TestOpenAIClientChatNoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(oaiResponse{
			Choices: []oaiChoice{},
			Usage:   oaiUsage{PromptTokens: 5, CompletionTokens: 0},
		})
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(server.URL+"/v1", "key")
	resp, err := client.Chat(context.Background(), ChatRequest{Model: "gpt-4", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if resp.Content != "" {
		t.Errorf("expected empty content for no choices, got %q", resp.Content)
	}
}

func TestOpenAIClientChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req oaiRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if !req.Stream {
			t.Error("expected stream=true for ChatStream")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		// Send text chunks
		chunks := []string{
			`{"choices":[{"delta":{"content":"Hello "}}]}`,
			`{"choices":[{"delta":{"content":"world!"}}]}`,
			`{"choices":[{"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`,
		}
		for _, chunk := range chunks {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(server.URL+"/v1", "key")
	ch, err := client.ChatStream(context.Background(), ChatRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatStream error: %v", err)
	}

	var events []StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}

	// Should have text events and a done event
	textCount := 0
	var doneEvent *StreamEvent
	for i := range events {
		if events[i].Type == "text" {
			textCount++
		}
		if events[i].Type == "done" {
			doneEvent = &events[i]
		}
	}

	if textCount != 2 {
		t.Errorf("expected 2 text events, got %d", textCount)
	}
	if doneEvent == nil {
		t.Fatal("expected a done event")
	}
	if doneEvent.Response == nil {
		t.Fatal("expected done event to have a Response")
	}
	if doneEvent.Response.Content != "Hello world!" {
		t.Errorf("expected accumulated content 'Hello world!', got %q", doneEvent.Response.Content)
	}
}

// --- OpenAI Client Constructor Tests ---

func TestNewOpenAIClient(t *testing.T) {
	client := NewOpenAIClient("sk-test")
	if client == nil {
		t.Fatal("expected non-nil OpenAIClient")
	}
	if client.baseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default base URL, got %q", client.baseURL)
	}
	if client.apiKey != "sk-test" {
		t.Errorf("expected apiKey='sk-test', got %q", client.apiKey)
	}
}

func TestNewOllamaClient(t *testing.T) {
	t.Run("default host", func(t *testing.T) {
		client := NewOllamaClient("")
		if client.baseURL != "http://localhost:11434/v1" {
			t.Errorf("expected default ollama URL, got %q", client.baseURL)
		}
	})

	t.Run("custom host", func(t *testing.T) {
		client := NewOllamaClient("http://myhost:1234")
		if client.baseURL != "http://myhost:1234/v1" {
			t.Errorf("expected 'http://myhost:1234/v1', got %q", client.baseURL)
		}
	})

	t.Run("trailing slash stripped", func(t *testing.T) {
		client := NewOllamaClient("http://myhost:1234/")
		if client.baseURL != "http://myhost:1234/v1" {
			t.Errorf("expected trailing slash removed, got %q", client.baseURL)
		}
	})
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	client := NewOpenAIClient("key", WithHTTPClient(custom))
	if client.httpClient != custom {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestNewOpenAICompatibleClient(t *testing.T) {
	client := NewOpenAICompatibleClient("http://custom-endpoint/api", "custom-key")
	if client.baseURL != "http://custom-endpoint/api" {
		t.Errorf("expected base URL without trailing slash, got %q", client.baseURL)
	}
	if client.apiKey != "custom-key" {
		t.Errorf("expected apiKey='custom-key', got %q", client.apiKey)
	}
}

// --- mapOAIStopReason Tests ---

func TestMapOAIStopReason(t *testing.T) {
	tests := []struct {
		input string
		want  StopReason
	}{
		{"stop", StopEndTurn},
		{"length", StopMaxTokens},
		{"tool_calls", StopToolUse},
		{"unknown", StopEndTurn},
		{"", StopEndTurn},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapOAIStopReason(tt.input)
			if got != tt.want {
				t.Errorf("mapOAIStopReason(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- NewClientForModel Tests ---

func TestNewClientForModel(t *testing.T) {
	t.Run("ollama prefix", func(t *testing.T) {
		t.Setenv("OLLAMA_HOST", "")
		client, modelName := NewClientForModel("ollama/llama3")
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		if modelName != "llama3" {
			t.Errorf("expected model='llama3', got %q", modelName)
		}
		if _, ok := client.(*OpenAIClient); !ok {
			t.Error("expected *OpenAIClient for ollama provider")
		}
	})

	t.Run("openai prefix", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "test-key")
		t.Setenv("OPENAI_BASE_URL", "")
		client, modelName := NewClientForModel("openai/gpt-4")
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		if modelName != "gpt-4" {
			t.Errorf("expected model='gpt-4', got %q", modelName)
		}
	})

	t.Run("openai with custom base URL", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "key")
		t.Setenv("OPENAI_BASE_URL", "http://custom/v1")
		client, _ := NewClientForModel("openai/gpt-4")
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		oaiClient, ok := client.(*OpenAIClient)
		if !ok {
			t.Fatal("expected *OpenAIClient")
		}
		if oaiClient.baseURL != "http://custom/v1" {
			t.Errorf("expected custom base URL, got %q", oaiClient.baseURL)
		}
	})

	t.Run("anthropic default", func(t *testing.T) {
		t.Setenv("OLLAMA_HOST", "")
		t.Setenv("OPENAI_API_KEY", "")
		t.Setenv("OPENAI_BASE_URL", "")
		client, modelName := NewClientForModel("anthropic/claude-3")
		if client == nil {
			t.Fatal("expected non-nil client")
		}
		if modelName != "claude-3" {
			t.Errorf("expected model='claude-3', got %q", modelName)
		}
		if _, ok := client.(*AnthropicClient); !ok {
			t.Error("expected *AnthropicClient for anthropic provider")
		}
	})
}

// --- MockClient with ToolCalls in stream ---

func TestMockClientChatStreamWithToolCalls(t *testing.T) {
	mock := NewMockClient(MockResponse{
		Content: "",
		ToolCalls: []ToolCall{
			{ID: "tc-1", Name: "search", Input: map[string]interface{}{"q": "test"}},
		},
		StopReason: StopToolUse,
	})

	ch, err := mock.ChatStream(context.Background(), ChatRequest{Model: "test"})
	if err != nil {
		t.Fatalf("ChatStream error: %v", err)
	}

	var events []StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}

	// Should have tool_call_start + done events (no text since content is empty)
	toolStartCount := 0
	for _, ev := range events {
		if ev.Type == "tool_call_start" {
			toolStartCount++
			if ev.ToolCall == nil {
				t.Error("expected ToolCall in tool_call_start event")
			} else if ev.ToolCall.Name != "search" {
				t.Errorf("expected tool name 'search', got %q", ev.ToolCall.Name)
			}
		}
	}
	if toolStartCount != 1 {
		t.Errorf("expected 1 tool_call_start event, got %d", toolStartCount)
	}
}

func TestMockClientChatStreamError(t *testing.T) {
	mock := NewMockClient(MockResponse{Error: fmt.Errorf("stream error")})

	_, err := mock.ChatStream(context.Background(), ChatRequest{Model: "test"})
	if err == nil {
		t.Fatal("expected error from ChatStream, got nil")
	}
	if err.Error() != "stream error" {
		t.Errorf("expected 'stream error', got %q", err.Error())
	}
}

// --- OpenAI Client with no API key (Ollama use case) ---

func TestOpenAIClientNoAuthHeader(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(oaiResponse{
			Choices: []oaiChoice{{Message: oaiMessage{Role: "assistant", Content: "ok"}, FinishReason: "stop"}},
		})
	}))
	defer server.Close()

	// No API key -- should not send Authorization header
	client := NewOpenAICompatibleClient(server.URL+"/v1", "")
	_, err := client.Chat(context.Background(), ChatRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if authHeader != "" {
		t.Errorf("expected no Authorization header, got %q", authHeader)
	}
}

// --- OpenAI Client with bad tool arguments in response ---

func TestOpenAIClientChatBadToolArguments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := oaiResponse{
			Choices: []oaiChoice{
				{
					Message: oaiMessage{
						Role: "assistant",
						ToolCalls: []oaiToolCall{
							{
								ID:   "tc-1",
								Type: "function",
								Function: oaiToolCallFunc{
									Name:      "bad_tool",
									Arguments: "not valid json",
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(server.URL+"/v1", "key")
	resp, err := client.Chat(context.Background(), ChatRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	// Should still return tool call, but with _error in input
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if _, hasError := resp.ToolCalls[0].Input["_error"]; !hasError {
		t.Error("expected _error key in tool call input for bad arguments")
	}
}
