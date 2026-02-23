package integration_tests

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/secrets"
)

func TestEnvResolverSuccess(t *testing.T) {
	t.Setenv("TEST_SECRET", "super-secret-value")

	resolver := secrets.NewEnvResolver()
	val, err := resolver.Resolve(context.Background(), "env(TEST_SECRET)")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if val != "super-secret-value" {
		t.Errorf("expected 'super-secret-value', got %q", val)
	}
}

func TestEnvResolverMissing(t *testing.T) {
	resolver := secrets.NewEnvResolver()
	_, err := resolver.Resolve(context.Background(), "env(NONEXISTENT_VAR_XYZ)")
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
	if !strings.Contains(err.Error(), "not set") {
		t.Errorf("expected 'not set' in error, got %v", err)
	}
}

func TestEnvResolverBadFormat(t *testing.T) {
	resolver := secrets.NewEnvResolver()
	_, err := resolver.Resolve(context.Background(), "vault(secret/key)")
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' in error, got %v", err)
	}
}

func TestRedactFilterScrubsMessages(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	filter := secrets.NewRedactFilter(inner)

	filter.AddSecret("my-api-key-12345")
	filter.AddSecret("another-secret")

	logger := slog.New(filter)
	logger.Info("Connecting with key my-api-key-12345 to service")
	logger.Info("Using token", "api_key", "my-api-key-12345")
	logger.Info("Secret: another-secret in use")

	output := buf.String()

	if strings.Contains(output, "my-api-key-12345") {
		t.Errorf("secret 'my-api-key-12345' was not redacted from logs:\n%s", output)
	}
	if strings.Contains(output, "another-secret") {
		t.Errorf("secret 'another-secret' was not redacted from logs:\n%s", output)
	}
	if !strings.Contains(output, "***REDACTED***") {
		t.Errorf("expected ***REDACTED*** in output:\n%s", output)
	}
}

func TestRedactFilterNoSecretsPassthrough(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	filter := secrets.NewRedactFilter(inner)

	logger := slog.New(filter)
	logger.Info("Normal log message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "Normal log message") {
		t.Errorf("expected passthrough for non-secret messages:\n%s", output)
	}
}

func TestRedactString(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, nil)
	filter := secrets.NewRedactFilter(inner)

	filter.AddSecret("sk-12345")

	result := filter.RedactString("Authorization: Bearer sk-12345")
	if strings.Contains(result, "sk-12345") {
		t.Errorf("expected secret to be redacted, got %q", result)
	}
	if !strings.Contains(result, "***REDACTED***") {
		t.Errorf("expected ***REDACTED***, got %q", result)
	}
}

func TestRedactEmptySecret(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, nil)
	filter := secrets.NewRedactFilter(inner)

	// Empty secret should be ignored
	filter.AddSecret("")

	logger := slog.New(filter)
	logger.Info("Normal message")

	output := buf.String()
	if strings.Contains(output, "***REDACTED***") {
		t.Error("empty secret should not cause redaction")
	}
}
