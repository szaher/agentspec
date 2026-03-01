package secrets

import (
	"context"
	"testing"
)

func TestEnvResolver_Resolve_ValidRefEnvSet(t *testing.T) {
	t.Setenv("MY_VAR", "secret-value-123")

	r := NewEnvResolver()
	got, err := r.Resolve(context.Background(), "env(MY_VAR)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "secret-value-123" {
		t.Errorf("got %q, want %q", got, "secret-value-123")
	}
}

func TestEnvResolver_Resolve_ValidRefEnvNotSet(t *testing.T) {
	r := NewEnvResolver()
	_, err := r.Resolve(context.Background(), "env(AGENTSPEC_TEST_UNSET_VAR)")
	if err == nil {
		t.Fatal("expected error for unset env var, got nil")
	}
	if want := `environment variable "AGENTSPEC_TEST_UNSET_VAR" not set`; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestEnvResolver_Resolve_MalformedRefWrongPrefix(t *testing.T) {
	r := NewEnvResolver()
	_, err := r.Resolve(context.Background(), "notenv(VAR)")
	if err == nil {
		t.Fatal("expected error for malformed ref, got nil")
	}
	if want := "unsupported secret reference format"; !contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
}

func TestEnvResolver_Resolve_MalformedRefUnclosed(t *testing.T) {
	r := NewEnvResolver()
	_, err := r.Resolve(context.Background(), "env(")
	if err == nil {
		t.Fatal("expected error for unclosed ref, got nil")
	}
	if want := "unsupported secret reference format"; !contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
}

func TestEnvResolver_Resolve_EmptyVarName(t *testing.T) {
	// env() is syntactically valid but the empty-string env var is unlikely to
	// be set, so we expect an "environment variable not set" error.
	r := NewEnvResolver()
	_, err := r.Resolve(context.Background(), "env()")
	if err == nil {
		t.Fatal("expected error for empty var name (env not set), got nil")
	}
}

// contains is a small helper to check substring presence.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
