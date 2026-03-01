# Research: State & Data Integrity

**Feature**: 008-state-data-integrity
**Date**: 2026-03-01

## R1: Atomic File Write in Go

**Decision**: Use write-to-temp-file → fsync → rename pattern.

**Rationale**: `os.Rename` on Unix is atomic at the filesystem level (POSIX guarantee). Writing to a temp file in the same directory ensures same-filesystem rename (no cross-device copy). Calling `file.Sync()` (fsync) before rename guarantees data hits disk before the metadata swap.

**Implementation**:
```go
// 1. Write to temp file in same directory
tmp, _ := os.CreateTemp(filepath.Dir(path), ".state-*.tmp")
tmp.Write(data)
tmp.Sync()  // fsync
tmp.Close()
// 2. Rename atomically
os.Rename(tmp.Name(), path)
```

**Alternatives considered**:
- Direct `os.WriteFile`: Not crash-safe — partial write on crash.
- Write + sync without rename: Still leaves window where file is partially written.
- SQLite WAL: Overkill for a single JSON file.

---

## R2: flock Behavior on Unix

**Decision**: Use `syscall.Flock` with `LOCK_EX` (blocking) and configurable timeout via goroutine + timer.

**Rationale**: flock is advisory locking — it only works if all processes use it. Since `agentspec` controls all state file access, this is sufficient. flock works identically on Linux and macOS. The existing `Lock()`/`Unlock()` in `local.go` already use flock but are never called — we wire them into the apply flow.

**Behavior notes**:
- `LOCK_EX | LOCK_NB` returns `EWOULDBLOCK` immediately if locked (current impl).
- For waiting with timeout: use a goroutine that calls blocking `LOCK_EX`, cancel via context.
- flock is per-file-descriptor, released on `Close()` or process exit (including crash).
- NFS: flock does NOT work on NFS. Document as known limitation.

**Alternatives considered**:
- `fcntl` locks: More portable but process-level (not fd-level), harder to manage.
- PID-based lock files: Not atomic — race between check and create.
- Cross-platform `lockfile` library: Adds dependency; unnecessary given Unix-only scope.

---

## R3: Stale Lock Detection

**Decision**: Write PID + timestamp to lock file. On acquisition failure, check if PID is alive and if lock age exceeds timeout.

**Rationale**: flock is released automatically on process exit (crash), so most stale locks self-resolve. But for edge cases (NFS, zombie processes), the PID check provides an additional safety net. The lock file content (JSON with PID, timestamp, hostname) enables debugging.

**Lock file format** (`.agentspec.state.json.lock`):
```json
{"pid": 12345, "created": "2026-03-01T10:00:00Z", "hostname": "ci-runner-7"}
```

**Stale detection algorithm**:
1. Try `LOCK_EX | LOCK_NB` — if succeeds, write lock info, proceed.
2. If `EWOULDBLOCK`: read lock file content.
3. If PID not alive (`kill(pid, 0)` returns error): break lock, warn, re-acquire.
4. If lock age > configured timeout (default 5 min): break lock, warn, re-acquire.
5. Otherwise: wait with configurable timeout, fail if exceeded.

**Alternatives considered**:
- Lock file without PID: Can't detect stale locks reliably.
- Heartbeat-based locks: Complex, overkill for CLI tool.

---

## R4: Redis RPUSH for Session Messages

**Decision**: Replace Redis `GET key → unmarshal → append → marshal → SET key` with per-message `RPUSH` to a Redis list.

**Rationale**: Redis lists natively support atomic append via `RPUSH`. This eliminates the read-modify-write race in `redis_store.go:153-163`. `LRANGE` retrieves messages without deserializing the entire value. Each message is stored as a separate list element (JSON-encoded).

**Key changes**:
- `SaveMessages`: `RPUSH session:{id}:messages <msg1> <msg2> ...` — atomic, O(1) per message.
- `LoadMessages`: `LRANGE session:{id}:messages 0 -1` — returns all messages.
- TTL: Set on list key via `EXPIRE` after each `RPUSH` to maintain session expiry.
- Migration: If existing data uses string key, detect and migrate on first access.

**Alternatives considered**:
- Redis streams: More features than needed; adds complexity.
- Redis transactions (WATCH/MULTI/EXEC): Solves race but still does full read-write.
- Lua scripting: Atomic but harder to maintain.

---

## R5: State File Corruption Detection

**Decision**: Validate JSON on load. If invalid, log ERROR and fall back to `.bak` file.

**Rationale**: Simple `json.Unmarshal` detects truncated/corrupted JSON. The backup file (created before each write) provides the most recent valid state. No schema versioning needed for this feature — the state format is already stable.

**Recovery flow**:
1. Load state file → `json.Unmarshal`.
2. If valid: proceed.
3. If invalid: log `ERROR "state file corrupted, falling back to backup"`.
4. Load `.bak` file → `json.Unmarshal`.
5. If valid: copy `.bak` → state file, proceed with warning.
6. If invalid: return error "both state and backup corrupted, manual recovery needed".

**Backup strategy**:
- Before each atomic write: rename current state to `.bak` (also atomic).
- Sequence: `rename(state, state.bak)` → `rename(tmp, state)`.
- On crash after first rename but before second: `.bak` has good state, main file missing → load from `.bak`.

**Alternatives considered**:
- Checksum in state file: Adds complexity; JSON validation is sufficient.
- Multiple backup generations: Overkill for this use case.
- WAL-based recovery: Complex; atomic rename + backup is simpler.
