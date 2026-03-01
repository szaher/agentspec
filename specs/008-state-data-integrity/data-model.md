# Data Model: State & Data Integrity

**Feature**: 008-state-data-integrity
**Date**: 2026-03-01

## Entities

### StateFile

The JSON file tracking all deployed resources. Must survive crashes and concurrent access.

| Field | Type | Description |
|-------|------|-------------|
| path | string | Absolute path to `.agentspec.state.json` |
| entries | []StateEntry | List of deployed resource entries (existing) |

**Constraints**:
- Written atomically via temp-file + fsync + rename
- Always accompanied by a `.bak` backup
- JSON must be valid; corruption triggers backup fallback

**State transitions**:
```
[empty/missing] → Load → [valid state]
[valid state] → Save → [temp write] → [fsync] → [backup old] → [rename new] → [valid state]
[corrupted] → Load → [detect corruption] → [fallback to backup] → [valid state]
[corrupted + corrupted backup] → Load → [error: manual recovery needed]
```

---

### StateLock

File-based lock preventing concurrent state modifications.

| Field | Type | Description |
|-------|------|-------------|
| path | string | Lock file path: `{state_path}.lock` |
| pid | int | PID of the process holding the lock |
| created | time.Time | Timestamp when lock was acquired |
| hostname | string | Hostname of the locking machine |
| timeout | time.Duration | Configurable stale lock timeout (default 5m) |
| fd | *os.File | File descriptor for flock (Unix) |

**Lock file content** (JSON):
```json
{"pid": 12345, "created": "2026-03-01T10:00:00Z", "hostname": "ci-runner-7"}
```

**Constraints**:
- Uses `syscall.Flock(LOCK_EX)` for kernel-level locking
- PID + timestamp for stale detection
- Auto-released on process exit/crash (flock behavior)

**State transitions**:
```
[unlocked] → Acquire(LOCK_NB) → [locked by self]
[locked by other] → Wait(timeout) → [locked by self] OR [timeout error]
[locked by dead PID] → Break(warn) → [locked by self]
[locked by stale timestamp] → Break(warn) → [locked by self]
[locked by self] → Release → [unlocked]
```

---

### StateBackup

Previous valid state file used for corruption recovery.

| Field | Type | Description |
|-------|------|-------------|
| path | string | Backup file path: `{state_path}.bak` |
| content | []byte | Copy of previous valid state |

**Constraints**:
- Created atomically via rename before each state write
- Contains the most recent VALID state (not the current write)
- Recovery sequence: if state corrupted → load backup → restore

**Lifecycle**:
```
[state write requested] → rename(state, state.bak) → rename(tmp, state) → [backup = previous state]
```

---

### SessionMessages (Redis)

Ordered list of conversation messages per session stored as a Redis list.

| Field | Type | Description |
|-------|------|-------------|
| key | string | Redis key: `session:{id}:messages` |
| elements | []string | JSON-encoded llm.Message objects |
| ttl | time.Duration | Session expiry (matches session TTL) |

**Constraints**:
- Append via `RPUSH` (atomic, O(1))
- Read via `LRANGE 0 -1` (returns all messages)
- TTL refreshed on each RPUSH via `EXPIRE`
- No read-modify-write pattern

**Operations**:
```
SaveMessages: RPUSH session:{id}:messages <msg1_json> <msg2_json> ...
              EXPIRE session:{id}:messages <ttl>
LoadMessages: LRANGE session:{id}:messages 0 -1
              → unmarshal each element
```

## Relationships

```
StateFile 1──1 StateLock     (one lock per state file)
StateFile 1──1 StateBackup   (one backup per state file)
Session   1──* SessionMessages (one message list per session, Redis only)
```

## Validation Rules

- StateLock PID must be > 0 and correspond to a running process (or be stale)
- StateLock created timestamp must be parseable as RFC 3339
- StateFile content must be valid JSON (json.Unmarshal succeeds)
- StateBackup must be valid JSON if it exists
- SessionMessages elements must be valid JSON-encoded llm.Message objects
