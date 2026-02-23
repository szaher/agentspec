package integration_tests

import (
	"context"
	"os/exec"
	"testing"

	"github.com/szaher/designs/agentz/internal/adapters/kubernetes"
	"github.com/szaher/designs/agentz/internal/ir"
)

func isKubectlAvailable() bool {
	cmd := exec.Command("kubectl", "version", "--client")
	return cmd.Run() == nil
}

func TestKubernetesManifestGeneration(t *testing.T) {
	resources := []ir.Resource{
		{Kind: "Agent", Name: "bot", FQN: "test/Agent/bot", Attributes: map[string]interface{}{"model": "claude-sonnet-4-20250514"}},
	}
	config := map[string]interface{}{
		"namespace": "agents",
		"replicas":  float64(3),
		"port":      float64(8080),
		"image":     "agentspec:v1",
	}

	manifests := kubernetes.GenerateManifests(resources, config)

	if manifests.Namespace == nil {
		t.Fatal("expected namespace manifest for non-default namespace")
	}
	ns, _ := manifests.Namespace["metadata"].(map[string]interface{})
	if ns["name"] != "agents" {
		t.Errorf("expected namespace 'agents', got %v", ns["name"])
	}

	if manifests.Deployment == nil {
		t.Fatal("expected deployment manifest")
	}
	spec, _ := manifests.Deployment["spec"].(map[string]interface{})
	if spec["replicas"] != 3 {
		t.Errorf("expected 3 replicas, got %v", spec["replicas"])
	}

	if manifests.Service == nil {
		t.Fatal("expected service manifest")
	}

	if manifests.ConfigMap == nil {
		t.Fatal("expected configmap manifest")
	}
}

func TestKubernetesManifestDefaultNamespace(t *testing.T) {
	resources := []ir.Resource{
		{Kind: "Agent", Name: "bot", FQN: "test/Agent/bot", Attributes: map[string]interface{}{"model": "claude-sonnet-4-20250514"}},
	}
	config := map[string]interface{}{
		"namespace": "default",
	}

	manifests := kubernetes.GenerateManifests(resources, config)

	if manifests.Namespace != nil {
		t.Error("expected no namespace manifest for default namespace")
	}
}

func TestKubernetesManifestWithHPA(t *testing.T) {
	resources := []ir.Resource{
		{Kind: "Agent", Name: "bot", FQN: "test/Agent/bot", Attributes: map[string]interface{}{"model": "claude-sonnet-4-20250514"}},
	}
	config := map[string]interface{}{
		"namespace": "agents",
		"autoscale": map[string]interface{}{
			"min_replicas": float64(2),
			"max_replicas": float64(20),
			"target_cpu":   float64(70),
		},
	}

	manifests := kubernetes.GenerateManifests(resources, config)

	if manifests.HPA == nil {
		t.Fatal("expected HPA manifest with autoscale config")
	}
	spec, _ := manifests.HPA["spec"].(map[string]interface{})
	if spec["minReplicas"] != 2 {
		t.Errorf("expected minReplicas=2, got %v", spec["minReplicas"])
	}
	if spec["maxReplicas"] != 20 {
		t.Errorf("expected maxReplicas=20, got %v", spec["maxReplicas"])
	}
}

func TestKubernetesExport(t *testing.T) {
	a := &kubernetes.Adapter{}
	a.SetConfig(map[string]interface{}{
		"namespace": "test-ns",
		"replicas":  float64(2),
		"port":      float64(9090),
		"image":     "agentspec:test",
	})

	outDir := t.TempDir()
	resources := []ir.Resource{
		{Kind: "Agent", Name: "bot", FQN: "test/Agent/bot", Attributes: map[string]interface{}{"model": "claude-sonnet-4-20250514"}},
	}

	if err := a.Export(context.Background(), resources, outDir); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	checkFileExists(t, outDir+"/deployment.json")
	checkFileExists(t, outDir+"/service.json")
	checkFileExists(t, outDir+"/configmap.json")
	checkFileExists(t, outDir+"/namespace.json")
}

func TestKubernetesAdapterValidate(t *testing.T) {
	a := &kubernetes.Adapter{}

	err := a.Validate(context.Background(), []ir.Resource{
		{Kind: "Prompt", Name: "sys"},
	})
	if err == nil {
		t.Fatal("expected error for no agents")
	}

	err = a.Validate(context.Background(), []ir.Resource{
		{Kind: "Agent", Name: "bot"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKubernetesApplyRequiresCluster(t *testing.T) {
	if !isKubectlAvailable() {
		t.Skip("kubectl not available")
	}
	// This test would apply to a real cluster — skipped by default
	t.Skip("Requires a running Kubernetes cluster (kind/minikube)")
}
