# Tasks: Test Foundation & Quality Engineering

**Input**: Design documents from `/specs/009-test-quality-foundation/`
**Prerequisites**: plan.md (required), spec.md (required), research.md

**Tests**: Unit tests ARE the deliverable for this feature. Each user story produces test files as primary output.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Amend constitution and prepare shared test utilities.

- [x] T001 Amend constitution Testing Strategy in `.specify/memory/constitution.md` — update "Unit tests are permitted only when they unblock integration test development" to "Unit tests are a complementary quality gate alongside integration tests" per FR-012; keep "Unit-test-first development" in Non-Goals unchanged
- [x] T002 [P] Create shared test helper `internal/testutil/testutil.go` (NEW) — add `TempDir(t)`, `MustMarshalJSON(t, v)`, `AssertErrorContains(t, err, substr)` utilities used across multiple test files to reduce boilerplate

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: No foundational blocking work needed. All packages exist and are ready for testing.

**Checkpoint**: Foundation ready — user story implementation can now begin

---

## Phase 3: User Story 1 — Unit Tests for Security-Critical Packages (Priority: P1) MVP

**Goal**: Add unit tests for auth, secrets, policy, tools, session, and state packages achieving at least 80% line coverage each.

**Independent Test**: `go test ./internal/auth/... ./internal/secrets/... ./internal/policy/... ./internal/tools/... ./internal/session/... ./internal/state/... -cover -count=1`

### Implementation for User Story 1

- [x] T003 [P] [US1] Create `internal/auth/key_test.go` (NEW) — test `ValidateKey()` with matching keys (timing-safe), mismatched keys, empty keys, and `KeyFromEnv()` with set/unset env vars; minimum 4 test functions
- [x] T004 [P] [US1] Create `internal/auth/middleware_test.go` (NEW) — test auth middleware with valid Bearer token, invalid token, missing token, X-API-Key header, skipped paths (`/healthz`, `/static/*`, `/favicon.ico`), `ClientIPKeyFunc()` with X-Forwarded-For; use `httptest.NewRecorder` and `httptest.NewRequest`; minimum 6 test functions
- [x] T005 [P] [US1] Create `internal/auth/ratelimit_test.go` (NEW) — test `Allow()` with burst capacity, refill over time, token exhaustion; test `AuthFailure()` and `IsAuthBlocked()` for 10-failure blocking; test `AuthSuccess()` clears failures; test `AuthBlockRetryAfter()`; test `Middleware()` integration; test `DefaultRateLimitConfig()` and `RateLimitConfigFromEnv()`; minimum 8 test functions
- [x] T006 [P] [US1] Create `internal/secrets/env_test.go` (NEW) — test `EnvResolver.Resolve()` with `env(VAR_NAME)` format: existing var, missing var, malformed ref (no parens, wrong prefix), empty var name; minimum 4 test functions
- [x] T007 [P] [US1] Create `internal/secrets/vault_test.go` (NEW) — test `VaultResolver.Resolve()` with `vault(path#key)` format parsing, malformed refs, cache TTL behavior (mock HTTP for Vault API or test parsing logic only); test `NewVaultResolver()` defaults; minimum 4 test functions
- [x] T008 [P] [US1] Create `internal/secrets/redact_test.go` (NEW) — test `RedactFilter.AddSecret()` and `RedactString()` with single secret, multiple secrets, overlapping secrets, empty string; test `Handle()` with slog.Record containing secret values; test thread safety with concurrent `AddSecret()` and `RedactString()`; minimum 5 test functions
- [x] T009 [P] [US1] Create `internal/policy/policy_test.go` (NEW) — test `DefaultEngine.Evaluate()` with deny rules (matching/non-matching patterns), require rules (`pinned imports` with @version/@latest, `secret X` with present/absent metadata, `signed packages`), wildcard pattern matching via `matchesPattern()`, empty policies, empty resources; test `FormatViolations()` with ModeEnforce and ModeWarn; minimum 8 test functions
- [x] T010 [P] [US1] Create `internal/policy/enforce_test.go` (NEW) — test `Enforce()` converts violations to `validate.ValidationError` correctly, test with zero violations, multiple violations; minimum 3 test functions
- [x] T011 [P] [US1] Create `internal/tools/allowlist_test.go` (NEW) — test `ValidateBinary()` with allowed binary that exists on PATH, binary not in allowlist (`ErrBinaryNotAllowed`), binary in allowlist but not found (`ErrBinaryNotFound`), empty allowlist (`ErrNoAllowlist`); minimum 4 test functions
- [x] T012 [P] [US1] Create `internal/tools/command_test.go` (NEW) — test `CommandExecutor.Execute()` with echo command (valid input → stdout output), command with timeout, command that fails (nonzero exit → error with stderr), binary not in allowlist; test `NewCommandExecutor()` with config; minimum 5 test functions
- [x] T013 [P] [US1] Create `internal/tools/http_test.go` (NEW) — test `HTTPExecutor.Execute()` with mock HTTP server (GET, POST with body template), test `ReadBody()` with normal body, oversized body (>10MB), empty body; test `SafeBodyString()` with HTML content type (escapes `{{`/`}}`), plain text; minimum 6 test functions
- [x] T014 [P] [US1] Create `internal/tools/ssrf_test.go` (NEW) — test `IsPrivateIP()` with RFC 1918 ranges (10.x, 172.16.x, 192.168.x), loopback (127.0.0.1), link-local (169.254.x), IPv6 equivalents (::1, fc00::, fe80::), and valid public IPs; test `NewSafeTransport()` blocks private IPs; minimum 6 test functions
- [x] T015 [P] [US1] Create `internal/tools/env_test.go` (NEW) — test `SafeEnv()` includes PATH and HOME, includes provided secrets, excludes other env vars; minimum 3 test functions
- [x] T015b [P] [US1] Create `internal/tools/inline_test.go` (NEW) — test inline executor: execution with valid input, error handling, input validation; minimum 3 test functions
- [x] T016 [P] [US1] Create `internal/session/memory_store_test.go` (NEW) — test `MemoryStore` CRUD: Create/Get/Delete/List/Touch, test with agent name filter, test expiry (set short expiry → expired session not returned), test `generateSecureID()` uniqueness and "sess_" prefix; minimum 6 test functions
- [x] T017 [P] [US1] Create `internal/session/session_test.go` (NEW) — test `Manager` with mock Store and mock memory.Store: Create, LoadMessages, SaveMessages, Close, Get, List; minimum 5 test functions
- [x] T018 [P] [US1] Create `internal/state/local_test.go` (NEW) — test `LocalBackend` unit tests (complementing existing integration tests): `NewLocalBackend()` defaults, `WithLockConfig()` partial updates, `Get()` found and not-found, `List()` with and without status filter, `DefaultLockConfig()` values, `ErrStateCorrupted.Error()` messages, `ErrStateLocked.Error()` messages; minimum 6 test functions

### Coverage Verification for User Story 1

- [x] T019 [US1] Verify each security-critical package achieves >=80% line coverage — run `go test ./internal/{auth,secrets,policy,tools,session,state}/... -coverprofile` for each, add missing tests if any package is below 80%; this task runs after all T003-T018 complete

**Checkpoint**: Security-critical packages have comprehensive unit tests at 80%+ coverage. Verify with `go test ./internal/auth/... ./internal/secrets/... ./internal/policy/... ./internal/tools/... ./internal/session/... ./internal/state/... -cover -count=1`

---

## Phase 4: User Story 2 — Automated Security Scanning in CI (Priority: P1)

**Goal**: Add govulncheck and gosec to CI pipeline.

**Independent Test**: Push a branch and verify CI security scanning jobs run and pass.

### Implementation for User Story 2

- [x] T020 [US2] Add `security-scan` job to `.github/workflows/ci.yml` (MODIFY) — new job that runs in parallel with `build-and-test`: install `govulncheck` at a pinned version (e.g., `govulncheck@v1.1.4`), run `govulncheck ./...`; add `gosec` to golangci-lint run (or separate step with `--enable gosec`); job fails if vulnerabilities found per FR-002, FR-003, Constitution Principle XI (explicit references)
- [x] T021 [US2] Enable `gosec` linter in `.golangci.yml` (MODIFY) — add `gosec` to enabled linters list; configure exclusions for known-safe patterns (e.g., `G104` for deferred close errors already handled); fix any existing gosec violations in codebase per research R3

**Checkpoint**: CI security scanning detects vulnerabilities and insecure patterns. Verify with `govulncheck ./...` and `golangci-lint run --enable gosec ./...`

---

## Phase 5: User Story 3 — Test Coverage Visibility and Thresholds (Priority: P2)

**Goal**: Add coverage reporting and threshold enforcement to CI.

**Independent Test**: Run coverage locally and verify threshold check passes.

### Implementation for User Story 3

- [x] T022 [US3] Add `coverage` job to `.github/workflows/ci.yml` (MODIFY) — depends on `build-and-test`; runs `go test ./... -coverprofile=coverage.out -covermode=atomic`, runs `go tool cover -func=coverage.out`, parses total coverage percentage, compares against `COVERAGE_THRESHOLD` env var (initial value: 50), fails if below threshold, uploads `coverage.out` as artifact per FR-004, FR-005, research R9
- [x] T023 [US3] Add `COVERAGE_THRESHOLD` env variable to `.github/workflows/ci.yml` (MODIFY) — set at workflow level: `env: COVERAGE_THRESHOLD: 50`; add comment explaining manual ratchet process per clarification answer

**Checkpoint**: Coverage threshold enforced in CI. Verify locally: `go test ./... -coverprofile=coverage.out -covermode=atomic && go tool cover -func=coverage.out | grep total:`

---

## Phase 6: User Story 4 — Race Condition Detection in CI (Priority: P2)

**Goal**: Add race detection job to CI pipeline.

**Independent Test**: Run `go test ./... -race -count=1` locally and verify zero races.

### Implementation for User Story 4

- [x] T024 [US4] Add `race-detection` job to `.github/workflows/ci.yml` (MODIFY) — runs in parallel with `coverage` job; runs `go test ./... -race -count=1 -timeout=15m`; uses separate timeout (15m) from main test job (10m) per FR-006, research R5
- [x] T025 [US4] Verify zero existing data races — run `go test ./... -race -count=1 -timeout=15m` locally; fix any races found in existing code (likely in rate limiter, session store, or memory store)

**Checkpoint**: Race detection passes with zero races. Verify with `go test ./... -race -count=1`

---

## Phase 7: User Story 5 — Comprehensive Linting (Priority: P3)

**Goal**: Enable additional linters for HTTP body closure, context usage, and code correctness.

**Independent Test**: Run `golangci-lint run ./...` and verify zero violations.

### Implementation for User Story 5

- [x] T026 [US5] Enable new linters in `.golangci.yml` (MODIFY) — add `bodyclose`, `noctx`, `contextcheck`, `gocritic`, `unconvert`, `misspell` to enabled linters per FR-007, research R4; configure reasonable exclusions for any linters that produce excessive noise
- [x] T027 [US5] Fix `bodyclose` violations in codebase — search for unclosed HTTP response bodies across `internal/tools/http.go`, `internal/mcp/`, `internal/runtime/server.go`, and any other files making HTTP requests; add `defer resp.Body.Close()` where missing
- [x] T028 [US5] Fix `noctx` violations in codebase — search for HTTP requests not using `context.Context` (e.g., `http.Get()`, `http.Post()` without `http.NewRequestWithContext()`); replace with context-aware versions
- [x] T029 [US5] Fix remaining new linter violations — run `golangci-lint run ./...` with new config, fix `contextcheck`, `gocritic`, `unconvert`, `misspell` violations; use `//nolint:` with justification for intentional suppressions only

**Checkpoint**: All linters pass with zero violations. Verify with `golangci-lint run ./...`

---

## Phase 8: User Story 6 — Code Deduplication (Priority: P3)

**Goal**: Consolidate duplicate rate limiters, session ID generators, and replace linear scans with indexed lookups.

**Independent Test**: Verify single implementations via grep and all tests pass.

### Implementation for User Story 6

- [x] T030 [US6] Consolidate rate limiter — remove `rateLimiter`, `tokenBucket`, `newRateLimiter()`, `allow()` from `internal/runtime/server.go` (lines ~746-820); import and use `auth.NewRateLimiter()` with per-agent key function; update `rateLimitMiddleware()` to use `auth.RateLimiter.Allow()` per FR-008, research R6
- [x] T031 [US6] Consolidate session ID generator — export `GenerateID(prefix string) string` from `internal/session/id.go`; replace `generateID()` in `internal/telemetry/traces.go` with `session.GenerateID("tr_")`; replace inline ID in `internal/runtime/server.go` line ~673 with `session.GenerateID("cf_")`; replace inline ID in `internal/telemetry/logger.go` with `session.GenerateID("cor_")` per FR-009, research R7
- [x] T032 [US6] Add indexed agent/pipeline lookups — add `agentsByName map[string]*AgentConfig` and `pipelinesByName map[string]*PipelineConfig` fields to `Server` struct in `internal/runtime/server.go`; populate maps in `NewServer()`; replace `findAgent()` calls (6 locations) with map lookup `s.agentsByName[name]`; replace `findPipeline()` calls with `s.pipelinesByName[name]`; delete `findAgent()` and `findPipeline()` methods per FR-010, research R8
- [x] T033 [US6] Add unit tests for consolidated components — test `session.GenerateID()` with various prefixes in `internal/session/id_test.go` (extend T016); test indexed lookups work in `internal/runtime/server.go` (existing or new test); verify rate limiter used by runtime per FR-008

**Checkpoint**: Zero duplicate implementations. Verify with `grep -rn "type.*rateLimiter" internal/` (only auth package) and `grep -rn "func.*generateID\|func.*GenerateID" internal/` (only session package)

---

## Phase 9: User Story 1b — Unit Tests for Additional Packages (Priority: P2)

**Goal**: Add unit tests for loop, pipeline, llm, expr, memory, mcp, ir, validate, and compiler packages achieving at least 60% line coverage each.

**Independent Test**: `go test ./internal/loop/... ./internal/pipeline/... ./internal/llm/... ./internal/expr/... ./internal/memory/... ./internal/mcp/... ./internal/ir/... ./internal/validate/... ./internal/compiler/... -cover -count=1`

### Implementation for User Story 1b

- [x] T034 [P] [US1b] Create `internal/ir/ir_test.go` (NEW) — test IR types: Resource, Policy, Config struct construction and field validation; test JSON marshaling/unmarshaling round-trip; minimum 4 test functions
- [x] T035 [P] [US1b] Create `internal/validate/validate_test.go` (NEW) — test validation rules: required fields, type checks, reference resolution, error formatting with source location; minimum 5 test functions
- [x] T036 [P] [US1b] Create `internal/expr/expr_test.go` (NEW) — test expression evaluator: arithmetic, string interpolation, boolean logic, variable substitution, error on invalid expressions; minimum 5 test functions
- [x] T037 [P] [US1b] Create `internal/memory/memory_test.go` (NEW) — test memory Store interface: store/retrieve/delete entries, test with different memory backends if applicable; minimum 4 test functions
- [x] T038 [P] [US1b] Create `internal/llm/llm_test.go` (NEW) — test LLM types: Message construction, role validation; test client configuration; test request/response marshaling; minimum 4 test functions
- [x] T039 [P] [US1b] Create `internal/mcp/mcp_test.go` (NEW) — test MCP client: connection setup, tool listing, tool invocation request formatting, error handling; minimum 4 test functions
- [x] T040 [P] [US1b] Create `internal/loop/loop_test.go` (NEW) — test agent loop: message flow, tool call handling, max iterations, stop conditions; minimum 5 test functions
- [x] T041 [P] [US1b] Create `internal/pipeline/pipeline_test.go` (NEW) — test pipeline execution: step ordering, input/output passing between steps, error propagation, DAG validation; minimum 4 test functions
- [x] T042 [P] [US1b] Create `internal/compiler/compiler_test.go` (NEW) — test compiler: AST to IR compilation, resource resolution, import handling, error messages with source location; minimum 5 test functions
- [x] T043 [US1b] Verify each additional package achieves >=60% line coverage — run `go test ./internal/{ir,validate,expr,memory,llm,mcp,loop,pipeline,compiler}/... -coverprofile` for each, add missing tests if any package is below 60%

**Checkpoint**: Additional packages have unit tests at 60%+ coverage. Verify with `go test ./internal/loop/... ./internal/pipeline/... ./internal/llm/... ./internal/expr/... ./internal/memory/... ./internal/mcp/... ./internal/ir/... ./internal/validate/... ./internal/compiler/... -cover -count=1`

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Validate all changes work together and no regressions.

- [x] T044 Run full test suite with race detector: `go test ./... -race -count=1 -timeout=15m` — fix any remaining race conditions
- [x] T045 Run `make pre-commit` — verify all checks pass (fmt, vet, lint, test, build, validate)
- [x] T046 Run quickstart.md verification steps end-to-end — validate all 12 verification sections pass
- [x] T047 Verify build succeeds cleanly: `go build -o agentspec ./cmd/agentspec` — no compilation errors

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: No work needed
- **US1 (Phase 3)**: Depends on Phase 1 (T001 constitution amendment)
- **US2 (Phase 4)**: Independent — CI config only
- **US3 (Phase 5)**: Sequential after US2 — modifies same `.github/workflows/ci.yml`
- **US4 (Phase 6)**: Sequential after US3 — modifies same `.github/workflows/ci.yml`
- **US5 (Phase 7)**: Depends on US2 (T021 adds gosec; T026 adds more linters to same file)
- **US6 (Phase 8)**: Independent — different files from US1-US5
- **US1b (Phase 9)**: Depends on US1 (T002 shared test helpers)
- **Polish (Phase 10)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (Security-Critical Tests)**: Depends on T001 (constitution) and T002 (test helpers)
- **US2 (Security Scanning)**: Independent — CI config only (modifies `.github/workflows/ci.yml`)
- **US3 (Coverage)**: Sequential after US2 — modifies same `.github/workflows/ci.yml`; benefits from running after US1 adds tests
- **US4 (Race Detection)**: Sequential after US3 — modifies same `.github/workflows/ci.yml`
- **US5 (Linting)**: Should run after US2 (gosec already added); modifies same `.golangci.yml`
- **US6 (Deduplication)**: Independent — modifies runtime/server.go and session/id.go
- **US1b (Additional Tests)**: Should run after US1 (shared patterns established)

### Within Each User Story

- Tasks marked [P] within a story can run in parallel
- Non-[P] tasks must run sequentially
- Coverage verification tasks (T019, T043) run AFTER all test implementation tasks in the same story

### Parallel Opportunities

**Maximum parallelism (4 parallel streams after Phase 1)**:
- Stream A: US1 (T003-T019) → US1b (T034-T043)
- Stream B: US2 (T020-T021) → US5 (T026-T029)
- Stream C: US3 (T022-T023) + US4 (T024-T025)
- Stream D: US6 (T030-T033)
- After all streams: Polish (T044-T047)

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T002)
2. Complete Phase 3: User Story 1 — Security-Critical Package Tests (T003-T019)
3. **STOP and VALIDATE**: `go test ./internal/auth/... ./internal/secrets/... ./internal/policy/... ./internal/tools/... ./internal/session/... ./internal/state/... -cover -count=1`
4. Highest-impact quality improvement shipped (security packages tested at 80%+)

### Incremental Delivery (P1 Stories)

1. Setup → Constitution amended, test helpers ready
2. US1 (Security Tests) → 17 tasks → 6 packages at 80% coverage
3. US2 (Security Scanning) → 2 tasks → govulncheck + gosec in CI
4. **Milestone**: Both P1 stories complete (19 tasks)

### Full Delivery (All Stories)

5. US3 (Coverage) → 2 tasks → Threshold enforcement in CI
6. US4 (Race Detection) → 2 tasks → Race detector in CI
7. US5 (Linting) → 4 tasks → Comprehensive linter config
8. US6 (Deduplication) → 4 tasks → Single implementations
9. US1b (Additional Tests) → 10 tasks → 9 more packages at 60% coverage
10. Polish → 4 tasks → Final validation
11. **Milestone**: All 48 tasks complete, full quality foundation shipped

---

## Summary

| Phase | Story | Priority | Tasks | Parallelizable |
|-------|-------|----------|-------|----------------|
| 1 | Setup | — | 2 | T002 parallel |
| 2 | Foundational | — | 0 | — |
| 3 | US1: Security-Critical Tests | P1 | 18 | T003-T018+T015b all parallel |
| 4 | US2: Security Scanning | P1 | 2 | — |
| 5 | US3: Coverage Thresholds | P2 | 2 | — |
| 6 | US4: Race Detection | P2 | 2 | — |
| 7 | US5: Comprehensive Linting | P3 | 4 | — |
| 8 | US6: Code Deduplication | P3 | 4 | — |
| 9 | US1b: Additional Package Tests | P2 | 10 | T034-T042 all parallel |
| 10 | Polish | — | 4 | — |
| **Total** | | | **48** | **4 parallel streams** |

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- US1 test files (T003-T018) are all in different packages — maximum parallelism
- US1b test files (T034-T042) are all in different packages — maximum parallelism
- `.github/workflows/ci.yml` is modified by US2, US3, US4 — sequential within CI changes
- `.golangci.yml` is modified by US2 (T021) and US5 (T026) — US5 depends on US2
- `internal/runtime/server.go` is modified by US6 only
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
