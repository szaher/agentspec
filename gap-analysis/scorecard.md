# Feature Area Scorecard

**Last updated:** 2026-03-01

Scoring: 0 = Non-existent, 1 = Critical gaps, 2 = Major gaps, 3 = Functional with gaps, 4 = Good with minor issues, 5 = Excellent

## Scores by Feature Area

| Feature Area | Reliability | Security | Performance | Maintainability | UX Completeness | Observability | Avg |
|---|---|---|---|---|---|---|---|
| DSL Parsing | 4 | 5 | 3 | 3 | 4 | 4 | 3.8 |
| Validation | 4 | 4 | 4 | 4 | 4 | 3 | 3.8 |
| Planning | 4 | 4 | 3 | 4 | 4 | 3 | 3.7 |
| Applying | 3 | 3 | 3 | 4 | 3 | 3 | 3.2 |
| Runtime Server | 2 | 2 | 2 | 3 | 3 | 3 | 2.5 |
| Agent Loop | 3 | 3 | 3 | 3 | 3 | 3 | 3.0 |
| Tool Execution | 2 | 1 | 3 | 3 | 3 | 2 | 2.3 |
| Session Mgmt | 2 | 1 | 2 | 3 | 3 | 1 | 2.0 |
| Pipeline Exec | 3 | 3 | 2 | 3 | 3 | 2 | 2.7 |
| Secret Mgmt | 3 | 3 | 4 | 4 | 3 | 3 | 3.3 |
| Plugin System | 3 | 4 | 4 | 3 | 3 | 2 | 3.2 |
| Compilation | 3 | 4 | 4 | 3 | 2 | 1 | 2.8 |
| Policy Engine | 1 | 1 | 5 | 3 | 2 | 1 | 2.2 |
| Frontend UI | 3 | 2 | 3 | 3 | 2 | 1 | 2.3 |
| LLM Integration | 3 | 3 | 3 | 3 | 3 | 2 | 2.8 |
| Package Mgmt | 3 | 3 | 4 | 3 | 2 | 1 | 2.7 |

## Justifications

### DSL Parsing (Avg: 3.8)
- **Reliability 4:** Hand-written parser with good error recovery and position tracking; tested via golden tests
- **Security 5:** No external input execution; pure text processing
- **Performance 3:** 1,964-line monolith file; functional but hard to optimize in isolation (PERF-001)
- **Maintainability 3:** Single large file makes modifications risky; only 1 unit test file for parser
- **UX Completeness 4:** Good error messages with line/column; "did you mean?" suggestions
- **Observability 4:** Errors include source positions and context

### Validation (Avg: 3.8)
- **Reliability 4:** Structural + semantic passes catch many issues; tested
- **Security 4:** Validates policy blocks; does not validate policy enforcement (deferred to apply)
- **Performance 4:** Linear scan of AST; adequate for expected file sizes
- **Maintainability 4:** Clean split between structural.go and semantic.go
- **UX Completeness 4:** Clear error messages with file/line references
- **Observability 3:** Errors reported but no metrics on validation failures

### Planning (Avg: 3.7)
- **Reliability 4:** Deterministic diff via SHA-256 hashing; tested with golden fixtures
- **Security 4:** No sensitive data exposure in plan output
- **Performance 3:** Reads entire state file per Get() call (PERF-005)
- **Maintainability 4:** Clean separation of concerns (plan.go, 181 LOC)
- **UX Completeness 4:** Clear create/update/delete/noop output
- **Observability 3:** Plan output is user-visible but not logged as telemetry

### Applying (Avg: 3.2)
- **Reliability 3:** Partial failure handling exists, but state writes are not atomic (BUG-04/05)
- **Security 3:** Policy engine check is called but checkRequirement() is a stub (SEC-016)
- **Performance 3:** State file re-read on every operation (PERF-005)
- **Maintainability 4:** Clean code (126 LOC) with clear error handling
- **UX Completeness 3:** Status output per resource; no rollback on partial failure
- **Observability 3:** Status logged per resource

### Runtime Server (Avg: 2.5)
- **Reliability 2:** No HTTP timeouts (PERF-007); no request size limits (SEC-014); race in rate limiter (PERF-002)
- **Security 2:** Non-constant-time key comparison (SEC-002); silent open access (SEC-003); no TLS (SEC-013); CORS wildcard (SEC-012)
- **Performance 2:** Missing timeouts; unbounded rate limiter map growth; linear agent lookup (PERF-008)
- **Maintainability 3:** server.go is ~750 LOC; clear but growing
- **UX Completeness 3:** All documented endpoints exist; missing HTTPS
- **Observability 3:** /v1/metrics and /healthz endpoints; no request logging

### Agent Loop (Avg: 3.0)
- **Reliability 3:** ReAct and Reflexion strategies implemented; max iterations prevent infinite loops; silent error swallowing in reflexion (BUG-34)
- **Security 3:** Tool execution delegates to tool executors; no additional sandboxing
- **Performance 3:** Token budget checking exists but overrun possible (BUG-24)
- **Maintainability 3:** 7 files, ~700 LOC; clean strategy pattern
- **UX Completeness 3:** Streaming support via SSE; tool call visibility
- **Observability 3:** Event emission for tool calls and completions

### Tool Execution (Avg: 2.3)
- **Reliability 2:** Missing OS environment in command executor (BUG-09/SEC-011); inline has no resource limits
- **Security 1:** Inline execution without sandboxing (SEC-004); command without allowlist (SEC-005); HTTP without SSRF protection (SEC-006); no response size limit (SEC-007)
- **Performance 3:** Adequate for expected usage; no connection pooling for HTTP
- **Maintainability 3:** Clean executor interface; 4 implementations
- **UX Completeness 3:** All 4 tool types functional
- **Observability 2:** No tool execution metrics or logging

### Session Management (Avg: 2.0)
- **Reliability 2:** Predictable session IDs (SEC-001); no eviction of expired sessions (PERF-003); Redis SaveMessages race (PERF-010)
- **Security 1:** Timestamp-based IDs enable enumeration/hijacking (SEC-001)
- **Performance 2:** Memory store leaks expired sessions; Redis uses KEYS * (PERF-009)
- **Maintainability 3:** Clean Store interface; 2 implementations
- **UX Completeness 3:** CRUD operations implemented; list with agent filtering
- **Observability 1:** No session metrics or logging

### Pipeline Execution (Avg: 2.7)
- **Reliability 3:** DAG topological sort with cycle detection; step failure handling
- **Security 3:** No additional concerns beyond agent loop security
- **Performance 2:** Quadratic DAG sort (PERF-006); no parallelism within layers (sequential execution)
- **Maintainability 3:** Clean DAG + executor split
- **UX Completeness 3:** Pipeline execution works; no progress reporting
- **Observability 2:** No pipeline-level metrics or progress events

### Policy Engine (Avg: 2.2)
- **Reliability 1:** checkRequirement() always returns true (SEC-016) — policies are not enforced
- **Security 1:** Core purpose is security, but it's a no-op stub
- **Performance 5:** Trivially fast (because it does nothing)
- **Maintainability 3:** Simple code (100 LOC); clean structure for future implementation
- **UX Completeness 2:** Users can define policies, but they have no effect
- **Observability 1:** No logging when policies are applied or bypassed

### Frontend UI (Avg: 2.3)
- **Reliability 3:** SSE streaming works; basic chat interface functional
- **Security 2:** API key in sessionStorage (SEC-017); CORS wildcard (SEC-012)
- **Performance 3:** Lightweight vanilla JS; no framework overhead
- **Maintainability 3:** Single file (18.5KB); manageable for current scope
- **UX Completeness 2:** Missing loading, error, empty states (UX-004); no markdown rendering
- **Observability 1:** No frontend error reporting or analytics

### LLM Integration (Avg: 2.8)
- **Reliability 3:** Anthropic client functional; OpenAI streaming may be broken (BUG findings); swallowed JSON errors (BUG-03)
- **Security 3:** API keys from environment; no key rotation support
- **Performance 3:** Token counting for budget management
- **Maintainability 3:** Clean interface; 3 implementations (Anthropic, OpenAI, Mock)
- **UX Completeness 3:** Tool calling, streaming supported
- **Observability 2:** Token metrics tracked; no request-level logging

## Top Opportunities (Lowest Scores)

| Rank | Feature Area | Avg Score | Critical Gaps |
|------|-------------|-----------|---------------|
| 1 | Session Management | 2.0 | Predictable IDs, no eviction, Redis KEYS, race conditions |
| 2 | Policy Engine | 2.2 | checkRequirement() is a no-op; policies provide false security |
| 3 | Tool Execution | 2.3 | No sandboxing for inline; no SSRF protection; no allowlist |
| 4 | Frontend UI | 2.3 | Missing states; API key in sessionStorage; CORS wildcard |
| 5 | Runtime Server | 2.5 | No timeouts; timing attack; open access; no TLS |
| 6 | Pipeline Execution | 2.7 | Quadratic DAG; no parallelism; no progress reporting |
| 7 | Package Management | 2.7 | publish --sign is a no-op; limited documentation |
| 8 | Compilation | 2.8 | All targets generate "not implemented" tool stubs |
| 9 | LLM Integration | 2.8 | OpenAI streaming issues; swallowed errors |
| 10 | Agent Loop | 3.0 | Token budget overrun; silent error swallowing in reflexion |
