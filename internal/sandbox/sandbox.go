package sandbox

import (
	"context"
	"fmt"
)

// Sandbox wraps inline tool execution with resource and filesystem isolation.
type Sandbox interface {
	// Execute runs a script in a sandboxed environment.
	// Returns stdout, stderr, and any error (including sandbox violations).
	Execute(ctx context.Context, config ExecConfig) (stdout, stderr string, err error)

	// Available reports whether this sandbox backend can run on the current platform.
	Available() bool
}

// ExecConfig holds the execution parameters for a sandboxed tool.
type ExecConfig struct {
	Language   string            // "python", "node", "bash", "ruby"
	Script     string            // Script content to execute
	Env        map[string]string // Environment variables (secrets + safe env)
	MemoryMB   int               // Memory limit in MB
	TimeoutSec int               // Execution timeout in seconds
	AllowNet   bool              // Whether to allow network access
	WorkDir    string            // Working directory inside sandbox
}

// ErrSandboxViolation indicates a sandbox policy was violated.
type ErrSandboxViolation struct {
	Operation string // e.g., "filesystem read", "network connect"
	Path      string // filesystem path or network address attempted
}

func (e *ErrSandboxViolation) Error() string {
	return fmt.Sprintf("sandbox violation: %s denied for %s", e.Operation, e.Path)
}

// ErrResourceLimit indicates a resource limit was exceeded.
type ErrResourceLimit struct {
	Resource string // "memory" or "time"
	Limit    string // configured limit value
}

func (e *ErrResourceLimit) Error() string {
	return fmt.Sprintf("resource limit exceeded: %s (limit: %s)", e.Resource, e.Limit)
}
