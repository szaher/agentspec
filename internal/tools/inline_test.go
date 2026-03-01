package tools

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestInterpreterForLanguage_ViaExecute(t *testing.T) {
	// interpreterForLanguage is unexported, so we test it indirectly through
	// Execute behavior. For known languages, if the interpreter is available,
	// execution should succeed (or fail with "not found" if not installed).
	// For unknown languages, we get "unsupported language".
	tests := []struct {
		name        string
		language    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "bash",
			language: "bash",
			wantErr:  false,
		},
		{
			name:        "unknown language",
			language:    "cobol",
			wantErr:     true,
			errContains: "unsupported language",
		},
		{
			name:        "empty language",
			language:    "",
			wantErr:     true,
			errContains: "unsupported language",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.language == "bash" {
				// Skip if bash is not available on the system
				if _, err := exec.LookPath("bash"); err != nil {
					t.Skip("bash not available on this system")
				}
			}

			config := InlineConfig{
				Language: tc.language,
				Code:     "echo test",
				Timeout:  5 * time.Second,
			}
			executor := NewInlineExecutor(config, nil)
			_, err := executor.Execute(context.Background(), nil)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("expected error containing %q, got: %v", tc.errContains, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewInlineExecutor(t *testing.T) {
	config := InlineConfig{
		Language:    "python",
		Code:        "print('hello')",
		Timeout:     10 * time.Second,
		MemoryLimit: 128,
	}
	secrets := map[string]string{"API_KEY": "test123"}

	executor := NewInlineExecutor(config, secrets)
	if executor == nil {
		t.Fatal("NewInlineExecutor returned nil")
	}
	if executor.config.Language != "python" {
		t.Fatalf("expected language %q, got %q", "python", executor.config.Language)
	}
	if executor.config.Code != "print('hello')" {
		t.Fatalf("expected code %q, got %q", "print('hello')", executor.config.Code)
	}
	if executor.sandbox != nil {
		t.Fatal("expected nil sandbox when no sandbox argument provided")
	}
}

func TestInlineExecutor_BashExecution(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available on this system")
	}

	config := InlineConfig{
		Language: "bash",
		Code:     "echo hello",
		Timeout:  5 * time.Second,
	}
	executor := NewInlineExecutor(config, nil)

	output, err := executor.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "hello\n" {
		t.Fatalf("expected %q, got %q", "hello\n", output)
	}
}

func TestInlineExecutor_UnsupportedLanguage(t *testing.T) {
	config := InlineConfig{
		Language: "fortran",
		Code:     "PRINT *, 'Hello'",
		Timeout:  5 * time.Second,
	}
	executor := NewInlineExecutor(config, nil)

	_, err := executor.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for unsupported language, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported language") {
		t.Fatalf("expected error containing 'unsupported language', got: %v", err)
	}
}

func TestInlineExecutor_Timeout(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available on this system")
	}

	config := InlineConfig{
		Language: "bash",
		Code:     "sleep 10",
		Timeout:  1 * time.Millisecond,
	}
	executor := NewInlineExecutor(config, nil)

	_, err := executor.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "deadline") && !strings.Contains(errStr, "killed") && !strings.Contains(errStr, "signal") {
		t.Fatalf("expected deadline/killed error, got: %v", err)
	}
}

func TestInlineExecutor_WithEnvVars(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available on this system")
	}

	config := InlineConfig{
		Language: "bash",
		Code:     "echo $MY_VAR",
		Timeout:  5 * time.Second,
		Env: map[string]string{
			"MY_VAR": "custom_value",
		},
	}
	executor := NewInlineExecutor(config, nil)

	output, err := executor.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(output) != "custom_value" {
		t.Fatalf("expected %q, got %q", "custom_value", strings.TrimSpace(output))
	}
}
