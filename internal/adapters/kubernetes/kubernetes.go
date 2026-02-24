package kubernetes

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/ir"
)

func init() {
	adapters.Register("kubernetes", func() adapters.Adapter {
		return &Adapter{}
	})
}

// Adapter implements the Kubernetes adapter.
type Adapter struct {
	namespace string
	config    map[string]interface{}
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return "kubernetes" }

// SetConfig sets the deploy target configuration.
func (a *Adapter) SetConfig(config map[string]interface{}) {
	a.config = config
	a.namespace = stringFromConfig(config, "namespace", "default")
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
	var results []adapters.Result
	var resources []ir.Resource
	for _, action := range actions {
		if action.Resource != nil {
			resources = append(resources, *action.Resource)
		}
	}

	manifests := GenerateManifests(resources, a.config)

	// Apply each manifest using kubectl
	applyManifest := func(name string, manifest map[string]interface{}) error {
		if manifest == nil {
			return nil
		}
		cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-", "--server-side")
		data, err := marshalJSON(manifest)
		if err != nil {
			return err
		}
		cmd.Stdin = strings.NewReader(string(data))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("kubectl apply %s: %s: %w", name, string(out), err)
		}
		return nil
	}

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
		if err := applyManifest(item.name, item.manifest); err != nil {
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
			Artifact: fmt.Sprintf("namespace=%s", a.namespace),
		})
	}
	return results, nil
}

// Export generates Kubernetes manifests in the output directory.
func (a *Adapter) Export(_ context.Context, resources []ir.Resource, outDir string) error {
	manifests := GenerateManifests(resources, a.config)
	return WriteManifests(manifests, outDir)
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
