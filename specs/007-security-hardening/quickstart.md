# Quickstart Verification: Security Hardening & Compliance

**Branch**: `007-security-hardening` | **Date**: 2026-03-01 | **Spec**: [spec.md](./spec.md)

## Prerequisites

- Go 1.25+ installed
- Repository cloned and on branch `007-security-hardening`
- `agentspec` binary built: `go build -o agentspec ./cmd/agentspec`

## Verification Steps

### 1. Cryptographic Session IDs (FR-001)

```bash
# Run the session ID unit tests
go test ./internal/session/ -run TestSecureID -v -count=1

# Expected: IDs start with "sess_", are 27 chars long, and 10,000 concurrent
# creations produce zero collisions
```

### 2. Constant-Time Auth (FR-002)

```bash
# Run the auth comparison tests
go test ./internal/auth/ -run TestConstantTime -v -count=1
go test ./internal/runtime/ -run TestAuthComparison -v -count=1

# Expected: Both middleware and runtime server use crypto/subtle
```

### 3. No-Auth Warning (FR-003)

```bash
# Start server without API key and without --no-auth flag
./agentspec run examples/simple.ias 2>&1 | head -5

# Expected: Server refuses to start or logs a prominent WARNING
# With --no-auth flag, it should start with warning
./agentspec run examples/simple.ias --no-auth 2>&1 | head -5
```

### 4. Inline Tool Sandbox (FR-004, FR-005)

```bash
# Run sandbox integration tests
go test ./internal/sandbox/ -v -count=1
go test ./integration_tests/ -run TestInlineSandbox -v -count=1

# Expected: Tools that attempt filesystem/network access are blocked
# Tools exceeding memory limits are terminated
```

### 5. Policy Engine (FR-006, FR-007)

```bash
# Run policy enforcement tests
go test ./internal/policy/ -run TestCheckRequirement -v -count=1

# Test with a .ias file that has unpinned imports
cat > /tmp/test-policy.ias << 'EOF'
agent "test" {
  policy {
    require "pinned imports"
  }
  import "some-package"
}
EOF
./agentspec apply /tmp/test-policy.ias 2>&1

# Expected: Apply fails with "unpinned import" error
# With --policy=warn, apply succeeds with warning
./agentspec apply /tmp/test-policy.ias --policy=warn 2>&1
```

### 6. Server Timeouts (FR-008, FR-009)

```bash
# Run server timeout tests
go test ./internal/runtime/ -run TestServerTimeouts -v -count=1

# Expected: Server has ReadHeaderTimeout, ReadTimeout, IdleTimeout set
# Request bodies exceeding 10MB return 413
```

### 7. SSRF Protection (FR-011)

```bash
# Run SSRF validation tests
go test ./internal/tools/ -run TestSSRF -v -count=1

# Expected: Private IP ranges (127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12,
# 192.168.0.0/16, 169.254.0.0/16) are blocked
```

### 8. Command Tool Allowlist (FR-012)

```bash
# Run allowlist tests
go test ./internal/tools/ -run TestAllowlist -v -count=1

# Expected: No allowlist configured → all commands blocked
# Allowlisted binary → allowed; unlisted binary → blocked
```

### 9. CORS Configuration (FR-014)

```bash
# Run CORS tests
go test ./internal/frontend/ -run TestCORS -v -count=1
go test ./internal/runtime/ -run TestCORS -v -count=1

# Expected: Default rejects all cross-origin; dev mode auto-allows localhost
```

### 10. Race Condition Fixes (FR-015)

```bash
# Run all tests with race detector
go test ./internal/mcp/ -race -v -count=1
go test ./internal/secrets/ -race -v -count=1

# Expected: Zero race conditions reported
```

### 11. Error Transparency (FR-016, FR-017)

```bash
# Run LLM client error handling tests
go test ./internal/llm/ -run TestErrorPropagation -v -count=1

# Expected: JSON errors are propagated, not discarded
```

### 12. Auth Rate Limiting (FR-020)

```bash
# Run rate limiting tests
go test ./internal/auth/ -run TestAuthRateLimit -v -count=1

# Expected: After 10 failures in 1 minute, IP is blocked for 5 minutes
```

## Full Test Suite

```bash
# Run all tests with race detector enabled
go test ./... -race -count=1

# Expected: All tests pass with zero race conditions
```

## Quick Smoke Test

```bash
# Build and run a minimal test
go build -o agentspec ./cmd/agentspec && \
go test ./internal/session/ ./internal/auth/ ./internal/tools/ \
  ./internal/policy/ ./internal/runtime/ ./internal/mcp/ \
  ./internal/secrets/ ./internal/llm/ \
  -race -count=1 -short

# Expected: All packages pass
```
