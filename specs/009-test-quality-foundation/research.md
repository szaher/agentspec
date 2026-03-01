# Research: Test Foundation & Quality Engineering

**Feature**: 009-test-quality-foundation
**Date**: 2026-03-01

## R1: Go Coverage Tooling

**Decision**: Use `go test -coverprofile=coverage.out -covermode=atomic` with `go tool cover -func` for threshold checking.

**Rationale**: Built-in Go tooling is the most reliable and requires no external dependencies. `-covermode=atomic` is required when using `-race` flag. `go tool cover -func` outputs per-function coverage with a total at the bottom, which can be parsed with `awk` or `grep` for threshold enforcement.

**Alternatives considered**:
- Codecov / Coveralls: External services with GitHub integration. Adds external dependency and requires account setup. Can be added later for PR-level coverage diff visualization.
- `gocover-cobertura`: Converts Go coverage to Cobertura XML format. Useful for CI integrations but adds complexity.

## R2: govulncheck Integration

**Decision**: Use `golang.org/x/vuln/cmd/govulncheck` as a standalone CI step.

**Rationale**: `govulncheck` is the official Go vulnerability scanner maintained by the Go security team. It uses the Go Vulnerability Database and performs call graph analysis — only reporting vulnerabilities that are actually reachable in the code, reducing false positives.

**Alternatives considered**:
- Trivy: Language-agnostic scanner with broader coverage but less Go-specific analysis. No call graph analysis — reports all vulnerable packages regardless of reachability.
- Snyk: Commercial tool with free tier. Requires account and API key. More suitable for enterprise environments.
- `nancy` (Sonatype): OSS dependency scanner. Less maintained than govulncheck.

**Integration**: Install via `go install golang.org/x/vuln/cmd/govulncheck@latest` in CI, run `govulncheck ./...`.

## R3: gosec Integration via golangci-lint

**Decision**: Enable `gosec` linter in existing `.golangci.yml` rather than running it standalone.

**Rationale**: The project already uses `golangci-lint v2.10.1` in CI. Adding `gosec` as an enabled linter avoids a separate CI step and shares the existing linter configuration. `gosec` checks for common security issues: hardcoded credentials, SQL injection, weak crypto, file permissions, etc.

**Alternatives considered**:
- Standalone `gosec` binary: Requires separate installation and CI step. Less integrated with existing lint workflow.
- Semgrep: Language-agnostic with broader rule sets but requires separate infrastructure and rule configuration.

## R4: Additional Linters for FR-007

**Decision**: Enable `bodyclose`, `noctx`, `contextcheck`, `gocritic`, `unconvert`, `misspell` in `.golangci.yml`.

**Rationale**:
- `bodyclose`: Detects unclosed HTTP response bodies — a common resource leak in Go HTTP clients. Critical for `internal/tools/http.go` and any code making HTTP requests.
- `noctx`: Detects HTTP requests made without `context.Context` — important for request cancellation and timeout propagation.
- `contextcheck`: Detects functions that should accept `context.Context` but don't, or that break context propagation chains.
- `gocritic`: Broad code correctness checks including diagnostic, performance, and style rules.
- `unconvert`: Catches unnecessary type conversions that add noise.
- `misspell`: Catches common spelling mistakes in comments and strings.

**Alternatives considered**:
- `wrapcheck`: Checks that errors from external packages are wrapped. Useful but too strict for initial rollout — would produce many violations.
- `exhaustive`: Checks exhaustiveness of enum switch statements. Useful but not security-critical.

## R5: Race Detection in CI

**Decision**: Run `go test ./... -race -count=1 -timeout=15m` as a separate CI job.

**Rationale**: The race detector adds 5-10x slowdown to test execution. Running it as a separate job with extended timeout (15 minutes vs 10 minutes for normal tests) prevents blocking the main test pipeline. The `-count=1` flag disables test caching to ensure races are checked on every run.

**Alternatives considered**:
- Running race detection in the main test job: Would significantly slow down the CI feedback loop for every PR.
- Running race detection only on main branch: Misses races introduced in PRs, defeating the purpose.

## R6: Rate Limiter Consolidation Strategy

**Decision**: Consolidate to `internal/auth/ratelimit.go` as the canonical implementation. The `runtime/server.go` inline rate limiter is deleted and replaced with `auth.NewRateLimiter()`.

**Rationale**: The `auth` package rate limiter is the more complete implementation — it has exported types, configurable options, env var support, auth failure tracking, and middleware. The `runtime/server.go` version is a simplified duplicate with identical token bucket math.

**Migration**:
1. Ensure `auth.RateLimiter.Allow(key)` works for per-agent rate limiting (currently per-IP)
2. `runtime/server.go` creates `auth.NewRateLimiter(config)` with per-agent key function
3. Delete `rateLimiter`, `tokenBucket`, `newRateLimiter()`, `allow()` from `server.go`

## R7: Session ID Generator Consolidation Strategy

**Decision**: Export `GenerateSecureID()` from `internal/session/id.go`. Replace all inline ID generation across the codebase.

**Rationale**: The session package implementation uses `crypto/rand` which is the only cryptographically secure approach. The `telemetry/traces.go` implementation using `time.Now().UnixNano()` with `time.Sleep(time.Nanosecond)` is both insecure (predictable) and slow (blocks on sleep). The `runtime/server.go` inline ID is also time-based and predictable.

**Migration**:
1. Export `GenerateSecureID()` and add optional prefix parameter: `GenerateID(prefix string) string`
2. `telemetry/logger.go`: Use `session.GenerateID("cor_")` for correlation IDs
3. `telemetry/traces.go`: Use `session.GenerateID("tr_")` for trace IDs
4. `runtime/server.go`: Use `session.GenerateID("cf_")` for control flow IDs

## R8: Indexed Lookup Strategy

**Decision**: Add `map[string]*AgentConfig` and `map[string]*PipelineConfig` to `Server` struct, populated during `NewServer()`.

**Rationale**: `findAgent()` is called 6 times per request flow (invoke, stream, session create, session message, pipeline execution, delegation). With O(1) map lookup, this eliminates linear scan overhead. The maps are built once at server startup from the existing config slices.

**Migration**:
1. Add `agentsByName` and `pipelinesByName` maps to `Server` struct
2. Populate in `NewServer()` from `config.Agents` and `config.Pipelines`
3. Replace `findAgent(name)` with `s.agentsByName[name]`
4. Replace `findPipeline(name)` with `s.pipelinesByName[name]`
5. Delete `findAgent()` and `findPipeline()` methods

## R9: Coverage Threshold Enforcement Approach

**Decision**: Use a shell script step in CI that parses `go tool cover -func` output and compares the total against a threshold variable.

**Rationale**: The simplest approach with no external dependencies. The total coverage line from `go tool cover -func` follows the format: `total:  (statements)  XX.X%`. This can be parsed with `grep` and `awk`. The threshold is stored as a workflow-level `env` variable for easy manual updates.

**Implementation**:
```bash
COVERAGE=$(go tool cover -func=coverage.out | grep total: | awk '{print $3}' | tr -d '%')
if (( $(echo "$COVERAGE < $COVERAGE_THRESHOLD" | bc -l) )); then
  echo "Coverage ${COVERAGE}% is below threshold ${COVERAGE_THRESHOLD}%"
  exit 1
fi
```
