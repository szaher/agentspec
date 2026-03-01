package auth

import (
	"testing"
)

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name     string
		provided string
		expected string
		want     bool
	}{
		{
			name:     "correct key matches",
			provided: "correct",
			expected: "correct",
			want:     true,
		},
		{
			name:     "wrong key does not match",
			provided: "wrong",
			expected: "correct",
			want:     false,
		},
		{
			name:     "empty provided does not match",
			provided: "",
			expected: "correct",
			want:     false,
		},
		{
			name:     "empty expected always returns false",
			provided: "anything",
			expected: "",
			want:     false,
		},
		{
			name:     "both empty returns false",
			provided: "",
			expected: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateKey(tt.provided, tt.expected)
			if got != tt.want {
				t.Errorf("ValidateKey(%q, %q) = %v, want %v", tt.provided, tt.expected, got, tt.want)
			}
		})
	}
}

func TestKeyFromEnv(t *testing.T) {
	t.Run("returns value when AGENTSPEC_API_KEY is set", func(t *testing.T) {
		t.Setenv("AGENTSPEC_API_KEY", "test-secret-key")

		got := KeyFromEnv()
		if got != "test-secret-key" {
			t.Errorf("KeyFromEnv() = %q, want %q", got, "test-secret-key")
		}
	})

	t.Run("returns empty string when AGENTSPEC_API_KEY is unset", func(t *testing.T) {
		t.Setenv("AGENTSPEC_API_KEY", "")

		got := KeyFromEnv()
		if got != "" {
			t.Errorf("KeyFromEnv() = %q, want %q", got, "")
		}
	})
}
