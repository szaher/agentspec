package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// NoopSandbox provides no isolation. Used for testing or when no backend is available.
type NoopSandbox struct{}

// Available always returns true — the noop sandbox works on any platform.
func (n *NoopSandbox) Available() bool { return true }

// Execute runs the script without isolation.
func (n *NoopSandbox) Execute(ctx context.Context, config ExecConfig) (string, string, error) {
	interpreter, ext, err := interpreterForLanguage(config.Language)
	if err != nil {
		return "", "", err
	}

	// Create temp directory for script
	tmpDir, err := os.MkdirTemp("", "agentspec-noop-*")
	if err != nil {
		return "", "", fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	scriptPath := filepath.Join(tmpDir, "script"+ext)
	if err := os.WriteFile(scriptPath, []byte(config.Script), 0600); err != nil {
		return "", "", fmt.Errorf("write script: %w", err)
	}

	cmd := exec.CommandContext(ctx, interpreter, scriptPath)
	cmd.Dir = tmpDir

	// Build environment
	for k, v := range config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), stderr.String(), err
	}

	return stdout.String(), stderr.String(), nil
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
		return "", "", fmt.Errorf("unsupported language %q", lang)
	}
}
