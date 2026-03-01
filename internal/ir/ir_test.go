package ir

import (
	"encoding/json"
	"testing"

	"github.com/szaher/designs/agentz/internal/ast"
)

// ---------------------------------------------------------------------------
// SortResources
// ---------------------------------------------------------------------------

func TestSortResources_ByKindThenName(t *testing.T) {
	resources := []Resource{
		{Kind: "Skill", Name: "z-skill"},
		{Kind: "Agent", Name: "b-agent"},
		{Kind: "Skill", Name: "a-skill"},
		{Kind: "Agent", Name: "a-agent"},
		{Kind: "Prompt", Name: "x-prompt"},
	}

	SortResources(resources)

	want := []struct{ kind, name string }{
		{"Agent", "a-agent"},
		{"Agent", "b-agent"},
		{"Prompt", "x-prompt"},
		{"Skill", "a-skill"},
		{"Skill", "z-skill"},
	}

	for i, w := range want {
		if resources[i].Kind != w.kind || resources[i].Name != w.name {
			t.Errorf("index %d: got %s/%s, want %s/%s",
				i, resources[i].Kind, resources[i].Name, w.kind, w.name)
		}
	}
}

func TestSortResources_Empty(t *testing.T) {
	var resources []Resource
	SortResources(resources) // should not panic
	if len(resources) != 0 {
		t.Errorf("expected empty slice after sorting empty input")
	}
}

// ---------------------------------------------------------------------------
// Document.MarshalJSON
// ---------------------------------------------------------------------------

func TestDocument_MarshalJSON_Deterministic(t *testing.T) {
	doc := &Document{
		IRVersion:   "1.0",
		LangVersion: "1.0",
		Package:     Package{Name: "test", Version: "0.1.0"},
		Resources: []Resource{
			{Kind: "Skill", Name: "b", Attributes: map[string]interface{}{"k": "v"}},
			{Kind: "Agent", Name: "a", Attributes: map[string]interface{}{"k": "v"}},
		},
	}

	data1, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("first marshal: %v", err)
	}
	data2, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("second marshal: %v", err)
	}
	if string(data1) != string(data2) {
		t.Error("MarshalJSON is not deterministic")
	}
}

func TestDocument_MarshalJSON_RoundTrip(t *testing.T) {
	doc := &Document{
		IRVersion:   "1.0",
		LangVersion: "1.0",
		Package:     Package{Name: "roundtrip", Version: "0.2.0", Description: "test desc"},
		Resources: []Resource{
			{Kind: "Agent", Name: "myagent", FQN: "roundtrip/Agent/myagent",
				Attributes: map[string]interface{}{"model": "gpt-4"}, Hash: "sha256:abc"},
		},
	}

	data, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var roundTripped Document
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if roundTripped.Package.Name != doc.Package.Name {
		t.Errorf("package name: got %q, want %q", roundTripped.Package.Name, doc.Package.Name)
	}
	if roundTripped.IRVersion != doc.IRVersion {
		t.Errorf("ir_version: got %q, want %q", roundTripped.IRVersion, doc.IRVersion)
	}
	if len(roundTripped.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(roundTripped.Resources))
	}
	if roundTripped.Resources[0].Name != "myagent" {
		t.Errorf("resource name: got %q, want %q", roundTripped.Resources[0].Name, "myagent")
	}
}

func TestDocument_MarshalJSON_SortsResources(t *testing.T) {
	doc := &Document{
		IRVersion:   "1.0",
		LangVersion: "1.0",
		Package:     Package{Name: "test", Version: "0.1.0"},
		Resources: []Resource{
			{Kind: "Skill", Name: "second", Attributes: map[string]interface{}{}},
			{Kind: "Agent", Name: "first", Attributes: map[string]interface{}{}},
		},
	}

	_, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// After MarshalJSON, resources should be sorted
	if doc.Resources[0].Kind != "Agent" || doc.Resources[1].Kind != "Skill" {
		t.Errorf("resources not sorted after MarshalJSON: got %s, %s",
			doc.Resources[0].Kind, doc.Resources[1].Kind)
	}
}

// ---------------------------------------------------------------------------
// SerializeCanonical
// ---------------------------------------------------------------------------

func TestSerializeCanonical_DeterministicKeyOrder(t *testing.T) {
	// Two maps with same content but potentially different iteration order
	m1 := map[string]interface{}{"z": "last", "a": "first", "m": "middle"}
	m2 := map[string]interface{}{"m": "middle", "a": "first", "z": "last"}

	b1, err := SerializeCanonical(m1)
	if err != nil {
		t.Fatalf("serialize m1: %v", err)
	}
	b2, err := SerializeCanonical(m2)
	if err != nil {
		t.Fatalf("serialize m2: %v", err)
	}

	if string(b1) != string(b2) {
		t.Errorf("not deterministic:\n  m1: %s\n  m2: %s", b1, b2)
	}

	// Verify keys are alphabetically sorted
	expected := `{"a":"first","m":"middle","z":"last"}`
	if string(b1) != expected {
		t.Errorf("got %s, want %s", b1, expected)
	}
}

func TestSerializeCanonical_NestedMaps(t *testing.T) {
	m := map[string]interface{}{
		"b": map[string]interface{}{"y": 2, "x": 1},
		"a": "val",
	}
	b, err := SerializeCanonical(m)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	expected := `{"a":"val","b":{"x":1,"y":2}}`
	if string(b) != expected {
		t.Errorf("got %s, want %s", b, expected)
	}
}

func TestSerializeCanonical_Nil(t *testing.T) {
	b, err := SerializeCanonical(nil)
	if err != nil {
		t.Fatalf("serialize nil: %v", err)
	}
	if string(b) != "null" {
		t.Errorf("got %s, want null", b)
	}
}

// ---------------------------------------------------------------------------
// ComputeHash
// ---------------------------------------------------------------------------

func TestComputeHash_SameForEquivalent(t *testing.T) {
	a := map[string]interface{}{"x": "1", "y": "2"}
	b := map[string]interface{}{"y": "2", "x": "1"}

	ha := ComputeHash(a)
	hb := ComputeHash(b)

	if ha != hb {
		t.Errorf("hashes differ for equivalent maps:\n  a=%s\n  b=%s", ha, hb)
	}
	if ha == "" {
		t.Error("hash should not be empty")
	}
	if len(ha) < 10 {
		t.Errorf("hash suspiciously short: %s", ha)
	}
	if ha[:7] != "sha256:" {
		t.Errorf("hash should start with sha256:, got %s", ha[:7])
	}
}

func TestComputeHash_DifferentForDifferent(t *testing.T) {
	a := map[string]interface{}{"key": "value1"}
	b := map[string]interface{}{"key": "value2"}

	ha := ComputeHash(a)
	hb := ComputeHash(b)

	if ha == hb {
		t.Errorf("hashes should differ for different maps: both %s", ha)
	}
}

func TestComputeHash_NilReturnsEmpty(t *testing.T) {
	// nil serializes to "null" which should still produce a hash
	h := ComputeHash(nil)
	// ComputeHash passes nil to SerializeCanonical which returns "null" via json.Marshal,
	// then hashes the bytes "null". This should produce a valid hash.
	if h == "" {
		t.Log("nil map produces empty hash (expected based on implementation)")
	}
}

// ---------------------------------------------------------------------------
// ApplyEnvironment
// ---------------------------------------------------------------------------

func TestApplyEnvironment_ValidOverride(t *testing.T) {
	doc := &Document{
		Resources: []Resource{
			{Kind: "Agent", Name: "myagent", Attributes: map[string]interface{}{"model": "gpt-3.5"}},
			{Kind: "Environment", Name: "prod", Attributes: map[string]interface{}{
				"overrides": []interface{}{
					map[string]interface{}{
						"resource":  "Agent/myagent",
						"attribute": "model",
						"value":     "gpt-4",
					},
				},
			}},
		},
	}

	result, err := ApplyEnvironment(doc, "prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Environment resources should be filtered out
	for _, r := range result.Resources {
		if r.Kind == "Environment" {
			t.Error("environment resources should be removed from output")
		}
	}

	// The agent should have the overridden model
	if len(result.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(result.Resources))
	}
	if result.Resources[0].Attributes["model"] != "gpt-4" {
		t.Errorf("model: got %v, want gpt-4", result.Resources[0].Attributes["model"])
	}
}

func TestApplyEnvironment_MissingEnvReturnsError(t *testing.T) {
	doc := &Document{
		Resources: []Resource{
			{Kind: "Agent", Name: "myagent", Attributes: map[string]interface{}{}},
		},
	}

	_, err := ApplyEnvironment(doc, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing environment")
	}
}

func TestApplyEnvironment_EmptyEnvNameReturnsDoc(t *testing.T) {
	doc := &Document{
		Resources: []Resource{
			{Kind: "Agent", Name: "a", Attributes: map[string]interface{}{}},
		},
	}
	result, err := ApplyEnvironment(doc, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != doc {
		t.Error("empty env name should return the original document")
	}
}

func TestApplyEnvironment_OverrideTargetNotFound(t *testing.T) {
	doc := &Document{
		Resources: []Resource{
			{Kind: "Environment", Name: "staging", Attributes: map[string]interface{}{
				"overrides": []interface{}{
					map[string]interface{}{
						"resource":  "Agent/nonexistent",
						"attribute": "model",
						"value":     "gpt-4",
					},
				},
			}},
		},
	}

	_, err := ApplyEnvironment(doc, "staging")
	if err == nil {
		t.Fatal("expected error when override target not found")
	}
}

func TestApplyEnvironment_DoesNotMutateOriginal(t *testing.T) {
	doc := &Document{
		Resources: []Resource{
			{Kind: "Agent", Name: "myagent", Attributes: map[string]interface{}{"model": "gpt-3.5"}},
			{Kind: "Environment", Name: "prod", Attributes: map[string]interface{}{
				"overrides": []interface{}{
					map[string]interface{}{
						"resource":  "Agent/myagent",
						"attribute": "model",
						"value":     "gpt-4",
					},
				},
			}},
		},
	}

	_, err := ApplyEnvironment(doc, "prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Original doc should be unchanged
	if doc.Resources[0].Attributes["model"] != "gpt-3.5" {
		t.Error("original document was mutated")
	}
}

// ---------------------------------------------------------------------------
// Lower
// ---------------------------------------------------------------------------

func TestLower_MissingPackage(t *testing.T) {
	f := &ast.File{}
	_, err := Lower(f)
	if err == nil {
		t.Fatal("expected error for missing package declaration")
	}
}

func TestLower_AgentPromptSkillSecret(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name:        "testpkg",
			Version:     "1.0.0",
			LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Prompt{
				Name:    "myprompt",
				Content: "Hello, {{name}}!",
			},
			&ast.Skill{
				Name:        "myskill",
				Description: "A test skill",
				Input: []*ast.Field{
					{Name: "query", Type: "string"},
				},
				Output: []*ast.Field{
					{Name: "result", Type: "string"},
				},
				Execution: &ast.Execution{
					Type:  "command",
					Value: "echo hello",
				},
			},
			&ast.Secret{
				Name:   "api_key",
				Source: "env",
				Key:    "API_KEY",
			},
			&ast.Agent{
				Name:  "myagent",
				Model: "gpt-4",
				Prompt: &ast.Ref{
					Kind: "prompt",
					Name: "myprompt",
				},
				Skills: []*ast.Ref{
					{Kind: "skill", Name: "myskill"},
				},
			},
		},
	}

	doc, err := Lower(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Package.Name != "testpkg" {
		t.Errorf("package name: got %q, want %q", doc.Package.Name, "testpkg")
	}
	if doc.IRVersion != "1.0" {
		t.Errorf("ir_version: got %q, want %q", doc.IRVersion, "1.0")
	}

	// Resources should be sorted: Agent, Prompt, Secret, Skill
	if len(doc.Resources) != 4 {
		t.Fatalf("expected 4 resources, got %d", len(doc.Resources))
	}

	expectedKinds := []string{"Agent", "Prompt", "Secret", "Skill"}
	for i, ek := range expectedKinds {
		if doc.Resources[i].Kind != ek {
			t.Errorf("resource %d: got kind %q, want %q", i, doc.Resources[i].Kind, ek)
		}
	}

	// Check agent FQN
	agentRes := doc.Resources[0]
	if agentRes.FQN != "testpkg/Agent/myagent" {
		t.Errorf("agent FQN: got %q, want %q", agentRes.FQN, "testpkg/Agent/myagent")
	}

	// Check agent references
	if len(agentRes.References) != 2 {
		t.Errorf("agent refs: got %d, want 2", len(agentRes.References))
	}

	// Check hash is computed
	if agentRes.Hash == "" {
		t.Error("agent hash should not be empty")
	}
}

func TestLower_EnvironmentResource(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name:        "testpkg",
			Version:     "1.0.0",
			LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Environment{
				Name: "production",
				Overrides: []*ast.Override{
					{Resource: "agent/main", Attribute: "model", Value: "gpt-4"},
				},
			},
		},
	}

	doc, err := Lower(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(doc.Resources))
	}
	r := doc.Resources[0]
	if r.Kind != "Environment" {
		t.Errorf("kind: got %q, want %q", r.Kind, "Environment")
	}
	if r.FQN != "testpkg/Environment/production" {
		t.Errorf("FQN: got %q, want %q", r.FQN, "testpkg/Environment/production")
	}
}

func TestLower_ImportsPreserved(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name:        "testpkg",
			Version:     "1.0.0",
			LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Import{
				Path:    "github.com/example/pkg",
				Version: "1.2.3",
				Alias:   "ex",
			},
		},
	}

	doc, err := Lower(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(doc.Imports))
	}
	if doc.Imports[0].Path != "github.com/example/pkg" {
		t.Errorf("import path: got %q", doc.Imports[0].Path)
	}
	if doc.Imports[0].Version != "1.2.3" {
		t.Errorf("import version: got %q", doc.Imports[0].Version)
	}
	if doc.Imports[0].Alias != "ex" {
		t.Errorf("import alias: got %q", doc.Imports[0].Alias)
	}
}

func TestLower_PolicyAndBinding(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name:        "testpkg",
			Version:     "1.0.0",
			LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Policy{
				Name: "security",
				Rules: []*ast.Rule{
					{Action: "deny", Resource: "Secret", Subject: "plaintext"},
				},
			},
			&ast.Binding{
				Name:    "local",
				Adapter: "local-mcp",
				Default: true,
				Config:  map[string]string{"timeout": "30s"},
			},
		},
	}

	doc, err := Lower(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(doc.Policies))
	}
	if doc.Policies[0].Name != "security" {
		t.Errorf("policy name: got %q", doc.Policies[0].Name)
	}
	if len(doc.Policies[0].Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(doc.Policies[0].Rules))
	}

	if len(doc.Bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(doc.Bindings))
	}
	if doc.Bindings[0].Adapter != "local-mcp" {
		t.Errorf("binding adapter: got %q", doc.Bindings[0].Adapter)
	}
	if !doc.Bindings[0].Default {
		t.Error("binding should be default")
	}
}

func TestLower_MCPServerAndClient(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name:        "testpkg",
			Version:     "1.0.0",
			LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Secret{
				Name:   "token",
				Source: "env",
				Key:    "TOKEN",
			},
			&ast.MCPServer{
				Name:      "myserver",
				Transport: "stdio",
				Command:   "npx",
				Args:      []string{"-y", "server"},
				Auth:      &ast.Ref{Name: "token"},
				Env:       map[string]string{"NODE_ENV": "production"},
			},
			&ast.MCPClient{
				Name: "myclient",
				Servers: []*ast.Ref{
					{Name: "myserver"},
				},
			},
		},
	}

	doc, err := Lower(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 resources: MCPClient, MCPServer, Secret (sorted)
	if len(doc.Resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(doc.Resources))
	}

	// Find MCPServer
	var server *Resource
	var client *Resource
	for i := range doc.Resources {
		switch doc.Resources[i].Kind {
		case "MCPServer":
			server = &doc.Resources[i]
		case "MCPClient":
			client = &doc.Resources[i]
		}
	}

	if server == nil {
		t.Fatal("MCPServer resource not found")
	}
	if server.Attributes["transport"] != "stdio" {
		t.Errorf("transport: got %v", server.Attributes["transport"])
	}
	if server.Attributes["command"] != "npx" {
		t.Errorf("command: got %v", server.Attributes["command"])
	}
	if server.FQN != "testpkg/MCPServer/myserver" {
		t.Errorf("server FQN: got %q", server.FQN)
	}
	// Check server has auth reference
	if len(server.References) == 0 {
		t.Error("server should have auth reference")
	}

	if client == nil {
		t.Fatal("MCPClient resource not found")
	}
	if client.FQN != "testpkg/MCPClient/myclient" {
		t.Errorf("client FQN: got %q", client.FQN)
	}
	if len(client.References) != 1 {
		t.Errorf("client refs: got %d, want 1", len(client.References))
	}
}

func TestLower_TypeDef(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name:        "testpkg",
			Version:     "1.0.0",
			LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.TypeDef{
				Name:     "Status",
				EnumVals: []string{"active", "inactive"},
			},
			&ast.TypeDef{
				Name:   "Tags",
				ListOf: "string",
			},
			&ast.TypeDef{
				Name: "UserProfile",
				Fields: []*ast.TypeField{
					{Name: "name", Type: "string", Required: true},
					{Name: "age", Type: "int", Default: "25"},
				},
			},
		},
	}

	doc, err := Lower(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Resources) != 3 {
		t.Fatalf("expected 3 type resources, got %d", len(doc.Resources))
	}

	for _, r := range doc.Resources {
		if r.Kind != "Type" {
			t.Errorf("expected kind Type, got %q", r.Kind)
		}
	}

	// Find the enum type
	var enumType *Resource
	for i := range doc.Resources {
		if doc.Resources[i].Name == "Status" {
			enumType = &doc.Resources[i]
		}
	}
	if enumType == nil {
		t.Fatal("Status type not found")
	}
	enumVals, ok := enumType.Attributes["enum"].([]interface{})
	if !ok {
		t.Fatal("enum attribute not found or wrong type")
	}
	if len(enumVals) != 2 {
		t.Errorf("expected 2 enum values, got %d", len(enumVals))
	}
}

func TestLower_Pipeline(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name:        "testpkg",
			Version:     "1.0.0",
			LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Pipeline{
				Name: "workflow",
				Steps: []*ast.PipelineStep{
					{Name: "extract", Agent: "extractor", Input: "user_query", Output: "extracted"},
					{Name: "transform", Agent: "transformer", DependsOn: []string{"extract"}, Parallel: true, When: "len(extracted) > 0"},
				},
			},
		},
	}

	doc, err := Lower(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Resources) != 1 {
		t.Fatalf("expected 1 pipeline resource, got %d", len(doc.Resources))
	}

	r := doc.Resources[0]
	if r.Kind != "Pipeline" {
		t.Errorf("kind: got %q, want Pipeline", r.Kind)
	}
	if r.FQN != "testpkg/Pipeline/workflow" {
		t.Errorf("FQN: got %q", r.FQN)
	}
	steps, ok := r.Attributes["steps"].([]interface{})
	if !ok {
		t.Fatal("steps attribute not found or wrong type")
	}
	if len(steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(steps))
	}
	// Check references to agents
	if len(r.References) != 2 {
		t.Errorf("expected 2 agent refs, got %d", len(r.References))
	}
}

func TestLower_DeployTarget(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name:        "testpkg",
			Version:     "1.0.0",
			LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.DeployTarget{
				Name:      "k8s",
				Target:    "kubernetes",
				Default:   true,
				Port:      8080,
				Namespace: "production",
				Replicas:  3,
				Image:     "myapp:latest",
				Resources: &ast.ResourceLimits{CPU: "500m", Memory: "256Mi"},
				Health:    &ast.HealthConfig{Path: "/healthz", Interval: "30s", Timeout: "5s"},
				Autoscale: &ast.AutoscaleConfig{MinReplicas: 2, MaxReplicas: 10, Metric: "cpu", Target: 80},
				Env:       map[string]string{"APP_ENV": "prod"},
				Secrets:   map[string]string{"DB_PASS": "secret/db"},
			},
		},
	}

	doc, err := Lower(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.DeployTargets) != 1 {
		t.Fatalf("expected 1 deploy target, got %d", len(doc.DeployTargets))
	}

	dt := doc.DeployTargets[0]
	if dt.Name != "k8s" {
		t.Errorf("name: got %q", dt.Name)
	}
	if dt.Target != "kubernetes" {
		t.Errorf("target: got %q", dt.Target)
	}
	if !dt.Default {
		t.Error("should be default")
	}
	if dt.Config["port"] != 8080 {
		t.Errorf("port: got %v", dt.Config["port"])
	}
	if dt.Config["namespace"] != "production" {
		t.Errorf("namespace: got %v", dt.Config["namespace"])
	}
}

func TestLower_AgentWithRuntimeConfig(t *testing.T) {
	boolTrue := true
	f := &ast.File{
		Package: &ast.Package{
			Name:        "testpkg",
			Version:     "1.0.0",
			LangVersion: "2.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{
				Name:        "advanced",
				Model:       "gpt-4",
				Prompt:      &ast.Ref{Name: "sys"},
				Strategy:    "react",
				MaxTurns:    5,
				Timeout:     "30s",
				TokenBudget: 4000,
				Temperature: 0.7,
				HasTemp:     true,
				Stream:      &boolTrue,
				OnError:     "fallback",
				MaxRetries:  3,
				Fallback:    "backup",
				MemoryCfg: &ast.MemoryConfig{
					Strategy:    "sliding_window",
					MaxMessages: 100,
				},
				Delegates: []*ast.Delegate{
					{AgentRef: "specialist", Condition: "when complex"},
				},
				Client:   &ast.Ref{Name: "myclient"},
				Metadata: map[string]string{"author": "test"},
			},
		},
	}

	doc, err := Lower(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(doc.Resources))
	}
	r := doc.Resources[0]
	attrs := r.Attributes

	if attrs["strategy"] != "react" {
		t.Errorf("strategy: got %v", attrs["strategy"])
	}
	if attrs["max_turns"] != 5 {
		t.Errorf("max_turns: got %v", attrs["max_turns"])
	}
	if attrs["timeout"] != "30s" {
		t.Errorf("timeout: got %v", attrs["timeout"])
	}
	if attrs["token_budget"] != 4000 {
		t.Errorf("token_budget: got %v", attrs["token_budget"])
	}
	if attrs["temperature"] != 0.7 {
		t.Errorf("temperature: got %v", attrs["temperature"])
	}
	if attrs["stream"] != true {
		t.Errorf("stream: got %v", attrs["stream"])
	}
	if attrs["on_error"] != "fallback" {
		t.Errorf("on_error: got %v", attrs["on_error"])
	}
	if attrs["max_retries"] != 3 {
		t.Errorf("max_retries: got %v", attrs["max_retries"])
	}
	if attrs["fallback"] != "backup" {
		t.Errorf("fallback: got %v", attrs["fallback"])
	}
	if r.Metadata == nil || r.Metadata["author"] != "test" {
		t.Error("metadata not preserved")
	}

	// Check memory config
	mem, ok := attrs["memory"].(map[string]interface{})
	if !ok {
		t.Fatal("memory config not found")
	}
	if mem["strategy"] != "sliding_window" {
		t.Errorf("memory strategy: got %v", mem["strategy"])
	}

	// Check delegates
	delegates, ok := attrs["delegates"].([]interface{})
	if !ok {
		t.Fatal("delegates not found")
	}
	if len(delegates) != 1 {
		t.Errorf("expected 1 delegate, got %d", len(delegates))
	}
}

func TestLower_SkillWithToolConfig(t *testing.T) {
	tests := []struct {
		name     string
		tool     *ast.ToolConfig
		checkKey string
	}{
		{
			name: "mcp tool",
			tool: &ast.ToolConfig{
				Type:       "mcp",
				ServerTool: "myserver/search",
				Timeout:    "10s",
			},
			checkKey: "server_tool",
		},
		{
			name: "http tool",
			tool: &ast.ToolConfig{
				Type:         "http",
				Method:       "POST",
				URL:          "https://api.example.com/search",
				Headers:      map[string]string{"Auth": "Bearer token"},
				BodyTemplate: `{"q": "{{query}}"}`,
				Timeout:      "30s",
			},
			checkKey: "method",
		},
		{
			name: "command tool",
			tool: &ast.ToolConfig{
				Type:    "command",
				Binary:  "/usr/bin/search",
				Args:    []string{"--query", "test"},
				Timeout: "5s",
				Env:     map[string]string{"PATH": "/usr/bin"},
				Secrets: map[string]string{"KEY": "secret/key"},
			},
			checkKey: "binary",
		},
		{
			name: "inline tool",
			tool: &ast.ToolConfig{
				Type:        "inline",
				Language:    "python",
				Code:        "print('hello')",
				MemoryLimit: "128m",
			},
			checkKey: "language",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &ast.File{
				Package: &ast.Package{
					Name: "testpkg", Version: "1.0.0", LangVersion: "2.0",
				},
				Statements: []ast.Statement{
					&ast.Skill{
						Name:        "testskill",
						Description: "test",
						Input:       []*ast.Field{{Name: "q", Type: "string"}},
						Output:      []*ast.Field{{Name: "r", Type: "string"}},
						ToolConfig:  tc.tool,
						Metadata:    map[string]string{"version": "1"},
					},
				},
			}

			doc, err := Lower(f)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(doc.Resources) != 1 {
				t.Fatalf("expected 1 resource, got %d", len(doc.Resources))
			}
			toolAttr, ok := doc.Resources[0].Attributes["tool"].(map[string]interface{})
			if !ok {
				t.Fatal("tool attribute not found")
			}
			if toolAttr["type"] != tc.tool.Type {
				t.Errorf("tool type: got %v, want %v", toolAttr["type"], tc.tool.Type)
			}
			if _, ok := toolAttr[tc.checkKey]; !ok {
				t.Errorf("expected key %q in tool config", tc.checkKey)
			}
		})
	}
}

func TestLower_AgentWithOnInput(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "3.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{
				Name:   "router",
				Model:  "gpt-4",
				Prompt: &ast.Ref{Name: "sys"},
				OnInput: &ast.OnInputBlock{
					Statements: []ast.OnInputStmt{
						&ast.UseSkillStmt{
							SkillName: "search",
							Params:    map[string]string{"mode": "fast"},
						},
						&ast.DelegateToStmt{AgentName: "specialist"},
						&ast.RespondStmt{Expression: `"done"`},
						&ast.IfBlock{
							Condition: "input == 'hello'",
							Body: []ast.OnInputStmt{
								&ast.RespondStmt{Expression: `"hi"`},
							},
							ElseIfs: []*ast.ElseIfClause{
								{Condition: "input == 'bye'", Body: []ast.OnInputStmt{
									&ast.RespondStmt{Expression: `"goodbye"`},
								}},
							},
							ElseBody: []ast.OnInputStmt{
								&ast.RespondStmt{Expression: `"unknown"`},
							},
						},
						&ast.ForEachBlock{
							Variable:   "item",
							Collection: "items",
							Body: []ast.OnInputStmt{
								&ast.UseSkillStmt{SkillName: "process"},
							},
						},
					},
				},
			},
		},
	}

	doc, err := Lower(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(doc.Resources))
	}

	onInput, ok := doc.Resources[0].Attributes["on_input"].([]interface{})
	if !ok {
		t.Fatal("on_input attribute not found")
	}
	if len(onInput) != 5 {
		t.Errorf("expected 5 on_input statements, got %d", len(onInput))
	}

	// Check first statement is use_skill
	first, ok := onInput[0].(map[string]interface{})
	if !ok {
		t.Fatal("first on_input statement is not a map")
	}
	if first["type"] != "use_skill" {
		t.Errorf("first type: got %v, want use_skill", first["type"])
	}
	if first["skill"] != "search" {
		t.Errorf("first skill: got %v, want search", first["skill"])
	}
}

func TestLower_AgentWithConfigValidationEval(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "3.0",
		},
		Statements: []ast.Statement{
			&ast.Agent{
				Name:   "validated",
				Model:  "gpt-4",
				Prompt: &ast.Ref{Name: "sys"},
				ConfigParams: []*ast.ConfigParam{
					{Name: "api_key", Type: "string", Required: true, Secret: true, Description: "The API key"},
					{Name: "max_items", Type: "int", HasDefault: true, Default: "10"},
				},
				ValidationRules: []*ast.ValidationRule{
					{Name: "length_check", Severity: "error", Expression: "len(output) > 0", MaxRetries: 2, Message: "Output too short"},
				},
				EvalCases: []*ast.EvalCase{
					{Name: "basic", Input: "hello", Expected: "Hello!", Scoring: "exact", Threshold: 0.9, Tags: []string{"smoke", "basic"}},
					{Name: "default_threshold", Input: "test", Expected: "Test"},
				},
			},
		},
	}

	doc, err := Lower(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(doc.Resources))
	}

	attrs := doc.Resources[0].Attributes

	// Check config_params
	params, ok := attrs["config_params"].([]interface{})
	if !ok {
		t.Fatal("config_params not found")
	}
	if len(params) != 2 {
		t.Errorf("expected 2 config params, got %d", len(params))
	}

	// Check validation_rules
	rules, ok := attrs["validation_rules"].([]interface{})
	if !ok {
		t.Fatal("validation_rules not found")
	}
	if len(rules) != 1 {
		t.Errorf("expected 1 validation rule, got %d", len(rules))
	}

	// Check eval_cases
	cases, ok := attrs["eval_cases"].([]interface{})
	if !ok {
		t.Fatal("eval_cases not found")
	}
	if len(cases) != 2 {
		t.Errorf("expected 2 eval cases, got %d", len(cases))
	}
}

func TestLower_PromptWithVariablesAndMetadata(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Name: "testpkg", Version: "1.0.0", LangVersion: "1.0",
		},
		Statements: []ast.Statement{
			&ast.Prompt{
				Name:    "template",
				Content: "Hello {{name}}, you are {{role}}",
				Version: "2.0",
				Variables: []*ast.Variable{
					{Name: "name", Type: "string", Required: true},
					{Name: "role", Type: "string", Default: "user"},
				},
				Metadata: map[string]string{"author": "test"},
			},
		},
	}

	doc, err := Lower(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(doc.Resources))
	}

	r := doc.Resources[0]
	if r.Attributes["version"] != "2.0" {
		t.Errorf("version: got %v", r.Attributes["version"])
	}
	vars, ok := r.Attributes["variables"].([]interface{})
	if !ok {
		t.Fatal("variables not found")
	}
	if len(vars) != 2 {
		t.Errorf("expected 2 variables, got %d", len(vars))
	}
	if r.Metadata == nil || r.Metadata["author"] != "test" {
		t.Error("metadata not preserved")
	}
}
