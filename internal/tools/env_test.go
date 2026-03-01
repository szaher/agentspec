package tools

import (
	"os"
	"strings"
	"testing"
)

func TestSafeEnv_NoSecrets(t *testing.T) {
	env := SafeEnv(nil)

	hasPath := false
	hasHome := false
	for _, entry := range env {
		if strings.HasPrefix(entry, "PATH=") {
			hasPath = true
		}
		if strings.HasPrefix(entry, "HOME=") {
			hasHome = true
		}
	}

	if !hasPath {
		t.Fatal("expected PATH entry in safe env, not found")
	}
	if !hasHome {
		t.Fatal("expected HOME entry in safe env, not found")
	}
}

func TestSafeEnv_WithSecrets(t *testing.T) {
	secrets := map[string]string{
		"API_KEY":     "secret-key-123",
		"DB_PASSWORD": "hunter2",
	}

	env := SafeEnv(secrets)

	hasPath := false
	hasHome := false
	hasAPIKey := false
	hasDBPassword := false

	for _, entry := range env {
		switch {
		case strings.HasPrefix(entry, "PATH="):
			hasPath = true
		case strings.HasPrefix(entry, "HOME="):
			hasHome = true
		case entry == "API_KEY=secret-key-123":
			hasAPIKey = true
		case entry == "DB_PASSWORD=hunter2":
			hasDBPassword = true
		}
	}

	if !hasPath {
		t.Fatal("expected PATH entry in safe env")
	}
	if !hasHome {
		t.Fatal("expected HOME entry in safe env")
	}
	if !hasAPIKey {
		t.Fatal("expected API_KEY secret in env")
	}
	if !hasDBPassword {
		t.Fatal("expected DB_PASSWORD secret in env")
	}
}

func TestSafeEnv_DoesNotLeakHostEnv(t *testing.T) {
	// Set a custom env var on the host and verify it does NOT appear
	// in the safe environment (only PATH, HOME, and secrets should be present)
	customVar := "AGENTSPEC_TEST_LEAK_CHECK_XYZ"
	customVal := "should_not_appear"
	t.Setenv(customVar, customVal)

	// Verify the var is set on the host
	if os.Getenv(customVar) != customVal {
		t.Fatalf("failed to set test env var %s", customVar)
	}

	env := SafeEnv(nil)

	for _, entry := range env {
		if strings.HasPrefix(entry, customVar+"=") {
			t.Fatalf("safe env leaked host env var %s: found %q", customVar, entry)
		}
	}
}

func TestSafeEnv_EmptySecretsMap(t *testing.T) {
	env := SafeEnv(map[string]string{})

	hasPath := false
	hasHome := false
	for _, entry := range env {
		if strings.HasPrefix(entry, "PATH=") {
			hasPath = true
		}
		if strings.HasPrefix(entry, "HOME=") {
			hasHome = true
		}
	}

	if !hasPath {
		t.Fatal("expected PATH entry in safe env with empty secrets map")
	}
	if !hasHome {
		t.Fatal("expected HOME entry in safe env with empty secrets map")
	}

	// Should only have PATH and HOME (2 entries at most)
	if len(env) > 2 {
		t.Fatalf("expected at most 2 entries (PATH, HOME) with empty secrets, got %d: %v", len(env), env)
	}
}
