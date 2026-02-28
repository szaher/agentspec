package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/ir"
)

func init() {
	adapters.Register("docker", func() adapters.Adapter {
		return &Adapter{}
	})
}

// Adapter implements the Docker adapter.
type Adapter struct {
	containerID string
	imageName   string
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return "docker" }

// Validate checks whether resources can be deployed as Docker containers.
func (a *Adapter) Validate(_ context.Context, resources []ir.Resource) error {
	hasAgent := false
	for _, r := range resources {
		if r.Kind == "Agent" {
			hasAgent = true
			break
		}
	}
	if !hasAgent {
		return fmt.Errorf("docker adapter requires at least one agent")
	}
	return nil
}

// Apply builds and starts a Docker container for the agent runtime.
func (a *Adapter) Apply(ctx context.Context, actions []adapters.Action) ([]adapters.Result, error) {
	var results []adapters.Result

	// Collect resources from actions
	var resources []ir.Resource
	for _, action := range actions {
		if action.Resource != nil {
			resources = append(resources, *action.Resource)
		}
	}

	port := agentPort(resources)
	a.imageName = "agentspec-runtime"

	// Write runtime config to the build context (current directory)
	// so Docker can COPY it into the image.
	configData, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile("runtime-config.json", configData, 0644); err != nil {
		return nil, fmt.Errorf("write config: %w", err)
	}
	defer os.Remove("runtime-config.json")

	tmpDir, err := os.MkdirTemp("", "agentspec-docker-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	dockerfile := GenerateDockerfile(resources, port)
	if err := os.WriteFile(tmpDir+"/Dockerfile", []byte(dockerfile), 0644); err != nil {
		return nil, fmt.Errorf("write Dockerfile: %w", err)
	}

	// Build image
	buildCmd := exec.CommandContext(ctx, "docker", "build", "-t", a.imageName, "-f", tmpDir+"/Dockerfile", ".")
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		for _, action := range actions {
			results = append(results, adapters.Result{
				FQN:    action.FQN,
				Action: action.Type,
				Status: adapters.ResultFailed,
				Error:  fmt.Sprintf("docker build failed: %s: %v", string(buildOut), err),
			})
		}
		return results, nil
	}

	// Run container
	runCmd := exec.CommandContext(ctx, "docker", "run", "-d",
		"-p", fmt.Sprintf("%d:%d", port, port),
		"--name", "agentspec-runtime",
		a.imageName,
	)
	out, err := runCmd.Output()
	if err != nil {
		for _, action := range actions {
			results = append(results, adapters.Result{
				FQN:    action.FQN,
				Action: action.Type,
				Status: adapters.ResultFailed,
				Error:  fmt.Sprintf("docker run failed: %v", err),
			})
		}
		return results, nil
	}

	a.containerID = strings.TrimSpace(string(out))

	for _, action := range actions {
		results = append(results, adapters.Result{
			FQN:      action.FQN,
			Action:   action.Type,
			Status:   adapters.ResultSuccess,
			Artifact: fmt.Sprintf("container=%s port=%d", a.containerID[:12], port),
		})
	}
	return results, nil
}

// Export generates a Dockerfile and runtime config in the output directory.
func (a *Adapter) Export(_ context.Context, resources []ir.Resource, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	port := agentPort(resources)
	dockerfile := GenerateDockerfile(resources, port)
	if err := os.WriteFile(outDir+"/Dockerfile", []byte(dockerfile), 0644); err != nil {
		return err
	}

	configData, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outDir+"/runtime-config.json", configData, 0644)
}

// Status returns the status of the Docker container.
func (a *Adapter) Status(ctx context.Context) ([]adapters.ResourceStatus, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a",
		"--filter", "name=agentspec-runtime",
		"--format", "{{.ID}}\t{{.Status}}\t{{.Ports}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps: %w", err)
	}

	var statuses []adapters.ResourceStatus
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 3)
		if len(parts) < 2 {
			continue
		}

		state := "stopped"
		health := "unknown"
		if strings.Contains(parts[1], "Up") {
			state = "running"
			health = "healthy"
		}

		rs := adapters.ResourceStatus{
			FQN:    "agentspec-runtime/" + parts[0],
			Name:   "agentspec-runtime",
			Kind:   "Container",
			State:  state,
			Health: health,
		}
		if len(parts) > 2 {
			rs.Endpoint = parts[2]
		}
		statuses = append(statuses, rs)
	}
	return statuses, nil
}

// Logs streams Docker container logs to the writer.
func (a *Adapter) Logs(ctx context.Context, w io.Writer, opts adapters.LogOptions) error {
	args := []string{"logs"}
	if opts.Follow {
		args = append(args, "--follow")
	}
	if opts.Tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", opts.Tail))
	}
	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}
	args = append(args, "agentspec-runtime")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

// Destroy stops and removes the Docker container and image.
func (a *Adapter) Destroy(ctx context.Context) ([]adapters.Result, error) {
	// Stop container
	stopCmd := exec.CommandContext(ctx, "docker", "stop", "agentspec-runtime")
	_ = stopCmd.Run()

	// Remove container
	rmCmd := exec.CommandContext(ctx, "docker", "rm", "agentspec-runtime")
	_ = rmCmd.Run()

	return []adapters.Result{
		{
			FQN:    "docker/agentspec-runtime",
			Action: adapters.ActionDelete,
			Status: adapters.ResultSuccess,
		},
	}, nil
}
