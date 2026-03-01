package integration_tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/szaher/designs/agentz/internal/compiler"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/loop"
	"github.com/szaher/designs/agentz/internal/memory"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/runtime"
	"github.com/szaher/designs/agentz/internal/session"
	"github.com/szaher/designs/agentz/internal/telemetry"
	"github.com/szaher/designs/agentz/internal/tools"
	"github.com/szaher/designs/agentz/internal/validate"
)

func TestCompilePipeline(t *testing.T) {
	// Test the full pipeline: parse → validate → lower → config conversion
	iasContent := `package "compile-test" version "1.0.0" lang "3.0"

prompt "sys" {
  content "You are a test agent."
}

agent "test-agent" {
  model "claude-sonnet-4-20250514"
  prompt "sys"
  strategy "react"
  max_turns 5

  config {
    api_key string required secret "API key"
    mode string default "test" "Operating mode"
  }

  validate {
    rule not_empty error "Output must not be empty" when output != ""
  }

  eval {
    case basic_test input "hello" expect "Hi there!" scoring contains threshold 0.5
  }
}
`

	// Phase 1: Parse
	f, parseErrs := parser.Parse(iasContent, "test.ias")
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	// Phase 2: Validate
	valErrs := validate.ValidateStructural(f)
	if len(valErrs) > 0 {
		t.Fatalf("validation errors: %v", valErrs)
	}

	// Phase 3: Lower to IR
	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("lowering failed: %v", err)
	}

	// Verify IR contains expected resources
	var agentRes *ir.Resource
	for i, r := range doc.Resources {
		if r.Kind == "Agent" && r.Name == "test-agent" {
			agentRes = &doc.Resources[i]
			break
		}
	}
	if agentRes == nil {
		t.Fatal("agent resource not found in IR")
	}

	// Verify config params in IR
	configParams, ok := agentRes.Attributes["config_params"].([]interface{})
	if !ok {
		t.Fatal("config_params not found in agent attributes")
	}
	if len(configParams) != 2 {
		t.Errorf("expected 2 config params, got %d", len(configParams))
	}

	// Verify validation rules in IR
	valRules, ok := agentRes.Attributes["validation_rules"].([]interface{})
	if !ok {
		t.Fatal("validation_rules not found in agent attributes")
	}
	if len(valRules) != 1 {
		t.Errorf("expected 1 validation rule, got %d", len(valRules))
	}

	// Verify eval cases in IR
	evalCases, ok := agentRes.Attributes["eval_cases"].([]interface{})
	if !ok {
		t.Fatal("eval_cases not found in agent attributes")
	}
	if len(evalCases) != 1 {
		t.Errorf("expected 1 eval case, got %d", len(evalCases))
	}

	// Phase 4: Convert to RuntimeConfig
	config, err := runtime.FromIR(doc)
	if err != nil {
		t.Fatalf("config conversion failed: %v", err)
	}

	if len(config.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(config.Agents))
	}

	agent := config.Agents[0]
	if agent.Name != "test-agent" {
		t.Errorf("expected agent name 'test-agent', got %q", agent.Name)
	}
	if agent.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected model 'claude-sonnet-4-20250514', got %q", agent.Model)
	}
	if len(agent.ConfigParams) != 2 {
		t.Errorf("expected 2 config params, got %d", len(agent.ConfigParams))
	}
	if len(agent.ValidationRules) != 1 {
		t.Errorf("expected 1 validation rule, got %d", len(agent.ValidationRules))
	}
	if len(agent.EvalCases) != 1 {
		t.Errorf("expected 1 eval case, got %d", len(agent.EvalCases))
	}

	// Verify prompt was resolved
	if agent.System == "" {
		t.Error("expected system prompt to be resolved from prompt reference")
	}
}

func TestConfigResolver(t *testing.T) {
	params := []runtime.ConfigParamDef{
		{Name: "api_key", Type: "string", Required: true, Secret: true},
		{Name: "mode", Type: "string", Required: false, HasDefault: true, Default: "test"},
		{Name: "port", Type: "int", Required: false, HasDefault: true, Default: "8080"},
	}

	// Test missing required param
	resolver, err := runtime.NewConfigResolver("")
	if err != nil {
		t.Fatalf("creating resolver: %v", err)
	}

	_, err = resolver.Resolve("test-agent", params)
	if err == nil {
		t.Fatal("expected error for missing required param")
	}
	if !strings.Contains(err.Error(), "api_key") {
		t.Errorf("error should mention missing param name, got: %v", err)
	}

	// Test with env var set
	t.Setenv("AGENTSPEC_TEST_AGENT_API_KEY", "sk-test-123")

	resolved, err := resolver.Resolve("test-agent", params)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if resolved.Values["api_key"] != "sk-test-123" {
		t.Errorf("expected api_key 'sk-test-123', got %v", resolved.Values["api_key"])
	}
	if resolved.Values["mode"] != "test" {
		t.Errorf("expected mode 'test', got %v", resolved.Values["mode"])
	}
	if resolved.Values["port"] != 8080 {
		t.Errorf("expected port 8080, got %v", resolved.Values["port"])
	}
}

func TestConfigResolverFromFile(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configData := map[string]interface{}{
		"test-agent": map[string]interface{}{
			"api_key": "file-key-123",
		},
	}
	data, _ := json.Marshal(configData)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	params := []runtime.ConfigParamDef{
		{Name: "api_key", Type: "string", Required: true},
	}

	resolver, err := runtime.NewConfigResolver(configPath)
	if err != nil {
		t.Fatalf("creating resolver: %v", err)
	}

	resolved, err := resolver.Resolve("test-agent", params)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if resolved.Values["api_key"] != "file-key-123" {
		t.Errorf("expected api_key 'file-key-123', got %v", resolved.Values["api_key"])
	}
}

func TestConfigRefGeneration(t *testing.T) {
	refs := []compiler.AgentConfigRef{
		{
			AgentName: "test-agent",
			Params: []runtime.ConfigParamDef{
				{Name: "api_key", Type: "string", Required: true, Secret: true, Description: "API key"},
				{Name: "mode", Type: "string", HasDefault: true, Default: "test", Description: "Mode"},
			},
		},
	}

	output := compiler.GenerateConfigRef(refs, "test-binary")

	if output == "" {
		t.Fatal("empty config ref output")
	}

	if !strings.Contains(output, "test-agent") {
		t.Error("config ref should contain agent name")
	}
	if !strings.Contains(output, "AGENTSPEC_TEST_AGENT_API_KEY") {
		t.Error("config ref should contain env var name")
	}
	if !strings.Contains(output, "**Yes**") {
		t.Error("config ref should mark required params")
	}
}

func TestCompiledAgentHealthz(t *testing.T) {
	// Test that a RuntimeConfig-based server serves /healthz correctly
	config := &runtime.RuntimeConfig{
		PackageName: "test-pkg",
		Agents: []runtime.AgentConfig{
			{
				Name:     "test-agent",
				FQN:      "test-pkg/Agent/test-agent",
				Model:    "claude-sonnet-4-20250514",
				Strategy: "react",
				MaxTurns: 5,
				Stream:   true,
			},
		},
		Prompts: map[string]string{},
	}

	mockClient := llm.NewMockClient(llm.MockResponse{
		Content:    "Hello!",
		StopReason: llm.StopEndTurn,
	})

	registry := tools.NewRegistry()
	sessionStore := session.NewMemoryStore(30 * time.Minute)
	memoryStore := memory.NewSlidingWindow(50)
	sessionMgr := session.NewManager(sessionStore, memoryStore)
	strategy := &loop.ReActStrategy{}

	srv := runtime.NewServer(config, mockClient, registry, sessionMgr, strategy, runtime.WithNoAuth(true))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, err := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/healthz", nil)
	if err != nil {
		t.Fatalf("creating healthz request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("healthz request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if body["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", body["status"])
	}
}

// TestCompileDeterminism verifies that compiling the same .ias file twice
// produces identical output (T088).
func TestCompileDeterminism(t *testing.T) {
	iasContent := `package "determinism-test" version "1.0.0" lang "3.0"

prompt "sys" {
  content "You are a deterministic agent."
}

agent "det-agent" {
  model "claude-sonnet-4-20250514"
  prompt "sys"
  strategy "react"
  max_turns 3
}
`
	// Parse, lower, and compile twice
	compile := func() *runtime.RuntimeConfig {
		f, errs := parser.Parse(iasContent, "determinism.ias")
		if errs != nil {
			t.Fatalf("parse error: %v", errs)
		}
		doc, err := ir.Lower(f)
		if err != nil {
			t.Fatalf("lower error: %v", err)
		}
		config, err := runtime.FromIR(doc)
		if err != nil {
			t.Fatalf("config error: %v", err)
		}
		return config
	}

	config1 := compile()
	config2 := compile()

	// Verify identical agent configs
	if len(config1.Agents) != len(config2.Agents) {
		t.Fatalf("agent count mismatch: %d vs %d", len(config1.Agents), len(config2.Agents))
	}

	a1 := config1.Agents[0]
	a2 := config2.Agents[0]

	if a1.Name != a2.Name {
		t.Errorf("name mismatch: %q vs %q", a1.Name, a2.Name)
	}
	if a1.Model != a2.Model {
		t.Errorf("model mismatch: %q vs %q", a1.Model, a2.Model)
	}
	if a1.System != a2.System {
		t.Errorf("system prompt mismatch")
	}
	if a1.Strategy != a2.Strategy {
		t.Errorf("strategy mismatch: %q vs %q", a1.Strategy, a2.Strategy)
	}
	if a1.MaxTurns != a2.MaxTurns {
		t.Errorf("max_turns mismatch: %d vs %d", a1.MaxTurns, a2.MaxTurns)
	}

	// Verify JSON serialization is identical
	json1, _ := json.Marshal(config1)
	json2, _ := json.Marshal(config2)
	if string(json1) != string(json2) {
		t.Error("JSON serialization of configs not identical")
	}
}

// TestProcessAdapterCompiledAgent verifies that a compiled agent config
// can create a functional server without external dependencies (T093).
func TestProcessAdapterCompiledAgent(t *testing.T) {
	config := &runtime.RuntimeConfig{
		PackageName: "process-test",
		Agents: []runtime.AgentConfig{
			{
				Name:     "local-agent",
				FQN:      "process-test/Agent/local-agent",
				Model:    "test-model",
				Strategy: "react",
				MaxTurns: 3,
				System:   "You are a test agent running as a local process.",
			},
		},
	}

	registry := tools.NewRegistry()
	sessionMgr := session.NewManager(session.NewMemoryStore(0), nil)
	strategy := &loop.ReActStrategy{}
	metrics := telemetry.NewMetrics()

	// Create server with UI enabled (simulating compiled agent)
	srv := runtime.NewServer(config, nil, registry, sessionMgr, strategy,
		runtime.WithUI(true),
		runtime.WithMetrics(metrics),
		runtime.WithNoAuth(true),
	)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Verify healthz
	healthReq, err := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/healthz", nil)
	if err != nil {
		t.Fatalf("creating healthz request: %v", err)
	}
	resp, err := http.DefaultClient.Do(healthReq)
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz: expected 200, got %d", resp.StatusCode)
	}

	// Verify agents API
	agentsReq, err := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/v1/agents", nil)
	if err != nil {
		t.Fatalf("creating agents request: %v", err)
	}
	resp, err = http.DefaultClient.Do(agentsReq)
	if err != nil {
		t.Fatalf("agents: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("agents: expected 200, got %d", resp.StatusCode)
	}

	var agentResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&agentResp); err != nil {
		t.Fatalf("decoding agents response: %v", err)
	}
	agents, _ := agentResp["agents"].([]interface{})
	if len(agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(agents))
	}

	// Verify frontend is served
	frontendReq, err := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/", nil)
	if err != nil {
		t.Fatalf("creating frontend request: %v", err)
	}
	resp, err = http.DefaultClient.Do(frontendReq)
	if err != nil {
		t.Fatalf("frontend: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("frontend: expected 200, got %d", resp.StatusCode)
	}

	// Verify metrics endpoint
	metricsReq, err := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/v1/metrics", nil)
	if err != nil {
		t.Fatalf("creating metrics request: %v", err)
	}
	resp, err = http.DefaultClient.Do(metricsReq)
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("metrics: expected 200, got %d", resp.StatusCode)
	}
}
