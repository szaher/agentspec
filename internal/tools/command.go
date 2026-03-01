package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// CommandConfig configures a command tool executor.
type CommandConfig struct {
	Binary    string            `json:"binary"`
	Args      []string          `json:"args,omitempty"`
	Timeout   time.Duration     `json:"timeout,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Allowlist []string          `json:"allowlist,omitempty"`
}

// CommandExecutor executes tools as subprocesses.
type CommandExecutor struct {
	config  CommandConfig
	secrets map[string]string
}

// NewCommandExecutor creates a command tool executor.
func NewCommandExecutor(config CommandConfig, secrets map[string]string) *CommandExecutor {
	return &CommandExecutor{config: config, secrets: secrets}
}

// Execute runs the configured command with input passed via stdin as JSON.
// Validates the binary against the allowlist before execution.
// Uses a minimal safe environment (PATH, HOME, secrets only).
func (e *CommandExecutor) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	// Validate binary against allowlist
	if err := ValidateBinary(e.config.Binary, e.config.Allowlist); err != nil {
		return "", fmt.Errorf("command tool: %w", err)
	}

	if e.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.config.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, e.config.Binary, e.config.Args...)

	// Use safe environment — do NOT inherit host env
	cmd.Env = SafeEnv(e.secrets)
	// Add tool-specific env vars
	for k, v := range e.config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Pass input as JSON on stdin
	inputData, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("command tool: marshal input: %w", err)
	}
	cmd.Stdin = bytes.NewReader(inputData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("command tool %s: %w: %s", e.config.Binary, err, stderr.String())
	}

	return stdout.String(), nil
}
