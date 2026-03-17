# Missing / Incomplete Features List

**Last updated:** 2026-03-01

## Summary

| Priority | Count |
|----------|-------|
| P0 | 4 |
| P1 | 8 |
| P2 | 7 |
| P3 | 5 |
| **Total** | **24** |

## P0 — Must Fix

| ID | Feature/Requirement | Where | Current Behavior | Gap | Evidence | Acceptance Criteria | Complexity |
|---|---|---|---|---|---|---|---|
| GAP-001 | Policy engine must actually enforce `require` rules | `agentspec apply`, `internal/policy/policy.go` | `checkRequirement()` always returns `true` | Policies are defined but never enforced; users get false security assurance | `internal/policy/policy.go:77-85` — default case returns `true` | Given a policy `require pinned imports`, when an import lacks a version pin, then apply MUST reject the resource | M |
| GAP-002 | State file writes must be atomic | `internal/state/local.go` | `os.WriteFile` writes directly to state file | Crash during write corrupts state; all infrastructure tracking lost | `internal/state/local.go` — no temp-file pattern | Given a crash during state save, when the system restarts, then the previous valid state MUST be intact | S |
| GAP-003 | Session IDs must be cryptographically random | `internal/session/memory_store.go`, `redis_store.go` | IDs from `time.Now().UnixNano()` | Predictable IDs enable session hijacking | `memory_store.go:102-104`, `redis_store.go:181-183` | Given session creation, when an ID is generated, then it MUST contain >= 128 bits of cryptographic randomness | S |
| GAP-004 | Inline tool execution must be sandboxed | `internal/tools/inline.go` | Writes code to temp file, executes via system interpreter | No filesystem, network, or resource isolation; full host access | `inline.go:34-89` — `exec.CommandContext` with no restrictions | Given inline tool execution, when code runs, then it MUST be isolated from host filesystem, network, and system resources | XL |

## P1 — High Priority

| ID | Feature/Requirement | Where | Current Behavior | Gap | Evidence | Acceptance Criteria | Complexity |
|---|---|---|---|---|---|---|---|
| GAP-005 | HTTP server must have connection timeouts | `internal/runtime/server.go` | `http.Server{}` created with no timeout fields | Vulnerable to slow-loris; connections held indefinitely | `server.go:114-121` | Given server start, when configured, then ReadHeaderTimeout, ReadTimeout, and IdleTimeout MUST be set | S |
| GAP-006 | API key comparison must be constant-time in server | `internal/runtime/server.go` | Uses `!=` string comparison | Timing side-channel enables key discovery | `server.go:158` — `key != s.apiKey` | Given an auth check, when comparing keys, then `subtle.ConstantTimeCompare` MUST be used | S |
| GAP-007 | Server must warn when running without authentication | `internal/runtime/server.go` | Silently allows all requests when no API key set | Users unknowingly run unauthenticated servers | `server.go:145-148` — no log, no flag | Given server start without API key, when the server binds, then a WARNING MUST be logged | S |
| GAP-008 | HTTP tool must have SSRF protection | `internal/tools/http.go` | Any URL accepted without validation | Can reach internal services, cloud metadata endpoints | `http.go:43-44` — no URL validation | Given an HTTP tool call, when the URL resolves to a private IP, then the request MUST be rejected | M |
| GAP-009 | Request bodies must have size limits | `internal/runtime/server.go` | `json.NewDecoder(r.Body).Decode()` with no limit | OOM via large request body | `server.go:206,310,402,465` | Given an API request, when the body exceeds 10MB, then the server MUST return 413 | S |
| GAP-010 | HTTP tool responses must have size limits | `internal/tools/http.go` | `io.ReadAll(resp.Body)` with no limit | OOM via large response body | `http.go:82` | Given an HTTP tool call, when response exceeds 10MB, then it MUST be truncated with error | S |
| GAP-011 | Redis session store must use SCAN instead of KEYS | `internal/session/redis_store.go` | `KEYS prefix*` blocks Redis server | Performance degradation; Redis blocking under high session counts | `redis_store.go:106` | Given session listing, when querying Redis, then cursor-based SCAN MUST be used | S |
| | **Status**: Implemented in `010-memory-performance` (2026-03-04) | | | | | | |
| GAP-012 | Command tool must validate binary against allowlist | `internal/tools/command.go` | Any binary name executed directly | Arbitrary command execution from `.ias` files | `command.go:39` — `exec.CommandContext(ctx, e.config.Binary, ...)` | Given a command tool config, when the binary is not in the allowlist, then execution MUST fail with descriptive error | M |

## P2 — Medium Priority

| ID | Feature/Requirement | Where | Current Behavior | Gap | Evidence | Acceptance Criteria | Complexity |
|---|---|---|---|---|---|---|---|
| GAP-013 | README must document all CLI commands | `README.md` | 11 of 19 commands documented | 8 commands undiscoverable: init, compile, publish, install, eval, status, logs, destroy | `README.md:56-71` vs `main.go:50-69` | Given the README, when a user reads CLI commands, then ALL 19 commands MUST be listed with descriptions | S |
| GAP-014 | Compiler targets must generate functional tool stubs | `internal/compiler/targets/` | All targets emit `return "not implemented"` | Generated code looks complete but fails at runtime | `crewai.go:218-241`, `langgraph.go:142`, `llamastack.go:128`, `llamaindex.go:142` | Given compilation, when tools are present, then generated code MUST include a working skeleton that calls the tool backend | L |
| GAP-015 | Frontend must have loading, error, and empty states | `internal/frontend/web/` | Blank screen on load; no error UI; no empty state | Poor UX during initial load, connection failure, and fresh sessions | `app.js` — no loading spinner, no error banner, no welcome message | Given frontend load, when agents are fetching, then a loading indicator MUST display; when no messages exist, then a welcome/help message MUST display | M |
| GAP-016 | `eval` command must support live agent invocation | `cmd/agentspec/eval.go` | Uses stub invoker that always returns error | Cannot evaluate agent behavior end-to-end | `eval.go:56-57,138-143` — `stubInvoker` always errors | Given eval with `--live` flag, when an expression invokes an agent, then a real LLM client MUST be used | L |
| GAP-017 | Expired memory store sessions must be evicted proactively | `internal/session/memory_store.go` | Expired sessions only deleted on individual `Get()` | Memory leak from never-accessed expired sessions | `memory_store.go:54-57` — lazy deletion only | Given expired sessions, when eviction timer fires, then all expired sessions MUST be removed | S |
| | **Status**: Implemented in `010-memory-performance` (2026-03-04) | | | | | | |
| GAP-018 | Redis SaveMessages must be concurrent-safe | `internal/session/redis_store.go` | Read-modify-write without locking | Concurrent saves to same session can lose messages | `redis_store.go:153-163` | Given concurrent message saves, when two saves execute, then NO messages MUST be lost | M |
| GAP-019 | CORS must be configurable (not wildcard) | `internal/frontend/sse.go` | Hardcoded `Access-Control-Allow-Origin: *` | Any website can invoke agents | `sse.go:32` | Given server config, when CORS origin is specified, then only that origin MUST be allowed; default MUST NOT be `*` | S |

## P3 — Nice to Have

| ID | Feature/Requirement | Where | Current Behavior | Gap | Evidence | Acceptance Criteria | Complexity |
|---|---|---|---|---|---|---|---|
| GAP-020 | `publish --sign` must work or be removed | `cmd/agentspec/publish.go` | Flag exists but prints "not yet implemented" | False security signal; user expects signing | `publish.go:25,31-33` | Given `--sign` flag, when publishing, then the package MUST be cryptographically signed; OR the flag MUST be removed | L |
| GAP-021 | Dev mode should use filesystem events, not polling | `cmd/agentspec/dev.go` | 2-second polling with `filepath.Walk` | High latency; CPU overhead on large dirs | `dev.go:100,116-128` | Given a file change, when in dev mode, then reload MUST trigger within 500ms | M |
| GAP-022 | `run` command naming should be clarified | `cmd/agentspec/run.go` | One-shot invocation (not a server) | README says "Start the agent runtime server" but `run` does one-shot; `dev` starts server | `run.go` vs README.md | Given `agentspec run`, when executed, then behavior MUST match documentation; or rename to `invoke` | S |
| GAP-023 | Server must support TLS | `internal/runtime/server.go` | HTTP only; no TLS support | API keys transmitted in cleartext | `server.go:114-121` | Given TLS config, when cert/key paths provided, then server MUST use HTTPS | M |
| GAP-024 | Rate limiter must evict stale buckets | `internal/auth/ratelimit.go` | Bucket map grows without bound | Memory leak under diverse client traffic | `ratelimit.go:71-98` | Given no traffic from a client for >10min, when eviction runs, then the client's bucket MUST be removed | S |
| | **Status**: Implemented in `010-memory-performance` (2026-03-04) | | | | | | |
