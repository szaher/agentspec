package session

import (
	"strings"
	"testing"
)

func TestGenerateID(t *testing.T) {
	// Test different prefixes
	prefixes := []string{"sess_", "tr_", "cor_", "cf_", ""}
	for _, prefix := range prefixes {
		id := GenerateID(prefix)
		if !strings.HasPrefix(id, prefix) {
			t.Errorf("GenerateID(%q) = %q, want prefix %q", prefix, id, prefix)
		}
		if len(id) <= len(prefix) {
			t.Errorf("GenerateID(%q) = %q, too short", prefix, id)
		}
	}

	// Uniqueness
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateID("u_")
		if seen[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		seen[id] = true
	}
}
