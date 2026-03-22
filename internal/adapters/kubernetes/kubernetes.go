package kubernetes

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/szaher/agentspec/internal/adapters"
	"github.com/szaher/agentspec/internal/ir"
	"github.com/szaher/agentspec/internal/k8s/converter"

	"sigs.k8s.io/yaml"
)

func init() {
	adapters.Register("kubernetes", func() adapters.Adapter {
		return &Adapter{}
	})
}

// DeployMode controls whether the adapter uses raw manifests or the AgentSpec operator CRDs.
type DeployMode string

const (
	// DeployModeDirect generates raw Deployments, Services, ConfigMaps (no operator needed).
	DeployModeDirect DeployMode = "direct"

	// DeployModeOperator generates AgentSpec CRDs (Agent, ToolBinding, Workflow, etc.) and
	// delegates workload management to the operator controller.
	DeployModeOperator DeployMode = "operator"

	// DeployModeAuto detects whether the operator CRDs are installed and picks the right mode.
	DeployModeAuto DeployMode = "auto"
)

// Adapter implements the Kubernetes adapter.
type Adapter struct {
	namespace string
	config    map[string]interface{}
	mode      DeployMode
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return "kubernetes" }

// SetConfig sets the deploy target configuration.
func (a *Adapter) SetConfig(config map[string]interface{}) {
	a.config = config
	a.namespace = stringFromConfig(config, "namespace", "default")

	// Determine deploy mode from config (default: auto).
	switch stringFromConfig(config, "mode", "auto") {
	case "direct":
		a.mode = DeployModeDirect
	case "operator":
		a.mode = DeployModeOperator
	default:
		a.mode = DeployModeAuto
	}
}

// Validate checks whether resources can be deployed to Kubernetes.
func (a *Adapter) Validate(_ context.Context, resources []ir.Resource) error {
	hasAgent := false
	for _, r := range resources {
		if r.Kind == "Agent" {
			hasAgent = true
			break
		}
	}
	if !hasAgent {
		return fmt.Errorf("kubernetes adapter requires at least one agent")
	}
	return nil
}

// Apply generates manifests and applies them to the cluster using kubectl.
func (a *Adapter) Apply(ctx context.Context, actions []adapters.Action) ([]adapters.Result, error) {
	var resources []ir.Resource
	for _, action := range actions {
		if action.Resource != nil {
			resources = append(resources, *action.Resource)
		}
	}

	mode := a.resolveMode(ctx)
	if mode == DeployModeOperator {
		return a.applyOperatorCRDs(ctx, resources, actions)
	}
	return a.applyDirect(ctx, resources, actions)
}

// applyDirect applies raw Deployments/Services/ConfigMaps (no operator).
func (a *Adapter) applyDirect(ctx context.Context, resources []ir.Resource, actions []adapters.Action) ([]adapters.Result, error) {
	var results []adapters.Result

	manifests := GenerateManifests(resources, a.config)

	applyOrder := []struct {
		name     string
		manifest map[string]interface{}
	}{
		{"namespace", manifests.Namespace},
		{"configmap", manifests.ConfigMap},
		{"deployment", manifests.Deployment},
		{"service", manifests.Service},
		{"hpa", manifests.HPA},
	}

	for _, item := range applyOrder {
		if item.manifest == nil {
			continue
		}
		if err := a.kubectlApply(ctx, item.name, item.manifest); err != nil {
			for _, action := range actions {
				results = append(results, adapters.Result{
					FQN:    action.FQN,
					Action: action.Type,
					Status: adapters.ResultFailed,
					Error:  err.Error(),
				})
			}
			return results, nil
		}
	}

	for _, action := range actions {
		results = append(results, adapters.Result{
			FQN:      action.FQN,
			Action:   action.Type,
			Status:   adapters.ResultSuccess,
			Artifact: fmt.Sprintf("namespace=%s,mode=direct", a.namespace),
		})
	}
	return results, nil
}

// applyOperatorCRDs converts IR resources to AgentSpec CRDs and applies them.
func (a *Adapter) applyOperatorCRDs(ctx context.Context, resources []ir.Resource, actions []adapters.Action) ([]adapters.Result, error) {
	var results []adapters.Result

	// Build an IR Document from the resources.
	doc := &ir.Document{Resources: resources}
	crds, err := converter.ConvertDocument(doc, a.namespace)
	if err != nil {
		for _, action := range actions {
			results = append(results, adapters.Result{
				FQN:    action.FQN,
				Action: action.Type,
				Status: adapters.ResultFailed,
				Error:  fmt.Sprintf("CRD conversion: %v", err),
			})
		}
		return results, nil
	}

	for _, crd := range crds {
		yamlData, err := yaml.Marshal(crd.Raw)
		if err != nil {
			for _, action := range actions {
				results = append(results, adapters.Result{
					FQN:    action.FQN,
					Action: action.Type,
					Status: adapters.ResultFailed,
					Error:  fmt.Sprintf("marshal %s/%s: %v", crd.Kind, crd.Name, err),
				})
			}
			return results, nil
		}
		if err := a.kubectlApplyRaw(ctx, crd.Kind+"/"+crd.Name, yamlData); err != nil {
			for _, action := range actions {
				results = append(results, adapters.Result{
					FQN:    action.FQN,
					Action: action.Type,
					Status: adapters.ResultFailed,
					Error:  err.Error(),
				})
			}
			return results, nil
		}
	}

	for _, action := range actions {
		results = append(results, adapters.Result{
			FQN:      action.FQN,
			Action:   action.Type,
			Status:   adapters.ResultSuccess,
			Artifact: fmt.Sprintf("namespace=%s,mode=operator,crds=%d", a.namespace, len(crds)),
		})
	}
	return results, nil
}

// resolveMode determines whether to use operator or direct mode.
func (a *Adapter) resolveMode(ctx context.Context) DeployMode {
	if a.mode != DeployModeAuto {
		return a.mode
	}
	// Auto-detect: check if the agents.agentspec.io CRD exists.
	cmd := exec.CommandContext(ctx, "kubectl", "api-resources", "--api-group=agentspec.io", "-o", "name")
	out, err := cmd.Output()
	if err == nil && strings.Contains(string(out), "agents.agentspec.io") {
		return DeployModeOperator
	}
	return DeployModeDirect
}

// kubectlApply applies a JSON manifest via kubectl.
func (a *Adapter) kubectlApply(ctx context.Context, name string, manifest map[string]interface{}) error {
	data, err := marshalJSON(manifest)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-", "--server-side")
	cmd.Stdin = strings.NewReader(string(data))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply %s: %s: %w", name, string(out), err)
	}
	return nil
}

// kubectlApplyRaw applies raw YAML data via kubectl.
func (a *Adapter) kubectlApplyRaw(ctx context.Context, name string, data []byte) error {
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-", "--server-side")
	cmd.Stdin = strings.NewReader(string(data))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply %s: %s: %w", name, string(out), err)
	}
	return nil
}

// Export generates Kubernetes manifests in the output directory.
func (a *Adapter) Export(ctx context.Context, resources []ir.Resource, outDir string) error {
	mode := a.resolveMode(ctx)
	if mode == DeployModeOperator {
		return a.exportOperatorCRDs(resources, outDir)
	}
	manifests := GenerateManifests(resources, a.config)
	return WriteManifests(manifests, outDir)
}

// exportOperatorCRDs writes AgentSpec CRD manifests to the output directory.
func (a *Adapter) exportOperatorCRDs(resources []ir.Resource, outDir string) error {
	doc := &ir.Document{Resources: resources}
	crds, err := converter.ConvertDocument(doc, a.namespace)
	if err != nil {
		return fmt.Errorf("CRD conversion: %w", err)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	for _, crd := range crds {
		yamlData, err := yaml.Marshal(crd.Raw)
		if err != nil {
			return fmt.Errorf("marshal %s/%s: %w", crd.Kind, crd.Name, err)
		}
		filename := fmt.Sprintf("%s_%s.yaml", crd.Kind, crd.Name)
		if err := os.WriteFile(filepath.Join(outDir, filename), yamlData, 0644); err != nil {
			return err
		}
	}
	return nil
}

// Status returns the status of Kubernetes deployments.
func (a *Adapter) Status(ctx context.Context) ([]adapters.ResourceStatus, error) {
	args := []string{"get", "deployments", "-l", "app.kubernetes.io/managed-by=agentspec",
		"-o", "custom-columns=NAME:.metadata.name,READY:.status.readyReplicas,DESIRED:.spec.replicas,AVAILABLE:.status.availableReplicas",
		"--no-headers"}
	if a.namespace != "" {
		args = append(args, "-n", a.namespace)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl get deployments: %w", err)
	}

	var statuses []adapters.ResourceStatus
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		state := "pending"
		health := "unknown"
		if fields[1] == fields[2] && fields[1] != "<none>" {
			state = "running"
			health = "healthy"
		} else if fields[1] != "<none>" {
			state = "degraded"
			health = "unhealthy"
		}

		rs := adapters.ResourceStatus{
			FQN:      fmt.Sprintf("%s/Deployment/%s", a.namespace, fields[0]),
			Name:     fields[0],
			Kind:     "Deployment",
			State:    state,
			Health:   health,
			Replicas: fmt.Sprintf("%s/%s", fields[1], fields[2]),
		}
		statuses = append(statuses, rs)
	}
	return statuses, nil
}

// Logs streams Kubernetes pod logs to the writer.
func (a *Adapter) Logs(ctx context.Context, w io.Writer, opts adapters.LogOptions) error {
	args := []string{"logs", "-l", "app.kubernetes.io/name=agentspec-runtime"}
	if a.namespace != "" {
		args = append(args, "-n", a.namespace)
	}
	if opts.Follow {
		args = append(args, "--follow")
	}
	if opts.Tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", opts.Tail))
	}
	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

// Destroy removes all AgentSpec-managed Kubernetes resources.
func (a *Adapter) Destroy(ctx context.Context) ([]adapters.Result, error) {
	args := []string{"delete", "all", "-l", "app.kubernetes.io/managed-by=agentspec"}
	if a.namespace != "" {
		args = append(args, "-n", a.namespace)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("kubectl delete: %s: %w", string(out), err)
	}

	return []adapters.Result{
		{
			FQN:    fmt.Sprintf("kubernetes/%s", a.namespace),
			Action: adapters.ActionDelete,
			Status: adapters.ResultSuccess,
		},
	}, nil
}

func marshalJSON(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
