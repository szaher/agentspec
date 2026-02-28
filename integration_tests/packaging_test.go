package integration_tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/adapters/compose"
	"github.com/szaher/designs/agentz/internal/adapters/docker"
	"github.com/szaher/designs/agentz/internal/adapters/kubernetes"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
)

// TestDockerfileFromBinary verifies Dockerfile generation for a compiled binary.
func TestDockerfileFromBinary(t *testing.T) {
	dockerfile := docker.GenerateDockerfileFromBinary("/build/myagent", 8080)

	if !strings.Contains(dockerfile, "FROM alpine") {
		t.Error("expected alpine base image")
	}
	if !strings.Contains(dockerfile, "COPY myagent /app/myagent") {
		t.Error("expected COPY of binary")
	}
	if !strings.Contains(dockerfile, "EXPOSE 8080") {
		t.Error("expected EXPOSE 8080")
	}
	if !strings.Contains(dockerfile, "/healthz") {
		t.Error("expected health check")
	}
	if !strings.Contains(dockerfile, "ENTRYPOINT") {
		t.Error("expected ENTRYPOINT")
	}
}

// TestDockerfileFromIR verifies Dockerfile generation from IR resources.
func TestDockerfileFromIR(t *testing.T) {
	input := readTestFile(t, "testdata/valid.ias")
	f, _ := parser.Parse(input, "valid.ias")
	doc, _ := ir.Lower(f)

	dockerfile := docker.GenerateDockerfile(doc.Resources, 8080)

	if !strings.Contains(dockerfile, "FROM golang") {
		t.Error("expected Go builder base image")
	}
	if !strings.Contains(dockerfile, "HEALTHCHECK") {
		t.Error("expected HEALTHCHECK")
	}
}

// TestKubernetesManifests verifies K8s manifest generation.
func TestKubernetesManifests(t *testing.T) {
	input := readTestFile(t, "testdata/valid.ias")
	f, _ := parser.Parse(input, "valid.ias")
	doc, _ := ir.Lower(f)

	config := map[string]interface{}{
		"image":     "myagent:1.0",
		"port":      8080,
		"namespace": "agents",
		"replicas":  2,
	}

	manifests := kubernetes.GenerateManifests(doc.Resources, config)

	// Verify all manifests are generated
	if manifests.Deployment == nil {
		t.Error("expected Deployment manifest")
	}
	if manifests.Service == nil {
		t.Error("expected Service manifest")
	}
	if manifests.ConfigMap == nil {
		t.Error("expected ConfigMap manifest")
	}
	if manifests.Namespace == nil {
		t.Error("expected Namespace manifest (non-default namespace)")
	}

	// Verify Deployment has probes
	spec, _ := manifests.Deployment["spec"].(map[string]interface{})
	tmpl, _ := spec["template"].(map[string]interface{})
	tmplSpec, _ := tmpl["spec"].(map[string]interface{})
	containers, _ := tmplSpec["containers"].([]map[string]interface{})
	if len(containers) == 0 {
		t.Fatal("expected containers in deployment")
	}
	if containers[0]["livenessProbe"] == nil {
		t.Error("expected livenessProbe")
	}
	if containers[0]["readinessProbe"] == nil {
		t.Error("expected readinessProbe")
	}

	// Write and verify
	outputDir := t.TempDir()
	if err := kubernetes.WriteManifests(manifests, outputDir); err != nil {
		t.Fatalf("write manifests error: %v", err)
	}

	expectedFiles := []string{"deployment.json", "service.json", "configmap.json", "namespace.json"}
	for _, f := range expectedFiles {
		if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %q to exist", f)
		}
	}
}

// TestHelmChart verifies Helm chart generation.
func TestHelmChart(t *testing.T) {
	chart := kubernetes.GenerateHelmChart("myagent", "myagent:1.0.0", 8080)

	if chart.ChartYAML == "" {
		t.Error("expected non-empty Chart.yaml")
	}
	if !strings.Contains(chart.ChartYAML, "name: myagent") {
		t.Error("expected chart name in Chart.yaml")
	}

	if chart.ValuesYAML == "" {
		t.Error("expected non-empty values.yaml")
	}
	if !strings.Contains(chart.ValuesYAML, "repository: myagent") {
		t.Error("expected image repository in values.yaml")
	}
	if !strings.Contains(chart.ValuesYAML, "tag: \"1.0.0\"") {
		t.Error("expected image tag in values.yaml")
	}
	if !strings.Contains(chart.ValuesYAML, "/healthz") {
		t.Error("expected health check path in values.yaml")
	}

	if chart.DeploymentYAML == "" {
		t.Error("expected non-empty deployment template")
	}
	if chart.ServiceYAML == "" {
		t.Error("expected non-empty service template")
	}

	// Write and verify
	outputDir := t.TempDir()
	chartDir := filepath.Join(outputDir, "chart")
	if err := kubernetes.WriteHelmChart(chart, chartDir); err != nil {
		t.Fatalf("write helm chart error: %v", err)
	}

	expectedFiles := []string{
		"Chart.yaml",
		"values.yaml",
		"templates/deployment.yaml",
		"templates/service.yaml",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(chartDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q to exist", f)
		}
	}
}

// TestDockerCompose verifies Docker Compose generation for multi-agent pipelines.
func TestDockerCompose(t *testing.T) {
	agents := []compose.AgentBinary{
		{Name: "researcher", Image: "researcher:1.0", HostPort: 8080, ContainerPort: 8080},
		{Name: "summarizer", Image: "summarizer:1.0", HostPort: 8081, ContainerPort: 8080},
	}

	content := compose.GenerateComposeFromBinaries(agents, "agent-net")

	if !strings.Contains(content, "researcher:") {
		t.Error("expected researcher service")
	}
	if !strings.Contains(content, "summarizer:") {
		t.Error("expected summarizer service")
	}
	if !strings.Contains(content, "agent-net") {
		t.Error("expected agent-net network")
	}
	if !strings.Contains(content, "healthcheck:") {
		t.Error("expected healthcheck")
	}

	// Write to disk
	outputDir := t.TempDir()
	if err := compose.GenerateComposeFile(agents, outputDir, "agent-net"); err != nil {
		t.Fatalf("generate compose file error: %v", err)
	}

	composePath := filepath.Join(outputDir, "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Error("expected docker-compose.yml to exist")
	}

	data, _ := os.ReadFile(composePath)
	if !strings.Contains(string(data), "8081:8080") {
		t.Error("expected port mapping 8081:8080 for summarizer")
	}
}

// TestDockerComposeNoNetwork verifies Compose generation without custom network.
func TestDockerComposeNoNetwork(t *testing.T) {
	agents := []compose.AgentBinary{
		{Name: "single-agent", Image: "agent:latest", HostPort: 8080, ContainerPort: 8080, NoAuth: true},
	}

	content := compose.GenerateComposeFromBinaries(agents, "")

	if strings.Contains(content, "networks:") {
		t.Error("should not contain networks section when no network name given")
	}
	if !strings.Contains(content, "AGENTSPEC_NO_AUTH=true") {
		t.Error("expected AGENTSPEC_NO_AUTH=true for noAuth agent")
	}
}
