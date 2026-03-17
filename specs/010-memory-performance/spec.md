# Feature Specification: Memory & Performance Management

**Feature Branch**: `010-memory-performance`
**Created**: 2026-03-01
**Status**: Draft
**Input**: Fix unbounded memory growth in rate limiters, session stores, and conversation memory. Improve performance of state lookups, Redis operations, DAG execution, and request tracing.

**Gap Analysis References**: BUG-007, BUG-008, BUG-014, BUG-035, GAP-011, GAP-017, GAP-024, PERF-001 through PERF-010, FEAT-010

## Out of Scope

- **Distributed tracing (OpenTelemetry)**: Correlation IDs are in-process only; OpenTelemetry integration is a separate future feature.
- **Cross-process state cache sharing**: The state file cache is per-process; shared caching across concurrent CLI invocations is not addressed here.
- **Redis Cluster/Sentinel support**: Redis operations target single-instance Redis; Cluster and Sentinel topologies are not supported in this feature.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Bounded Memory Usage Under Load (Priority: P1)

An engineer runs an agent server in production serving thousands of distinct clients over several days. The server's memory usage must remain stable over time — not grow linearly with the number of unique clients or expired sessions.

**Why this priority**: Unbounded memory growth is the most impactful performance issue. Rate limiter buckets, expired sessions, and conversation histories accumulate without eviction, eventually causing out-of-memory crashes in production.

**Independent Test**: Can be tested by simulating 10,000 unique clients making requests, then waiting past the eviction interval, and verifying memory usage returns to baseline.

**Acceptance Scenarios**:

1. **Given** a server receiving requests from 10,000 unique clients, **When** no client has been active for 10 minutes, **Then** rate limiter memory for those clients is reclaimed.
2. **Given** a session that has been expired for longer than the TTL, **When** an eviction cycle runs, **Then** the session is removed from memory regardless of whether it was accessed.
3. **Given** a conversation memory store configured with a max of 1,000 sessions (reduced from default for testing), **When** the maximum session count is reached, **Then** the least-recently-used session is evicted to make room for new sessions.
4. **Given** the server has been running for 24 hours under sustained load, **When** memory usage is measured, **Then** it is within 20% of the 1-hour mark (stable, not growing).

---

### User Story 2 - Efficient Session Store Operations (Priority: P1)

An engineer uses the session store (backed by Redis or in-memory) for a high-traffic agent. Session listing must not block the entire store, and message retrieval must scale with the number of sessions.

**Why this priority**: The Redis `KEYS` command blocks the entire server during execution — a single session list call can freeze all Redis-dependent operations. This is a documented Redis anti-pattern that must be resolved before production use.

**Independent Test**: Can be tested by creating 10,000 sessions in Redis, then calling list — the operation must complete without blocking other Redis commands.

**Acceptance Scenarios**:

1. **Given** 10,000 sessions in Redis, **When** listing sessions, **Then** the operation uses cursor-based iteration and does not block other Redis clients.
2. **Given** a session with 500 messages, **When** appending a new message, **Then** the operation completes in constant time (not proportional to existing message count).
3. **Given** the memory store, **When** listing sessions, **Then** expired sessions are cleaned up during the listing operation.

---

### User Story 3 - Fast State File Access (Priority: P2)

An engineer runs `agentspec plan` or `apply` on a project with 100+ deployed resources. State file lookups should not re-read and re-parse the entire file for each resource query.

**Why this priority**: The state file is currently re-read from disk and fully parsed for every `Get()` call. For operations that check multiple resources (like plan and apply), this causes unnecessary I/O and parsing overhead.

**Independent Test**: Can be tested by measuring `Get()` call latency with a 100-resource state file — second and subsequent calls should be significantly faster than the first.

**Acceptance Scenarios**:

1. **Given** a state file with 100 resources, **When** `Get()` is called multiple times, **Then** only the first call reads from disk; subsequent calls use cached data.
2. **Given** a `Save()` operation, **When** new state is written, **Then** the cache is invalidated and the next `Get()` reads fresh data.
3. **Given** a state file with 500 resources, **When** `Get()` is called by FQN, **Then** the lookup is done via index (not linear scan).

---

### User Story 4 - Request Traceability (Priority: P2)

An SRE investigating a production issue needs to trace a single user request through the agent server — from the initial HTTP request, through the LLM call, tool execution, and session operations. Every log entry in that flow must share a common correlation identifier.

**Why this priority**: Current log entries are disconnected. When multiple requests are processed concurrently, it's impossible to correlate which log entries belong to which request, making debugging extremely difficult.

**Independent Test**: Can be tested by sending a request and verifying all log entries generated during that request share the same correlation ID.

**Acceptance Scenarios**:

1. **Given** an incoming HTTP request, **When** the server processes it, **Then** a unique correlation ID is generated and included in all log entries for that request.
2. **Given** a tool call within an agent invocation, **When** the tool logs output, **Then** the log entry includes the request's correlation ID.
3. **Given** the correlation ID, **When** filtering logs, **Then** the complete request lifecycle is visible: auth → session load → LLM call → tool calls → response.

---

### User Story 5 - Efficient Pipeline Execution (Priority: P3)

An engineer configures a multi-agent pipeline with 20+ steps and complex dependencies. The pipeline scheduler must determine execution order efficiently and execute independent steps in parallel.

**Why this priority**: The current DAG topological sort has quadratic complexity. While adequate for small pipelines, this becomes noticeable with larger configurations.

**Independent Test**: Can be tested by measuring sort time for a 100-step pipeline — it should complete in under 10ms.

**Acceptance Scenarios**:

1. **Given** a pipeline with 100 steps, **When** topological sort is computed, **Then** the operation completes in under 10ms.
2. **Given** a pipeline with independent steps in the same layer, **When** execution reaches that layer, **Then** independent steps execute concurrently.
3. **Given** a server processing agent invocations, **When** looking up agents by name, **Then** the lookup completes in constant time (not proportional to agent count).

---

### Edge Cases

- What happens when the eviction goroutine falls behind under extreme load? Eviction is bounded and non-blocking; if it can't keep up, the memory cap serves as a hard limit.
- What happens when the state file cache becomes stale due to external modification? The cache includes a file modification timestamp check.
- What happens when Redis is temporarily unavailable during session operations? Operations fail with descriptive errors; no silent fallback to different behavior.
- What happens when correlation ID generation itself becomes a bottleneck? IDs are generated in-process with no external dependency; overhead is negligible.

## Clarifications

### Session 2026-03-04

- Q: Should the spec require memory management components to emit operational metrics? → A: Yes — require counters (eviction count, cache hit/miss, active sessions) via structured log entries.
- Q: What default maximum should rate limiter bucket count (FR-002) and conversation memory session count (FR-004) use? → A: 10,000 for both.
- Q: What format should correlation IDs use? → A: ULID (time-sortable, 128-bit, Crockford Base32 encoded).
- Q: Which items should be explicitly declared out of scope? → A: All of: distributed tracing (OpenTelemetry), cross-process state cache sharing, Redis Cluster/Sentinel support.
- Q: What concurrency safety model should eviction-enabled stores use? → A: `sync.RWMutex` — read-heavy optimization with exclusive lock only during eviction writes.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Rate limiter MUST evict buckets that have not been accessed within a configurable interval (default 10 minutes).
- **FR-002**: Rate limiter MUST have a configurable maximum bucket count (default 10,000); when reached, the oldest buckets are evicted.
- **FR-003**: Memory session store MUST proactively evict expired sessions via background cleanup (not only on access).
- **FR-004**: Conversation memory store MUST enforce a maximum number of concurrent sessions (default 10,000) with LRU eviction.
- **FR-005**: Redis session store MUST use cursor-based iteration for listing sessions (not blocking full-scan commands).
- **FR-006**: Redis session store MUST append messages without reading the entire history.
- **FR-007**: State file backend MUST cache loaded entries in memory, invalidating on save.
- **FR-008**: State file lookups by FQN MUST use an indexed data structure.
- **FR-009**: All HTTP request processing MUST include a ULID-formatted correlation ID in every log entry generated during the request.
- **FR-010**: Pipeline DAG topological sort MUST have linear time complexity relative to steps and edges.
- **FR-011**: Agent and pipeline lookups MUST use indexed data structures with constant-time access.
- **FR-012**: Rate limiter eviction MUST emit structured log entries with eviction count per cycle.
- **FR-013**: Memory session stores, Redis session stores, and conversation memory stores MUST emit structured log entries with active session/entry count and eviction count per cleanup cycle.
- **FR-014**: State file cache MUST emit structured log entries with cache hit and miss counts.
- **FR-015**: In-memory stores with background eviction (rate limiter, session store, conversation memory) MUST use `sync.RWMutex` for concurrency safety — read lock for lookups, write lock for eviction and mutations.
- **FR-016**: Redis session store MUST transparently migrate existing message data from String-type (JSON array) to List-type storage on first access, preserving all existing messages without data loss.

### Key Entities

- **EvictionPolicy**: Configuration for when and how stale entries are removed from in-memory stores (interval, max count, TTL).
- **CorrelationID**: ULID-formatted unique identifier generated per HTTP request, propagated via context through all log entries in the request's lifecycle. Time-sortable for log range queries.
- **StateCache**: In-memory copy of state file entries, indexed by FQN, with modification-time-based invalidation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Server memory usage remains stable (within 20% of baseline) after 24 hours of sustained load with 10,000+ unique clients.
- **SC-002**: Rate limiter evicts stale buckets within 2x the configured eviction interval.
- **SC-003**: Session listing with 10,000+ sessions completes without blocking other store operations.
- **SC-004**: State file `Get()` with warm cache is at least 10x faster than cold read from disk.
- **SC-005**: All log entries within a single request share the same correlation ID.
- **SC-006**: Pipeline topological sort for 100-step DAG completes in under 10ms.
- **SC-007**: Agent lookup by name completes in constant time regardless of agent count.

## Assumptions

- The eviction goroutine runs on a fixed interval (configurable, default 5 minutes) and completes within a fraction of the interval.
- The state file cache is per-process (not shared across concurrent CLI invocations — that's handled by file locking in feature 008).
- Correlation IDs are generated in-process and do not require external services.
- The Redis session store's append-based message storage is compatible with existing message retrieval patterns.
