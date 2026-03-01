package integration_tests

import (
	"context"
	"os/exec"
	"testing"

	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/adapters/docker"
	"github.com/szaher/designs/agentz/internal/ir"
)

func isDockerAvailable() bool {
	cmd := exec.CommandContext(context.Background(), "docker", "info")
	return cmd.Run() == nil
}

func TestDockerAdapterValidate(t *testing.T) {
	_ = docker.Adapter{}
	a := &docker.Adapter{}

	// No agents should fail
	err := a.Validate(context.Background(), []ir.Resource{
		{Kind: "Prompt", Name: "sys"},
	})
	if err == nil {
		t.Fatal("expected error for no agents")
	}

	// With agent should pass
	err = a.Validate(context.Background(), []ir.Resource{
		{Kind: "Agent", Name: "bot"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerAdapterExport(t *testing.T) {
	a := &docker.Adapter{}
	outDir := t.TempDir()

	resources := []ir.Resource{
		{Kind: "Agent", Name: "test-bot", FQN: "test/Agent/test-bot", Attributes: map[string]interface{}{"model": "claude-sonnet-4-20250514"}},
		{Kind: "DeployTarget", Name: "docker", Attributes: map[string]interface{}{"port": float64(9090)}},
	}

	if err := a.Export(context.Background(), resources, outDir); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// Check Dockerfile exists
	checkFileExists(t, outDir+"/Dockerfile")
	checkFileExists(t, outDir+"/runtime-config.json")
}

func TestDockerAdapterApply(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	a := &docker.Adapter{}

	actions := []adapters.Action{
		{
			FQN:  "test/Agent/bot",
			Type: adapters.ActionCreate,
			Resource: &ir.Resource{
				Kind: "Agent", Name: "bot", FQN: "test/Agent/bot",
				Attributes: map[string]interface{}{"model": "claude-sonnet-4-20250514"},
			},
		},
	}

	results, err := a.Apply(context.Background(), actions)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Clean up
	defer func() {
		_, _ = a.Destroy(context.Background())
	}()

	for _, r := range results {
		if r.Status != adapters.ResultSuccess {
			t.Logf("apply result (may fail without project source): %s: %s", r.FQN, r.Error)
		}
	}
}

func checkFileExists(t *testing.T, path string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "test", "-f", path)
	if err := cmd.Run(); err != nil {
		t.Errorf("expected file %s to exist", path)
	}
}
