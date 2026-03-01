# Tasks: State & Data Integrity

**Input**: Design documents from `/specs/008-state-data-integrity/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Integration tests are included per project constitution (Testing Strategy: "Integration tests are the primary quality gate").

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: No new packages needed. Verify existing dependencies and prepare shared error types.

- [x] T001 Add shared error types to `internal/state/local.go`: `ErrStateCorrupted` struct (Path, BackupUsed, Err fields) for US1 corruption detection, and `ErrStateLocked` struct (HolderPID, Hostname, LockedAt fields) for US2 lock timeout — both implement `error` interface with descriptive `Error()` methods per state-backend-contract error types

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: No additional foundational work needed — all existing packages and dependencies are in place. Error types created in Setup (T001) unblock both US1 and US2.

**Checkpoint**: Foundation ready — user story implementation can now begin

---

## Phase 3: User Story 1 — Crash-Safe State Persistence (Priority: P1) MVP

**Goal**: Replace non-atomic `os.WriteFile` with crash-safe temp-file → fsync → rename, add `.bak` backup before each write, and detect/recover corrupted state files on load.

**Independent Test**: Simulate crash during state file write 1000 times and verify zero corruptions. Corrupt state file manually and verify backup fallback works.

### Implementation for User Story 1

- [x] T002 [US1] Implement atomic `Save()` in `internal/state/local.go` — replace `os.WriteFile(b.Path, data, 0644)` with: `os.CreateTemp(same dir)` → write JSON → `file.Sync()` (fsync) → close → `os.Rename(b.Path, b.Path+".bak")` → `os.Rename(tmp, b.Path)` per research R1
- [x] T003 [US1] Implement corruption detection in `Load()` in `internal/state/local.go` — if `json.Unmarshal` fails: log ERROR "state file corrupted", attempt load from `.bak` file, if `.bak` valid: restore it to state path and return entries with warning, if both corrupted: return `ErrStateCorrupted` per research R5
- [x] T004 [US1] Add logging for backup and corruption events in `internal/state/local.go` — import `log/slog`, log INFO on backup creation, ERROR on corruption detection, ERROR on backup fallback per FR-009 (covers backup/corruption subset of FR-009; lock lifecycle logging completed separately in T010)

### Integration Test for User Story 1

- [x] T005 [US1] Create `integration_tests/state_test.go` (NEW) — test atomic write (write + verify valid JSON), test backup creation (verify `.bak` exists after Save), test corruption recovery (truncate state file → Load() falls back to `.bak`), test both-corrupted error (truncate both → Load() returns ErrStateCorrupted), test deleted state file during runtime (next Save creates new file), test disk-full error handling (Save returns error, original state untouched) per SC-001, SC-005

**Checkpoint**: State files are crash-safe with backup recovery. Verify with `go test ./internal/state/ ./integration_tests/ -run TestState -v -count=1`

---

## Phase 4: User Story 2 — Concurrent Apply Serialization (Priority: P1)

**Goal**: Wire existing Lock/Unlock into apply flow, add stale lock detection with PID/timestamp, and add configurable lock timeout.

**Independent Test**: Run 10 concurrent apply operations on the same state file and verify all complete with all resources tracked correctly.

**Dependency**: US1 (T002) must complete before US2 starts — both modify `internal/state/local.go` Save() method. US2 Lock/Unlock wrap the Save call.

### Implementation for User Story 2

- [x] T006 [US2] Enhance `Lock()` in `internal/state/local.go` — accept `context.Context` and `LockConfig` (timeout, stale threshold); on acquire: write JSON lock info (PID, timestamp, hostname) to lock file; on `EWOULDBLOCK`: read lock info, check if PID alive via `syscall.Kill(pid, 0)`, check if age > stale threshold; if stale: break lock with WARN log, re-acquire; if not stale: wait with context deadline; on timeout: return `ErrStateLocked` with holder details per research R2, R3
- [x] T007 [US2] Add `LockConfig` struct to `internal/state/local.go` — fields: `LockTimeout time.Duration` (default 30s), `StaleThreshold time.Duration` (default 5m); add `WithLockConfig(LockConfig)` option or method on LocalBackend
- [x] T008 [US2] Wire `Lock()`/`Unlock()` into apply flow in `internal/apply/apply.go` — call `backend.Lock(ctx)` before `backend.Load()`, defer `backend.Unlock()`, pass context with deadline per state-backend-contract
- [x] T009 [US2] Add `--lock-timeout` flag (default 30s) to `cmd/agentspec/apply.go` — pass timeout to apply flow, create context with deadline from flag value
- [x] T010 [US2] Add lock lifecycle logging in `internal/state/local.go` — INFO for lock acquired (with PID), INFO for lock released (with held duration), INFO for lock wait started (with holder info), WARN for stale lock broken (with stale PID and age), ERROR for lock timeout per FR-009 (covers lock lifecycle subset of FR-009; backup/corruption logging completed separately in T004)

### Integration Test for User Story 2

- [x] T011 [US2] Extend `integration_tests/state_test.go` with concurrent apply tests — test two goroutines calling Lock() simultaneously (one waits), test stale lock detection (create lock file with dead PID → Lock() breaks it), test lock timeout (hold lock → second Lock() times out with ErrStateLocked), test 10 concurrent saves with Lock/Unlock (all succeed, no corruption) per SC-002, SC-004

**Checkpoint**: Concurrent applies are serialized via file locking. Verify with `go test ./internal/state/ ./integration_tests/ -run "TestState|TestConcurrent" -v -count=1`

---

## Phase 5: User Story 3 — Concurrent Session Message Safety (Priority: P2)

**Goal**: Replace Redis read-modify-write pattern in SaveMessages with atomic RPUSH, replace LoadMessages with LRANGE, add migration for existing string keys, and fix error handling in process adapter.

**Independent Test**: Send 100 messages concurrently to the same session and verify all 100 are present in the conversation history.

### Implementation for User Story 3

- [x] T012 [US3] Rewrite `SaveMessages()` in `internal/session/redis_store.go` — replace `LoadMessages() → append → Set()` pattern with: marshal each message to JSON, `RPUSH session:{id}:messages msg1 msg2 ...`, `EXPIRE session:{id}:messages ttl`; return error on failure (satisfies FR-008 error reporting requirement) per research R4 and session-store-contract
- [x] T013 [US3] Rewrite `LoadMessages()` in `internal/session/redis_store.go` — replace `Get() → Unmarshal` with: `LRANGE session:{id}:messages 0 -1`, unmarshal each element individually, return error on Redis failure (not nil), log WARN and skip on individual unmarshal failure per session-store-contract
- [x] T014 [US3] Add migration logic for existing string-based sessions in `internal/session/redis_store.go` — in `LoadMessages()`: check key type with `TYPE` command, if string: unmarshal, delete string key, RPUSH each message to list key, log INFO "migrated session to list format" per session-store-contract migration section
- [x] T015 [US3] Fix silent state save errors in `internal/adapters/process/process.go` — replace `_ = a.stateBackend.Save(entries)` at line ~143 and `_ = a.stateBackend.Save(remaining)` at line ~238 with proper error handling: log ERROR, return error to caller

### Integration Test for User Story 3

- [x] T016 [US3] Add concurrent session message tests to `integration_tests/state_test.go` or new `integration_tests/session_save_test.go` — test 100 concurrent RPUSH saves to same session (all messages present), test LoadMessages returns error on Redis failure (mock), test constant-time append (not proportional to history size) per SC-003

**Checkpoint**: Redis session messages use atomic RPUSH. Verify with `go test ./internal/session/ ./integration_tests/ -run "TestSession|TestConcurrent" -v -count=1`

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validate all changes work together and no regressions

- [x] T017 Run full test suite with race detector: `go test ./... -race -count=1` — fix any remaining race conditions
- [x] T018 Run quickstart.md verification steps end-to-end — validate all 7 verification sections pass
- [x] T019 Audit modified files for remaining `_ = ` error-discard patterns in `internal/state/` and `internal/session/` — grep and fix any overlooked silent error discards
- [x] T020 Verify build succeeds cleanly: `go build -o agentspec ./cmd/agentspec` — no compilation errors

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup (T001) — BLOCKS user stories
- **User Stories (Phase 3–5)**: All depend on Phase 1 completion
  - US1 (P1) starts first — modifies Save() and Load()
  - US2 (P1) starts after US1 T002 completes — both modify `local.go`, US2 adds Lock/Unlock around Save
  - US3 (P2) can start in parallel with US1/US2 — different package (`internal/session/`)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (Crash-Safe State)**: Independent — no dependencies on other stories
- **US2 (Concurrent Apply)**: **Depends on US1 T002** — US2 wraps Lock/Unlock around the atomic Save() from US1; both modify `internal/state/local.go`
- **US3 (Session Safety)**: Independent — entirely in `internal/session/redis_store.go` and `internal/adapters/process/process.go`

### Within Each User Story

- Tasks marked [P] within a story can run in parallel
- Non-[P] tasks must run sequentially
- Integration test tasks run AFTER all implementation tasks in the same story

### Parallel Opportunities

**Maximum parallelism (2 parallel streams after Phase 1)**:
- Stream A: US1 (T002 → T003 → T004 → T005) → US2 (T006 → T007 → T008 → T009 → T010 → T011)
- Stream B: US3 (T012 → T013 → T014 → T015 → T016)
- Streams A and B run in parallel since they touch different packages

---

## Parallel Example: Cross-Story

```bash
# After Phase 1 (T001), both streams start:
Stream A: US1 (T002 → T003 → T004 → T005) → US2 (T006 → T007 → T008 → T009 → T010 → T011)
Stream B: US3 (T012 → T013 → T014 → T015 → T016)
# After both streams complete:
Polish: T017 → T018 → T019 → T020
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001)
2. Complete Phase 3: User Story 1 — Crash-Safe State Persistence (T002–T005)
3. **STOP and VALIDATE**: `go test ./internal/state/ ./integration_tests/ -run TestState -v -count=1`
4. Highest-impact data integrity fix shipped (non-atomic writes eliminated)

### Incremental Delivery (P1 Stories)

1. Setup → Error types ready
2. US1 (Crash-Safe State) → 4 tasks → Atomic writes + backup + corruption recovery
3. US2 (Concurrent Apply) → 6 tasks → File locking + stale detection
4. **Milestone**: Both P1 stories complete (11 tasks including tests)

### Full Delivery (All Stories)

5. US3 (Session Safety) → 5 tasks → Redis RPUSH + error handling
6. Polish → 4 tasks → Final validation
7. **Milestone**: All 20 tasks complete, full state & data integrity shipped

---

## Summary

| Phase | Story | Priority | Tasks | Parallelizable |
|-------|-------|----------|-------|----------------|
| 1 | Setup | — | 1 | — |
| 2 | Foundational | — | 0 | — |
| 3 | US1: Crash-Safe State | P1 | 4 | — |
| 4 | US2: Concurrent Apply | P1 | 6 | — |
| 5 | US3: Session Safety | P2 | 5 | — |
| 6 | Polish | — | 4 | — |
| **Total** | | | **20** | **US3 parallel with US1+US2** |

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- `internal/state/local.go` is modified by US1 (T002–T004) and US2 (T006–T007, T010) — US2 must wait for US1 T002
- `internal/session/redis_store.go` is only modified by US3 — fully parallel with US1/US2
- `integration_tests/state_test.go` is created by US1 (T005) and extended by US2 (T011) — sequential
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
