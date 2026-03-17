# Data Model: Memory & Performance Management

**Branch**: `010-memory-performance` | **Date**: 2026-03-04

## Entities

### EvictionPolicy

Configuration for when and how stale entries are removed from in-memory stores.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `MaxEntries` | `int` | `10000` | Maximum number of entries before forced eviction |
| `TTL` | `time.Duration` | `10m` | Time-to-live for entries since last access |
| `EvictionInterval` | `time.Duration` | `5m` | How often the background eviction goroutine runs |

**Validation rules**:
- `MaxEntries` must be > 0
- `TTL` must be > 0
- `EvictionInterval` must be > 0 and < `TTL` (otherwise entries expire before eviction runs)

**Used by**: Rate limiter (bucket eviction), Memory session store (session eviction), Conversation memory (session-level LRU eviction)

---

### RateLimiterBucket (modified)

Token bucket with access tracking for eviction.

| Field | Type | Description |
|-------|------|-------------|
| `tokens` | `float64` | Current token count |
| `lastRefill` | `time.Time` | Last token refill timestamp |
| `lastAccess` | `time.Time` | **NEW** — Last time `Allow()` was called for this bucket |
| `rate` | `float64` | Tokens per second (per-agent variant only) |
| `burst` | `int` | Max burst size (per-agent variant only) |

**State transitions**:
```
Created (Allow first call) → Active (Allow subsequent) → Stale (no access > TTL) → Evicted (background goroutine)
                                                        → Evicted (max cap reached, oldest-first)
```

---

### Session (unchanged interface, new internal behavior)

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Unique session identifier (`sess_<nanos>`) |
| `AgentName` | `string` | Associated agent |
| `CreatedAt` | `time.Time` | Creation timestamp |
| `LastActive` | `time.Time` | Last `Touch()` timestamp |
| `Metadata` | `map[string]string` | Arbitrary metadata |

**State transitions**:
```
Created → Active (Touch updates LastActive) → Expired (LastActive + TTL < now) → Evicted (background goroutine)
```

**Memory store behavior change**: Background goroutine proactively scans and removes expired sessions instead of relying on lazy eviction in `Get()`/`List()`.

---

### ConversationMemorySession (new internal tracking)

LRU tracking for session-level eviction in conversation memory stores.

| Field | Type | Description |
|-------|------|-------------|
| `sessionID` | `string` | Session identifier (map key) |
| `messages` | `[]llm.Message` | Conversation history |
| `lruElement` | `*list.Element` | Pointer to position in LRU doubly-linked list |

**State transitions**:
```
Created (Save/Load first call) → Active (promoted to MRU on access) → LRU (least recently used) → Evicted (max sessions exceeded)
```

**Data structure**: `map[string]*ConversationMemorySession` + `container/list.List` for O(1) LRU operations.

---

### StateCache (new)

In-memory cache layer for the local state file backend.

| Field | Type | Description |
|-------|------|-------------|
| `entries` | `[]Entry` | Cached copy of all state entries |
| `index` | `map[string]*Entry` | FQN → Entry pointer for O(1) lookup |
| `modTime` | `time.Time` | File modification time when cache was populated |
| `hits` | `uint64` | Cache hit counter (for metrics) |
| `misses` | `uint64` | Cache miss counter (for metrics) |

**State transitions**:
```
Cold (no cache) → Warm (after first Load) → Stale (file mtime changed or Save called) → Cold
```

**Invalidation rules**:
- `Save()` clears cache immediately (entries = nil, index = nil)
- `Get()`/`Load()` checks file mtime before using cache; if mtime differs, reloads

---

### CorrelationID

| Field | Type | Description |
|-------|------|-------------|
| value | `string` | ULID string (26 chars, Crockford Base32) |

**Format**: ULID — 128-bit identifier with 48-bit millisecond timestamp prefix + 80-bit random suffix.

**Lifecycle**:
```
HTTP Request arrives → Middleware checks X-Correlation-ID header
  → If present: use provided ID
  → If absent: generate new ULID
→ Inject into request context via telemetry.WithCorrelationID()
→ Set X-Correlation-ID response header
→ All downstream log entries include correlation_id field
→ Request completes
```

**Propagation**: Via `context.Context` using existing `telemetry.WithCorrelationID()` / `telemetry.CorrelationID()` helpers.

---

### DAG (modified)

| Field | Type | Description |
|-------|------|-------------|
| `Steps` | `map[string]*Step` | Index of all steps |
| `Order` | `[][]string` | Topological layers |
| `Incoming` | `map[string]int` | In-degree count per step |
| `Adjacency` | `map[string][]string` | **NEW** — Forward edges: step → list of dependents |

**Algorithm change**: `topologicalSort()` uses `Adjacency` to decrement in-degree of dependents in O(1) per edge instead of scanning all steps.

---

### Server (modified)

| Field | Type | Description |
|-------|------|-------------|
| `agentIndex` | `map[string]*AgentConfig` | **NEW** — Agent name → config for O(1) lookup |
| `pipelineIndex` | `map[string]*PipelineConfig` | **NEW** — Pipeline name → config for O(1) lookup |

**Built at**: `NewServer()` initialization from config slices. Immutable after construction (no mutex needed).

## Relationships

```
Server
 ├── agentIndex [1:N] → AgentConfig
 ├── pipelineIndex [1:N] → PipelineConfig
 ├── RateLimiter
 │    ├── EvictionPolicy (config)
 │    └── buckets [1:N] → RateLimiterBucket
 ├── SessionManager
 │    ├── MemoryStore (or RedisStore)
 │    │    ├── EvictionPolicy (config, memory store only)
 │    │    └── sessions [1:N] → Session
 │    └── ConversationMemory (SlidingWindow or Summary)
 │         ├── EvictionPolicy (config, max sessions)
 │         └── sessions [1:N] → ConversationMemorySession
 └── StateBackend (LocalBackend)
      └── StateCache
           └── index [1:N] → Entry
```

## Redis Key Schema (modified)

| Key Pattern | Type | Description |
|-------------|------|-------------|
| `agentspec:session:<id>` | String (JSON) | Session metadata |
| `agentspec:session:<id>:messages` | **List** (was String) | Message history — each element is JSON-serialized `llm.Message` |

**Migration note**: Existing message data stored as JSON arrays (String type) must be handled. On first `LoadMessages()`, if the key is a String type, read and delete it. On next `SaveMessages()`, data will be stored as a List. This provides transparent migration.
