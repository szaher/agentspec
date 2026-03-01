package tools

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// ErrBinaryNotAllowed indicates a binary is not in the allowlist.
type ErrBinaryNotAllowed struct {
	Binary string
}

func (e *ErrBinaryNotAllowed) Error() string {
	return fmt.Sprintf("binary %q not in allowlist", e.Binary)
}

// ErrBinaryNotFound indicates a binary is in the allowlist but not on the system.
type ErrBinaryNotFound struct {
	Binary string
}

func (e *ErrBinaryNotFound) Error() string {
	return fmt.Sprintf("binary %q not found on system", e.Binary)
}

// ErrNoAllowlist indicates no allowlist is configured — all execution is blocked.
type ErrNoAllowlist struct{}

func (e *ErrNoAllowlist) Error() string {
	return "command tool execution blocked: no allowlist configured. Configure allowed_commands in agent or server config."
}

// ValidateBinary checks if a binary name is permitted by the allowlist.
// Returns an error if the binary is not allowed or not found.
// When allowlist is nil or empty, ALL binaries are blocked (secure default).
func ValidateBinary(binary string, allowlist []string) error {
	if len(allowlist) == 0 {
		return &ErrNoAllowlist{}
	}

	// Extract basename for comparison
	baseName := filepath.Base(binary)

	// Check if binary is in the allowlist
	found := false
	for _, allowed := range allowlist {
		if allowed == baseName {
			found = true
			break
		}
	}

	if !found {
		return &ErrBinaryNotAllowed{Binary: baseName}
	}

	// Verify binary exists on the system
	if _, err := exec.LookPath(binary); err != nil {
		return &ErrBinaryNotFound{Binary: baseName}
	}

	return nil
}
