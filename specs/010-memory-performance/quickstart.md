# Quickstart: Verifying Memory & Performance Changes

**Branch**: `010-memory-performance`

## Build

```bash
go build -o agentspec ./cmd/agentspec
```

## Run Tests

```bash
# All tests
go test ./... -count=1

# Specific packages affected by this feature
go test ./internal/auth/... -count=1 -v
go test ./internal/session/... -count=1 -v
go test ./internal/memory/... -count=1 -v
go test ./internal/state/... -count=1 -v
go test ./internal/pipeline/... -count=1 -v
go test ./internal/runtime/... -count=1 -v
go test ./internal/telemetry/... -count=1 -v

# Integration tests
go test ./integration_tests/... -count=1 -v
```

## Verify Rate Limiter Eviction (FR-001, FR-002)

```bash
# Run the server with rate limiting enabled
AGENTSPEC_RATE_LIMIT="10:20" ./agentspec run example.ias

# In another terminal, simulate many unique clients:
for i in $(seq 1 100); do
  curl -s -H "X-API-Key: test" -H "X-Forwarded-For: 10.0.0.$i" \
    http://localhost:8080/v1/agents/myagent/invoke \
    -d '{"input":"test"}' > /dev/null
done

# Wait past the eviction interval (default 10 min, or configure shorter for testing)
# Check structured logs for eviction entries:
# {"level":"INFO","msg":"rate limiter eviction","evicted":100,"remaining":0}
```

## Verify Session Store Eviction (FR-003)

```bash
# Check logs for background session cleanup entries:
# {"level":"INFO","msg":"session cleanup","evicted":5,"active":42}
```

## Verify State File Caching (FR-007, FR-008)

```bash
# Run plan on a project with multiple resources — second run should be faster
time ./agentspec plan example.ias
time ./agentspec plan example.ias

# Check logs for cache hit/miss entries:
# {"level":"DEBUG","msg":"state cache","hits":10,"misses":1}
```

## Verify Correlation IDs (FR-009)

```bash
# Send a request and check the response header
curl -v -H "X-API-Key: test" \
  http://localhost:8080/v1/agents/myagent/invoke \
  -d '{"input":"hello"}'

# Response should include:
# < X-Correlation-ID: 01HXYZ...  (ULID format)

# All log entries during that request will share the same correlation_id field.
```

## Verify DAG Sort Performance (FR-010)

```bash
# Run benchmarks
go test ./internal/pipeline/... -bench=BenchmarkTopologicalSort -benchmem
```

## Key Log Entries to Watch For

| Component | Log Message | Fields |
|-----------|-------------|--------|
| Rate limiter | `rate limiter eviction` | `evicted`, `remaining` |
| Session store | `session cleanup` | `evicted`, `active` |
| Conversation memory | `memory session eviction` | `evicted`, `remaining`, `max` |
| State cache | `state cache` | `hits`, `misses` |
| Correlation | `correlation_id` (in all request logs) | `correlation_id` |
