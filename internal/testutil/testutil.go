// Package testutil provides shared test helpers to reduce boilerplate across unit tests.
package testutil

import (
	"encoding/json"
	"strings"
	"testing"
)

// TempDir creates a temporary directory that is automatically cleaned up when the test finishes.
func TempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// MustMarshalJSON marshals v to JSON, failing the test if marshaling fails.
func MustMarshalJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return data
}

// AssertErrorContains asserts that err is non-nil and its message contains substr.
func AssertErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("expected error containing %q, got %q", substr, err.Error())
	}
}
