// Package process implements the local process adapter for the AgentSpec toolchain.
package process

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/state"
)

func init() {
	adapters.Register("process", func() adapters.Adapter {
		return &Adapter{}
	})
}

// Adapter implements the local process adapter.
type Adapter struct {
	stateBackend state.Backend
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return "process" }

// SetStateBackend sets the state backend for PID/port tracking.
func (a *Adapter) SetStateBackend(b state.Backend) {
	a.stateBackend = b
}

// Validate checks whether resources can be deployed as a local process.
func (a *Adapter) Validate(_ context.Context, resources []ir.Resource) error {
	hasAgent := false
	for _, r := range resources {
		if r.Kind == "Agent" {
			hasAgent = true
			break
		}
	}
	if !hasAgent {
		return fmt.Errorf("process adapter requires at least one agent")
	}
	return nil
}

// Apply starts the runtime process, waits for health check, and records state.
func (a *Adapter) Apply(ctx context.Context, actions []adapters.Action) ([]adapters.Result, error) {
	var results []adapters.Result

	// Check if runtime is already running
	if a.stateBackend != nil {
		entries, _ := a.stateBackend.Load()
		for _, e := range entries {
			if e.Adapter == "process" && e.Status == state.StatusApplied {
				// Check if process is still alive
				if e.Error != "" {
					pid, _ := strconv.Atoi(e.Error) // PID stored in Error field temporarily
					if pid > 0 {
						proc, err := os.FindProcess(pid)
						if err == nil && proc.Signal(nil) == nil {
							// Process still running, skip restart
							for _, action := range actions {
								results = append(results, adapters.Result{
									FQN:    action.FQN,
									Action: adapters.ActionNoop,
									Status: adapters.ResultSuccess,
								})
							}
							return results, nil
						}
					}
				}
			}
		}
	}

	// Find the binary path
	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = "agentspec"
	}

	// Start runtime as a subprocess
	port := 8080
	cmd := exec.CommandContext(ctx, binaryPath, "serve", "--port", strconv.Itoa(port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
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

	// Wait for health check
	if err := WaitForHealth(ctx, fmt.Sprintf("http://localhost:%d", port), 30*time.Second); err != nil {
		_ = cmd.Process.Kill()
		for _, action := range actions {
			results = append(results, adapters.Result{
				FQN:    action.FQN,
				Action: action.Type,
				Status: adapters.ResultFailed,
				Error:  fmt.Sprintf("health check failed: %v", err),
			})
		}
		return results, nil
	}

	// Record success
	for _, action := range actions {
		results = append(results, adapters.Result{
			FQN:      action.FQN,
			Action:   action.Type,
			Status:   adapters.ResultSuccess,
			Artifact: fmt.Sprintf("pid=%d port=%d", cmd.Process.Pid, port),
		})
	}

	// Save PID to state
	if a.stateBackend != nil {
		entries, _ := a.stateBackend.Load()
		entries = append(entries, state.Entry{
			FQN:         "runtime/process",
			Hash:        fmt.Sprintf("pid:%d", cmd.Process.Pid),
			Status:      state.StatusApplied,
			LastApplied: time.Now(),
			Adapter:     "process",
			Error:       strconv.Itoa(cmd.Process.Pid),
		})
		_ = a.stateBackend.Save(entries)
	}

	return results, nil
}

// Export writes runtime configuration to the output directory.
func (a *Adapter) Export(_ context.Context, resources []ir.Resource, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outDir+"/runtime-config.json", data, 0644)
}
