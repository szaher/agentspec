# Implementation Plan: State & Data Integrity

**Branch**: `008-state-data-integrity` | **Date**: 2026-03-01 | **Spec**: `specs/008-state-data-integrity/spec.md`
**Input**: Feature specification from `/specs/008-state-data-integrity/spec.md`

## Summary

Replace the non-atomic `os.WriteFile` in `internal/state/local.go` with a crash-safe temp-file → fsync → rename pattern, wire the existing-but-unused `Lock()`/`Unlock()` into the apply flow, add stale lock detection with PID/timestamp checking, create `.bak` backups before each write with corruption-detection fallback, and migrate Redis session message storage from read-modify-write `GET`/`SET` to atomic `RPUSH`/`LRANGE` list operations.

## Technical Context

**Language/Version**: Go 1.25+ (existing)
**Primary Dependencies**: `syscall` (flock, existing), `go-redis` (existing), `crypto/rand` (existing)
**Storage**: Local JSON state file (`.agentspec.state.json`), Redis (session messages)
**Testing**: `go test` with integration tests (primary quality gate per constitution)
**Target Platform**: Linux, macOS (Unix-only — flock, no Windows support)
**Project Type**: CLI tool with HTTP runtime server
**Performance Goals**: Lock acquisition < 100ms (uncontested), RPUSH O(1) per message
**Constraints**: Advisory locking only (flock); NFS not supported
**Scale/Scope**: Small teams, CI pipelines (< 10 concurrent applies); < 10MB state files

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Determinism | PASS | State file is sorted by FQN (existing); atomic write preserves determinism |
| II. Idempotency | PASS | Lock serializes concurrent applies; idempotent apply remains unchanged |
| VI. Safe Defaults | PASS | Locking enabled by default; no opt-out mechanism |
| X. Strict Validation | PASS | JSON validation on load; corruption detected with actionable error |
| XII. No Hidden Behavior | PASS | Lock events logged at INFO/WARN; backup creation visible |
| Testing Strategy | PASS | Integration tests exercise crash simulation, concurrent applies, concurrent sessions |
| Observability | PASS | FR-009 mandates structured log events for lock lifecycle |

No violations. No complexity justification needed.

## Project Structure

### Documentation (this feature)

```text
specs/008-state-data-integrity/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   ├── state-backend-contract.md
│   └── session-store-contract.md
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
internal/state/
├── local.go             # MODIFY: atomic write, backup, corruption detection, lock wiring
└── state.go             # READ: existing interface (no changes expected)

internal/session/
├── redis_store.go       # MODIFY: RPUSH/LRANGE message storage
├── memory_store.go      # READ: already safe (no changes expected)
├── session.go           # READ: manager (no changes expected)
└── store.go             # READ: interface (no changes expected)

internal/apply/
└── apply.go             # MODIFY: wire Lock/Unlock around Save

cmd/agentspec/
└── apply.go             # MODIFY: pass lock timeout config, defer Unlock

internal/adapters/process/
└── process.go           # MODIFY: handle state save errors (replace _ = )

integration_tests/
├── state_test.go        # NEW: atomic write, crash sim, concurrent applies, backup/recovery
└── session_test.go      # NEW or MODIFY: concurrent message saves (Redis RPUSH)
```

**Structure Decision**: All changes are within existing packages. No new packages needed — state locking infrastructure already exists in `internal/state/local.go` but is unused. The primary work is fixing existing code, not creating new abstractions.

## Architecture

### State Write Flow (After)

```
apply.go → Lock(ctx) → Load() → compute changes → Save(entries) → Unlock()
                                                        │
                                                        ▼
                                                marshal(entries)
                                                        │
                                                        ▼
                                              createTemp(same dir)
                                                        │
                                                        ▼
                                               write → fsync → close
                                                        │
                                                        ▼
                                              rename(state, state.bak)
                                                        │
                                                        ▼
                                              rename(tmp, state)
```

### Lock Acquisition Flow

```
Lock(ctx) ──► flock(LOCK_NB) ──► acquired? ──► write lock info ──► return nil
                                      │
                                      ▼ blocked
                              read lock file
                                      │
                              ┌───────┼───────┐
                              ▼               ▼
                         PID alive?     age > timeout?
                              │               │
                              no              yes
                              ▼               ▼
                         break lock      break lock
                         warn log        warn log
                              │               │
                              └───────┬───────┘
                                      ▼
                              re-acquire flock

                         (if still alive & fresh)
                              ▼
                         wait with ctx deadline
                              │
                         timeout? → return ErrStateLocked
```

### Redis Session Messages (After)

```
SaveMessages(sessionID, msgs)
    │
    ▼
RPUSH session:{id}:messages msg1_json msg2_json ...
    │
    ▼
EXPIRE session:{id}:messages ttl
    │
    ▼
return nil (or error)
```
