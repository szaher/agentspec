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
	Binary  string            `json:"binary"`
	Args    []string          `json:"args,omitempty"`
	Timeout time.Duration     `json:"timeout,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
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
func (e *CommandExecutor) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	if e.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.config.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, e.config.Binary, e.config.Args...)

	// Set environment
	for k, v := range e.config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range e.secrets {
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
