# Contract: State Backend

**Feature**: 008-state-data-integrity
**Scope**: `internal/state/local.go` — LocalBackend interface

## Interface

The LocalBackend MUST implement these operations with the specified guarantees:

### Save(entries []StateEntry) error

**Preconditions**:
- Lock MUST be held by caller (via `Lock()`)
- entries MUST be serializable to JSON

**Postconditions**:
- On success: state file contains exactly `entries`, flushed to disk
- Previous valid state preserved in `.bak` file
- On failure: original state file untouched, error returned

**Atomicity guarantee**:
```
1. Marshal entries to JSON with indentation
2. Write JSON to temp file in same directory (.state-*.tmp)
3. fsync temp file (file.Sync())
4. Close temp file
5. Rename current state file to .bak (atomic)
6. Rename temp file to state file (atomic)
```

If crash occurs at any step:
- Steps 1-4: temp file may be partial, state file intact
- Step 5: .bak has old state, state file gone → recovery loads .bak
- Step 6: .bak has old state, state file = new state → consistent

### Load() ([]StateEntry, error)

**Postconditions**:
- On success: returns valid state entries
- On JSON error: attempts .bak recovery, logs ERROR
- On .bak recovery success: restores state file from backup, returns entries
- On both corrupted: returns error with "manual recovery" message

### Lock(ctx context.Context) error

**Behavior**:
1. Open/create lock file (`{path}.lock`)
2. Try `flock(LOCK_EX | LOCK_NB)`
3. If acquired: write lock info (PID, timestamp, hostname), return nil
4. If blocked: check stale lock (dead PID or age > timeout)
5. If stale: break lock, warn via slog, re-acquire
6. If not stale: wait with context deadline
7. On timeout: return error with lock holder details

**Lock info format**: `{"pid": <int>, "created": "<RFC3339>", "hostname": "<string>"}`

### Unlock() error

**Behavior**:
1. Release flock
2. Close lock file descriptor
3. Remove lock file
4. Errors during cleanup are logged but not returned (best-effort)

## Logging Contract

| Event | Level | Fields |
|-------|-------|--------|
| Lock acquired | INFO | path, pid |
| Lock released | INFO | path, pid, held_duration |
| Lock wait started | INFO | path, holder_pid, holder_hostname |
| Stale lock broken | WARN | path, stale_pid, stale_age, stale_hostname |
| Lock timeout | ERROR | path, holder_pid, wait_duration |
| State corruption detected | ERROR | path, json_error |
| Backup fallback | ERROR | path, backup_path |
| Backup also corrupted | ERROR | path, backup_path |

## Error Types

```go
type ErrStateLocked struct {
    HolderPID  int
    Hostname   string
    LockedAt   time.Time
}

type ErrStateCorrupted struct {
    Path       string
    BackupUsed bool
    Err        error
}
```
