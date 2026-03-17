# Implementation Plan: Memory & Performance Management

**Branch**: `010-memory-performance` | **Date**: 2026-03-04 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/010-memory-performance/spec.md`

## Summary

Fix unbounded memory growth in rate limiters, session stores, and conversation memory by adding eviction policies with configurable limits and TTLs. Improve performance of state file lookups (caching + indexing), Redis session operations (SCAN + RPUSH), DAG topological sort (Kahn's O(V+E)), and agent/pipeline lookups (map indexing). Add ULID-formatted correlation IDs as HTTP middleware for end-to-end request tracing. Emit structured log counters for all eviction and cache operations.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `log/slog` (structured logging), `sync` (RWMutex), `oklog/ulid` (correlation IDs), existing Redis client interface, `encoding/json` (state file)
**Storage**: Local JSON state file (`.agentspec.state.json`), Redis (session store, opt-in), in-memory maps (rate limiter, session store default, conversation memory)
**Testing**: `go test ./... -count=1`, integration tests in `integration_tests/`
**Target Platform**: Linux/macOS server (CLI + HTTP runtime)
**Project Type**: CLI + embedded HTTP server
**Performance Goals**: State cache 10x speedup, DAG sort <10ms for 100 steps, constant-time agent lookup, stable memory over 24h
**Constraints**: No new external dependencies beyond ULID library, single-instance Redis only, per-process caching only
**Scale/Scope**: 10,000 unique clients, 10,000 sessions, 500 resources in state file, 100-step pipelines

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Determinism | PASS | Eviction order is deterministic (oldest-first, LRU). State cache reads produce same results as disk reads. |
| II. Idempotency | PASS | Eviction is idempotent — running twice removes nothing extra. Cache invalidation on save preserves idempotency of apply. |
| III. Portability | PASS | No platform-specific changes. Redis remains opt-in. |
| IV. Separation of Concerns | PASS | Eviction policy is a separate concern from store logic. Correlation ID middleware is isolated from business logic. |
| V. Reproducibility | PASS | No change to build or export determinism. |
| VI. Safe Defaults | PASS | Default limits (10,000) prevent unbounded growth out of the box. |
| VII. Minimal Surface Area | PASS | No new CLI commands or DSL keywords. Configuration via existing patterns (env vars, constructor options). |
| VIII. English-Friendly Syntax | N/A | No DSL changes. |
| IX. Canonical Formatting | N/A | No DSL changes. |
| X. Strict Validation | PASS | Eviction config validated at construction time. |
| XI. Explicit References | N/A | No dependency resolution changes. |
| XII. No Hidden Behavior | PASS | Eviction emits structured log entries. Cache behavior is observable via hit/miss counters. |
| Observability (Operational) | PASS | Correlation IDs satisfy "every operation MUST carry a correlation ID". Structured log counters for eviction/cache satisfy metrics requirement. |

**Gate result: PASS** — No violations. All changes align with constitution principles.

## Project Structure

### Documentation (this feature)

```text
specs/010-memory-performance/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
internal/
├── eviction/
│   └── policy.go              # Shared EvictionPolicy struct (used by auth, session, memory packages)
├── auth/
│   └── ratelimit.go           # FR-001, FR-002, FR-012, FR-015: Add eviction, max cap, metrics, RWMutex
├── session/
│   ├── store.go               # No changes (interface stable)
│   ├── memory_store.go        # FR-003, FR-013, FR-015: Background eviction goroutine, metrics, RWMutex
│   └── redis_store.go         # FR-005, FR-006, FR-013: SCAN iteration, RPUSH append, metrics
├── memory/
│   ├── memory.go              # No changes (interface stable)
│   ├── sliding.go             # FR-004, FR-015: Max session count with LRU eviction, RWMutex
│   └── summary.go             # FR-004, FR-015: Max session count with LRU eviction, RWMutex
├── state/
│   ├── state.go               # No changes (interface stable)
│   └── local.go               # FR-007, FR-008, FR-014: In-memory cache, FQN index, hit/miss counters
├── pipeline/
│   └── dag.go                 # FR-010: Replace O(V²) sort with O(V+E) Kahn's using adjacency list
├── runtime/
│   └── server.go              # FR-009, FR-011: Correlation ID middleware, agent/pipeline index maps
├── telemetry/
│   ├── logger.go              # Minor: Ensure RequestLogger includes correlation ID consistently
│   └── correlation.go         # FR-009: ULID generation, HTTP middleware for correlation ID injection
└── events/
    └── events.go              # No changes

integration_tests/
├── telemetry_test.go          # Update rate limiter tests for eviction behavior
├── session_test.go            # Add session eviction and Redis SCAN tests
├── state_test.go              # Add cache hit/miss benchmark tests
└── pipeline_test.go           # Add DAG sort performance benchmark
```

**Structure Decision**: This feature modifies existing packages in-place. Two new files are created: `internal/eviction/policy.go` (shared EvictionPolicy struct to avoid import cycles between auth/session/memory) and `internal/telemetry/correlation.go` (ULID middleware). The existing package boundaries are well-suited for the remaining changes.

## Complexity Tracking

> No constitution violations to justify.

No violations detected. All changes use existing patterns (mutex-protected maps, slog structured logging, HTTP middleware chain).
