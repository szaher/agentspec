# Contract: Tool Execution Security

**Feature**: 007-security-hardening | **Date**: 2026-03-01

## Overview

Defines the security contracts for command tool allowlist validation, HTTP tool SSRF protection, response size limits, safe environment handling, and body serialization.

## Interface: Command Tool Allowlist

**Package**: `internal/tools`
**File**: `allowlist.go`

```go
// ValidateBinary checks if a binary name is permitted by the allowlist.
// Returns an error if the binary is not allowed or not found.
func ValidateBinary(binary string, allowlist []string) error
```

**Contract**:
- When `allowlist` is nil or empty: ALL binaries are blocked with error `"command tool execution blocked: no allowlist configured. Configure allowed_commands in agent or server config."`
- When `allowlist` is non-empty and binary is NOT in list: return error `"binary %q not in allowlist"`
- When binary is in allowlist but not on system: return error `"binary %q not found on system"` (via `exec.LookPath`)
- When binary is in allowlist and exists: return nil
- Binary matching is exact basename comparison (no path components)
- Two distinct error types allow callers to distinguish "not allowed" from "not found"

## Interface: SSRF Validator

**Package**: `internal/tools`
**File**: `ssrf.go`

```go
// NewSafeTransport returns an http.Transport that validates resolved IPs
// against private/internal ranges before connecting.
func NewSafeTransport() *http.Transport

// IsPrivateIP checks if an IP address is in a private/internal range.
func IsPrivateIP(ip net.IP) bool
```

**Contract**:
- MUST check resolved IP at dial time (inside `DialContext`), not at URL parse time
- MUST block all RFC 1918, RFC 3927, loopback, and IPv6 equivalents (see data-model.md SSRFBlocklist)
- MUST return error `"SSRF: private network access denied for %s (%s)"` including both hostname and resolved IP
- MUST handle DNS rebinding by checking after resolution, before connection
- MUST support both IPv4 and IPv6 addresses
- `IsPrivateIP` MUST be exported for use in tests

## Interface: HTTP Tool Response Limits

**Package**: `internal/tools`
**File**: `http.go`

```go
// Modified: ReadBody reads the response body with a size limit.
func ReadBody(body io.Reader, limit int64) ([]byte, bool, error)
// Returns (data, truncated, error)
```

**Contract**:
- MUST use `io.LimitReader` with configurable max (default 10MB = 10485760 bytes)
- When body exceeds limit: return data up to limit with `truncated = true`
- MUST NOT use `io.ReadAll` without a limit (current vulnerability)
- Truncation MUST be reported to the caller (not silent)

## Interface: Safe Environment

**Package**: `internal/tools`
**Files**: `command.go`, `inline.go`

```go
// SafeEnv returns a minimal environment for tool execution.
// Includes only PATH, HOME, and configured secrets.
func SafeEnv(secrets map[string]string) []string
```

**Contract**:
- MUST NOT inherit the full host environment
- MUST include: `PATH` (system default), `HOME` (current user's home)
- MUST include configured secrets as environment variables
- MUST NOT include: API keys, cloud credentials, shell history paths, editor configs
- Applied to both command tools and inline tools

## Interface: Safe Body Serialization

**Package**: `internal/tools`
**File**: `http.go`

```go
// SafeBodyString converts an HTTP response body to a safe string representation.
func SafeBodyString(body []byte, contentType string) string
```

**Contract**:
- For JSON content type: return raw JSON (safe for template rendering)
- For HTML content type: escape HTML entities before rendering
- For other types: return raw string with no template interpolation
- MUST NOT pass raw body through Go template execution (prevents template injection)
- MUST sanitize `{{` and `}}` sequences in body content

## Integration Points

### Command Tool Execution Flow

```
1. Parse binary name from tool config
2. ValidateBinary(binary, allowlist) → error if blocked
3. exec.LookPath(binary) → already done by ValidateBinary
4. Build SafeEnv(secrets)
5. Execute with safe env
```

### HTTP Tool Execution Flow

```
1. Parse URL from tool config
2. Create client with NewSafeTransport()
3. Execute request
4. ReadBody(resp.Body, maxBytes) → check truncation
5. SafeBodyString(body, contentType) → safe rendering
```

## Error Responses

| Scenario | Error Message |
|----------|---------------|
| No allowlist configured | `command tool execution blocked: no allowlist configured` |
| Binary not in allowlist | `binary "rm" not in allowlist` |
| Binary not found | `binary "custom-tool" not found on system` |
| SSRF blocked | `SSRF: private network access denied for metadata.internal (169.254.169.254)` |
| Response too large | `response body truncated at 10MB limit` |
| Interpreter not found | `interpreter not found: python3` |
