# Tasks: Production Readiness & Advanced Features

**Input**: Design documents from `/specs/012-production-readiness/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add shared infrastructure needed by multiple user stories

- [x] T001 Add version injection via `-ldflags` in `Makefile` — add `VERSION` variable from `git describe --tags`, pass `-ldflags "-X main.version=$(VERSION)"` to `go build`
- [x] T002 Add `version`, `commit`, `date` variables to `cmd/agentspec/main.go` and update `newVersionCmd()` to print them (currently may be hardcoded)
- [x] T003 Create `internal/cost/pricing.go` — embed a default model pricing table (`map[string]ModelPricing`) with entries for claude-sonnet-4-20250514, claude-haiku, gpt-4o, gpt-4o-mini. Include `LookupPrice(model string) (inputPerMTok, outputPerMTok float64)` function
- [x] T004 Create `internal/cost/tracker.go` — implement `CostTracker` struct with `RecordUsage(agent, model string, inputTokens, outputTokens int)` that calculates cost from pricing table and accumulates per-agent totals. Include `GetAgentCost(agent string) float64` and `Reset()`

---

## Phase 2: DSL & IR Foundation (Blocking — needed by US2, US4, US5, US6)

**Purpose**: Extend the parser, AST, IR, validator, and formatter for new blocks and fields. This MUST complete before user stories that depend on the new DSL syntax.

**Independent Test**: Parse a `.ias` file with `user`, `guardrail`, `models`, and `budget` blocks — verify no parse errors and IR contains the new fields.

- [x] T005 Add new token types to `internal/parser/token.go` — add `TokenUser`, `TokenGuardrail`, `TokenModels`, `TokenBudget`, `TokenAgents`, `TokenKeyword`, `TokenPatterns`, `TokenFallbackMsg` (or equivalent keyword tokens for the new block fields)
- [x] T006 Add `UserNode` and `GuardrailNode` AST types to `internal/ast/ast.go` — `UserNode { Name, KeyRef, Agents []string, Role }`, `GuardrailNode { Name, Mode, Keywords []string, Patterns []string, FallbackMsg }`
- [x] T007 Extend `AgentNode` in `internal/ast/ast.go` — add `Models []string`, `BudgetDaily float64`, `BudgetMonthly float64`, `GuardrailRefs []string` fields
- [x] T008 Implement `parseUser()` in `internal/parser/parser.go` — parse `user "name" { key secret("ENV") agents ["a1", "a2"] role "invoke" }` into `UserNode`. Register in the top-level dispatch
- [x] T009 Implement `parseGuardrail()` in `internal/parser/parser.go` — parse `guardrail "name" { mode "block" keywords [...] patterns [...] fallback "msg" }` into `GuardrailNode`. Register in the top-level dispatch
- [x] T010 Extend `parseAgent()` in `internal/parser/parser.go` — add parsing for `models [...]`, `budget daily N`, `budget monthly N`, and `uses guardrail "name"` within agent blocks
- [x] T011 Add `UserDef` and `GuardrailDef` to IR types in `internal/ir/` — `UserDef { Name, KeySecretRef, Agents []string, Role }`, `GuardrailDef { Name, Mode, Keywords, Patterns, FallbackMsg }`
- [x] T012 Extend agent IR in `internal/ir/` — add `Models []string`, `BudgetDaily float64`, `BudgetMonthly float64`, `Guardrails []string` fields to the agent IR struct
- [x] T013 Implement IR lowering for user and guardrail in `internal/ir/lower.go` — lower `UserNode` → `UserDef`, `GuardrailNode` → `GuardrailDef`, and new agent fields to IR
- [x] T014 Add structural validation for user and guardrail blocks in `internal/validate/structural.go` — validate required fields, valid mode values, at least one keyword/pattern
- [x] T015 Add semantic validation in `internal/validate/semantic.go` — validate user key references resolve to declared secrets, user agent references resolve to declared agents, guardrail references in agents resolve to declared guardrails, `model` and `models` are mutually exclusive
- [x] T016 Extend formatter in `internal/formatter/formatter.go` — add formatting rules for `user` and `guardrail` blocks, `models` list, `budget` fields

**Checkpoint**: `agentspec validate` and `agentspec fmt` work with the new DSL blocks.

---

## Phase 3: User Story 1 — Encrypted Communications (Priority: P1)

**Goal**: Server supports TLS with configurable cert/key. Rejects plain HTTP when TLS is enabled. Certificate hot-reload.

**Independent Test**: Start server with `--tls-cert` and `--tls-key`, verify HTTPS works and HTTP is rejected.

- [x] T017 [US1] Add `--tls-cert` and `--tls-key` flags to `cmd/agentspec/run.go` — register string flags, pass to runtime options
- [x] T018 [US1] Add `TLSCert` and `TLSKey` fields to `runtime.Options` in `internal/runtime/runtime.go`
- [x] T019 [US1] Implement TLS server startup in `internal/runtime/server.go` — when TLS cert/key are provided, use `tls.LoadX509KeyPair()` and `http.Server.ListenAndServeTLS()`. When not provided, use `ListenAndServe()` with a warning log
- [x] T020 [US1] Implement certificate hot-reload in `internal/runtime/server.go` — use `tls.Config.GetCertificate` callback that reads the cert file on each TLS handshake (with caching). Watch cert file via fsnotify, invalidate cache on change, log reload
- [x] T021 [US1] Validate TLS cert/key at startup in `internal/runtime/server.go` — if only one of cert/key is provided, return an error. If cert is invalid/expired, return a clear error message
- [x] T022 [US1] Add TLS integration test in `integration_tests/tls_test.go` — test HTTPS startup, cert validation error, and missing cert/key behavior

**Checkpoint**: `agentspec run --tls-cert cert.pem --tls-key key.pem` serves HTTPS.

---

## Phase 4: User Story 2 — Multi-User Access Control (Priority: P1)

**Goal**: Per-user API keys with agent-level permissions and audit logging.

**Independent Test**: Two users with different permissions — verify access control and audit log entries.

### Sub-phase 4a: User Resolution & Auth

- [x] T023 [US2] Create `internal/auth/users.go` — implement `UserStore` struct with `Resolve(apiKey string) (*User, bool)` that maps API keys to user identities. Load from IR `UserDef` list. Include `User { Name, Agents []string, Role }` struct
- [x] T024 [US2] Extend auth middleware in `internal/runtime/server.go` — when `UserStore` is configured, resolve API key to user identity. Set user name in request context. Check agent access on invocation endpoints (403 if not authorized). Fall back to existing single-key mode when no users configured (FR-004)
- [x] T025 [US2] Wire `UserStore` initialization in `internal/runtime/runtime.go` — create `UserStore` from `RuntimeConfig.Users`, resolve secret references to actual key values at startup

### Sub-phase 4b: Audit Logging

- [x] T026 [US2] Create `internal/auth/audit.go` — implement `AuditLogger` struct with `Log(entry AuditEntry)` that writes one JSON line per invocation to the audit log file. Use `log/slog` with a file handler. Include `AuditEntry` struct matching data model
- [x] T027 [US2] Add `--audit-log` flag to `cmd/agentspec/run.go` — register string flag (default: `agentspec-audit.log`), pass to runtime options
- [x] T028 [US2] Integrate audit logging in `internal/runtime/server.go` — after each agent invocation (invoke/stream), write an audit entry with user name, agent name, session ID, token counts, duration, status, and correlation ID
- [x] T029 [US2] Add multi-user auth integration test in `integration_tests/auth_test.go` — test user resolution, permission enforcement (403), audit log output, backward-compatible single-key mode, and hot-reload of user permission changes (modify `.ias` user block, verify new permissions take effect without restart)

**Checkpoint**: Per-user auth works, 403 on unauthorized access, audit log records invocations.

---

## Phase 5: User Story 3 — Agent Observability Dashboard (Priority: P2)

**Goal**: Prometheus metrics endpoint enhanced with cost/fallback/guardrail metrics + Grafana dashboard template.

**Independent Test**: Import Grafana dashboard JSON, verify it references the correct metric names.

- [x] T030 [US3] Add cost metrics and per-model token labels to `internal/telemetry/metrics.go` — add `agentspec_cost_dollars_total` (counter by agent, model) and `agentspec_budget_usage_ratio` (gauge by agent, period) metrics. Also extend existing `agentspec_tokens_total` to include a `model` label (currently only has agent, type) to satisfy FR-006 per-model token tracking
- [x] T031 [US3] Add fallback metrics to `internal/telemetry/metrics.go` — add `agentspec_fallback_total` (counter by agent, from_model, to_model) metric
- [x] T032 [US3] Add guardrail metrics to `internal/telemetry/metrics.go` — add `agentspec_guardrail_violations_total` (counter by agent, guardrail, mode) metric
- [x] T033 [P] [US3] Create `dashboards/agentspec-overview.json` — Grafana dashboard JSON with panels for: invocation rate, latency percentiles (p50/p95/p99), token consumption, cost tracking, tool call patterns, fallback events, guardrail violations, budget usage
- [x] T034 [US3] Add metrics integration test in `integration_tests/metrics_test.go` — verify `/v1/metrics` endpoint includes all new metric names in Prometheus text format

**Checkpoint**: `/v1/metrics` returns all metrics. Grafana dashboard imports successfully.

---

## Phase 6: User Story 4 — Cost Tracking and Budgets (Priority: P2)

**Goal**: Per-agent cost tracking with daily/monthly budgets that pause agents when exceeded.

**Independent Test**: Set a daily budget, exhaust it, verify 429 response.

- [x] T035 [US4] Extend state file schema in `internal/state/local.go` — add `Budgets []BudgetEntry` to the state struct. Implement `LoadBudgets()`, `SaveBudgets()`, `UpdateBudget(agent, period string, usedDollars float64)` methods
- [x] T036 [US4] Implement budget enforcement in `internal/cost/tracker.go` — add `CheckBudget(agent string) error` that returns a budget-exceeded error if the agent's daily or monthly budget is exhausted. Add `RecordAndCheck(agent, model string, inputTokens, outputTokens int) error`
- [x] T037 [US4] Integrate budget checks in `internal/runtime/server.go` — before agent invocation, check budget. On budget exceeded, return 429 with budget details and `Retry-After` header. After successful invocation, update budget usage
- [x] T038 [US4] Implement budget period reset in `internal/cost/tracker.go` — on each budget check, compare current time with `ResetAt`. If past, reset `UsedDollars` to 0 and advance `ResetAt` by one period
- [x] T039 [US4] Wire budget config from IR to runtime in `internal/runtime/runtime.go` — extract `BudgetDaily` and `BudgetMonthly` from agent IR, create budget entries in state file on first apply
- [x] T040 [US4] Add 80% budget warning in `internal/cost/tracker.go` — when usage crosses 80% of limit, log a warning (once per period, tracked by `WarnedAt` field)
- [x] T041 [US4] Add budget integration test in `integration_tests/budget_test.go` — test budget enforcement (429), budget reset, 80% warning, and budget persistence across simulated restart

**Checkpoint**: Budget enforcement works end-to-end. 429 on exceeded budget.

---

## Phase 7: User Story 5 — Multi-Model Fallback (Priority: P2)

**Goal**: Agent tries multiple models in order, falling back on failure/rate-limiting.

**Independent Test**: Simulate primary model failure, verify fallback to secondary.

- [x] T042 [US5] Create `internal/llm/fallback.go` — implement `FallbackClient` struct wrapping `[]Client`. `Chat()` tries each client in order; on error, logs a warning and tries next. Returns error with all failures if all fail. Implements `Client` interface
- [x] T043 [US5] Wire fallback client in `internal/runtime/runtime.go` — when agent config has `Models` (len > 1), create `FallbackClient` with one `Client` per model. When single `Model`, use existing behavior
- [x] T044 [US5] Record fallback events in telemetry — when fallback occurs, increment `agentspec_fallback_total` metric and log warning with from_model and to_model
- [x] T045 [US5] Add fallback integration test in `integration_tests/fallback_test.go` — test successful fallback, all-models-fail error, and fallback metric recording

**Checkpoint**: Fallback chains work. Warning logged. Metrics recorded.

---

## Phase 8: User Story 6 — Agent Guardrails (Priority: P3)

**Goal**: Content filtering on agent output with warn and block modes.

**Independent Test**: Configure keyword blocklist, verify blocked responses are filtered.

- [x] T046 [US6] Create `internal/loop/guardrail.go` — implement `GuardrailFilter` struct with `Check(output string) (filtered string, violations []Violation, err error)`. Match keywords (case-insensitive `strings.Contains`) and patterns (`regexp.MatchString`). In block mode, replace output with fallback message. In warn mode, return original output with violations
- [x] T047 [US6] Wire guardrail filter in `internal/loop/react.go` — after agent produces output, apply guardrail filters from agent config. Log violations. In block mode, replace response. In warn mode, log and pass through
- [x] T048 [US6] Record guardrail violations in telemetry — increment `agentspec_guardrail_violations_total` metric per violation
- [x] T049 [US6] Add guardrail integration test in `integration_tests/guardrail_test.go` — test keyword blocking, regex pattern matching, warn mode (pass-through with log), and multiple guardrails on one agent

**Checkpoint**: Guardrails filter output correctly in both modes.

---

## Phase 9: User Story 7 — Agent Versioning and Rollback (Priority: P3)

**Goal**: Version history stored in state file. Rollback restores previous version.

**Independent Test**: Apply twice, rollback, verify previous version restored.

- [x] T050 [US7] Extend state file schema in `internal/state/local.go` — add `AgentVersions map[string][]VersionEntry` to state struct. Implement `SaveVersion(agent string, entry VersionEntry)` with 10-version retention, and `GetVersions(agent string) []VersionEntry`
- [x] T051 [US7] Record versions on apply in `internal/apply/apply.go` — after successful agent apply, compute change summary (diff of fields), save IR snapshot as new version entry
- [x] T052 [US7] Create `cmd/agentspec/rollback.go` — implement `newRollbackCmd()` with `--agent` flag. Load version history, restore previous version's IR snapshot by writing it directly to the state file as the current agent state entry (bypasses parse/validate pipeline since the IR was already validated when originally applied), create a new version entry for the rollback action
- [x] T053 [US7] Create `cmd/agentspec/history.go` — implement `newHistoryCmd()` with `--agent` flag. Load version history, print table with Version, Timestamp, Summary columns
- [x] T054 [US7] Register rollback and history commands in `cmd/agentspec/main.go`
- [x] T055 [US7] Add versioning integration test in `integration_tests/versioning_test.go` — test version creation on apply, history listing, rollback, and 10-version retention limit

**Checkpoint**: `agentspec history` lists versions. `agentspec rollback` restores previous.

---

## Phase 10: User Story 8 — Release Automation (Priority: P3)

**Goal**: GoReleaser config + GitHub Actions workflow for automated cross-platform releases.

**Independent Test**: Run `goreleaser check` to validate config. Verify workflow YAML is valid.

- [x] T056 [P] [US8] Create `.goreleaser.yaml` — configure build targets (linux/darwin/windows × amd64/arm64), ldflags for version injection, archive formats (.tar.gz/.zip), checksum generation, changelog auto-generation
- [x] T057 [P] [US8] Create `.github/workflows/release.yaml` — GitHub Actions workflow triggered on `v*.*.*` tag push. Steps: checkout, setup-go, run `goreleaser release --clean`
- [x] T058 [US8] Update `cmd/agentspec/main.go` — ensure `version`, `commit`, `date` variables are set via ldflags and printed by `version` command

**Checkpoint**: `goreleaser check` passes. Workflow YAML is valid.

---

## Phase 11: Bug Fix — Tool Call ID Correlation (FR-015 / BUG-010)

**Purpose**: Fix tool result-action correlation to use tool call IDs instead of index-based correlation.

- [x] T059 Wire tool call IDs end-to-end in `internal/loop/react.go` — ensure `ToolCallRecord.ID` is populated from the LLM response `tool_use` block ID, passed through tool execution, and included in the tool result sent back to the LLM. Remove any index-based correlation logic
- [x] T060 Add tool call ID correlation test in `integration_tests/loop_test.go` — test that concurrent tool calls are correctly correlated by ID, not by position

**Checkpoint**: Tool results match tool calls by ID, not index.

---

## Phase 12: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and cleanup across all user stories

- [x] T061 [P] Create example file `examples/production-agent/production-agent.ias` — demonstrate user, guardrail, models, budget blocks together
- [x] T062 [P] Update `README.md` — add rollback, history to CLI commands table. Add cost tracking, guardrails, TLS to feature list
- [x] T063 [P] Update `docs/user-guide/cli/` — add `run.md` TLS flags, create `rollback.md` and `history.md` pages, update `index.md` commands table
- [x] T064 Update integration test for CLI doc completeness in `integration_tests/cli_doc_test.go` — add `rollback` and `history` to the registered commands list
- [x] T065 Run full test suite (`go test ./... -count=1`), linter (`golangci-lint run ./...`), and formatter (`gofmt -l .`) — fix any issues
- [x] T066 Run all quickstart.md scenarios end-to-end to validate the feature works as documented

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1 (Setup)
    │
    └──> Phase 2 (DSL & IR Foundation) ──> Phase 3 (US1: TLS)
              │                              Phase 4 (US2: Auth)
              │                              Phase 6 (US4: Budgets)
              │                              Phase 7 (US5: Fallback)
              │                              Phase 8 (US6: Guardrails)
              │                              Phase 9 (US7: Versioning)
              │
              └──> Phase 5 (US3: Observability) [depends on metrics from US4/US5/US6]

    Phase 10 (US8: Release) ──> independent, can start after Phase 1
    Phase 11 (BUG-010) ──> independent, can start anytime
    Phase 12 (Polish) ──> depends on all user stories complete
```

### Parallel Opportunities

- **After Phase 2**: US1 (TLS), US2 (Auth), US7 (Versioning) can run in parallel (different files)
- **After Phase 2**: US4 (Budgets) and US5 (Fallback) can run in parallel (different files)
- **Anytime**: US8 (Release) and BUG-010 are fully independent
- **Within Phase 12**: T061, T062, T063 can run in parallel (different files)

### User Story Dependencies

- **US3 (Observability)**: Depends on US4, US5, US6 for new metrics (cost, fallback, guardrail)
- **US4 (Budgets)**: Depends on Phase 1 (cost tracker) and Phase 2 (budget DSL)
- **All others**: Depend only on Phase 2 (DSL foundation)

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup
2. Complete Phase 2: DSL & IR Foundation
3. Complete Phase 3: TLS (US1) — encrypted communications
4. Complete Phase 4: Auth (US2) — multi-user access control
5. **STOP and VALIDATE**: TLS works, per-user auth enforced, audit log written
6. Run tests and lint

### Incremental Delivery

1. Setup + DSL Foundation → Foundation ready
2. Add US1 (TLS) + US2 (Auth) → P1 stories complete → Demo
3. Add US4 (Budgets) + US5 (Fallback) → P2 core complete → Demo
4. Add US3 (Observability) → P2 complete (depends on metrics from US4/US5) → Demo
5. Add US6 (Guardrails) + US7 (Versioning) → P3 stories complete → Demo
6. Add US8 (Release) + BUG-010 → All stories complete
7. Polish → Feature complete

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Phase 2 (DSL Foundation) is the main blocker — it enables 6 of 8 user stories
- Total tasks: 66 (T001-T066)
- Commit after each phase or logical group
- No new external Go dependencies for core features (GoReleaser is CI-only)
