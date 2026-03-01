# Quickstart: Test Foundation & Quality Engineering

**Feature**: 009-test-quality-foundation
**Date**: 2026-03-01

## Prerequisites

- Go 1.25+ installed
- golangci-lint v2.10.1+ installed
- Repository cloned and on `009-test-quality-foundation` branch

## Verification Steps

### 1. Build Succeeds

```bash
go build -o agentspec ./cmd/agentspec
# Expected: Clean build, no errors
```

### 2. Unit Tests Pass for Security-Critical Packages

```bash
go test ./internal/auth/... ./internal/secrets/... ./internal/policy/... ./internal/tools/... ./internal/session/... ./internal/state/... -v -count=1
# Expected: All tests pass
```

### 3. Security-Critical Package Coverage (80% minimum)

```bash
for pkg in auth secrets policy tools session state; do
  echo "=== internal/$pkg ==="
  go test ./internal/$pkg/... -coverprofile=/tmp/${pkg}.out -covermode=atomic
  go tool cover -func=/tmp/${pkg}.out | grep total:
done
# Expected: Each package shows >= 80.0% total coverage
```

### 4. Additional Package Tests Pass

```bash
go test ./internal/loop/... ./internal/pipeline/... ./internal/llm/... ./internal/expr/... ./internal/memory/... ./internal/mcp/... ./internal/ir/... ./internal/validate/... ./internal/compiler/... -v -count=1
# Expected: All tests pass
```

### 5. Additional Package Coverage (60% minimum)

```bash
for pkg in loop pipeline llm expr memory mcp ir validate compiler; do
  echo "=== internal/$pkg ==="
  go test ./internal/$pkg/... -coverprofile=/tmp/${pkg}.out -covermode=atomic
  go tool cover -func=/tmp/${pkg}.out | grep total:
done
# Expected: Each package shows >= 60.0% total coverage
```

### 6. Full Test Suite with Race Detector

```bash
go test ./... -race -count=1 -timeout=15m
# Expected: All tests pass, zero data races detected
```

### 7. Overall Coverage Threshold

```bash
go test ./... -coverprofile=/tmp/total.out -covermode=atomic
go tool cover -func=/tmp/total.out | grep total:
# Expected: Total coverage >= 50%
```

### 8. Lint with New Linters

```bash
golangci-lint run ./...
# Expected: Zero violations (including gosec, bodyclose, noctx, contextcheck)
```

### 9. Security Scanning

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
# Expected: No known vulnerabilities found
```

### 10. Code Deduplication Verification

```bash
# Verify single rate limiter implementation
grep -rn "type.*rateLimiter\|type.*RateLimiter" internal/
# Expected: Only internal/auth/ratelimit.go

# Verify single ID generator
grep -rn "func.*generateID\|func.*GenerateID\|func.*generateSecureID\|func.*GenerateSecureID" internal/
# Expected: Only internal/session/id.go (exported)

# Verify indexed lookups
grep -n "findAgent\|findPipeline" internal/runtime/server.go
# Expected: No results (functions removed, replaced with map lookups)
```

### 11. Constitution Amendment

```bash
grep -A 5 "complementary quality gate" .specify/memory/constitution.md
# Expected: Testing Strategy section mentions unit tests as complementary quality gate
```

### 12. Pre-commit Check

```bash
make pre-commit
# Expected: All checks pass
```
