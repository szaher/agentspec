package tools

import (
	"errors"
	"testing"
)

func TestValidateBinary_EmptyAllowlist(t *testing.T) {
	err := ValidateBinary("echo", nil)
	if err == nil {
		t.Fatal("expected error for nil allowlist, got nil")
	}

	var noAllowlist *ErrNoAllowlist
	if !errors.As(err, &noAllowlist) {
		t.Fatalf("expected *ErrNoAllowlist, got %T: %v", err, err)
	}

	// Also test with empty slice (not just nil)
	err = ValidateBinary("echo", []string{})
	if err == nil {
		t.Fatal("expected error for empty allowlist, got nil")
	}
	if !errors.As(err, &noAllowlist) {
		t.Fatalf("expected *ErrNoAllowlist for empty slice, got %T: %v", err, err)
	}
}

func TestValidateBinary_BinaryInAllowlistAndOnPath(t *testing.T) {
	// "echo" is a standard binary available on all Unix-like systems
	err := ValidateBinary("echo", []string{"echo", "cat", "ls"})
	if err != nil {
		t.Fatalf("expected nil error for allowed binary on PATH, got: %v", err)
	}
}

func TestValidateBinary_BinaryNotInAllowlist(t *testing.T) {
	tests := []struct {
		name      string
		binary    string
		allowlist []string
	}{
		{
			name:      "simple binary not allowed",
			binary:    "curl",
			allowlist: []string{"echo", "cat"},
		},
		{
			name:      "full path binary not allowed",
			binary:    "/usr/bin/curl",
			allowlist: []string{"echo", "cat"},
		},
		{
			name:      "single item allowlist",
			binary:    "rm",
			allowlist: []string{"echo"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateBinary(tc.binary, tc.allowlist)
			if err == nil {
				t.Fatal("expected error for binary not in allowlist, got nil")
			}

			var notAllowed *ErrBinaryNotAllowed
			if !errors.As(err, &notAllowed) {
				t.Fatalf("expected *ErrBinaryNotAllowed, got %T: %v", err, err)
			}
		})
	}
}

func TestValidateBinary_BinaryInAllowlistButNotFound(t *testing.T) {
	fakeBinary := "nonexistent_binary_xyz_12345"
	err := ValidateBinary(fakeBinary, []string{fakeBinary, "echo"})
	if err == nil {
		t.Fatal("expected error for binary not found on system, got nil")
	}

	var notFound *ErrBinaryNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("expected *ErrBinaryNotFound, got %T: %v", err, err)
	}

	if notFound.Binary != fakeBinary {
		t.Fatalf("expected Binary=%q, got %q", fakeBinary, notFound.Binary)
	}
}

func TestValidateBinary_FullPathUsesBasename(t *testing.T) {
	// When a full path is provided, filepath.Base is used for allowlist comparison.
	// "/usr/bin/echo" should match allowlist entry "echo".
	tests := []struct {
		name      string
		binary    string
		allowlist []string
		wantErr   bool
	}{
		{
			name:      "full path matches basename in allowlist",
			binary:    "/bin/echo",
			allowlist: []string{"echo"},
			wantErr:   false,
		},
		{
			name:      "full path does not match if basename not in allowlist",
			binary:    "/bin/echo",
			allowlist: []string{"cat"},
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateBinary(tc.binary, tc.allowlist)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got: %v", err)
			}
		})
	}
}
