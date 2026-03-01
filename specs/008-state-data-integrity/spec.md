# Feature Specification: State & Data Integrity

**Feature Branch**: `008-state-data-integrity`
**Created**: 2026-03-01
**Status**: Draft
**Input**: Address state file corruption risks, non-atomic writes, missing file locking, and session store data races from gap analysis.

**Gap Analysis References**: BUG-004, BUG-005, BUG-022, GAP-002, GAP-018

## Clarifications

### Session 2026-03-01

- Q: Should file locking support Windows (LockFileEx) or is Unix-only (flock) acceptable? → A: Unix-only (flock) — Windows is out of scope for now.
- Q: Should lock, backup, and corruption events produce log output? → A: Yes — INFO for lock acquire/release, WARN for stale lock break, ERROR for corruption fallback.
- Q: Does FR-007 (append without read-modify-write) apply to all session stores or only Redis? → A: Redis only — memory store is already safe via mutex; fix Redis to use native RPUSH.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Crash-Safe State Persistence (Priority: P1)

An engineer runs `agentspec apply` to deploy agents to production. If the process crashes, loses power, or is killed during the apply operation, the state file must not be corrupted. The previous valid state must be recoverable.

**Why this priority**: The state file (`.agentspec.state.json`) is the single source of truth for all deployed infrastructure. Corruption means losing track of every deployed resource — requiring manual recovery of all agent deployments.

**Independent Test**: Can be tested by simulating a crash during state file write and verifying the file is either the old valid state or the new valid state, never a partial write.

**Acceptance Scenarios**:

1. **Given** an apply operation in progress, **When** the process is killed during state file write, **Then** the state file contains either the previous valid state or the new complete state.
2. **Given** a corrupted state file (e.g., truncated JSON), **When** the system starts, **Then** it detects the corruption and falls back to the most recent valid backup.
3. **Given** a successful apply, **When** the state is saved, **Then** the file is flushed to disk before the operation is reported as complete.

---

### User Story 2 - Concurrent Apply Serialization (Priority: P1)

Two engineers or CI pipelines simultaneously run `agentspec apply` on the same project directory. The state file must not be corrupted by concurrent read-modify-write operations. One apply must wait for the other to complete.

**Why this priority**: In team environments and CI/CD pipelines, concurrent applies are likely. Without file locking, the second apply can overwrite the first's state changes, causing deployed resources to become untracked.

**Independent Test**: Can be tested by running two apply operations in parallel on the same state file and verifying both complete successfully with all resources tracked.

**Acceptance Scenarios**:

1. **Given** two concurrent apply operations, **When** both try to acquire the state file lock, **Then** one acquires it and the other waits.
2. **Given** a lock held by a crashed process, **When** a new apply detects a stale lock (older than configurable timeout), **Then** the lock is broken and the new apply proceeds with a warning.
3. **Given** a locked state file, **When** the waiting apply exceeds a timeout, **Then** it fails with a descriptive error explaining the lock situation.

---

### User Story 3 - Concurrent Session Message Safety (Priority: P2)

Multiple users interact with the same agent session concurrently (e.g., a shared support session). When both users send messages at the same time, no messages should be lost due to concurrent read-modify-write patterns in the session store.

**Why this priority**: Message loss in conversations is a data integrity issue that degrades user trust. The current read-all/append/write-all pattern in Redis can lose messages under concurrent writes.

**Independent Test**: Can be tested by sending 100 messages concurrently to the same session and verifying all 100 are present in the conversation history.

**Acceptance Scenarios**:

1. **Given** two users sending messages to the same session simultaneously, **When** both messages are saved, **Then** both messages appear in the conversation history.
2. **Given** a session with 1000 messages, **When** a new message is appended, **Then** the operation completes in constant time (not proportional to total message count).
3. **Given** a session store failure during save, **When** the save fails, **Then** the error is reported to the caller and the existing messages remain intact.

---

### Edge Cases

- What happens when disk space runs out during state file write? The system reports the error and the previous state file remains intact.
- What happens when the state file is deleted while the system is running? The system creates a new state file on next save.
- What happens when the lock file is orphaned (process crashed without releasing)? A configurable stale lock timeout (default 5 minutes) allows automatic recovery.
- What happens when the state file grows very large (10,000+ resources)? The system continues to function; performance optimization is addressed in feature 010.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST write state files atomically using a write-to-temporary-file-then-rename pattern.
- **FR-002**: System MUST flush the temporary file to disk (fsync) before renaming to the final path.
- **FR-003**: System MUST implement file-based locking to serialize concurrent state file access.
- **FR-004**: System MUST detect and recover from stale locks (configurable timeout, default 5 minutes).
- **FR-005**: System MUST keep one backup copy of the previous valid state file (`.agentspec.state.json.bak`).
- **FR-006**: System MUST detect corrupted state files (invalid JSON) and fall back to the backup.
- **FR-007**: The Redis session store MUST append messages using native list operations (RPUSH) without requiring read-modify-write of the entire history. The in-memory store is already safe via mutex-protected slice append.
- **FR-008**: Session stores MUST report save errors to callers rather than silently discarding them.
- **FR-009**: System MUST log lock lifecycle events: INFO for acquire/release, WARN for stale lock break with PID/age details, ERROR for corruption detection and backup fallback.

### Key Entities

- **StateFile**: JSON file tracking all deployed resources; must survive crashes and concurrent access.
- **StateLock**: File-based lock preventing concurrent state modifications; includes PID and timestamp for stale detection.
- **StateBackup**: Previous valid state file; used for recovery from corruption.
- **SessionMessages**: Ordered list of conversation messages per session; must support concurrent append.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: State file survives 1000 simulated crash-during-write scenarios with zero corruption (100% recovery rate).
- **SC-002**: Concurrent apply operations (10 simultaneous) complete without data loss — all resources tracked correctly.
- **SC-003**: Concurrent message saves (100 simultaneous to same session) result in zero lost messages.
- **SC-004**: Stale locks are automatically recovered within the configured timeout period.
- **SC-005**: State file backup is always present and contains the most recent valid state.

## Assumptions

- File system supports atomic rename operations (Linux, macOS). Windows is out of scope for this feature.
- File-based locking via `flock` (Unix-only) is sufficient for the expected concurrency level (small teams, CI pipelines).
- The Redis session store can use list-based append operations instead of key-value read-modify-write.
- State file sizes will remain manageable (< 10MB) for the write-then-rename pattern.
