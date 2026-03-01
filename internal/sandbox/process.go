package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// ProcessSandbox implements Sandbox using OS-level process isolation.
type ProcessSandbox struct{}

// Available reports whether OS-level sandboxing is supported on the current platform.
func (p *ProcessSandbox) Available() bool {
	// Supported on Linux and macOS — requires bash and ulimit
	return runtime.GOOS == "linux" || runtime.GOOS == "darwin"
}

// Execute runs a script in a sandboxed environment using OS-level process isolation.
// Uses ulimit for memory, context timeout + process kill for CPU, tmpdir for filesystem.
func (p *ProcessSandbox) Execute(ctx context.Context, config ExecConfig) (string, string, error) {
	if !p.Available() {
		return "", "", fmt.Errorf("process sandbox not available on %s", runtime.GOOS)
	}

	interpreter, ext, err := interpreterForLanguage(config.Language)
	if err != nil {
		return "", "", err
	}

	// Verify interpreter exists
	if _, err := exec.LookPath(interpreter); err != nil {
		return "", "", fmt.Errorf("interpreter not found: %s", config.Language)
	}

	// Create isolated temp directory for sandbox
	sandboxDir, err := os.MkdirTemp("", "agentspec-sandbox-*")
	if err != nil {
		return "", "", fmt.Errorf("create sandbox dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(sandboxDir) }()

	// Write script to sandbox directory
	scriptPath := filepath.Join(sandboxDir, "script"+ext)
	if err := os.WriteFile(scriptPath, []byte(config.Script), 0600); err != nil {
		return "", "", fmt.Errorf("write script: %w", err)
	}

	// Build the sandboxed command using bash wrapper with ulimit
	timeoutSec := config.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	// Apply context timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Build wrapper script with ulimit restrictions
	wrapperScript := p.buildWrapperScript(interpreter, scriptPath, config)
	wrapperPath := filepath.Join(sandboxDir, "wrapper.sh")
	if err := os.WriteFile(wrapperPath, []byte(wrapperScript), 0700); err != nil {
		return "", "", fmt.Errorf("write wrapper: %w", err)
	}

	cmd := exec.CommandContext(execCtx, "bash", wrapperPath)
	cmd.Dir = sandboxDir

	// Set minimal safe environment — do NOT inherit host environment
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		fmt.Sprintf("HOME=%s", sandboxDir),
		fmt.Sprintf("TMPDIR=%s", sandboxDir),
	}
	for k, v := range config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// Check if context timed out
		if execCtx.Err() == context.DeadlineExceeded {
			return stdout.String(), stderr.String(), &ErrResourceLimit{
				Resource: "time",
				Limit:    fmt.Sprintf("%ds", timeoutSec),
			}
		}
		// Check for memory limit errors (exit code 137 = OOM kill)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 137 {
				return stdout.String(), stderr.String(), &ErrResourceLimit{
					Resource: "memory",
					Limit:    fmt.Sprintf("%dMB", config.MemoryMB),
				}
			}
		}
		return stdout.String(), stderr.String(), err
	}

	return stdout.String(), stderr.String(), nil
}

func (p *ProcessSandbox) buildWrapperScript(interpreter, scriptPath string, config ExecConfig) string {
	var script bytes.Buffer
	script.WriteString("#!/bin/bash\nset -e\n")

	// Memory limit via ulimit (in KB)
	if config.MemoryMB > 0 {
		memKB := config.MemoryMB * 1024
		fmt.Fprintf(&script, "ulimit -v %d 2>/dev/null || true\n", memKB)
	}

	// File size limit (16MB)
	script.WriteString("ulimit -f 16384 2>/dev/null || true\n")

	// Max processes limit
	script.WriteString("ulimit -u 64 2>/dev/null || true\n")

	// Restrict working directory
	fmt.Fprintf(&script, "cd %s\n", strconv.Quote(filepath.Dir(scriptPath)))

	// Execute the script
	fmt.Fprintf(&script, "exec %s %s\n", interpreter, strconv.Quote(scriptPath))

	return script.String()
}
