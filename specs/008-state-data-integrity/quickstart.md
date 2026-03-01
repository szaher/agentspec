# Quickstart Verification: State & Data Integrity

**Feature**: 008-state-data-integrity
**Date**: 2026-03-01

## Prerequisites

- Go 1.25+
- Redis instance (for session store tests)
- Unix system (Linux or macOS — flock required)

## Verification Steps

### 1. Atomic State File Write

```bash
# Build the CLI
go build -o agentspec ./cmd/agentspec

# Create a test state by applying an example
./agentspec apply examples/basic-agent/basic-agent.ias

# Verify state file exists
ls -la .agentspec.state.json
# Expected: file exists with valid JSON

# Verify backup exists after second apply
./agentspec apply examples/basic-agent/basic-agent.ias
ls -la .agentspec.state.json.bak
# Expected: .bak file exists with previous state
```

### 2. Crash Recovery

```bash
# Corrupt the state file
echo "corrupted{" > .agentspec.state.json

# Run apply — should detect corruption and recover from backup
./agentspec apply examples/basic-agent/basic-agent.ias 2>&1
# Expected: ERROR log "state file corrupted, falling back to backup"
# Expected: apply succeeds using backup state

# Verify state file is restored
python3 -c "import json; json.load(open('.agentspec.state.json'))"
# Expected: no error (valid JSON)
```

### 3. File Locking

```bash
# Run two applies simultaneously
./agentspec apply examples/basic-agent/basic-agent.ias &
./agentspec apply examples/basic-agent/basic-agent.ias &
wait
# Expected: both complete successfully
# Expected: INFO logs showing lock acquire/release
# Expected: one apply waits for the other

# Verify no corruption
python3 -c "import json; json.load(open('.agentspec.state.json'))"
# Expected: valid JSON
```

### 4. Stale Lock Recovery

```bash
# Simulate a stale lock by creating a lock file with a dead PID
echo '{"pid": 99999, "created": "2020-01-01T00:00:00Z", "hostname": "test"}' > .agentspec.state.json.lock

# Run apply — should detect stale lock and break it
./agentspec apply examples/basic-agent/basic-agent.ias 2>&1
# Expected: WARN log "stale lock detected" with PID and age
# Expected: apply proceeds successfully
```

### 5. Concurrent Session Messages (Integration Test)

```bash
# Run the concurrent session message test
go test ./integration_tests/ -run TestConcurrentSessionMessages -v -count=1
# Expected: 100 concurrent saves, zero lost messages
# Expected: all messages present in correct order
```

### 6. Full Test Suite

```bash
# Run all tests with race detector
go test ./... -race -count=1
# Expected: all pass, zero data races

# Run state-specific tests
go test ./internal/state/ ./internal/session/ ./integration_tests/ -v -count=1 -run "TestState|TestSession|TestConcurrent"
# Expected: all pass
```

### 7. Build Verification

```bash
go build -o agentspec ./cmd/agentspec
# Expected: clean build, no errors
```

## Success Criteria Verification

| SC | Test | Expected |
|----|------|----------|
| SC-001 | Crash simulation test (1000 iterations) | 0 corruptions |
| SC-002 | 10 concurrent applies | All resources tracked |
| SC-003 | 100 concurrent message saves | 0 lost messages |
| SC-004 | Stale lock with dead PID | Auto-recovered |
| SC-005 | After any successful apply | .bak file present and valid |
