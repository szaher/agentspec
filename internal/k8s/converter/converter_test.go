package converter

import (
	"testing"

	"github.com/szaher/agentspec/internal/ir"
)

func TestConvertDocument(t *testing.T) {
	tests := []struct {
		name      string
		doc       *ir.Document
		namespace string
		wantCount int
		wantErr   bool
	}{
		{
			name: "empty document",
			doc: &ir.Document{
				Resources: []ir.Resource{},
			},
			namespace: "default",
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "single agent",
			doc: &ir.Document{
				Resources: []ir.Resource{
					{
						Kind: "Agent",
						Name: "test-agent",
						Attributes: map[string]interface{}{
							"model": "claude-3-5-sonnet-20241022",
						},
					},
				},
			},
			namespace: "default",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "multiple resources",
			doc: &ir.Document{
				Resources: []ir.Resource{
					{
						Kind: "Agent",
						Name: "agent1",
						Attributes: map[string]interface{}{
							"model": "claude-3-5-sonnet-20241022",
						},
					},
					{
						Kind: "Prompt",
						Name: "prompt1",
						Attributes: map[string]interface{}{
							"content": "You are a helpful assistant",
						},
					},
					{
						Kind: "Skill",
						Name: "skill1",
						Attributes: map[string]interface{}{
							"description": "Test skill",
							"tool": map[string]interface{}{
								"type":   "command",
								"binary": "echo",
							},
						},
					},
				},
			},
			namespace: "test-ns",
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "unsupported resource kinds are skipped",
			doc: &ir.Document{
				Resources: []ir.Resource{
					{
						Kind: "Agent",
						Name: "agent1",
						Attributes: map[string]interface{}{
							"model": "claude-3-5-sonnet-20241022",
						},
					},
					{
						Kind:       "Type",
						Name:       "unsupported1",
						Attributes: map[string]interface{}{},
					},
					{
						Kind:       "Environment",
						Name:       "unsupported2",
						Attributes: map[string]interface{}{},
					},
					{
						Kind: "Prompt",
						Name: "prompt1",
						Attributes: map[string]interface{}{
							"content": "test",
						},
					},
				},
			},
			namespace: "default",
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertDocument(tt.doc, tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertDocument() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("ConvertDocument() got %d resources, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestConvertAgent(t *testing.T) {
	tests := []struct {
		name      string
		res       ir.Resource
		namespace string
		wantErr   bool
		validate  func(t *testing.T, resources []Resource)
	}{
		{
			name: "minimal agent",
			res: ir.Resource{
				Kind: "Agent",
				Name: "minimal-agent",
				Attributes: map[string]interface{}{
					"model": "claude-3-5-sonnet-20241022",
				},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]
				if r.Kind != "Agent" {
					t.Errorf("expected Kind=Agent, got %s", r.Kind)
				}
				if r.Name != "minimal-agent" {
					t.Errorf("expected Name=minimal-agent, got %s", r.Name)
				}
				if r.Namespace != "default" {
					t.Errorf("expected Namespace=default, got %s", r.Namespace)
				}
				if r.APIVersion != "agentspec.io/v1alpha1" {
					t.Errorf("expected APIVersion=agentspec.io/v1alpha1, got %s", r.APIVersion)
				}
			},
		},
		{
			name: "agent with all attributes",
			res: ir.Resource{
				Kind: "Agent",
				Name: "full-agent",
				Attributes: map[string]interface{}{
					"model":     "claude-3-5-sonnet-20241022",
					"strategy":  "react",
					"prompt":    "system-prompt",
					"max_turns": float64(10),
					"skills":    []interface{}{"skill1", "skill2", "skill3"},
				},
			},
			namespace: "test-ns",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]

				spec, ok := r.Raw["spec"].(map[string]interface{})
				if !ok {
					t.Fatal("spec not found in Raw")
				}

				if spec["model"] != "claude-3-5-sonnet-20241022" {
					t.Errorf("expected model=claude-3-5-sonnet-20241022, got %v", spec["model"])
				}
				if spec["strategy"] != "react" {
					t.Errorf("expected strategy=react, got %v", spec["strategy"])
				}
				if spec["promptRef"] != "system-prompt" {
					t.Errorf("expected promptRef=system-prompt, got %v", spec["promptRef"])
				}

				maxTurns, ok := spec["maxTurns"].(float64)
				if !ok || maxTurns != 10 {
					t.Errorf("expected maxTurns=10, got %v", spec["maxTurns"])
				}

				skillRefs, ok := spec["skillRefs"].([]interface{})
				if !ok {
					t.Fatal("skillRefs not found or wrong type")
				}
				if len(skillRefs) != 3 {
					t.Errorf("expected 3 skillRefs, got %d", len(skillRefs))
				}
			},
		},
		{
			name: "agent with empty skills list",
			res: ir.Resource{
				Kind: "Agent",
				Name: "agent-no-skills",
				Attributes: map[string]interface{}{
					"model":  "claude-3-5-sonnet-20241022",
					"skills": []interface{}{},
				},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertAgent(tt.res, tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestConvertPrompt(t *testing.T) {
	tests := []struct {
		name      string
		res       ir.Resource
		namespace string
		wantErr   bool
		validate  func(t *testing.T, resources []Resource)
	}{
		{
			name: "basic prompt",
			res: ir.Resource{
				Kind: "Prompt",
				Name: "test-prompt",
				Attributes: map[string]interface{}{
					"content": "You are a helpful AI assistant.",
				},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]
				if r.Kind != "ConfigMap" {
					t.Errorf("expected Kind=ConfigMap, got %s", r.Kind)
				}
				if r.APIVersion != "v1" {
					t.Errorf("expected APIVersion=v1, got %s", r.APIVersion)
				}

				data, ok := r.Raw["data"].(map[string]interface{})
				if !ok {
					t.Fatal("data not found in Raw")
				}
				if data["system-prompt"] != "You are a helpful AI assistant." {
					t.Errorf("expected system-prompt content, got %v", data["system-prompt"])
				}
			},
		},
		{
			name: "empty prompt",
			res: ir.Resource{
				Kind: "Prompt",
				Name: "empty-prompt",
				Attributes: map[string]interface{}{
					"content": "",
				},
			},
			namespace: "test-ns",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]
				data, ok := r.Raw["data"].(map[string]interface{})
				if !ok {
					t.Fatal("data not found in Raw")
				}
				if data["system-prompt"] != "" {
					t.Errorf("expected empty system-prompt, got %v", data["system-prompt"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertPrompt(tt.res, tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestConvertSkill(t *testing.T) {
	tests := []struct {
		name      string
		res       ir.Resource
		namespace string
		wantErr   bool
		validate  func(t *testing.T, resources []Resource)
	}{
		{
			name: "inline tool",
			res: ir.Resource{
				Kind: "Skill",
				Name: "inline-skill",
				Attributes: map[string]interface{}{
					"description": "Execute inline bash",
					"tool": map[string]interface{}{
						"type": "inline",
						"code": "echo hello",
					},
				},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]
				if r.Kind != "ToolBinding" {
					t.Errorf("expected Kind=ToolBinding, got %s", r.Kind)
				}

				spec, ok := r.Raw["spec"].(map[string]interface{})
				if !ok {
					t.Fatal("spec not found in Raw")
				}
				if spec["toolType"] != "command" {
					t.Errorf("expected toolType=command, got %v", spec["toolType"])
				}

				command, ok := spec["command"].(map[string]interface{})
				if !ok {
					t.Fatal("command not found in spec")
				}
				if command["binary"] != "sh" {
					t.Errorf("expected binary=sh, got %v", command["binary"])
				}

				args, ok := command["args"].([]interface{})
				if !ok || len(args) != 2 {
					t.Fatalf("expected 2 args, got %v", args)
				}
				if args[0] != "-c" || args[1] != "echo hello" {
					t.Errorf("expected args=[-c, echo hello], got %v", args)
				}
			},
		},
		{
			name: "mcp tool",
			res: ir.Resource{
				Kind: "Skill",
				Name: "mcp-skill",
				Attributes: map[string]interface{}{
					"description": "MCP server tool",
					"tool": map[string]interface{}{
						"type":        "mcp",
						"server_tool": "mcp-server-1",
					},
				},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]

				spec, ok := r.Raw["spec"].(map[string]interface{})
				if !ok {
					t.Fatal("spec not found in Raw")
				}
				if spec["toolType"] != "mcp" {
					t.Errorf("expected toolType=mcp, got %v", spec["toolType"])
				}

				mcp, ok := spec["mcp"].(map[string]interface{})
				if !ok {
					t.Fatal("mcp not found in spec")
				}
				if mcp["serverRef"] != "mcp-server-1" {
					t.Errorf("expected serverRef=mcp-server-1, got %v", mcp["serverRef"])
				}
			},
		},
		{
			name: "command tool",
			res: ir.Resource{
				Kind: "Skill",
				Name: "command-skill",
				Attributes: map[string]interface{}{
					"description": "Run external command",
					"tool": map[string]interface{}{
						"type":   "command",
						"binary": "kubectl",
						"args":   []interface{}{"get", "pods"},
					},
				},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]

				spec, ok := r.Raw["spec"].(map[string]interface{})
				if !ok {
					t.Fatal("spec not found in Raw")
				}
				if spec["toolType"] != "command" {
					t.Errorf("expected toolType=command, got %v", spec["toolType"])
				}

				command, ok := spec["command"].(map[string]interface{})
				if !ok {
					t.Fatal("command not found in spec")
				}
				if command["binary"] != "kubectl" {
					t.Errorf("expected binary=kubectl, got %v", command["binary"])
				}

				args, ok := command["args"].([]interface{})
				if !ok || len(args) != 2 {
					t.Fatalf("expected 2 args, got %v", args)
				}
			},
		},
		{
			name: "http tool",
			res: ir.Resource{
				Kind: "Skill",
				Name: "http-skill",
				Attributes: map[string]interface{}{
					"description": "HTTP API call",
					"tool": map[string]interface{}{
						"type":   "http",
						"url":    "https://api.example.com/data",
						"method": "GET",
					},
				},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]

				spec, ok := r.Raw["spec"].(map[string]interface{})
				if !ok {
					t.Fatal("spec not found in Raw")
				}
				if spec["toolType"] != "http" {
					t.Errorf("expected toolType=http, got %v", spec["toolType"])
				}

				http, ok := spec["http"].(map[string]interface{})
				if !ok {
					t.Fatal("http not found in spec")
				}
				if http["url"] != "https://api.example.com/data" {
					t.Errorf("expected url=https://api.example.com/data, got %v", http["url"])
				}
				if http["method"] != "GET" {
					t.Errorf("expected method=GET, got %v", http["method"])
				}
			},
		},
		{
			name: "unknown tool type",
			res: ir.Resource{
				Kind: "Skill",
				Name: "unknown-skill",
				Attributes: map[string]interface{}{
					"description": "Unknown tool type",
					"tool": map[string]interface{}{
						"type": "custom",
					},
				},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]

				spec, ok := r.Raw["spec"].(map[string]interface{})
				if !ok {
					t.Fatal("spec not found in Raw")
				}
				if spec["toolType"] != "custom" {
					t.Errorf("expected toolType=custom, got %v", spec["toolType"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertSkill(tt.res, tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertSkill() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestConvertSecret(t *testing.T) {
	tests := []struct {
		name      string
		res       ir.Resource
		namespace string
		wantErr   bool
		validate  func(t *testing.T, resources []Resource)
	}{
		{
			name: "basic secret",
			res: ir.Resource{
				Kind: "Secret",
				Name: "api-key",
				Attributes: map[string]interface{}{
					"key": "sk-1234567890",
				},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]
				if r.Kind != "Secret" {
					t.Errorf("expected Kind=Secret, got %s", r.Kind)
				}
				if r.APIVersion != "v1" {
					t.Errorf("expected APIVersion=v1, got %s", r.APIVersion)
				}

				if r.Raw["type"] != "Opaque" {
					t.Errorf("expected type=Opaque, got %v", r.Raw["type"])
				}

				stringData, ok := r.Raw["stringData"].(map[string]interface{})
				if !ok {
					t.Fatal("stringData not found in Raw")
				}
				if stringData["key"] != "sk-1234567890" {
					t.Errorf("expected key=sk-1234567890, got %v", stringData["key"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertSecret(tt.res, tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestConvertPipeline(t *testing.T) {
	tests := []struct {
		name      string
		res       ir.Resource
		namespace string
		wantErr   bool
		validate  func(t *testing.T, resources []Resource)
	}{
		{
			name: "simple pipeline",
			res: ir.Resource{
				Kind: "Pipeline",
				Name: "test-pipeline",
				Attributes: map[string]interface{}{
					"steps": []interface{}{
						map[string]interface{}{
							"name":  "step1",
							"agent": "agent1",
							"input": "Do task 1",
						},
						map[string]interface{}{
							"name":  "step2",
							"agent": "agent2",
							"input": "Do task 2",
						},
					},
				},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]
				if r.Kind != "Workflow" {
					t.Errorf("expected Kind=Workflow, got %s", r.Kind)
				}

				spec, ok := r.Raw["spec"].(map[string]interface{})
				if !ok {
					t.Fatal("spec not found in Raw")
				}

				steps, ok := spec["steps"].([]interface{})
				if !ok {
					t.Fatal("steps not found in spec")
				}
				if len(steps) != 2 {
					t.Errorf("expected 2 steps, got %d", len(steps))
				}
			},
		},
		{
			name: "pipeline with dependencies",
			res: ir.Resource{
				Kind: "Pipeline",
				Name: "dependent-pipeline",
				Attributes: map[string]interface{}{
					"steps": []interface{}{
						map[string]interface{}{
							"name":  "step1",
							"agent": "agent1",
							"input": "Do task 1",
						},
						map[string]interface{}{
							"name":       "step2",
							"agent":      "agent2",
							"input":      "Do task 2",
							"depends_on": []interface{}{"step1"},
						},
						map[string]interface{}{
							"name":       "step3",
							"agent":      "agent3",
							"input":      "Do task 3",
							"depends_on": []interface{}{"step1", "step2"},
						},
					},
				},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]

				spec, ok := r.Raw["spec"].(map[string]interface{})
				if !ok {
					t.Fatal("spec not found in Raw")
				}

				steps, ok := spec["steps"].([]interface{})
				if !ok {
					t.Fatal("steps not found in spec")
				}
				if len(steps) != 3 {
					t.Errorf("expected 3 steps, got %d", len(steps))
				}

				step2, ok := steps[1].(map[string]interface{})
				if !ok {
					t.Fatal("step2 is not a map")
				}
				deps, ok := step2["dependsOn"].([]interface{})
				if !ok || len(deps) != 1 {
					t.Errorf("expected step2 to have 1 dependency, got %v", deps)
				}

				step3, ok := steps[2].(map[string]interface{})
				if !ok {
					t.Fatal("step3 is not a map")
				}
				deps3, ok := step3["dependsOn"].([]interface{})
				if !ok || len(deps3) != 2 {
					t.Errorf("expected step3 to have 2 dependencies, got %v", deps3)
				}
			},
		},
		{
			name: "empty pipeline",
			res: ir.Resource{
				Kind: "Pipeline",
				Name: "empty-pipeline",
				Attributes: map[string]interface{}{
					"steps": []interface{}{},
				},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]

				spec, ok := r.Raw["spec"].(map[string]interface{})
				if !ok {
					t.Fatal("spec not found in Raw")
				}

				// Empty slices may be omitted or nil in JSON marshaling
				if steps, ok := spec["steps"]; ok && steps != nil {
					stepsList, ok := steps.([]interface{})
					if ok && len(stepsList) != 0 {
						t.Errorf("expected 0 steps, got %d", len(stepsList))
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertPipeline(tt.res, tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertPipeline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestConvertMCPServer(t *testing.T) {
	tests := []struct {
		name      string
		res       ir.Resource
		namespace string
		wantErr   bool
		validate  func(t *testing.T, resources []Resource)
	}{
		{
			name: "basic MCP server",
			res: ir.Resource{
				Kind:       "MCPServer",
				Name:       "test-mcp",
				Attributes: map[string]interface{}{},
			},
			namespace: "default",
			wantErr:   false,
			validate: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("expected 1 resource, got %d", len(resources))
				}
				r := resources[0]
				if r.Kind != "ToolBinding" {
					t.Errorf("expected Kind=ToolBinding, got %s", r.Kind)
				}
				if r.Name != "test-mcp" {
					t.Errorf("expected Name=test-mcp, got %s", r.Name)
				}

				spec, ok := r.Raw["spec"].(map[string]interface{})
				if !ok {
					t.Fatal("spec not found in Raw")
				}
				if spec["toolType"] != "mcp" {
					t.Errorf("expected toolType=mcp, got %v", spec["toolType"])
				}

				mcp, ok := spec["mcp"].(map[string]interface{})
				if !ok {
					t.Fatal("mcp not found in spec")
				}
				if mcp["serverRef"] != "test-mcp" {
					t.Errorf("expected serverRef=test-mcp, got %v", mcp["serverRef"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertMCPServer(tt.res, tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertMCPServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestStringAttr(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]interface{}
		key   string
		want  string
	}{
		{
			name: "existing string value",
			attrs: map[string]interface{}{
				"key": "value",
			},
			key:  "key",
			want: "value",
		},
		{
			name: "non-existent key",
			attrs: map[string]interface{}{
				"other": "value",
			},
			key:  "key",
			want: "",
		},
		{
			name: "non-string value",
			attrs: map[string]interface{}{
				"key": 123,
			},
			key:  "key",
			want: "",
		},
		{
			name:  "nil attrs",
			attrs: nil,
			key:   "key",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringAttr(tt.attrs, tt.key)
			if got != tt.want {
				t.Errorf("stringAttr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntAttr(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]interface{}
		key   string
		want  int
	}{
		{
			name: "float64 value",
			attrs: map[string]interface{}{
				"key": float64(42),
			},
			key:  "key",
			want: 42,
		},
		{
			name: "int value",
			attrs: map[string]interface{}{
				"key": 123,
			},
			key:  "key",
			want: 123,
		},
		{
			name: "non-existent key",
			attrs: map[string]interface{}{
				"other": 42,
			},
			key:  "key",
			want: 0,
		},
		{
			name: "non-numeric value",
			attrs: map[string]interface{}{
				"key": "not a number",
			},
			key:  "key",
			want: 0,
		},
		{
			name:  "nil attrs",
			attrs: nil,
			key:   "key",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := intAttr(tt.attrs, tt.key)
			if got != tt.want {
				t.Errorf("intAttr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringFromMap(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "existing string value",
			m: map[string]interface{}{
				"key": "value",
			},
			key:  "key",
			want: "value",
		},
		{
			name: "non-existent key",
			m: map[string]interface{}{
				"other": "value",
			},
			key:  "key",
			want: "",
		},
		{
			name: "non-string value",
			m: map[string]interface{}{
				"key": 456,
			},
			key:  "key",
			want: "",
		},
		{
			name: "nil map",
			m:    nil,
			key:  "key",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringFromMap(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("stringFromMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
