# Bug List (Defects)

**Last updated:** 2026-03-01

## Summary

| Severity | Count |
|----------|-------|
| Critical | 5 |
| High | 9 |
| Medium | 11 |
| Low | 13 |
| **Total** | **38** |

## Critical

| ID | Title | Severity | User Impact | Evidence | Root Cause | Proposed Fix | Tests to Add |
|---|---|---|---|---|---|---|---|
| BUG-001 | Race condition in MCP connection pool | Critical | Concurrent tool calls may get wrong connection or panic on nil map access | `internal/mcp/pool.go` — shared `connections` map accessed without mutex in `Get()` and `Close()` | Missing synchronization on shared map | Add `sync.RWMutex`; lock on read/write; use `sync.Map` or channel-based pool | `TestMCPPoolConcurrentAccess`, `TestMCPPoolRaceDetector` |
| BUG-002 | Session ID collision from timestamp | Critical | Two sessions created in same nanosecond get same ID; second overwrites first, losing conversation history | `internal/session/memory_store.go:102-104`, `redis_store.go:181-183` — `fmt.Sprintf("sess_%d", time.Now().UnixNano())` | Timestamp-only ID generation without entropy | Use `crypto/rand` with 128+ bits: `sess_` + base64url(16 random bytes) | `TestSessionIDUniqueness`, `TestConcurrentSessionCreation` |
| BUG-003 | Swallowed JSON unmarshal errors in LLM clients | Critical | Corrupted or unexpected LLM responses silently produce empty/wrong data; tool calls may be lost | `internal/llm/anthropic.go:174` — `_ = json.Unmarshal(block.Input, &input)`; `openai.go:290` — `args, _ := json.Marshal(tc.Input)` | Errors explicitly discarded with `_` | Return errors; wrap with context: `fmt.Errorf("unmarshal tool input: %w", err)` | `TestAnthropicMalformedToolInput`, `TestOpenAIMarshalFailure` |
| BUG-004 | State file has locking infrastructure but never calls Lock/Unlock | Critical | Concurrent `apply` invocations can corrupt the state file (read-modify-write race) | `internal/state/local.go` — `LocalBackend` has no mutex or file locking; `Lock()`/`Unlock()` stubs exist in interface but are no-ops | Locking was designed but never implemented | Implement `flock`-based file locking in `Lock()`/`Unlock()`; call them around `Save()` | `TestConcurrentApply`, `TestStateLocking` |
| BUG-005 | State file writes are not atomic | Critical | Power loss or crash during `os.WriteFile` produces truncated/empty state file; all deployed resource tracking lost | `internal/state/local.go` — `os.WriteFile(b.path, data, 0644)` writes directly | No write-then-rename pattern | Write to temp file, fsync, then `os.Rename` to final path | `TestStateFileAtomicWrite`, `TestStateFileCrashRecovery` |

## High

| ID | Title | Severity | User Impact | Evidence | Root Cause | Proposed Fix | Tests to Add |
|---|---|---|---|---|---|---|---|
| BUG-006 | Command/Inline tools missing OS environment | High | Tools fail to find binaries (no PATH), read configs, or access HOME | `internal/tools/command.go:42-47`, `inline.go:66-71` — `cmd.Env` set to non-nil without `os.Environ()` base | Go exec: non-nil `Env` replaces parent env entirely | Prepend `os.Environ()` before appending secrets, or construct minimal safe env with PATH | `TestCommandToolHasPath`, `TestInlineToolEnvironment` |
| BUG-007 | Rate limiter bucket map grows without bound | High | Memory leak proportional to unique client IPs; eventually OOM under sustained diverse traffic | `internal/auth/ratelimit.go:71-98`, `runtime/server.go:694-746` — maps never evict stale entries | No eviction or max-size cap | Add periodic sweep goroutine (e.g., every 5min remove buckets idle >10min) | `TestRateLimiterEviction`, `TestRateLimiterMemoryBound` |
| | **Status**: Fixed in `010-memory-performance` (2026-03-04) | | | | | | |
| BUG-008 | Memory store sessions never evicted when expired | High | Memory leak; expired sessions accumulate forever unless individually accessed via `Get()` | `internal/session/memory_store.go:54-57` — only `Get()` deletes expired; `List()` skips but doesn't delete | Lazy expiration only on read; no background cleanup | Add background goroutine or evict during `List()` | `TestMemoryStoreExpiredEviction` |
| | **Status**: Fixed in `010-memory-performance` (2026-03-04) | | | | | | |
| BUG-009 | Shared RedactFilter mutated without synchronization | High | Concurrent goroutines adding secrets while filtering may cause race or partial redaction | `internal/secrets/redact.go` — `RedactFilter` struct with string slice, no mutex | Missing synchronization on shared mutable state | Add `sync.RWMutex`; lock writes in `AddSecret()`, read-lock in `Filter()` | `TestRedactFilterConcurrent` |
| BUG-010 | Result-action index correlation may be wrong | High | Tool results mapped to wrong action; agent sees incorrect tool outputs | `internal/loop/react.go` — tool results correlated by index position in response blocks | Index-based correlation assumes ordering; out-of-order or filtered blocks break it | Correlate by tool call ID (already present in response) instead of index | `TestToolResultCorrelation`, `TestOutOfOrderToolCalls` |
| BUG-011 | OpenAI streaming SSE parsing incomplete | High | Streaming responses from OpenAI may be truncated or miss events | `internal/llm/openai.go` — SSE reader implementation may not handle multi-line data fields or reconnection | Simplified SSE parser | Use a proper SSE client library or fully implement SSE spec | `TestOpenAIStreamingEdgeCases` |
| BUG-012 | Missing context cancellation propagation in tool execution | High | Tool executions don't respect parent context cancellation; agent hangs on long-running tools | `internal/tools/` — some executors create commands without propagating deadline | Context not forwarded consistently | Ensure all `exec.CommandContext` uses the passed context; add timeout fallback | `TestToolCancellation` |
| BUG-013 | Pipeline executor ignores DAG ordering | High | Steps that should run in topological order may execute in arbitrary order | `internal/pipeline/executor.go` — execution may not respect layer ordering from `TopologicalSort()` | Sort result not used correctly in execution loop | Verify execution follows `TopologicalSort()` layer ordering | `TestPipelineExecutionOrder` |
| BUG-014 | Sliding window memory unbounded across sessions | High | All session histories stored in single map without cap; OOM under many sessions | `internal/memory/sliding.go` — `sessions map[string][]llm.Message` grows without limit | No max-sessions cap or LRU eviction | Cap sessions; evict LRU; use session store for overflow | `TestSlidingWindowSessionLimit` |
| | **Status**: Fixed in `010-memory-performance` (2026-03-04) | | | | | | |

## Medium

| ID | Title | Severity | User Impact | Evidence | Root Cause | Proposed Fix | Tests to Add |
|---|---|---|---|---|---|---|---|
| BUG-015 | Multiple .ias files on validate/plan silently use only first | Medium | User thinks all files are validated but only first is processed | `cmd/agentspec/validate.go`, `plan.go` — `args[0]` used; rest ignored | CLI accepts variadic args but only processes first | Either process all files or return error if >1 provided | `TestValidateMultipleFiles` |
| BUG-016 | Token budget can be exceeded | Medium | Agent runs may consume more tokens than configured budget allows | `internal/loop/react.go` — budget checked after LLM call returns, not before | Post-hoc check; single call can blow budget | Estimate tokens before calling LLM; abort if remaining budget < estimated need | `TestTokenBudgetEnforcement` |
| BUG-017 | Go template injection in HTTP tool body | Medium | LLM-controlled inputs rendered via `text/template` without escaping | `internal/tools/http.go:47-54` — `template.New("body").Parse(e.config.BodyTemplate)` | Using `text/template` with untrusted input | Use `json.Marshal` for JSON bodies; sanitize template inputs | `TestHTTPToolBodyInjection` |
| BUG-018 | Timing side-channel in auth key comparison | Medium | API key can be discovered character-by-character | `internal/runtime/server.go:158` — `key != s.apiKey` (plain comparison) | Not using constant-time comparison | Use `subtle.ConstantTimeCompare` or existing `auth.ValidateKey()` | `TestAuthTimingAttack` |
| BUG-019 | CORS wildcard allows cross-origin agent invocation | Medium | Any website can invoke agents on a user's local server | `internal/frontend/sse.go:32` — `Access-Control-Allow-Origin: *` | Hardcoded wildcard | Make configurable; restrict to frontend origin | `TestCORSRestriction` |
| BUG-020 | Silent error swallowing in reflexion strategy | Medium | Self-reflection failures silently ignored; agent proceeds with potentially wrong analysis | `internal/loop/` — reflexion error handling path | Error not propagated to caller | Log warning and include error context in next iteration | `TestReflexionErrorHandling` |
| BUG-021 | Session save failures silently dropped | Medium | Conversation history lost without user notification | `internal/runtime/server.go:265,435` — `_ = s.sessions.SaveMessages(...)` | Error explicitly discarded | Log error; consider returning 500 to client on save failure | `TestSessionSaveFailure` |
| BUG-022 | Redis SaveMessages has read-modify-write race | Medium | Two concurrent saves to same session can lose messages | `internal/session/redis_store.go:153-163` — loads all, appends, saves back | No locking on read-modify-write | Use Redis `RPUSH` for append; or add distributed lock | `TestRedisConcurrentSaveMessages` |
| BUG-023 | HTTP tool no response body size limit | Medium | Malicious endpoint returns huge response; OOM | `internal/tools/http.go:82` — `io.ReadAll(resp.Body)` | No size limit | Use `io.LimitReader(resp.Body, 10*1024*1024)` | `TestHTTPToolLargeResponse` |
| BUG-024 | No request body size limit on API endpoints | Medium | Attacker sends huge request body; OOM | `internal/runtime/server.go:206,310,402,465` — `json.NewDecoder(r.Body).Decode()` | No `http.MaxBytesReader` | Wrap body with `http.MaxBytesReader(w, r.Body, maxSize)` | `TestRequestBodySizeLimit` |
| BUG-025 | No HTTP server timeouts | Medium | Slow-loris attack; connection exhaustion | `internal/runtime/server.go:114-121` — no timeout fields set | Missing timeout configuration | Set `ReadHeaderTimeout`, `ReadTimeout`, `IdleTimeout` | `TestServerTimeouts` |

## Low

| ID | Title | Severity | User Impact | Evidence | Root Cause | Proposed Fix | Tests to Add |
|---|---|---|---|---|---|---|---|
| BUG-026 | UTF-8 splitting in sliding window | Low | Multi-byte characters at message boundary may be corrupted | `internal/memory/sliding.go` — slicing by message count, not character-aware | Unlikely with message-level slicing but possible with token-level | Validate message integrity at boundaries | `TestSlidingWindowUTF8` |
| BUG-027 | Unsanitized LLM output used as identifiers | Low | LLM-generated content used in map keys or file paths without sanitization | Various loop/tool files | Trust in LLM output format | Sanitize all LLM-provided identifiers | `TestLLMOutputSanitization` |
| BUG-028 | Polling-based file watcher in dev mode | Low | 2-second minimum latency; CPU overhead on large dirs | `cmd/agentspec/dev.go:100,116-128` — `time.NewTicker(2 * time.Second)` + `filepath.Walk` | No fsnotify usage | Use `fsnotify` library | `TestDevModeFileWatch` |
| BUG-029 | Scanner errors ignored in parser | Low | Very large files or malformed input may cause scanner errors that are silently ignored | `internal/parser/` — scanner error handling | Missing error check | Add scanner error handling | `TestParserScannerErrors` |
| BUG-030 | IP + port used in rate limiter key | Low | Multiple clients behind NAT share rate limit; port-specific keys allow bypass | `internal/auth/ratelimit.go` — key includes port | Using raw RemoteAddr | Extract IP only from RemoteAddr; handle X-Forwarded-For | `TestRateLimiterIPExtraction` |
| BUG-031 | X-Forwarded-For spoofing in rate limiter | Low | Client can set fake X-Forwarded-For to bypass rate limits | `internal/auth/ratelimit.go` — may trust X-Forwarded-For | No trusted proxy configuration | Only trust X-Forwarded-For from known proxies | `TestRateLimiterXFFSpoofing` |
| BUG-032 | Vault token stored as plaintext string | Low | Memory dumps expose Vault token | `internal/secrets/vault.go:21` — `Token string` | Go string immutability | Accept; document as known limitation; consider token renewal | — |
| BUG-033 | WASM plugin stdout goes to host stdout | Low | Plugin output mixed with AgentSpec output | `internal/plugins/host.go:51-53` — `WithStdout(os.Stdout)` | Direct stdout passthrough | Capture to buffer; log separately | `TestPluginStdoutIsolation` |
| BUG-034 | API key stored in sessionStorage in frontend | Low | XSS can exfiltrate API key | `internal/frontend/web/app.js:9` — `sessionStorage.getItem("agentspec_api_key")` | Browser-side storage | Document as dev-only; use HttpOnly cookie for prod | — |
| BUG-035 | State file Get() reads entire file per call | Low | Performance degradation with large state files | `internal/state/local.go:62-74` — `Load()` called in `Get()` | No in-memory caching | Cache entries after Load(); invalidate on Save() | `TestStateGetCaching` |
| | **Status**: Fixed in `010-memory-performance` (2026-03-04) | | | | | | |
| BUG-036 | Duplicate rate limiter implementations | Low | Bug fixes must be applied in two places | `internal/auth/ratelimit.go` vs `internal/runtime/server.go:694-746` | Copy-paste | Consolidate into single generic rate limiter | `TestRateLimiterGeneric` |
| BUG-037 | findAgent/findPipeline use linear scan | Low | O(n) lookup per request for agents/pipelines | `internal/runtime/server.go:676-692` | Slice-based storage | Build `map[string]*AgentConfig` at server creation | `TestAgentLookupPerformance` |
| | **Status**: Fixed in `010-memory-performance` (2026-03-04) | | | | | | |
| BUG-038 | Ignored marshal error in Anthropic client | Low | Tool input schema silently empty if marshal fails | `internal/llm/anthropic.go:140` — `schemaBytes, _ := json.Marshal(t.InputSchema)` | Error discarded | Return error | `TestAnthropicSchemaMarshal` |
