# Research: Memory & Performance Management

**Branch**: `010-memory-performance` | **Date**: 2026-03-04

## R-001: Rate Limiter Eviction Strategy

**Decision**: Time-based eviction with max bucket cap, using `sync.RWMutex` and a background goroutine.

**Rationale**: The current implementation (`internal/auth/ratelimit.go` and `internal/runtime/server.go`) stores buckets in `map[string]*bucket` protected by `sync.Mutex`. Buckets are created on first access and never removed. Under sustained load with many unique client IPs, this map grows unboundedly.

The eviction approach adds:
- A `lastAccess time.Time` field to each bucket for staleness detection.
- A background goroutine running every N minutes (configurable, default 5 minutes) that scans and removes buckets not accessed within the eviction interval (default 10 minutes).
- A hard cap of 10,000 buckets — when reached, the oldest buckets are evicted immediately on insertion.
- `sync.RWMutex` replaces `sync.Mutex` so `Allow()` calls use `RLock()` (read path) and eviction uses `Lock()` (write path). Note: `Allow()` updates `lastAccess` and `tokens`, so it requires a write lock on the individual bucket or a promotion to write lock. Approach: use `RWMutex` at the map level — `RLock` for lookups of existing buckets, `Lock` for bucket creation and eviction.

**Alternatives considered**:
- `sync.Map`: Higher per-operation overhead; `Range()` required for eviction is O(n) regardless. No advantage over explicit RWMutex for this access pattern.
- Sharded maps: Adds complexity without proportional benefit at 10K scale. Could revisit at 100K+.
- LRU cache library (e.g., `hashicorp/golang-lru`): Adds external dependency. The eviction logic here is simple enough to implement inline.

**Codebase impact**: Two implementations need updating:
1. `internal/auth/ratelimit.go` — generic rate limiter (used by auth middleware)
2. `internal/runtime/server.go` — per-agent rate limiter (embedded in server struct)

Consider consolidating to a single implementation in `internal/auth/ratelimit.go` and having the server use it.

---

## R-002: Memory Session Store Background Eviction

**Decision**: Add a background cleanup goroutine to `MemoryStore` that proactively evicts expired sessions on a fixed interval.

**Rationale**: Current `MemoryStore` (`internal/session/memory_store.go`) only checks expiry lazily — on `Get()` and `List()` calls. Sessions that are never accessed again after expiry persist indefinitely in the map. The store uses `sync.Mutex` for all operations.

Changes:
- Replace `sync.Mutex` with `sync.RWMutex`.
- Add a `Start(ctx context.Context)` method that launches a cleanup goroutine.
- The goroutine runs every `evictionInterval` (configurable, default 5 minutes), acquires write lock, scans for expired sessions, removes them, and emits a structured log with eviction count and remaining session count.
- The goroutine exits when the context is cancelled (server shutdown).
- Modify `Get()` and `List()` to use `RLock()` for read paths, with lazy eviction on `Get()` promoting to write lock only when an expired session is found.

**Alternatives considered**:
- Expiry-only on access (current approach): Leaves expired sessions in memory indefinitely if not accessed. Unacceptable for long-running servers.
- Separate expiry heap: More complex, marginal benefit over periodic scan at 10K scale.

**Codebase impact**: `internal/session/memory_store.go`. Runtime initialization in `internal/runtime/runtime.go` must call `Start(ctx)`.

---

## R-003: Redis Session Store — SCAN and RPUSH

**Decision**: Replace `KEYS` with `SCAN` for listing, and `RPUSH` + Redis List for message storage.

**Rationale**:
1. **KEYS anti-pattern** (`internal/session/redis_store.go:106`): The `List()` method calls `s.client.Keys(ctx, s.prefix+"*")` which blocks the entire Redis server during execution. Replace with cursor-based `SCAN` iteration.

2. **Full message reload** (`redis_store.go:152-163`): `SaveMessages()` calls `LoadMessages()` to get all existing messages, appends, then re-serializes the entire array. Replace with Redis List (`RPUSH`) for O(1) append. `LoadMessages()` uses `LRANGE` to retrieve.

Changes to `RedisClient` interface:
- Add `Scan(ctx, cursor, pattern, count) (keys, nextCursor, error)` method.
- Add `RPush(ctx, key, values...) error` method.
- Add `LRange(ctx, key, start, stop) ([]string, error)` method.

**Alternatives considered**:
- Redis Streams for messages: Over-engineered for append-only message log. Lists are simpler and sufficient.
- Pagination via `SORT`: Still requires loading all keys. `SCAN` is the documented Redis solution.

**Codebase impact**: `internal/session/redis_store.go`. The `RedisClient` interface gains 3 new methods — all implementations must be updated.

---

## R-004: Conversation Memory LRU Eviction

**Decision**: Add session-level LRU eviction to `SlidingWindow` and `Summary` memory stores with a configurable max session count (default 10,000).

**Rationale**: Both `SlidingWindow` (`internal/memory/sliding.go`) and `Summary` (`internal/memory/summary.go`) store per-session message histories in `map[string][]llm.Message`. Neither limits the number of sessions. With 10K+ sessions, each holding up to 50 messages, memory consumption grows unboundedly.

Approach:
- Add a session access order tracker (doubly-linked list + map for O(1) LRU operations).
- On `Save()` or `Load()`, promote the session to most-recently-used.
- When session count exceeds the max, evict the least-recently-used session.
- Use `sync.RWMutex` for concurrency safety.
- Emit structured log on eviction with evicted session count and remaining count.

**Alternatives considered**:
- External LRU library: Simple enough to implement with `container/list` from stdlib.
- Time-based eviction only: Sessions with long-running but infrequent conversations would be evicted prematurely. LRU better preserves active sessions.

**Codebase impact**: `internal/memory/sliding.go`, `internal/memory/summary.go`. Consider extracting shared LRU logic into a helper in `internal/memory/`.

---

## R-005: State File Caching and Indexing

**Decision**: Add in-memory cache with FQN index to `LocalBackend`, invalidated on `Save()` and external modification (mtime check).

**Rationale**: `LocalBackend` (`internal/state/local.go`) re-reads and re-parses the JSON file on every `Get()` call. For `plan` and `apply` operations that check 100+ resources, this causes O(n) I/O per lookup.

Changes:
- Add cached fields: `entries []Entry`, `index map[string]*Entry` (FQN → Entry pointer), `modTime time.Time`.
- `Load()` checks file mtime against cached mtime. If unchanged, returns cached entries.
- `Get(fqn)` uses `index[fqn]` for O(1) lookup after first load.
- `Save()` invalidates cache (sets entries to nil, clears index).
- Emit structured log with cache hit/miss on each `Get()` call.

**Alternatives considered**:
- File watcher (fsnotify): Adds external dependency and complexity for marginal benefit. Mtime check is simpler and sufficient.
- Memory-mapped file: Over-engineered for JSON state files at 500-resource scale.

**Codebase impact**: `internal/state/local.go` only. Interface (`state.go`) unchanged.

---

## R-006: Correlation ID Middleware

**Decision**: Add HTTP middleware that generates a ULID correlation ID for each request, injects it into the request context, and includes it in all downstream log entries.

**Rationale**: The telemetry package (`internal/telemetry/`) already has `WithCorrelationID()` and `CorrelationID()` context helpers, plus `RequestLogger()` that includes correlation IDs. However, no middleware currently injects correlation IDs into HTTP requests. The infrastructure exists but is unwired.

Changes:
- Add `CorrelationMiddleware(next http.Handler) http.Handler` that:
  1. Checks for `X-Correlation-ID` header (allow callers to propagate).
  2. If absent, generates a new ULID.
  3. Calls `telemetry.WithCorrelationID(r.Context(), id)`.
  4. Sets `X-Correlation-ID` response header.
- Wire into server middleware chain between auth and route handling.
- Use `oklog/ulid` or `github.com/oklog/ulid/v2` for ULID generation with `math/rand` source (cryptographic randomness not required for correlation IDs).

**Alternatives considered**:
- UUID v4: Not time-sortable. ULIDs enable time-range log queries.
- In-handler generation: Duplicates logic across every handler. Middleware is the standard HTTP pattern.
- OpenTelemetry trace ID: Explicitly out of scope per spec.

**Codebase impact**: New file `internal/telemetry/correlation.go` (or extend `logger.go`). `internal/runtime/server.go` middleware chain updated. New dependency: `oklog/ulid/v2`.

---

## R-007: DAG Topological Sort Optimization

**Decision**: Replace the current O(V²+E) topological sort with proper O(V+E) Kahn's algorithm using adjacency lists.

**Rationale**: The current implementation (`internal/pipeline/dag.go`) uses Kahn's algorithm but with an O(V) scan to find zero-in-degree nodes on each iteration, making the overall complexity O(V²+E). The fix is straightforward:
- Build an adjacency list (`map[string][]string`) mapping each step to its dependents.
- Initialize a queue with all zero-in-degree nodes.
- Process from queue, decrementing in-degree of dependents and enqueuing when they reach zero.
- Group by layers for parallel execution.

**Alternatives considered**:
- DFS-based topological sort: Produces a single ordering, not layered. Kahn's naturally produces layers for parallel execution.
- Priority queue: Unnecessary — simple FIFO queue is sufficient for BFS layering.

**Codebase impact**: `internal/pipeline/dag.go`. The `DAG` struct gains an `Adjacency map[string][]string` field. `topologicalSort()` function rewritten.

---

## R-008: Agent and Pipeline Index Maps

**Decision**: Replace linear-scan `findAgent()` and `findPipeline()` with `map[string]*AgentConfig` and `map[string]*PipelineConfig` built at server initialization.

**Rationale**: `findAgent()` and `findPipeline()` (`internal/runtime/server.go`) iterate through config slices for every request. With many agents, this is O(n) per request. Building index maps at startup converts to O(1) lookup.

Changes:
- Add `agentIndex map[string]*AgentConfig` and `pipelineIndex map[string]*PipelineConfig` fields to `Server` struct.
- Populate during `NewServer()` from config slices.
- Replace all `findAgent(name)` calls with `s.agentIndex[name]`.
- Replace all `findPipeline(name)` calls with `s.pipelineIndex[name]`.

**Alternatives considered**:
- Keep slice + binary search: Requires maintaining sorted order. Map is simpler and O(1).
- Existing tool registry pattern: Already uses `map[string]Executor` with `sync.RWMutex`. Agent/pipeline config is static after init, so no mutex needed.

**Codebase impact**: `internal/runtime/server.go` only. ~20 lines changed.
