# Tasks: Memory & Performance Management

**Input**: Design documents from `/specs/010-memory-performance/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add new dependency and project initialization

- [x] T001 Add `github.com/oklog/ulid/v2` dependency via `go get github.com/oklog/ulid/v2` in go.mod

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared data structures used by multiple user stories. MUST be complete before any user story work begins.

- [x] T002 Create EvictionPolicy struct with MaxEntries, TTL, EvictionInterval fields, DefaultPolicy() constructor, and Validate() method in internal/eviction/policy.go per contracts/eviction-policy.go and data-model.md (shared package — avoids import cycles between auth, session, and memory packages)
- [x] T003 Create LRU session tracker helper using `container/list` and `map[string]*list.Element` with Promote(), Evict(), Len() methods in internal/memory/lru.go per data-model.md ConversationMemorySession entity

**Checkpoint**: Foundation ready — user story implementation can now begin in parallel

---

## Phase 3: User Story 1 — Bounded Memory Usage Under Load (Priority: P1)

**Goal**: Server memory usage remains stable over time — rate limiter buckets, expired sessions, and conversation histories are evicted when stale or when limits are exceeded.

**Independent Test**: Simulate 10,000 unique clients, wait past eviction interval, verify memory usage returns to baseline.

### Implementation for User Story 1

- [x] T004 [P] [US1] Add `lastAccess time.Time` field to bucket struct, replace `sync.Mutex` with `sync.RWMutex`, update `Allow()` to track access time and use RLock for existing bucket lookup in internal/auth/ratelimit.go per research.md R-001
- [x] T005 [US1] Add EvictionPolicy field to RateLimiter, add `Start(ctx context.Context)` method launching background eviction goroutine that removes buckets exceeding TTL, enforce MaxEntries cap on bucket creation (oldest-first eviction) in internal/auth/ratelimit.go per FR-001, FR-002 (depends on T004)
- [x] T006 [US1] Add structured log emission (eviction count, remaining buckets) to rate limiter eviction cycle using slog in internal/auth/ratelimit.go per FR-012
- [x] T007 [US1] Consolidate per-agent rate limiter in internal/runtime/server.go to use the shared RateLimiter from internal/auth/ratelimit.go — remove duplicate rateLimiter struct and tokenBucket struct from server.go, update WithRateLimit option and rateLimitMiddleware to use auth.RateLimiter per research.md R-001
- [x] T008 [P] [US1] Replace `sync.Mutex` with `sync.RWMutex` in MemoryStore, add `Start(ctx context.Context)` method launching background cleanup goroutine on configurable interval, proactively evict expired sessions on each cycle in internal/session/memory_store.go per FR-003, FR-015
- [x] T009 [US1] Add structured log emission (eviction count, active session count) to MemoryStore cleanup cycle using slog in internal/session/memory_store.go per FR-013
- [x] T010 [P] [US1] Add LRU session tracking to SlidingWindow using internal/memory/lru.go helper, enforce max session count (default 10,000), evict LRU session on overflow, replace `sync.Mutex` with `sync.RWMutex`, promote session on Save()/Load() in internal/memory/sliding.go per FR-004, FR-015
- [x] T011 [P] [US1] Add LRU session tracking to Summary using internal/memory/lru.go helper, enforce max session count (default 10,000), evict LRU session on overflow, replace `sync.Mutex` with `sync.RWMutex`, promote session on Save()/Load() in internal/memory/summary.go per FR-004, FR-015
- [x] T012 [US1] Add structured log emission (evicted count, remaining, max) to conversation memory eviction in both internal/memory/sliding.go and internal/memory/summary.go per FR-013
- [x] T013 [US1] Wire eviction goroutine lifecycle into runtime — call `rateLimiter.Start(ctx)` and `sessionStore.Start(ctx)` during server startup in internal/runtime/runtime.go, ensure goroutines stop on context cancellation

- [x] T013A [US1] Add integration test verifying rate limiter eviction — create many unique clients, wait past TTL, verify buckets are reclaimed in integration_tests/telemetry_test.go

**Checkpoint**: Rate limiter, session store, and conversation memory all bound memory with configurable limits and background eviction. Metrics emitted via structured logs.

---

## Phase 4: User Story 2 — Efficient Session Store Operations (Priority: P1)

**Goal**: Redis session listing uses cursor-based SCAN instead of blocking KEYS. Message append is O(1) via Redis List instead of full-array reload.

**Independent Test**: Create 10,000 sessions in Redis, list without blocking. Append message to session with 500 messages in constant time.

### Implementation for User Story 2

- [x] T014 [US2] Extend RedisClient interface with Scan(ctx, cursor, pattern, count), RPush(ctx, key, values...), and LRange(ctx, key, start, stop) methods in internal/session/redis_store.go per contracts/redis-client-interface.go
- [x] T015 [US2] Replace `Keys()` call in RedisStore.List() with cursor-based SCAN iteration loop in internal/session/redis_store.go per FR-005
- [x] T016 [US2] Replace SaveMessages() full-array reload+rewrite with RPush() for O(1) message append, replace LoadMessages() JSON-array Get() with LRange() for Redis List retrieval in internal/session/redis_store.go per FR-006
- [x] T017 [US2] Add transparent migration in LoadMessages() — detect String-type key (legacy JSON array), read it, delete it, and re-store as List elements via RPush for next access in internal/session/redis_store.go per FR-016, data-model.md Redis Key Schema migration note
- [x] T018 [US2] Add structured log emission (active session count, eviction count) to RedisStore cleanup operations using slog in internal/session/redis_store.go per FR-013
- [x] T019 [US2] Update MemoryStore List() to also clean expired sessions during listing operation (lazy eviction on list) in internal/session/memory_store.go per acceptance scenario US2-3 (depends on T008 completing RWMutex migration in same file)

- [x] T019A [US2] Add integration test verifying session eviction and Redis SCAN — verify background cleanup removes expired sessions, verify List() uses SCAN not KEYS in integration_tests/session_test.go

**Checkpoint**: Redis operations are non-blocking. Message append is O(1). Memory store list cleans expired sessions.

---

## Phase 5: User Story 3 — Fast State File Access (Priority: P2)

**Goal**: State file Get() uses in-memory cache with FQN index. Second+ calls skip disk I/O. Cache invalidates on Save() and external file modification.

**Independent Test**: Measure Get() latency with 100-resource state file — warm cache at least 10x faster than cold read.

### Implementation for User Story 3

- [x] T020 [US3] Add cache fields (entries []Entry, index map[string]*Entry, modTime time.Time, hits/misses uint64) to LocalBackend struct in internal/state/local.go per data-model.md StateCache entity
- [x] T021 [US3] Modify Load() to check file mtime against cached modTime, return cached entries if unchanged, rebuild index on cache miss in internal/state/local.go per FR-007
- [x] T022 [US3] Modify Get(fqn) to use index map for O(1) lookup after ensuring cache is warm (call Load if needed), increment hits/misses counters in internal/state/local.go per FR-008
- [x] T023 [US3] Modify Save() to invalidate cache (set entries=nil, clear index, reset modTime) after writing to disk in internal/state/local.go per FR-007
- [x] T024 [US3] Add structured log emission (cache hits, cache misses) on Get() calls using slog in internal/state/local.go per FR-014

- [x] T024A [US3] Add state cache benchmark test — measure Get() latency with warm vs cold cache, verify 10x speedup target in integration_tests/state_test.go

**Checkpoint**: State file lookups are O(1) with warm cache. Cache invalidates correctly on save and external modification.

---

## Phase 6: User Story 4 — Request Traceability (Priority: P2)

**Goal**: Every HTTP request gets a ULID correlation ID. All log entries during that request share the same ID for end-to-end tracing.

**Independent Test**: Send a request, verify all log entries share the same correlation ID. Check X-Correlation-ID response header.

### Implementation for User Story 4

- [x] T025 [US4] Create CorrelationMiddleware function that checks X-Correlation-ID request header, generates ULID if absent, injects into context via WithCorrelationID(), sets response header in internal/telemetry/correlation.go per FR-009, research.md R-006
- [x] T026 [US4] Wire CorrelationMiddleware into server HTTP handler chain — insert between authMiddleware and mux routing in internal/runtime/server.go per FR-009
- [x] T027 [US4] Verify all HTTP handlers use telemetry.RequestLogger() to create request-scoped loggers that include correlation_id field — audit handleInvoke, handleStream, handleCreateSession, handleSessionMessage, handlePipelineRun in internal/runtime/server.go per FR-009

**Checkpoint**: All request log entries share a ULID correlation ID. Response header X-Correlation-ID is set.

---

## Phase 7: User Story 5 — Efficient Pipeline Execution (Priority: P3)

**Goal**: DAG topological sort runs in O(V+E) time. Agent and pipeline lookups are O(1).

**Independent Test**: Measure sort time for 100-step pipeline — under 10ms. Agent lookup is constant time regardless of count.

### Implementation for User Story 5

- [x] T028 [P] [US5] Add Adjacency map[string][]string field to DAG struct, populate forward edges during BuildDAG() from DependsOn relationships in internal/pipeline/dag.go per FR-010, data-model.md DAG entity
- [x] T029 [US5] Rewrite topologicalSort() to use queue-based Kahn's algorithm — initialize queue with zero-in-degree nodes, process from queue decrementing dependents via Adjacency map, group by layers for parallel execution in internal/pipeline/dag.go per FR-010, research.md R-007
- [x] T030 [P] [US5] Add agentIndex map[string]*AgentConfig and pipelineIndex map[string]*PipelineConfig fields to Server struct, populate during NewServer() from config slices in internal/runtime/server.go per FR-011, data-model.md Server entity
- [x] T031 [US5] Replace all findAgent(name) and findPipeline(name) calls with agentIndex/pipelineIndex map lookups, remove findAgent() and findPipeline() functions from internal/runtime/server.go per FR-011, research.md R-008

- [x] T031A [US5] Add DAG sort performance benchmark — verify 100-step pipeline sorts in under 10ms in integration_tests/pipeline_test.go

**Checkpoint**: DAG sort is O(V+E). Agent/pipeline lookups are O(1). Pipeline execution with independent steps runs concurrently.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Integration testing, validation, and cleanup across all stories

- [x] T032 Run full test suite `go test ./... -count=1` and fix any regressions
- [x] T033 Run quickstart.md validation scenarios end-to-end

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Stories (Phase 3–7)**: All depend on Foundational phase completion
  - US1 and US2 can proceed in parallel (both P1, different files)
  - US3 and US4 can proceed in parallel (both P2, different files)
  - US5 can proceed independently (P3)
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Can start after Phase 2 — requires EvictionPolicy (T002) and LRU helper (T003)
- **US2 (P1)**: Can start after Phase 2 — T019 depends on T008 (US1) completing RWMutex migration in memory_store.go. Other US2 tasks (T014–T018) are independent of US1.
- **US3 (P2)**: Can start after Phase 2 — fully independent (internal/state/local.go only)
- **US4 (P2)**: Can start after Phase 1 (needs ULID dep) — fully independent (telemetry + server middleware)
- **US5 (P3)**: Can start after Phase 2 — no dependencies on other stories

### Within Each User Story

- Models/data structures before business logic
- Core implementation before metrics/logging
- Metrics/logging before integration wiring
- Story complete before moving to next priority

### Parallel Opportunities

- T004 (rate limiter) can run in parallel with T008 (session store) and T010, T011 (conversation memory) — different files. T005 runs after T004 (same file).
- T020–T024 (state cache) can run in parallel with T025–T027 (correlation ID) — different packages
- T028–T029 (DAG) can run in parallel with T030–T031 (index maps) — different functions in different files
- Per-story integration tests (T013A, T019A, T024A, T031A) run within their story phases

---

## Parallel Example: User Story 1

```bash
# Launch all independent rate limiter + session + memory tasks together:
Task: "T004 [P] [US1] Add lastAccess to bucket, RWMutex in internal/auth/ratelimit.go"
Task: "T008 [P] [US1] Background eviction goroutine in internal/session/memory_store.go"
Task: "T010 [P] [US1] LRU session tracking in internal/memory/sliding.go"
Task: "T011 [P] [US1] LRU session tracking in internal/memory/summary.go"

# Then sequentially within each file:
Task: "T005 [US1] EvictionPolicy + Start() in internal/auth/ratelimit.go" (after T004)
Task: "T006 [US1] Structured log emission in internal/auth/ratelimit.go" (after T005)
Task: "T007 [US1] Consolidate server rate limiter in internal/runtime/server.go" (after T006)
```

## Parallel Example: User Story 5

```bash
# Launch DAG and index map tasks in parallel (different files):
Task: "T028 [P] [US5] Add Adjacency field in internal/pipeline/dag.go"
Task: "T030 [P] [US5] Add agentIndex/pipelineIndex in internal/runtime/server.go"

# Then sequentially:
Task: "T029 [US5] Rewrite topologicalSort() in internal/pipeline/dag.go" (after T028)
Task: "T031 [US5] Replace findAgent/findPipeline calls in internal/runtime/server.go" (after T030)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (add ULID dependency)
2. Complete Phase 2: Foundational (EvictionPolicy + LRU helper)
3. Complete Phase 3: User Story 1 (rate limiter + session + memory eviction)
4. **STOP and VALIDATE**: Test memory stability under load
5. Deploy/demo if ready — memory leaks are fixed

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add US1 (bounded memory) → Test independently → MVP!
3. Add US2 (Redis efficiency) → Test independently → Deploy
4. Add US3 (state cache) + US4 (correlation IDs) → Test independently → Deploy
5. Add US5 (DAG + lookups) → Test independently → Deploy
6. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (memory bounds)
   - Developer B: User Story 2 (Redis efficiency)
3. After P1 stories complete:
   - Developer A: User Story 3 (state cache)
   - Developer B: User Story 4 (correlation IDs)
   - Developer C: User Story 5 (DAG + lookups)
4. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- T019 explicitly depends on T008 (same file, RWMutex migration must complete first)
