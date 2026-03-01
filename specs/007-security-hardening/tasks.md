# Tasks: Security Hardening & Compliance

**Input**: Design documents from `/specs/007-security-hardening/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Integration tests are included per project constitution (Testing Strategy: "Integration tests are the primary quality gate").

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create new packages, files, and add dependencies required by multiple user stories

- [x] T001 Create `internal/sandbox/` package directory with empty `sandbox.go`, `process.go`, `noop.go` files
- [x] T002 Add `golang.org/x/sync` dependency for `singleflight` package (run `go get golang.org/x/sync` if not already present)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared utilities that MUST be complete before user story implementation

**Warning**: No user story work can begin until this phase is complete

- [x] T003 Create `SafeEnv(secrets map[string]string) []string` utility function in `internal/tools/env.go` — returns minimal environment with only PATH, HOME, and configured secrets; used by both command and inline tools (FR-013)

**Checkpoint**: Foundation ready — user story implementation can now begin

---

## Phase 3: User Story 1 — Secure Session Management (Priority: P1) MVP

**Goal**: Replace predictable timestamp-based session IDs with cryptographically random 128-bit identifiers

**Independent Test**: Create 10,000 sessions concurrently and verify zero collisions, all IDs start with `sess_`, and contain >= 128 bits of randomness

### Implementation for User Story 1

- [x] T004 [P] [US1] Create shared `generateSecureID() string` function in `internal/session/id.go` — use `crypto/rand.Read(16 bytes)` + base64url encoding + `sess_` prefix per research R1
- [x] T005 [US1] Replace `generateID()` call (line ~102-104) with `generateSecureID()` in `internal/session/memory_store.go` — remove the `fmt.Sprintf("sess_%d", time.Now().UnixNano())` pattern
- [x] T006 [US1] Replace `generateSessionID()` call (line ~181-183) with `generateSecureID()` in `internal/session/redis_store.go` — remove the `fmt.Sprintf("sess_%d", time.Now().UnixNano())` pattern

### Integration Test for User Story 1

- [x] T007 [US1] Add session ID security tests to `integration_tests/security_test.go` (NEW) — test 10,000 concurrent session creations with zero collisions, verify `sess_` prefix, verify IDs are not timestamp-derived (SC-001)

**Checkpoint**: Session IDs are now cryptographically random. Verify with `go test ./internal/session/ ./integration_tests/ -run TestSession -v -count=1`

---

## Phase 4: User Story 2 — API Key Authentication Hardening (Priority: P1)

**Goal**: Eliminate timing side-channel in auth, add rate limiting for brute-force protection, require explicit opt-in for no-auth mode

**Independent Test**: Measure response times for correct vs incorrect keys of varying prefix lengths — timing variance must be < 1% over 10,000 measurements

### Implementation for User Story 2

- [x] T008 [US2] Add `authFailures map[string]*authBucket` with `authBucket` struct (IP, Failures, WindowStart, BlockedUntil) to rate limiter in `internal/auth/ratelimit.go` — per data-model AuthFailureBucket
- [x] T009 [US2] Implement `AuthFailure(ip)`, `IsAuthBlocked(ip)`, `AuthSuccess(ip)` methods on `RateLimiter` in `internal/auth/ratelimit.go` — 10 failures/min threshold, 5-min block, mutex-protected, with stale entry eviction per auth-contract
- [x] T010 [US2] Update auth middleware to check `IsAuthBlocked()` before key validation, call `AuthFailure()`/`AuthSuccess()` after, and return 429 with `Retry-After` header for blocked IPs in `internal/auth/middleware.go`
- [x] T011 [US2] Replace `key != s.apiKey` (line ~158) with `auth.ValidateKey(key, s.apiKey)` in `internal/runtime/server.go` — import `internal/auth` package
- [x] T012 [US2] Add `--no-auth` flag to dev command in `cmd/agentspec/dev.go` — when no API key and no `--no-auth`: log WARNING and reject all requests; when `--no-auth`: allow with startup warning per FR-003

### Integration Test for User Story 2

- [x] T013 [US2] Extend `integration_tests/auth_test.go` with constant-time auth verification, rate limiting (10 failures → 429), no-auth flag behavior, and startup warning tests (SC-002, FR-020)

**Checkpoint**: Auth is constant-time, rate-limited, and requires explicit no-auth opt-in. Verify with `go test ./internal/auth/ ./internal/runtime/ ./integration_tests/ -run TestAuth -v -count=1`

---

## Phase 5: User Story 3 — Inline Tool Sandboxing (Priority: P1)

**Goal**: Sandbox Python/Node/Bash/Ruby inline tool execution with filesystem, network, and memory isolation

**Independent Test**: Create inline tools that attempt to read `/etc/passwd`, make network requests, or exceed memory — all must fail with sandbox violation errors

### Implementation for User Story 3

- [x] T014 [P] [US3] Define `Sandbox` interface, `ExecConfig` struct, `ErrSandboxViolation`, and `ErrResourceLimit` error types in `internal/sandbox/sandbox.go` — per sandbox-contract
- [x] T015 [P] [US3] Implement `NoopSandbox` (Available always true, Execute runs without isolation) in `internal/sandbox/noop.go` — for testing and fallback
- [x] T016 [US3] Implement `ProcessSandbox` with OS-level isolation in `internal/sandbox/process.go` — ulimit for memory, context timeout + process kill for CPU, tmpdir for filesystem, network blocking; platform detection for Linux/macOS/Windows fallback per research R3
- [x] T017 [US3] Integrate `Sandbox` into inline tool executor in `internal/tools/inline.go` — wrap `exec.CommandContext` with sandbox.Execute, apply `SafeEnv()` from T003, add `--sandbox` flag detection
- [x] T018 [US3] Capture inline tool stdout/stderr to buffers instead of passing to host in `internal/tools/inline.go` — return captured output, not os.Stdout/os.Stderr

### Integration Test for User Story 3

- [x] T019 [US3] Add inline sandbox tests to `integration_tests/security_test.go` — test filesystem read outside sandbox fails, network access blocked, memory limit enforced, all 4 languages (Python, Node, Bash, Ruby) sandbox uniformly (SC-004)

**Checkpoint**: Inline tools run in sandbox when enabled. Verify with `go test ./internal/sandbox/ ./internal/tools/ ./integration_tests/ -run TestSandbox -v -count=1`

---

## Phase 6: User Story 4 — Policy Engine Enforcement (Priority: P1)

**Goal**: Replace stub `checkRequirement()` that always returns true with actual validation for 4 requirement types

**Independent Test**: Define `require pinned imports` policy and attempt apply with unpinned import — must fail with specific error

### Implementation for User Story 4

- [x] T020 [US4] Add `Violation` struct (Rule, Resource, Message, Details) and `EvalMode` type (enforce/warn) to `internal/policy/policy.go` — per policy-contract
- [x] T021 [US4] Implement `checkRequirement()` with dispatch to 4 handlers in `internal/policy/policy.go` — `pinned imports`: check all References for version/SHA pin; `secret`: verify Subject exists in secret resolvers; `deny command`: match Subject against command tool configs; `signed packages`: stub with WARNING log per research R4. Collect ALL violations (not just first). Return error for unknown types.
- [x] T022 [US4] Wire evaluation mode into apply flow in `cmd/agentspec/apply.go` — add `--policy=warn` flag; in enforce mode block on violations; in warn mode log violations and proceed; format output grouped by resource with bracketed type prefix per policy-contract error format

### Integration Test for User Story 4

- [x] T023 [US4] Create `integration_tests/policy_test.go` (NEW) — test all 4 requirement types: unpinned import rejected, missing secret rejected, denied command blocked, signed packages stub warns; test --policy=warn mode proceeds with warnings; test multiple violations reported together (SC-003)

**Checkpoint**: Policy engine validates all 4 requirement types. Verify with `go test ./internal/policy/ ./integration_tests/ -run TestPolicy -v -count=1`

---

## Phase 7: User Story 5 — HTTP Server Production Hardening (Priority: P2)

**Goal**: Protect server from slow-loris, oversized bodies, idle connections, and cross-origin attacks

**Independent Test**: Send a slow-drip request (1 byte/sec) — server must time out within ReadHeaderTimeout

**Dependency**: T011 (US2) must complete before T025 starts — both modify `internal/runtime/server.go`

### Implementation for User Story 5

- [x] T024 [US5] Set `ReadHeaderTimeout` (10s), `ReadTimeout` (30s), `IdleTimeout` (120s) on `http.Server` in `internal/runtime/server.go` — per data-model ServerTimeouts
- [x] T025 [US5] Wrap all API endpoint handlers with `http.MaxBytesReader(w, r.Body, 10MB)` in `internal/runtime/server.go` — return 413 when exceeded per FR-009
- [x] T026 [US5] Replace hardcoded `Access-Control-Allow-Origin: *` with configurable origin allowlist in `internal/frontend/sse.go` — match Origin header against list, set to matched origin (not wildcard) per FR-014
- [x] T027 [US5] Add CORS middleware to server mux that checks Origin header and sets appropriate headers in `internal/runtime/server.go` — integrate with CORSConfig from data-model
- [x] T028 [US5] Add `--cors-origins` flag (comma-separated) to dev command in `cmd/agentspec/dev.go` — in dev mode auto-add `http://localhost:<port>` and `http://127.0.0.1:<port>` per research R8

**Checkpoint**: Server has timeouts, body limits, and proper CORS. Verify with `go test ./internal/runtime/ ./internal/frontend/ -v -count=1`

---

## Phase 8: User Story 6 — Tool Execution Security (Priority: P2)

**Goal**: Add command binary allowlist (block-by-default) and SSRF protection for HTTP tools

**Independent Test**: Configure command tool with unlisted binary — must fail. HTTP tool to `169.254.169.254` — must be blocked.

### Implementation for User Story 6

- [x] T029 [P] [US6] Create `IsPrivateIP(ip net.IP) bool` and `NewSafeTransport() *http.Transport` in `internal/tools/ssrf.go` — check resolved IP in DialContext against all RFC 1918/3927/loopback/IPv6 ranges per research R5 and data-model SSRFBlocklist
- [x] T030 [P] [US6] Create `ValidateBinary(binary string, allowlist []string) error` in `internal/tools/allowlist.go` — nil/empty allowlist blocks all; not-in-list vs not-found distinct errors; exact basename match per tool-security-contract
- [x] T031 [US6] Replace `http.DefaultTransport` with `NewSafeTransport()` in HTTP tool executor in `internal/tools/http.go` — SSRF protection at dial time
- [x] T032 [US6] Replace `io.ReadAll(resp.Body)` with `ReadBody(resp.Body, 10MB)` in `internal/tools/http.go` — return truncation status to caller per FR-010
- [x] T033 [US6] Add `SafeBodyString(body, contentType)` to HTTP tool response rendering in `internal/tools/http.go` — sanitize `{{`/`}}` sequences, escape HTML content per tool-security-contract
- [x] T034 [US6] Add allowlist validation and `SafeEnv()` to command tool execution in `internal/tools/command.go` — call `ValidateBinary()` before execution, replace inherited env with `SafeEnv(secrets)` per FR-012/FR-013

### Integration Test for User Story 6

- [x] T035 [US6] Extend `integration_tests/tools_test.go` with allowlist tests (no allowlist blocks all, unlisted binary rejected, listed binary allowed), SSRF tests (169.254.169.254 blocked, 10.x.x.x blocked, public IP allowed), and response size limit tests (SC-006)

**Checkpoint**: Command tools are allowlisted, HTTP tools are SSRF-protected with size limits. Verify with `go test ./internal/tools/ ./integration_tests/ -run TestTool -v -count=1`

---

## Phase 9: User Story 7 — Concurrent Access Safety (Priority: P2)

**Goal**: Fix data races in MCP connection pool and secret redaction filter

**Independent Test**: Run all tests with `-race` flag — zero data race warnings

### Implementation for User Story 7

- [x] T036 [P] [US7] Fix TOCTOU race in `Connect()` by using `sync.Map` with `singleflight.Group` for per-key deduplication in `internal/mcp/pool.go` — `LoadOrStore` atomically checks, `singleflight.Do` ensures single connection creation per research R6
- [x] T037 [P] [US7] Fix `WithAttrs()`/`WithGroup()` in `internal/secrets/redact.go` — share the parent's mutex reference instead of creating a new independent `sync.RWMutex` when cloning the filter

**Checkpoint**: No data races. Verify with `go test ./internal/mcp/ ./internal/secrets/ -race -v -count=1`

---

## Phase 10: User Story 8 — Error Transparency (Priority: P3)

**Goal**: Stop silently discarding JSON errors in LLM clients, log session save failures, capture plugin output

**Independent Test**: Send malformed JSON tool input — error must be logged at WARNING and reported to LLM

### Implementation for User Story 8

- [x] T038 [P] [US8] Replace `_ = json.Unmarshal(...)` and `_ = json.Marshal(...)` with proper error handling at lines ~140, ~174 in `internal/llm/anthropic.go` — log at WARNING, skip tool call with error message to LLM per FR-016
- [x] T039 [P] [US8] Replace `_ = json.Unmarshal(...)` and `_ = json.Marshal(...)` with proper error handling at lines ~236, ~290, ~388 in `internal/llm/openai.go` — log at WARNING, propagate error to caller per FR-016
- [x] T040 [P] [US8] Replace `WithStdout(os.Stdout).WithStderr(os.Stderr)` with buffer capture in `internal/plugins/host.go` (line ~51-53) — create `bytes.Buffer` for stdout/stderr, use after execution per FR-018
- [x] T041 [US8] Add ERROR-level logging for session save failures at `internal/runtime/server.go:265` (`_ = s.sessions.SaveMessages(...)`) and `internal/runtime/server.go:435` (`_ = s.sessions.SaveMessages(...)`) — replace `_ =` with error check, log at ERROR level via `slog.Error()` per FR-017
- [x] T042 [US8] Add `Warning` HTTP response header when session save fails in `internal/runtime/server.go` — after logging the save error (T041), set `w.Header().Set("Warning", "199 - \"session save failed\"")` on the response per US8 acceptance scenario 2

**Checkpoint**: No silent error discards. Verify with `go test ./internal/llm/ ./internal/plugins/ ./internal/session/ ./internal/runtime/ -v -count=1`

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: Validate all changes work together and no regressions

- [x] T043 Run full test suite with race detector: `go test ./... -race -count=1` — fix any remaining race conditions (SC-007)
- [x] T044 Run quickstart.md verification steps end-to-end — validate all 12 verification sections pass
- [x] T045 Audit modified files for remaining `_ = json.` or `_ = err` patterns — grep and fix any overlooked silent error discards (SC-008)
- [x] T046 Verify build succeeds cleanly: `go build -o agentspec ./cmd/agentspec` — no compilation errors or warnings

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS user stories US3 and US6 (both need SafeEnv)
- **User Stories (Phase 3–10)**: All depend on Foundational phase completion
  - P1 stories (US1–US4) can proceed in parallel
  - P2 stories (US5–US7) can proceed in parallel after P1 or concurrently if staffed
  - P3 story (US8) can proceed independently
- **Polish (Phase 11)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (Session IDs)**: Independent — no dependencies on other stories
- **US2 (Auth Hardening)**: Independent — touches `runtime/server.go` (US5 must wait for T011 to complete before starting T024)
- **US3 (Sandbox)**: Depends on T003 (SafeEnv) — no dependencies on other stories
- **US4 (Policy Engine)**: Independent — self-contained in `internal/policy/` + `cmd/agentspec/apply.go`
- **US5 (Server Hardening)**: **Depends on US2 T011** — both modify `internal/runtime/server.go`; US5 tasks on server.go (T024, T025, T027) must run after T011 completes
- **US6 (Tool Security)**: Depends on T003 (SafeEnv) — no dependencies on other stories
- **US7 (Concurrency)**: Independent — self-contained in `internal/mcp/` and `internal/secrets/`
- **US8 (Error Transparency)**: Partially depends on US5 completion for T042 (Warning header in server.go), but can start T038-T040 independently

### Within Each User Story

- Tasks marked [P] within a story can run in parallel
- Non-[P] tasks must run sequentially (they depend on prior tasks in the same story)
- Integration test tasks run AFTER all implementation tasks in the same story

### Parallel Opportunities

**Maximum parallelism (7 parallel streams after Phase 2)**:
- US1, US2, US3, US4, US6, US7 can all start in parallel
- US5 must wait for US2 T011 (server.go dependency)
- US8 T038-T040 can start in parallel with other stories; T041-T042 depend on US5 server.go work
- Within US1: T004 first, then T005+T006 in parallel, then T007
- Within US3: T014+T015 in parallel, then T016, T017, T018 sequentially, then T019
- Within US6: T029+T030 in parallel, then T031–T034 sequentially, then T035
- Within US7: T036+T037 fully parallel (different packages)
- Within US8: T038+T039+T040 fully parallel (different packages), then T041, T042

---

## Parallel Example: User Story 1

```bash
# First: create the shared ID generator
Task: "T004 [US1] Create generateSecureID() in internal/session/id.go"

# Then: update both stores in parallel (different files, same interface)
Task: "T005 [US1] Replace generateID() in internal/session/memory_store.go"
Task: "T006 [US1] Replace generateSessionID() in internal/session/redis_store.go"

# Then: integration test
Task: "T007 [US1] Add session ID security tests to integration_tests/security_test.go"
```

## Parallel Example: User Story 6

```bash
# First: create both new utility files in parallel (no dependencies)
Task: "T029 [US6] Create SSRF validator in internal/tools/ssrf.go"
Task: "T030 [US6] Create allowlist validator in internal/tools/allowlist.go"

# Then: integrate into existing tools sequentially (same file for HTTP tasks)
Task: "T031 [US6] Integrate SSRF into HTTP tool in internal/tools/http.go"
Task: "T032 [US6] Add response size limit in internal/tools/http.go"
Task: "T033 [US6] Add safe body serialization in internal/tools/http.go"
Task: "T034 [US6] Integrate allowlist into command tool in internal/tools/command.go"

# Then: integration test
Task: "T035 [US6] Extend integration_tests/tools_test.go with allowlist and SSRF tests"
```

## Parallel Example: Cross-Story

```bash
# After Phase 2, stories start (US5 waits for US2 T011):
Stream A: US1 (T004 → T005+T006 → T007)
Stream B: US2 (T008 → T009 → T010 → T011 → T012 → T013)
Stream C: US3 (T014+T015 → T016 → T017 → T018 → T019)
Stream D: US4 (T020 → T021 → T022 → T023)
Stream E: US6 (T029+T030 → T031 → T032 → T033 → T034 → T035)
Stream F: US7 (T036 + T037)
# After US2 T011 completes:
Stream G: US5 (T024 → T025 → T026 → T027 → T028)
# After US5 completes:
Stream H: US8 (T038+T039+T040 → T041 → T042)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (SafeEnv utility)
3. Complete Phase 3: User Story 1 — Secure Session IDs (including integration test T007)
4. **STOP and VALIDATE**: `go test ./internal/session/ ./integration_tests/ -run TestSession -v -count=1`
5. Highest-impact security fix shipped (predictable IDs eliminated)

### Incremental Delivery (P1 Stories)

1. Setup + Foundational → Foundation ready
2. US1 (Session IDs) → 4 tasks → Quick win
3. US2 (Auth Hardening) → 6 tasks → Brute-force protection + timing fix
4. US3 (Sandbox) → 6 tasks → Inline tool isolation
5. US4 (Policy Engine) → 4 tasks → Policy enforcement activated
6. **Milestone**: All P1 security hardening complete (23 tasks including tests)

### Full Delivery (All Stories)

7. US5 (Server Hardening) → 5 tasks → DoS protection + CORS
8. US6 (Tool Security) → 7 tasks → SSRF protection + command allowlist
9. US7 (Concurrency) → 2 tasks → Race condition fixes
10. US8 (Error Transparency) → 5 tasks → No silent error swallowing
11. Polish → 4 tasks → Final validation
12. **Milestone**: All 46 tasks complete, full security hardening shipped

---

## Summary

| Phase | Story | Priority | Tasks | Parallelizable |
|-------|-------|----------|-------|----------------|
| 1 | Setup | — | 2 | — |
| 2 | Foundational | — | 1 | — |
| 3 | US1: Session IDs | P1 | 4 | T005+T006 |
| 4 | US2: Auth Hardening | P1 | 6 | — |
| 5 | US3: Sandbox | P1 | 6 | T014+T015 |
| 6 | US4: Policy Engine | P1 | 4 | — |
| 7 | US5: Server Hardening | P2 | 5 | — |
| 8 | US6: Tool Security | P2 | 7 | T029+T030 |
| 9 | US7: Concurrency | P2 | 2 | T036+T037 |
| 10 | US8: Error Transparency | P3 | 5 | T038+T039+T040 |
| 11 | Polish | — | 4 | — |
| **Total** | | | **46** | **14 parallelizable** |

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- `runtime/server.go` is modified by US2 (T011), US5 (T024/T025/T027), and US8 (T041/T042) — US5 must wait for US2 T011; US8 T041/T042 should run after US5
- `internal/tools/http.go` has 3 tasks in US6 (T031-T033) — must be sequential (same file)
- `integration_tests/security_test.go` is created by US1 (T007) and extended by US3 (T019) — US3 test runs after US1 test
- US3 and US6 both depend on T003 (SafeEnv) from Phase 2
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
