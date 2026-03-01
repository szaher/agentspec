# Contract: Inline Tool Sandbox

**Feature**: 007-security-hardening | **Date**: 2026-03-01

## Overview

Defines the sandbox interface for isolating inline tool execution (Python, Node, Bash, Ruby) using OS-level process isolation.

## Interface: `Sandbox`

**Package**: `internal/sandbox`
**File**: `sandbox.go`

```go
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
    Language    string            // "python", "node", "bash", "ruby"
    Script      string            // Script content to execute
    Env         map[string]string // Environment variables (secrets + safe env)
    MemoryMB    int               // Memory limit in MB
    TimeoutSec  int               // Execution timeout in seconds
    AllowNet    bool              // Whether to allow network access
    WorkDir     string            // Working directory inside sandbox
}
```

**Contract**:
- MUST support all 4 languages: Python, Node, Bash, Ruby
- MUST create a temporary sandbox directory per execution, cleaned up after
- MUST restrict filesystem access to the sandbox directory only
- MUST enforce memory limit via OS mechanisms (ulimit on Linux/macOS)
- MUST enforce timeout via `context.WithTimeout` + process kill
- MUST block network access by default (unless `AllowNet` is true)
- MUST return a distinct `ErrSandboxViolation` error type for policy violations
- MUST return a distinct `ErrResourceLimit` error type for resource exhaustion

## Interface: `ProcessSandbox`

**Package**: `internal/sandbox`
**File**: `process.go`

```go
// ProcessSandbox implements Sandbox using OS-level process isolation.
type ProcessSandbox struct{}
```

**Contract**:
- On Linux: use `ulimit` for memory, `timeout` for CPU, tmpdir chroot for filesystem
- On macOS: use `ulimit` for memory, `timeout` (or `gtimeout`) for CPU, tmpdir for filesystem
- On Windows: fall back to `NoopSandbox` with warning
- MUST set process environment to minimal safe set (PATH, HOME) plus configured secrets
- MUST NOT inherit host environment variables
- MUST capture stdout/stderr to buffers (not pass to host stdout/stderr)

## Interface: `NoopSandbox`

**Package**: `internal/sandbox`
**File**: `noop.go`

```go
// NoopSandbox provides no isolation. Used for testing or when no backend is available.
type NoopSandbox struct{}
```

**Contract**:
- `Available()` always returns `true`
- `Execute()` runs the script without isolation
- Used ONLY in test environments or when sandbox is explicitly disabled

## Startup Behavior

**Contract**:
- If `--sandbox` flag or `sandbox: true` config is set:
  - Detect available sandbox backend
  - If none available: disable inline tools entirely with error: `"inline tools disabled: no sandbox backend available on this platform. Install coreutils for timeout support."`
- If sandbox is not configured:
  - Inline tools run without isolation (current behavior, for backward compatibility)
  - Log WARNING: `"inline tools running without sandbox isolation. Use --sandbox to enable."`

## Error Types

```go
// ErrSandboxViolation indicates a sandbox policy was violated.
type ErrSandboxViolation struct {
    Operation string // e.g., "filesystem read", "network connect"
    Path      string // filesystem path or network address attempted
}

// ErrResourceLimit indicates a resource limit was exceeded.
type ErrResourceLimit struct {
    Resource string // "memory" or "time"
    Limit    string // configured limit value
}
```

**Contract**:
- `ErrSandboxViolation` MUST include the specific operation and target
- `ErrResourceLimit` MUST include which resource and what the limit was
- Both error types MUST implement `error` interface with descriptive messages
- Error messages MUST be user-facing (not raw OS errors)

## Language-Specific Behavior

| Language | Interpreter | Notes |
|----------|------------|-------|
| Python | `python3` | Falls back to `python` if `python3` not found |
| Node | `node` | |
| Bash | `bash` | NOT `sh` — bash-specific features may be used |
| Ruby | `ruby` | |

**Contract**:
- Interpreter must be found via `exec.LookPath` (not hardcoded path)
- If interpreter not found: return error `"interpreter not found: <language>"` (distinct from sandbox violation)
