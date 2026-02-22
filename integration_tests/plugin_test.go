package integration_tests

import (
	"testing"

	"github.com/szaher/designs/agentz/internal/plugins"
)

func TestPluginManifestLoad(t *testing.T) {
	manifest, err := plugins.LoadManifestFromFile("../plugins/monitor/manifest.json")
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	if manifest.Name != "monitor" {
		t.Errorf("expected name 'monitor', got %q", manifest.Name)
	}
	if manifest.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", manifest.Version)
	}

	// Check capabilities
	if len(manifest.Capabilities.ResourceTypes) != 1 {
		t.Errorf("expected 1 resource type, got %d", len(manifest.Capabilities.ResourceTypes))
	}
	if manifest.Capabilities.ResourceTypes[0].Kind != "Monitor" {
		t.Errorf("expected resource type 'Monitor', got %q", manifest.Capabilities.ResourceTypes[0].Kind)
	}

	if len(manifest.Capabilities.Validators) != 1 {
		t.Errorf("expected 1 validator, got %d", len(manifest.Capabilities.Validators))
	}
	if len(manifest.Capabilities.Transforms) != 1 {
		t.Errorf("expected 1 transform, got %d", len(manifest.Capabilities.Transforms))
	}
	if len(manifest.Capabilities.Hooks) != 1 {
		t.Errorf("expected 1 hook, got %d", len(manifest.Capabilities.Hooks))
	}
}

func TestPluginDuplicateTypeConflict(t *testing.T) {
	plugin1 := &plugins.LoadedPlugin{
		Manifest: plugins.Manifest{
			Name: "plugin-a",
			Capabilities: plugins.Capabilities{
				ResourceTypes: []plugins.ResourceType{
					{Kind: "Monitor"},
				},
			},
		},
	}
	plugin2 := &plugins.LoadedPlugin{
		Manifest: plugins.Manifest{
			Name: "plugin-b",
			Capabilities: plugins.Capabilities{
				ResourceTypes: []plugins.ResourceType{
					{Kind: "Monitor"},
				},
			},
		},
	}

	err := plugins.CheckConflicts([]*plugins.LoadedPlugin{plugin1, plugin2})
	if err == nil {
		t.Fatal("expected conflict error for duplicate Monitor type")
	}
}

func TestPluginHookExecution(t *testing.T) {
	plugin := &plugins.LoadedPlugin{
		Manifest: plugins.Manifest{
			Name: "monitor",
			Capabilities: plugins.Capabilities{
				Hooks: []plugins.Hook{
					{Stage: "pre-apply", Name: "monitor-preflight", Description: "Check endpoints"},
				},
			},
		},
	}

	results, err := plugins.ExecuteHooks([]*plugins.LoadedPlugin{plugin}, plugins.StagePreApply, nil)
	if err != nil {
		t.Fatalf("hook execution failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 hook result, got %d", len(results))
	}
	if !results[0].Success {
		t.Error("hook should have succeeded")
	}
	if results[0].Plugin != "monitor" {
		t.Errorf("expected plugin 'monitor', got %q", results[0].Plugin)
	}
}

func TestPluginMissingError(t *testing.T) {
	_, err := plugins.ResolvePluginPath("nonexistent-plugin", "1.0.0")
	if err == nil {
		t.Fatal("expected error for missing plugin")
	}
}
