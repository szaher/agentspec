package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// InlineConfig configures an inline code executor.
type InlineConfig struct {
	Language    string            `json:"language"`
	Code        string            `json:"code"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
	MemoryLimit int               `json:"memory_limit,omitempty"` // in MB
	Env         map[string]string `json:"env,omitempty"`
}

// InlineExecutor runs embedded code in a sandboxed subprocess.
type InlineExecutor struct {
	config  InlineConfig
	secrets map[string]string
}

// NewInlineExecutor creates an inline code executor.
func NewInlineExecutor(config InlineConfig, secrets map[string]string) *InlineExecutor {
	return &InlineExecutor{config: config, secrets: secrets}
}

// Execute writes the code to a temp file and runs it as a subprocess.
func (e *InlineExecutor) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	timeout := e.config.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	// Determine interpreter and file extension
	interpreter, ext, err := interpreterForLanguage(e.config.Language)
	if err != nil {
		return "", err
	}

	// Write code to temp file
	tmpDir, err := os.MkdirTemp("", "agentspec-inline-*")
	if err != nil {
		return "", fmt.Errorf("inline tool: create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	scriptPath := filepath.Join(tmpDir, "script"+ext)
	if err := os.WriteFile(scriptPath, []byte(e.config.Code), 0600); err != nil {
		return "", fmt.Errorf("inline tool: write script: %w", err)
	}

	cmd := exec.CommandContext(ctx, interpreter, scriptPath)
	cmd.Dir = tmpDir

	// Build environment
	for k, v := range e.config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range e.secrets {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Pass input via stdin
	inputData, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("inline tool: marshal input: %w", err)
	}
	cmd.Stdin = bytes.NewReader(inputData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("inline tool (%s): %w: %s", e.config.Language, err, stderr.String())
	}

	return stdout.String(), nil
}

func interpreterForLanguage(lang string) (interpreter, ext string, err error) {
	switch lang {
	case "python", "python3":
		return "python3", ".py", nil
	case "javascript", "node":
		return "node", ".js", nil
	case "bash", "sh":
		return "bash", ".sh", nil
	case "ruby":
		return "ruby", ".rb", nil
	default:
		return "", "", fmt.Errorf("inline tool: unsupported language %q", lang)
	}
}
