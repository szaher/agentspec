# Risks and Unknowns

**Last updated:** 2026-03-01

## Top 10 Technical Risks

| Rank | Risk | Likelihood | Impact | Evidence | Mitigation |
|------|------|-----------|--------|----------|------------|
| 1 | **State file corruption** loses all infrastructure tracking | Medium | Critical | Non-atomic writes (BUG-005), no locking (BUG-004), single file is SPOF | Atomic writes, file locking, periodic backups, remote state backend option |
| 2 | **Inline tool execution** enables host compromise | High (if untrusted .ias used) | Critical | No sandbox (SEC-004), arbitrary binary execution via command tool (SEC-005) | WASM sandbox for inline, binary allowlist for command, `--no-inline` flag |
| 3 | **Session hijacking** via predictable IDs | Medium | High | Timestamp-based IDs (SEC-001), no session binding to IP/user | Cryptographic IDs, session-IP binding, short TTLs |
| 4 | **No unit test safety net** for 33/34 internal packages | Ongoing | High | Only integration tests exist (QE-001); refactoring is blind | Prioritized unit test campaign starting with security-critical packages |
| 5 | **Memory exhaustion** from unbounded growth in multiple maps | Medium (under load) | High | Rate limiter buckets (PERF-002), session store (PERF-003), memory store (PERF-004) | Eviction goroutines, max-size caps, LRU policies |
| | **Status**: Mitigated in `010-memory-performance` (2026-03-04) — eviction goroutines, max-size caps, and LRU policies implemented for rate limiter, session store, and conversation memory. | | | | |
| 6 | **Race conditions** in shared state | Medium | High | MCP pool (BUG-001), RedactFilter (BUG-009), Redis SaveMessages (PERF-010) | Mutex protection, race detector in CI, channel-based patterns |
| 7 | **Policy engine false security** — rules accepted but not enforced | High (by design) | Medium | `checkRequirement()` always returns true (SEC-016) | Implement actual checks; add integration tests for each policy type |
| 8 | **LLM client errors silently swallowed** | Ongoing | Medium | Discarded JSON errors (BUG-003), ignored session saves (BUG-021) | Handle all errors; log at warning level minimum |
| 9 | **HTTP server vulnerable to DoS** | Medium (if exposed) | Medium | No timeouts (PERF-007), no request limits (SEC-014), CORS wildcard (SEC-012) | Timeouts, MaxBytesReader, configurable CORS, reverse proxy docs |
| 10 | **Compiler generates non-functional code** | Ongoing | Medium | All targets emit "not implemented" stubs (UX-007) | Generate proper skeletons, or clearly mark generated code as template |

## Top 10 Product Risks

| Rank | Risk | Likelihood | Impact | Evidence | Mitigation |
|------|------|-----------|--------|----------|------------|
| 1 | **Users trust policy enforcement** that doesn't work | High | High | Policy engine is a no-op; users define `require` rules thinking they're enforced | Implement or remove; document limitations clearly |
| 2 | **Users deploy without auth** and don't realize it | Medium | High | Silent open access when no API key set (SEC-003) | Warning log, `--no-auth` flag requirement |
| 3 | **8 of 19 CLI commands undocumented** | Ongoing | Medium | README missing init, compile, publish, install, eval, status, logs, destroy | Update README, add `--help` text for all commands |
| 4 | **Compiled code fails at runtime** with "not implemented" | High (if users compile) | Medium | All framework targets produce stubs (UX-007) | Generate functional skeletons or document limitation |
| 5 | **`run` vs `dev` naming confusion** | Medium | Low | `run` does one-shot; `dev` starts server; naming is counterintuitive | Rename `run` to `invoke` or document clearly |
| 6 | **No rollback mechanism** for agent deployments | Medium | Medium | No version history or rollback command | Implement `agentspec rollback` |
| 7 | **No cost visibility** for LLM API usage | Medium | Medium | Token counts tracked but no cost translation or budgets | Implement cost estimation and budget alerts |
| 8 | **Frontend UI is confusing on first use** | High | Low | No empty state, no loading indicator, no error handling | Add UX states |
| 9 | **`publish --sign`** gives false security | Medium | Low | Flag accepted but does nothing (UX-002) | Implement or remove |
| 10 | **`eval` command** can't actually evaluate agents | High (if users try eval) | Medium | Stub invoker always errors (UX-003) | Implement live mode or document limitation |

## Unknowns List

Information needed to fully assess risks and plan remediation:

| # | Unknown | What's Needed | Impact on Analysis |
|---|---------|--------------|-------------------|
| 1 | **Actual deployment patterns** — Is the runtime server exposed to the internet, or always behind a proxy? | User survey or deployment docs | Determines urgency of TLS, CORS, DoS mitigations |
| 2 | **Production usage scale** — How many concurrent sessions, agents, and pipelines? | Load testing results or production metrics | Determines urgency of memory/performance fixes |
| 3 | **Trust model for .ias files** — Are they authored by trusted engineers only, or can untrusted users submit them? | Product requirements | Determines urgency of inline/command sandboxing |
| 4 | **OpenAI client usage** — Is the OpenAI provider actively used, or is it Anthropic-only? | Usage data | If unused, deprioritize OpenAI streaming fixes |
| 5 | **Compiler targets usage** — Are users compiling to CrewAI/LangGraph/etc., or is it aspirational? | Usage data | Determines urgency of fixing generated code |
| 6 | **Plugin ecosystem** — Are WASM plugins being built by third parties, or only internal? | Community activity | Determines urgency of plugin marketplace |
| 7 | **Redis deployment** — Is Redis used in production, or is memory store the default? | Deployment configs | Determines urgency of Redis-specific fixes |
| 8 | **Vault integration** — Is Vault actively used for secret management? | User configs | Determines urgency of Vault-specific improvements |
| 9 | **Multi-agent pipeline scale** — How many steps do typical pipelines have? | Example analysis, user data | Determines urgency of DAG performance fix |
| 10 | **Test execution frequency** — How often are tests run? Only CI or also locally? | Team practices | Determines investment in test speed/ergonomics |
| 11 | **Version compatibility** — Do users need to maintain `.ias` files across AgentSpec versions? | Product strategy | Determines need for schema versioning and migration |
| 12 | **Error handling preferences** — Should the tool fail fast or try to continue on errors? | User feedback | Guides error handling strategy across codebase |

## Architecture Risk Summary

### Strengths
- Clean separation of concerns (37 well-scoped internal packages)
- Deterministic IR with SHA-256 hashing for reliable change detection
- Pluggable patterns throughout (state backend, LLM provider, session store, deployment adapter)
- Hand-written parser with excellent error messages
- Comprehensive integration test suite (31 test files)

### Weaknesses
- No unit test isolation — a single integration test failure could be caused by any of 34 packages
- Security-critical code (auth, secrets, policy, tools) has zero unit tests
- Multiple concurrent-access issues in shared state
- Policy engine is a stub
- Several error paths silently swallow failures
- HTTP server missing production hardening (timeouts, TLS, size limits)

### Systemic Patterns Requiring Attention
1. **"Security surface exists but doesn't work"** pattern: Policy engine, publish --sign, open-access server
2. **"Unbounded growth"** pattern: Rate limiter maps, session stores, memory stores — **Mitigated in `010-memory-performance` (2026-03-04)**
3. **"Swallowed errors"** pattern: LLM clients, session saves, reflexion strategy
4. **"Missing production hardening"** pattern: No timeouts, no TLS, no request limits, no CORS config
