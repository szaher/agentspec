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

	"github.com/szaher/designs/agentz/internal/sandbox"
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
	sandbox sandbox.Sandbox
}

// NewInlineExecutor creates an inline code executor.
// If sb is non-nil and available, inline tools run in the sandbox.
func NewInlineExecutor(config InlineConfig, secrets map[string]string, sb ...sandbox.Sandbox) *InlineExecutor {
	e := &InlineExecutor{config: config, secrets: secrets}
	if len(sb) > 0 && sb[0] != nil {
		e.sandbox = sb[0]
	}
	return e
}

// Execute writes the code to a temp file and runs it as a subprocess.
// If a sandbox is configured and available, execution is sandboxed.
// Stdout/stderr are captured to buffers (never passed to host stdout/stderr).
func (e *InlineExecutor) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	// Build safe environment
	safeEnv := SafeEnv(e.secrets)
	envMap := make(map[string]string)
	for _, kv := range safeEnv {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				envMap[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	// Add tool-specific env vars
	for k, v := range e.config.Env {
		envMap[k] = v
	}

	// If sandbox is configured and available, use it
	if e.sandbox != nil && e.sandbox.Available() {
		timeoutSec := int(e.config.Timeout.Seconds())
		if timeoutSec <= 0 {
			timeoutSec = 30
		}
		sc := sandbox.ExecConfig{
			Language:   e.config.Language,
			Script:     e.config.Code,
			Env:        envMap,
			MemoryMB:   e.config.MemoryLimit,
			TimeoutSec: timeoutSec,
		}
		stdout, stderr, err := e.sandbox.Execute(ctx, sc)
		if err != nil {
			return "", fmt.Errorf("inline tool (%s): %w: %s", e.config.Language, err, stderr)
		}
		return stdout, nil
	}

	// Fallback: direct execution without sandbox
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

	// Use safe environment — do NOT inherit host env
	cmd.Env = SafeEnv(e.secrets)
	for k, v := range e.config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Pass input via stdin
	inputData, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("inline tool: marshal input: %w", err)
	}
	cmd.Stdin = bytes.NewReader(inputData)

	// Capture stdout/stderr to buffers — never pass to host
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
