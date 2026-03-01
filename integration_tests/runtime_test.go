package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/loop"
	"github.com/szaher/designs/agentz/internal/memory"
	"github.com/szaher/designs/agentz/internal/runtime"
	"github.com/szaher/designs/agentz/internal/session"
	"github.com/szaher/designs/agentz/internal/tools"
)

func newTestServer(t *testing.T, mockClient *llm.MockClient) *httptest.Server {
	t.Helper()

	config := &runtime.RuntimeConfig{
		PackageName: "test-pkg",
		Agents: []runtime.AgentConfig{
			{
				Name:     "helper",
				FQN:      "test-pkg/helper",
				Model:    "claude-sonnet-4-20250514",
				System:   "You are a helpful assistant.",
				Strategy: "react",
				MaxTurns: 5,
				Stream:   true,
			},
		},
		Prompts: map[string]string{
			"system": "You are a helpful assistant.",
		},
	}

	registry := tools.NewRegistry()
	sessionStore := session.NewMemoryStore(30 * time.Minute)
	memoryStore := memory.NewSlidingWindow(50)
	sessionMgr := session.NewManager(sessionStore, memoryStore)
	strategy := &loop.ReActStrategy{}

	server := runtime.NewServer(config, mockClient, registry, sessionMgr, strategy, runtime.WithNoAuth(true))

	return httptest.NewServer(server.Handler())
}

func TestRuntimeHealthEndpoint(t *testing.T) {
	mock := llm.NewMockClient(llm.MockResponse{
		Content:    "Hello!",
		StopReason: llm.StopEndTurn,
	})
	ts := newTestServer(t, mock)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", body["status"])
	}
}

func TestRuntimeListAgents(t *testing.T) {
	mock := llm.NewMockClient(llm.MockResponse{
		Content:    "Hello!",
		StopReason: llm.StopEndTurn,
	})
	ts := newTestServer(t, mock)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/v1/agents")
	if err != nil {
		t.Fatalf("list agents failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	agents, ok := body["agents"].([]interface{})
	if !ok || len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %v", body["agents"])
	}

	agent, _ := agents[0].(map[string]interface{})
	if agent["name"] != "helper" {
		t.Errorf("expected agent name 'helper', got %v", agent["name"])
	}
}

func TestRuntimeInvokeAgent(t *testing.T) {
	mock := llm.NewMockClient(llm.MockResponse{
		Content:    "I can help with that!",
		StopReason: llm.StopEndTurn,
		Usage: llm.TokenUsage{
			InputTokens:  10,
			OutputTokens: 5,
		},
	})
	ts := newTestServer(t, mock)
	defer ts.Close()

	reqBody, _ := json.Marshal(map[string]string{
		"message": "Help me with Go",
	})
	resp, err := http.Post(ts.URL+"/v1/agents/helper/invoke", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["output"] != "I can help with that!" {
		t.Errorf("expected 'I can help with that!', got %v", body["output"])
	}

	tokens, _ := body["tokens"].(map[string]interface{})
	total, _ := tokens["total"].(float64)
	if total != 15 {
		t.Errorf("expected total tokens 15, got %v", tokens["total"])
	}
}

func TestRuntimeInvokeAgentNotFound(t *testing.T) {
	mock := llm.NewMockClient()
	ts := newTestServer(t, mock)
	defer ts.Close()

	reqBody, _ := json.Marshal(map[string]string{
		"message": "hello",
	})
	resp, err := http.Post(ts.URL+"/v1/agents/nonexistent/invoke", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRuntimeSessionLifecycle(t *testing.T) {
	mock := llm.NewMockClient(
		llm.MockResponse{Content: "First response", StopReason: llm.StopEndTurn},
		llm.MockResponse{Content: "Second response", StopReason: llm.StopEndTurn},
	)
	ts := newTestServer(t, mock)
	defer ts.Close()

	// Create session
	resp, err := http.Post(ts.URL+"/v1/agents/helper/sessions", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var sessBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&sessBody); err != nil {
		t.Fatalf("decode session response: %v", err)
	}
	sessionID, _ := sessBody["session_id"].(string)

	if sessionID == "" {
		t.Fatal("expected non-empty session_id")
	}

	// Send message to session
	msgBody, _ := json.Marshal(map[string]string{"message": "Hello"})
	resp2, err := http.Post(
		fmt.Sprintf("%s/v1/agents/helper/sessions/%s", ts.URL, sessionID),
		"application/json",
		bytes.NewReader(msgBody),
	)
	if err != nil {
		t.Fatalf("session message: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}

	var msgResp map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&msgResp); err != nil {
		t.Fatalf("decode message response: %v", err)
	}
	if msgResp["output"] != "First response" {
		t.Errorf("expected 'First response', got %v", msgResp["output"])
	}
	if msgResp["session_id"] != sessionID {
		t.Errorf("expected session_id %q, got %v", sessionID, msgResp["session_id"])
	}

	// Delete session
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/agents/helper/sessions/%s", ts.URL, sessionID), nil)
	resp3, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete session: %v", err)
	}
	defer func() { _ = resp3.Body.Close() }()

	if resp3.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp3.StatusCode)
	}
}

func TestRuntimeAPIKeyAuth(t *testing.T) {
	config := &runtime.RuntimeConfig{
		PackageName: "test-pkg",
		Agents: []runtime.AgentConfig{
			{Name: "helper", Model: "claude-sonnet-4-20250514", Strategy: "react", MaxTurns: 5},
		},
		Prompts: map[string]string{},
	}

	mock := llm.NewMockClient(llm.MockResponse{Content: "ok", StopReason: llm.StopEndTurn})
	registry := tools.NewRegistry()
	sessionStore := session.NewMemoryStore(30 * time.Minute)
	memoryStore := memory.NewSlidingWindow(50)
	sessionMgr := session.NewManager(sessionStore, memoryStore)
	strategy := &loop.ReActStrategy{}

	server := runtime.NewServer(config, mock, registry, sessionMgr, strategy, runtime.WithAPIKey("test-key"))
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	// Health check should NOT require auth
	resp, _ := http.Get(ts.URL + "/healthz")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health check should not require auth, got %d", resp.StatusCode)
	}

	// List agents without key should fail
	resp, _ = http.Get(ts.URL + "/v1/agents")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without API key, got %d", resp.StatusCode)
	}

	// List agents with key should succeed
	req, _ := http.NewRequest("GET", ts.URL+"/v1/agents", nil)
	req.Header.Set("X-API-Key", "test-key")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with API key, got %d", resp.StatusCode)
	}

	// List agents with Bearer token should also work
	req2, _ := http.NewRequest("GET", ts.URL+"/v1/agents", nil)
	req2.Header.Set("Authorization", "Bearer test-key")
	resp2, _ := http.DefaultClient.Do(req2)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with Bearer token, got %d", resp2.StatusCode)
	}
}
