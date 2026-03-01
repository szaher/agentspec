package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCommandExecutor_SuccessfulExecution(t *testing.T) {
	executor := NewCommandExecutor(CommandConfig{
		Binary:    "echo",
		Args:      []string{"-n", "hello"},
		Allowlist: []string{"echo"},
	}, nil)

	output, err := executor.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected successful execution, got error: %v", err)
	}
	if output != "hello" {
		t.Fatalf("expected output %q, got %q", "hello", output)
	}
}

func TestCommandExecutor_CommandFailure(t *testing.T) {
	// "false" is a standard Unix command that always exits with status 1
	executor := NewCommandExecutor(CommandConfig{
		Binary:    "false",
		Allowlist: []string{"false"},
	}, nil)

	_, err := executor.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for non-zero exit, got nil")
	}
}

func TestCommandExecutor_BinaryNotInAllowlist(t *testing.T) {
	executor := NewCommandExecutor(CommandConfig{
		Binary:    "echo",
		Allowlist: []string{"cat", "ls"},
	}, nil)

	_, err := executor.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for binary not in allowlist, got nil")
	}
	if !strings.Contains(err.Error(), "not in allowlist") {
		t.Fatalf("expected error containing 'not in allowlist', got: %v", err)
	}
}

func TestCommandExecutor_Timeout(t *testing.T) {
	executor := NewCommandExecutor(CommandConfig{
		Binary:    "sleep",
		Args:      []string{"10"},
		Timeout:   1 * time.Millisecond,
		Allowlist: []string{"sleep"},
	}, nil)

	_, err := executor.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	// The error should indicate the context deadline was exceeded or signal killed
	errStr := err.Error()
	if !strings.Contains(errStr, "deadline") && !strings.Contains(errStr, "killed") && !strings.Contains(errStr, "signal") {
		t.Fatalf("expected deadline/killed error, got: %v", err)
	}
}

func TestCommandExecutor_InputPassedAsJSON(t *testing.T) {
	// Use "cat" to echo back stdin, verifying JSON is passed
	executor := NewCommandExecutor(CommandConfig{
		Binary:    "cat",
		Allowlist: []string{"cat"},
	}, nil)

	input := map[string]interface{}{
		"key":    "value",
		"number": float64(42),
	}

	output, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("expected successful execution, got error: %v", err)
	}

	// Verify that the output is valid JSON matching our input
	if !strings.Contains(output, `"key"`) || !strings.Contains(output, `"value"`) {
		t.Fatalf("expected JSON output containing input data, got: %s", output)
	}
}

func TestCommandExecutor_WithSecrets(t *testing.T) {
	// Use env to print the environment; verify secrets are present
	executor := NewCommandExecutor(CommandConfig{
		Binary:    "env",
		Allowlist: []string{"env"},
	}, map[string]string{
		"MY_SECRET": "super_secret_value",
	})

	output, err := executor.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected successful execution, got error: %v", err)
	}

	if !strings.Contains(output, "MY_SECRET=super_secret_value") {
		t.Fatalf("expected output to contain secret, got: %s", output)
	}
}

func TestCommandExecutor_EmptyAllowlist(t *testing.T) {
	executor := NewCommandExecutor(CommandConfig{
		Binary: "echo",
		// No allowlist configured — should block
	}, nil)

	_, err := executor.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty allowlist, got nil")
	}
	if !strings.Contains(err.Error(), "no allowlist configured") {
		t.Fatalf("expected error about no allowlist configured, got: %v", err)
	}
}
