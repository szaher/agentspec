# Implementation Plan: Test Foundation & Quality Engineering

**Branch**: `009-test-quality-foundation` | **Date**: 2026-03-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/009-test-quality-foundation/spec.md`

## Summary

Address quality engineering gaps: add unit tests for 15 internal packages (6 security-critical at 80% coverage, 9 additional at 60%), add CI security scanning (govulncheck + gosec), coverage reporting with manual threshold ratchet (50%→70%), race detection in CI, comprehensive linting, and code deduplication (rate limiter, session ID generator, indexed lookups). Amend project constitution to recognize unit tests as a complementary quality gate.

## Technical Context

**Language/Version**: Go 1.25+ (existing)
**Primary Dependencies**: golangci-lint v2.10.1 (existing), govulncheck (new), gosec (new via golangci-lint)
**Storage**: N/A (testing/CI infrastructure only)
**Testing**: `go test` with `-race`, `-coverprofile`, `-covermode=atomic`
**Target Platform**: Linux (CI), macOS/Linux (development)
**Project Type**: CLI tool with HTTP runtime server
**Performance Goals**: CI pipeline completes in <10 minutes; race detection job <15 minutes
**Constraints**: No new external services; coverage and security tools run in GitHub Actions
**Scale/Scope**: 15 internal packages, ~38 directories total, 1 CI workflow file

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Determinism | PASS | Tests are deterministic; coverage thresholds are fixed values |
| II. Idempotency | PASS | N/A — no apply/state changes |
| III. Portability | PASS | N/A — CI/testing infrastructure |
| IV. Separation of Concerns | PASS | Tests colocated with packages per Go convention |
| V. Reproducibility | PASS | Linter versions pinned; govulncheck pinned to specific version in CI (e.g., v1.1.4) |
| VI. Safe Defaults | PASS | No secret handling changes |
| VII. Minimal Surface Area | PASS | No new keywords or constructs |
| VIII. English-Friendly Syntax | PASS | N/A |
| IX. Canonical Formatting | PASS | N/A |
| X. Strict Validation | PASS | N/A |
| XI. Explicit References | PASS | Tool versions pinned in CI |
| XII. No Hidden Behavior | PASS | N/A |
| Testing Strategy | **AMENDMENT REQUIRED** | Constitution says "Unit tests are permitted only when they unblock integration test development." This feature amends the Testing Strategy to recognize unit tests as a complementary quality gate (FR-012). |
| Non-Goals | **COMPATIBLE** | "Unit-test-first development" remains a non-goal. This feature adds tests retroactively, not as TDD. |

**Gate Result**: PASS with one planned amendment (FR-012). The amendment is an explicit deliverable of this feature.

## Project Structure

### Documentation (this feature)

```text
specs/009-test-quality-foundation/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── quickstart.md        # Verification steps
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
# Unit test files (NEW — colocated with packages per Go convention)
internal/auth/
├── key_test.go              # Key validation tests
├── middleware_test.go       # Auth middleware tests
└── ratelimit_test.go        # Rate limiter tests

internal/secrets/
├── env_test.go              # Env resolver tests
├── vault_test.go            # Vault resolver tests
└── redact_test.go           # Redaction filter tests

internal/policy/
├── policy_test.go           # Policy engine tests
└── enforce_test.go          # Enforcement tests

internal/tools/
├── allowlist_test.go        # Binary allowlist tests
├── command_test.go          # Command executor tests
├── http_test.go             # HTTP executor tests
├── ssrf_test.go             # SSRF protection tests
├── inline_test.go           # Inline executor tests
└── env_test.go              # Safe env tests

internal/session/
├── memory_store_test.go     # Memory store tests
├── redis_store_test.go      # Redis store tests (with mock)
├── session_test.go          # Manager tests
└── id_test.go               # ID generation tests

internal/state/
└── local_test.go            # LocalBackend tests (atomic write, locking)

internal/testutil/
└── testutil.go              # Shared test helpers (TempDir, MustMarshalJSON, AssertErrorContains)

internal/loop/
└── loop_test.go             # Agent loop tests
internal/pipeline/
└── pipeline_test.go         # Pipeline execution tests
internal/llm/
└── llm_test.go              # LLM client tests
internal/expr/
└── expr_test.go             # Expression evaluator tests
internal/memory/
└── memory_test.go           # Memory store tests
internal/mcp/
└── mcp_test.go              # MCP client tests
internal/ir/
└── ir_test.go               # IR types tests
internal/validate/
└── validate_test.go         # Validation tests
internal/compiler/
└── compiler_test.go         # Compiler tests

# CI configuration (MODIFIED)
.github/workflows/ci.yml    # Add security, coverage, race detection jobs

# Linter configuration (MODIFIED)
.golangci.yml                # Add gosec, bodyclose, noctx, contextcheck linters

# Constitution (MODIFIED)
.specify/memory/constitution.md  # Amend Testing Strategy section

# Deduplication (MODIFIED)
internal/auth/ratelimit.go       # Canonical rate limiter (enhanced)
internal/runtime/server.go       # Use auth.RateLimiter, indexed lookups
internal/session/id.go           # Canonical ID generator (enhanced)
internal/telemetry/traces.go     # Use session ID generator or proper trace IDs
```

**Structure Decision**: Unit tests are colocated with their packages following Go convention (`*_test.go` in the same directory). One new package `internal/testutil/` provides shared test helpers to reduce boilerplate. CI jobs are added to the existing workflow file. Code deduplication consolidates into existing canonical locations. FR-011 (additional package tests at 60%) is organized as "US1b" in tasks.md — a continuation of US1 covering non-security packages.

## Deduplication Strategy

### Rate Limiter Consolidation

**Current state**: Two nearly identical token bucket implementations:
- `internal/auth/ratelimit.go` — exported `RateLimiter` with `RateLimitConfig`, auth failure tracking, middleware
- `internal/runtime/server.go` (lines 746-820) — unexported `rateLimiter` with inline rate/burst fields

**Target**: Single implementation in `internal/auth/ratelimit.go`. The `runtime/server.go` rate limiter is replaced by importing and using `auth.RateLimiter` with a per-agent key function.

### Session ID Generator Consolidation

**Current state**: Four different ID generation approaches:
1. `internal/session/id.go` — `generateSecureID()` using `crypto/rand` + base64 (128-bit, "sess_" prefix)
2. `internal/telemetry/logger.go` — inline `crypto/rand` + hex for correlation IDs
3. `internal/telemetry/traces.go` — `generateID()` using `time.Now().UnixNano()` (NOT cryptographically secure)
4. `internal/runtime/server.go` line 673 — inline `fmt.Sprintf("cf_%s_%d", ...)` with `time.Now().UnixNano()`

**Target**: Export `GenerateSecureID()` from `internal/session/id.go` (or create a shared `internal/id/` package). Replace all inline ID generation with the canonical function. The `telemetry/traces.go` implementation using `time.Now().UnixNano()` with `time.Sleep(time.Nanosecond)` is both insecure and slow — must be replaced.

### Indexed Lookups

**Current state**: Linear scans in `internal/runtime/server.go`:
- `findAgent(name)` — O(n) scan of `s.config.Agents` slice, called 6 times
- `findPipeline(name)` — O(n) scan of `s.config.Pipelines` slice, called 1 time

**Target**: Add `agentsByName map[string]*AgentConfig` and `pipelinesByName map[string]*PipelineConfig` maps, populated during `NewServer()` initialization. Replace `findAgent`/`findPipeline` with direct map lookups.

## CI Pipeline Design

### New Jobs (added to existing `.github/workflows/ci.yml`)

```text
Jobs (execution order):
├── build-and-test (EXISTING — enhanced)
│   ├── ... existing steps ...
│   └── (add) Upload coverage artifact
├── security-scan (NEW — parallel with build-and-test)
│   ├── govulncheck ./...
│   └── golangci-lint run (gosec-only config or --enable gosec)
├── coverage (NEW — depends on build-and-test)
│   ├── go test ./... -coverprofile=coverage.out -covermode=atomic
│   ├── go tool cover -func=coverage.out
│   └── Check threshold (50% initial)
└── race-detection (NEW — parallel with coverage)
    └── go test ./... -race -count=1 -timeout=15m
```

### Coverage Threshold Enforcement

The coverage threshold is stored as a CI variable or hardcoded value in the workflow. Engineers manually update it when ready to raise the bar. Initial: 50%.

```yaml
env:
  COVERAGE_THRESHOLD: 50
```

### Linter Additions

New linters to enable in `.golangci.yml`:
- `gosec` — security-focused static analysis (FR-003)
- `bodyclose` — checks HTTP response body closure (FR-007)
- `noctx` — checks HTTP requests made without context (FR-007)
- `contextcheck` — checks context propagation (FR-007)
- `gocritic` — broad code correctness checks
- `unconvert` — unnecessary type conversions
- `misspell` — common spelling mistakes in comments/strings

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Constitution amendment (Testing Strategy) | Unit tests needed as complementary quality gate for 33/34 untested packages | "Integration tests only" policy is insufficient for isolated function-level testing of security-critical code (auth, secrets, policy) |
