# Data Model: Security Hardening & Compliance

**Branch**: `007-security-hardening` | **Date**: 2026-03-01 | **Spec**: [spec.md](./spec.md)

## Entities

### SessionID

A cryptographically random identifier for user sessions.

| Field | Type | Description |
|-------|------|-------------|
| prefix | string | Fixed value `sess_` for debugging and log filtering |
| entropy | [16]byte | 128 bits of cryptographic randomness from `crypto/rand` |
| encoded | string | Base64url encoding of entropy bytes (no padding) |

**Format**: `sess_<base64url(16 random bytes)>` (e.g., `sess_dGhpcyBpcyAxNiBieXRl`)

**Validation Rules**:
- Prefix must be exactly `sess_`
- Entropy portion must decode to exactly 16 bytes
- Total length: 5 (prefix) + 22 (base64url of 16 bytes) = 27 characters
- Generated via `crypto/rand.Read()` — never timestamps, counters, or `math/rand`

**Relationships**: Used as key in `Store` interface (`Create`, `Get`, `Delete`, `Touch`)

---

### PolicyRule

A deny/require rule defined in .ias files, evaluated during `apply`.

| Field | Type | Description |
|-------|------|-------------|
| Type | string | One of: `deny`, `require` |
| Action | string | One of: `command`, `pinned imports`, `secret`, `signed packages` |
| Subject | string | Target of the rule (e.g., binary name for `deny command`, secret name for `require secret`) |

**Requirement Type Dispatch**:

| Requirement | Validation Logic |
|-------------|-----------------|
| `pinned imports` | All import declarations in IR have version or SHA pin |
| `secret` | Named secret exists in configured secret resolvers |
| `deny command` | No command tool uses the denied binary name |
| `signed packages` | Imported packages have valid signature in manifest |

**Validation Rules**:
- `Type` must be `deny` or `require`
- `Action` must be one of the 4 supported requirement types
- `Subject` is required for `deny command` and `require secret`; ignored for `pinned imports` and `signed packages`
- Unknown requirement types produce an error listing supported types

**State Transitions**:
- Evaluation modes: `enforce` (default) → blocks apply on violation; `warn` → logs violations, allows apply

**Relationships**: Evaluated by `policy.Engine` against `Resource` during `apply`

---

### ToolAllowlist

Configurable list of permitted binary names for command tools.

| Field | Type | Description |
|-------|------|-------------|
| Binaries | []string | Permitted binary names (e.g., `["curl", "jq", "git"]`) |
| Source | string | Where the allowlist is configured: agent config or server config |

**Validation Rules**:
- Empty allowlist (default) blocks ALL command tool execution
- Binary names are matched exactly (no path resolution — just basename)
- Allowlisted binary must also exist on the system (`exec.LookPath` check)
- Two distinct error types: "binary not in allowlist" vs "binary not found"

**Relationships**: Checked by `tools.CommandExecutor` before execution

---

### CORSConfig

Origin allowlist for cross-origin requests.

| Field | Type | Description |
|-------|------|-------------|
| AllowedOrigins | []string | Permitted origins (e.g., `["https://app.example.com"]`) |
| DevMode | bool | When true, auto-adds `http://localhost:<port>` and `http://127.0.0.1:<port>` |

**Validation Rules**:
- Default: empty list (deny all cross-origin requests)
- Wildcard `*` is not the default and must be explicitly configured
- `Access-Control-Allow-Origin` is set to the matched origin (not `*`)
- In dev mode, localhost origins are automatically added for the built-in UI port

**Relationships**: Used by CORS middleware in `runtime.Server` and `frontend.SSEWriter`

---

### SandboxConfig

Resource limits and permissions for inline tool execution.

| Field | Type | Description |
|-------|------|-------------|
| Enabled | bool | Whether sandboxing is active (default: false; enabled via `--sandbox` or config) |
| MemoryLimitMB | int | Maximum memory in MB for tool process (default: 256) |
| TimeoutSec | int | Maximum execution time in seconds (default: 30) |
| AllowNetwork | bool | Whether network access is permitted (default: false) |
| SandboxDir | string | Restricted filesystem root for tool execution |

**Validation Rules**:
- When `Enabled` is false AND no sandbox backend is available, inline tools are disabled entirely
- `MemoryLimitMB` must be > 0 and <= 4096
- `TimeoutSec` must be > 0 and <= 300
- `SandboxDir` is a temporary directory created per-execution, cleaned up after
- Applies uniformly to all 4 languages (Python, Node, Bash, Ruby)

**Relationships**: Used by `sandbox.Sandbox` interface wrapping `tools.InlineExecutor`

---

### AuthFailureBucket

Per-IP tracking of authentication failures for rate limiting.

| Field | Type | Description |
|-------|------|-------------|
| IP | string | Client IP address (key) |
| Failures | int | Count of failed auth attempts in current window |
| WindowStart | time.Time | Start of the current 1-minute counting window |
| BlockedUntil | time.Time | If set, IP is blocked until this time |

**Validation Rules**:
- Window resets after 1 minute of no failures
- After 10 failures in 1 minute → `BlockedUntil` set to now + 5 minutes
- Blocked IPs receive 429 regardless of key correctness
- Successful auth resets the failure counter (but does NOT clear an active block)

**State Transitions**:
```
Normal → (failure) → Tracking → (10th failure in 1min) → Blocked
Blocked → (5 min elapsed) → Normal
Tracking → (1 min without failure) → Normal
Tracking → (success) → Normal
```

**Relationships**: Stored in `auth.RateLimiter.authFailures` map, checked by `auth.Middleware`

---

### SSRFBlocklist

Hardcoded list of private/internal IP ranges blocked for HTTP tools.

| Range | CIDR | Description |
|-------|------|-------------|
| Loopback | `127.0.0.0/8` | IPv4 loopback |
| RFC 1918-A | `10.0.0.0/8` | Private network (Class A) |
| RFC 1918-B | `172.16.0.0/12` | Private network (Class B) |
| RFC 1918-C | `192.168.0.0/16` | Private network (Class C) |
| Link-local | `169.254.0.0/16` | Link-local / cloud metadata |
| IPv6 loopback | `::1/128` | IPv6 loopback |
| IPv6 ULA | `fc00::/7` | IPv6 unique local |
| IPv6 link-local | `fe80::/10` | IPv6 link-local |

**Validation Rules**:
- Checked at dial time (resolved IP), not URL parse time
- Prevents DNS rebinding attacks by inspecting resolved IP before connecting
- Custom `http.Transport` with `DialContext` performs the check

**Relationships**: Used by `tools.SSRFValidator` in HTTP tool execution

---

### ServerTimeouts

HTTP server timeout configuration for DoS protection.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| ReadHeaderTimeout | time.Duration | 10s | Max time to read request headers |
| ReadTimeout | time.Duration | 30s | Max time to read entire request |
| IdleTimeout | time.Duration | 120s | Max time for idle keep-alive connections |
| MaxBodyBytes | int64 | 10MB (10485760) | Max request body size |

**Validation Rules**:
- All timeouts must be > 0
- `MaxBodyBytes` must be > 0 and <= 100MB
- Applied to `http.Server` struct at startup
- `MaxBodyBytes` enforced via `http.MaxBytesReader` on all API endpoints

**Relationships**: Configured on `runtime.Server.httpServer`

## Entity Relationship Summary

```
┌──────────────┐     validates     ┌──────────────┐
│  PolicyRule   │────────────────→ │   Resource    │
└──────────────┘                   └──────────────┘
                                          │
                                   references
                                          ↓
┌──────────────┐                  ┌──────────────┐
│ToolAllowlist │←── checked by ──│CommandExecutor│
└──────────────┘                  └──────────────┘

┌──────────────┐     creates      ┌──────────────┐
│    Store     │────────────────→ │  SessionID    │
└──────────────┘                   └──────────────┘

┌──────────────┐     wraps        ┌──────────────┐
│   Sandbox    │────────────────→ │InlineExecutor │
└──────────────┘                   └──────────────┘
       ↑
  configured by
       │
┌──────────────┐
│SandboxConfig │
└──────────────┘

┌──────────────┐     protects     ┌──────────────┐
│SSRFBlocklist │────────────────→ │ HTTPExecutor  │
└──────────────┘                   └──────────────┘

┌──────────────────┐   tracks    ┌──────────────┐
│AuthFailureBucket │────────────→│auth.Middleware│
└──────────────────┘              └──────────────┘

┌──────────────┐   configures    ┌──────────────┐
│  CORSConfig  │────────────────→│runtime.Server│
└──────────────┘                  └──────────────┘

┌───────────────┐  configures    ┌──────────────┐
│ServerTimeouts │───────────────→│runtime.Server│
└───────────────┘                 └──────────────┘
```
