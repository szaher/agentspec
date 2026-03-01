# Research: Security Hardening & Compliance

**Branch**: `007-security-hardening` | **Date**: 2026-03-01

## R1: Cryptographic Session ID Generation

**Decision**: Use `crypto/rand` to generate 16 random bytes, encode as base64url, prefix with `sess_`.

**Rationale**: `crypto/rand` provides OS-level entropy (e.g., `/dev/urandom`). 16 bytes = 128 bits of randomness, which exceeds the OWASP recommendation of 64 bits minimum for session identifiers. Base64url encoding produces URL-safe IDs without padding. The `sess_` prefix aids debugging and log filtering.

**Alternatives considered**:
- `google/uuid` (v4): Adds a dependency for 122 bits of randomness. The UUID format includes fixed bits (version/variant), so actual entropy is lower than raw 128 bits. Rejected: unnecessary dependency.
- `math/rand`: Not cryptographically secure. Rejected: unsuitable for security-sensitive identifiers.
- `xid`/`ulid`: Shorter IDs but include timestamp components. Rejected: partial predictability contradicts requirements.

**Implementation**: Single shared function in `internal/session/id.go`, replacing both `generateID()` and `generateSessionID()`.

## R2: Constant-Time API Key Comparison

**Decision**: Replace `key != s.apiKey` in `runtime/server.go:158` with `auth.ValidateKey(key, s.apiKey)` from `internal/auth/key.go`.

**Rationale**: The `auth` package already implements constant-time comparison using `crypto/subtle.ConstantTimeCompare`. The runtime server currently reimplements auth inline without using this function. Reusing the existing function eliminates the timing side-channel and reduces code duplication.

**Alternatives considered**:
- Inline `subtle.ConstantTimeCompare` in server.go: Works but duplicates logic. Rejected: DRY principle.
- Replace runtime server auth with `auth.Middleware`: Larger refactor, changes HTTP handler chain. Deferred to a separate cleanup — the immediate fix is swapping the comparison.

## R3: Inline Tool Sandbox Approach

**Decision**: Use OS-level process isolation as the primary sandbox mechanism. Wrap inline tool execution with resource limits (ulimit/cgroups on Linux, sandbox-exec on macOS) and filesystem restriction (chroot-like tmpdir). Fall back to "disabled" on unsupported platforms.

**Rationale**: The existing wazero runtime sandboxes WASM modules effectively, but inline tools are Python/Node/Bash/Ruby scripts that run as native processes — they cannot be compiled to WASM without significant effort. OS-level isolation (ulimit for memory, timeout for CPU, tmpdir for filesystem) is the pragmatic approach that works across all 4 languages uniformly.

**Alternatives considered**:
- WASM compilation of scripts: Requires compiling Python/Node/Bash/Ruby runtimes to WASM. Rejected: massive effort, poor compatibility, slow startup.
- Docker container per execution: Strong isolation but heavy overhead (seconds to start). Rejected: too slow for interactive tool calls.
- gVisor/Firecracker: Excellent isolation but requires root/KVM and is Linux-only. Rejected: portability constraint.
- seccomp-bpf: Linux-only syscall filtering. Rejected: not portable to macOS/Windows.

**Implementation phases**:
1. Phase 1 (this feature): Process isolation with ulimit/timeout + restricted tmpdir. Disabled by default; enabled with `--sandbox` flag or `sandbox: true` in config.
2. Phase 2 (future): Container-based isolation as opt-in for stronger guarantees.

## R4: Policy Engine Requirement Types

**Decision**: Implement 4 requirement types in `checkRequirement()`:

1. **`pinned imports`**: Check that all import declarations in the IR have a version or SHA pin. Iterate `resource.References` for import references and verify each has a version field.
2. **`secret`**: Check that the named secret (from `rule.Subject`) is referenced in the resource's `Attributes`. Verify the secret exists in the configured secret resolvers.
3. **`deny command`**: Check that no command tool in the resource uses the denied binary name. Compare `rule.Subject` against tool configs.
4. **`signed packages`**: Check that imported packages have a valid signature. This requires the package manifest to include a signature field — stub with a "not yet implemented" log for MVP if package signing from feature 012 isn't ready.

**Rationale**: These 4 types cover the most critical security requirements for untrusted .ias files: supply-chain integrity (pinned imports, signed packages), secret management (require secret), and execution control (deny command).

**Alternatives considered**:
- Extensible plugin-based policy checker: Over-engineered for 4 types. Deferred to future if more types emerge.
- OPA/Rego integration: Powerful but adds a heavy dependency. Rejected: overkill for current needs.

## R5: SSRF Protection Strategy

**Decision**: Validate resolved IP addresses against RFC 1918 (private), RFC 3927 (link-local), and loopback ranges before allowing HTTP tool connections. Use a custom `http.Transport` with a `DialContext` that resolves DNS and checks the IP before connecting.

**Rationale**: URL-based validation alone is insufficient (DNS rebinding attacks). Checking the resolved IP at dial time catches both direct IP URLs and DNS-resolved IPs pointing to private ranges.

**Blocked ranges**:
- `127.0.0.0/8` (loopback)
- `10.0.0.0/8` (RFC 1918)
- `172.16.0.0/12` (RFC 1918)
- `192.168.0.0/16` (RFC 1918)
- `169.254.0.0/16` (link-local, cloud metadata)
- `::1/128` (IPv6 loopback)
- `fc00::/7` (IPv6 unique local)
- `fe80::/10` (IPv6 link-local)

**Alternatives considered**:
- URL pattern matching only: Vulnerable to DNS rebinding. Rejected.
- External proxy with SSRF protection: Adds infrastructure dependency. Rejected.
- Allowlist of permitted domains: Too restrictive for general-purpose HTTP tools. Rejected as default; could be an additional optional restriction.

## R6: MCP Pool Race Condition Fix

**Decision**: Use `sync.Map` with a per-key `singleflight.Group` to prevent duplicate connections. `LoadOrStore` atomically checks for an existing client; if absent, `singleflight.Do` ensures only one goroutine creates the connection while others wait.

**Rationale**: The current implementation has a TOCTOU race where two goroutines can both create connections for the same server name, leaking one. `singleflight` is the standard Go pattern for deduplicating concurrent operations.

**Alternatives considered**:
- Hold lock during connect: Blocks all pool operations while one connection is made. Rejected: too coarse.
- Per-key mutex map: More complex with similar semantics. Rejected: `singleflight` is simpler.

## R7: Auth Failure Rate Limiting

**Decision**: Add a dedicated auth failure counter per IP in the existing rate limiter infrastructure. Track failed auth attempts separately from general rate limiting. After 10 failures in 1 minute, block the IP for 5 minutes.

**Rationale**: Reuses the existing `RateLimiter` pattern (bucket map + mutex). A separate counter for auth failures prevents a legitimate high-traffic client from being confused with a brute-force attacker.

**Implementation**: Add `authFailures map[string]*authBucket` to the rate limiter with timestamp tracking. Check before auth comparison; increment on failure; reset on success.

## R8: CORS Configuration

**Decision**: Add `--cors-origins` flag (comma-separated list) to the server. Default: empty (reject all cross-origin). In dev mode (`agentspec dev`), auto-add `http://localhost:<port>` and `http://127.0.0.1:<port>`.

**Rationale**: The built-in frontend UI needs CORS access in dev mode, but production should be locked down. Auto-detecting the dev server's own address avoids requiring manual CORS config during development.

**Implementation**: Pass allowed origins to `SSEWriter` and add CORS middleware to the server mux. Check `Origin` header against the allowlist; set `Access-Control-Allow-Origin` to the matched origin (not wildcard).
