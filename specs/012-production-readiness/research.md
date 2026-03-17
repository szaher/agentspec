# Research: Production Readiness & Advanced Features

**Feature Branch**: `012-production-readiness`
**Date**: 2026-03-17

## Research Decisions

### 1. TLS Implementation in Go HTTP Server

- **Decision**: Use `http.Server.ListenAndServeTLS()` with `--tls-cert` and `--tls-key` CLI flags
- **Rationale**: Go stdlib provides production-grade TLS. No external dependency needed. The existing `runtime.Server` already configures `http.Server` with timeouts — adding TLS is a flag check at startup.
- **Alternatives considered**: Reverse proxy (nginx/envoy) in front — adds deployment complexity; `autocert` (Let's Encrypt) — out of scope per spec assumptions.
- **Certificate reload**: Use `tls.Config.GetCertificate` with a file watcher (fsnotify already a dependency) to support hot-reload without restart.

### 2. Per-User API Key Authentication

- **Decision**: Extend the existing auth middleware to support multiple API keys mapped to user identities. Keys defined inline in `.ias` files via a new `user` block.
- **Rationale**: The server already validates a single API key via `X-API-Key` or `Authorization: Bearer`. Extending to a key→user lookup table is minimal change. The `.ias` parser already supports block-based definitions.
- **Alternatives considered**: External user store (database) — too heavy for MVP; JWT tokens — adds token lifecycle complexity; basic auth — less secure than API keys.
- **DSL syntax**: `user "alice" { key secret("ALICE_API_KEY") agents ["support-agent", "search-agent"] role "invoke" }`

### 3. Audit Log Implementation

- **Decision**: Structured JSON log file (`agentspec-audit.log`) using `log/slog` with a dedicated `slog.Handler`
- **Rationale**: `slog` is already used throughout the codebase. A separate file handler ensures audit entries don't mix with application logs. JSON format is compatible with log aggregation tools (Loki, ELK, CloudWatch).
- **Alternatives considered**: State file — too much I/O contention; stdout — mixes with app logs; database — out of scope.
- **Configurable path**: `--audit-log` flag, defaults to `agentspec-audit.log` in the working directory.

### 4. Prometheus Metrics Endpoint

- **Decision**: Keep the existing custom Prometheus text formatter in `internal/telemetry/metrics.go`. Add new metrics for cost, fallback, and guardrail events.
- **Rationale**: The existing formatter already outputs valid Prometheus exposition format. Adding the `prometheus/client_golang` dependency would duplicate the existing implementation. The custom formatter is simpler and sufficient.
- **Alternatives considered**: `prometheus/client_golang` library — heavier dependency, existing formatter works; OpenTelemetry — more complex, targets a different audience.
- **New metrics**: `agentspec_cost_dollars_total`, `agentspec_fallback_total`, `agentspec_guardrail_violations_total`, `agentspec_budget_usage_ratio`.

### 5. Budget Tracking in State File

- **Decision**: Add a `budgets` section to the state file with per-agent daily/monthly usage counters and reset timestamps.
- **Rationale**: The state file already has atomic write semantics with file locking. Budget state is small and low-write-frequency (updated per invocation). Persisting in the state file means budget state survives restarts without new dependencies.
- **Alternatives considered**: Separate budget file — adds file management complexity; in-memory only — loses budget state on restart.
- **Schema**: `{"agent": "...", "period": "daily|monthly", "limit_dollars": 10.0, "used_dollars": 3.50, "reset_at": "2026-03-18T00:00:00Z"}`

### 6. Multi-Model Fallback Chain

- **Decision**: Add a `fallback` list to agent config in the IR. The LLM client tries each model in order. The existing `on_error "fallback"` + `fallback "agent"` pattern is for agent-level fallback; model fallback is a new list within the same agent.
- **Rationale**: The existing `llm.Client` interface has `Chat()` and `ChatStream()`. A wrapper client that tries multiple providers in sequence is a clean composition pattern. No changes to the loop layer needed.
- **Alternatives considered**: Circuit breaker pattern — too complex for MVP; load balancer — different use case (distribution vs. failover).
- **DSL syntax**: `models ["claude-sonnet-4-20250514", "gpt-4o-mini"]` in agent block — first is primary, rest are fallbacks.

### 7. Content Guardrails

- **Decision**: Post-processing filter applied to agent output in the loop layer, before returning to the caller. Configurable via a `guardrail` block in the `.ias` file.
- **Rationale**: Filtering at the loop level catches all output paths (sync and stream). Keyword matching is simple string search; regex is `regexp.Compile`. Topic restrictions use a deny-list of keywords (not LLM-based classification — too expensive and recursive).
- **Alternatives considered**: LLM-based content classification — adds latency and cost, recursive LLM calls; input filtering — doesn't prevent LLM from generating harmful output.
- **DSL syntax**: `guardrail "safety" { mode "block" keywords ["password", "credit card"] fallback "I cannot provide that information." }`

### 8. Agent Versioning in State File

- **Decision**: Store last 10 IR snapshots per agent in the state file under a `versions` key. Each version includes timestamp, IR hash, and a summary of changed fields.
- **Rationale**: The state file already tracks agent state. Adding version history is an extension of the existing schema. The IR is already content-hashed — comparing hashes detects changes. Limiting to 10 versions keeps the state file manageable.
- **Alternatives considered**: Git-based versioning (rely on git history) — requires git, doesn't work in containerized deployments; separate version database — overkill for 10 snapshots.

### 9. Release Automation with GoReleaser

- **Decision**: Use GoReleaser via GitHub Actions for cross-platform binary builds, checksums, and GitHub Releases with auto-generated changelog.
- **Rationale**: GoReleaser is the standard tool for Go binary releases. It handles cross-compilation, checksum generation, and GitHub Release creation in a single configuration file. The project already uses GitHub Actions for CI.
- **Alternatives considered**: Manual `go build` scripts — error-prone, no changelog; `goreleaser` alternatives (ko, xgo) — less mature for CLI tools.
- **Version injection**: `-ldflags "-X main.version=${GITHUB_REF_NAME}"` — standard Go pattern for build-time version strings.

### 10. Tool Call ID Correlation (FR-015 / BUG-010)

- **Decision**: Use the `ToolCallRecord.ID` field (already present) to correlate tool results with tool calls. Pass the ID from the LLM response through tool execution and back in the result.
- **Rationale**: The Anthropic API returns `tool_use` blocks with an `id` field. The existing `ToolCallRecord` struct has an `ID` field but it may not be wired end-to-end. Fixing this is a targeted change in the loop layer.
- **Alternatives considered**: Index-based correlation (current behavior) — fragile when tools execute concurrently or when some fail.
