# Contract: Authentication & Rate Limiting

**Feature**: 007-security-hardening | **Date**: 2026-03-01

## Overview

Defines the authentication middleware behavior, constant-time key validation, no-auth mode, and auth failure rate limiting for the runtime server.

## Interface: `auth.ValidateKey`

**Package**: `internal/auth`
**File**: `key.go`

```go
// ValidateKey compares a provided key against the expected key using
// constant-time comparison. Returns true if keys match.
// Both the runtime server and auth middleware MUST use this function.
func ValidateKey(provided, expected string) bool
```

**Contract**:
- MUST use `crypto/subtle.ConstantTimeCompare`
- MUST NOT short-circuit on length mismatch (pad or use constant-time length check)
- Timing variance between correct and incorrect keys MUST be statistically insignificant (<1% variance over 10,000 measurements)

## Interface: Auth Middleware

**Package**: `internal/auth`
**File**: `middleware.go`

```go
// Middleware returns an HTTP middleware that validates API keys.
// When apiKey is empty and noAuth is false, all requests are rejected.
// When apiKey is empty and noAuth is true, all requests are allowed with a startup warning.
func Middleware(apiKey string, noAuth bool, limiter *RateLimiter) func(http.Handler) http.Handler
```

**Contract**:
- MUST call `ValidateKey()` for key comparison (no inline `!=` or `==`)
- MUST check rate limiter before key validation (blocked IPs get 429 immediately)
- MUST increment auth failure counter on failed validation
- MUST reset auth failure counter on successful validation
- When `apiKey == ""` and `noAuth == false`: reject all requests with 401
- When `apiKey == ""` and `noAuth == true`: allow all requests; log WARNING at startup
- MUST return 401 for invalid keys (no information about which characters matched)
- MUST return 429 for rate-limited IPs with `Retry-After` header

## Interface: Auth Rate Limiter

**Package**: `internal/auth`
**File**: `ratelimit.go`

```go
// AuthFailure records a failed auth attempt for the given IP.
// Returns true if the IP is now blocked.
func (r *RateLimiter) AuthFailure(ip string) bool

// IsAuthBlocked checks if an IP is currently blocked due to auth failures.
func (r *RateLimiter) IsAuthBlocked(ip string) bool

// AuthSuccess resets the failure counter for the given IP.
func (r *RateLimiter) AuthSuccess(ip string)
```

**Contract**:
- MUST track failures per IP independently
- MUST block after 10 failures within a 1-minute window
- MUST block for exactly 5 minutes after threshold exceeded
- MUST be safe for concurrent access (mutex-protected)
- MUST evict stale entries to prevent unbounded memory growth
- `AuthSuccess` resets counter but does NOT clear an active block

## Runtime Server Integration

**File**: `internal/runtime/server.go`

**Contract**:
- MUST replace `key != s.apiKey` at line ~158 with `auth.ValidateKey(key, s.apiKey)`
- MUST add `--no-auth` flag to `cmd/agentspec/run.go`
- MUST log `WARNING: Server running without authentication. Use --no-auth to suppress this warning.` when no API key is configured

## Error Responses

| Scenario | HTTP Status | Body |
|----------|-------------|------|
| Missing API key header | 401 | `{"error": "authentication required"}` |
| Invalid API key | 401 | `{"error": "authentication failed"}` |
| IP rate-limited | 429 | `{"error": "too many failed attempts", "retry_after": <seconds>}` |
| No auth configured, no --no-auth flag | 401 | `{"error": "server requires authentication configuration"}` |
