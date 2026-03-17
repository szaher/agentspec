# Tasks: Product Completeness & UX

**Input**: Design documents from `/specs/011-product-completeness/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add new dependency and prepare for implementation

- [x] T001 Add `github.com/fsnotify/fsnotify` dependency via `go get github.com/fsnotify/fsnotify` and run `go mod tidy`

---

## Phase 2: Foundational — Command Rename (US5, Priority: P3 but blocking)

**Purpose**: Swap `run` ↔ `dev` command semantics. This MUST complete before documentation (US1) since it changes all command descriptions.

**Goal**: `run` starts the server (was `dev`), `dev` does one-shot invocation (was `run`). Deprecation aliases emit warnings.

**Independent Test**: Run `agentspec run --help` → should say "Start agent runtime server". Run `agentspec dev --help` → should say "One-shot agent invocation".

- [x] T002 [US5] Create deprecation alias helper function in `cmd/agentspec/main.go` — a function `newDeprecatedAlias(oldName, newName string, actual *cobra.Command) *cobra.Command` that wraps a command to print a stderr deprecation warning before delegating to the actual command
- [x] T003 [US5] Swap command logic: move server/hot-reload logic from `cmd/agentspec/dev.go` into `cmd/agentspec/run.go` (update Use, Short, Long fields to describe server behavior). Move one-shot invocation logic from `cmd/agentspec/run.go` into `cmd/agentspec/dev.go` (update Use, Short, Long fields to describe one-shot behavior). Update `runDevLoop` references accordingly
- [x] T004 [US5] Register deprecation aliases in `cmd/agentspec/main.go` — add aliases so old `run` (one-shot) behavior is accessible via a deprecated name and old `dev` (server) behavior is accessible via a deprecated name. Both must print deprecation warnings to stderr
- [x] T005 [US5] Update `cmd/agentspec/dev.go` help text to clearly differentiate one-shot behavior: `Short: "Invoke an agent and print the response"`, `Long: "One-shot agent invocation: parse, validate, invoke with LLM, print response, exit."`
- [x] T006 [US5] Update `cmd/agentspec/run.go` help text to clearly differentiate server behavior: `Short: "Start agent runtime server with hot reload"`, `Long: "Watches .ias files for changes and automatically restarts the runtime. Includes built-in web UI."`

**Checkpoint**: `agentspec run` starts server, `agentspec dev` does one-shot. Old names emit deprecation warnings.

---

## Phase 3: User Story 1 — Complete CLI Documentation (Priority: P1) 🎯 MVP

**Goal**: All 20 CLI commands documented in README with descriptions and usage examples. CI validates completeness.

**Independent Test**: Compare list of commands in README against commands registered in `cmd/agentspec/main.go` — they must match exactly.

### Implementation for User Story 1

- [x] T007 [US1] Update CLI commands table in `README.md` to include all 20 commands with correct descriptions (reflecting the run↔dev rename). Add entries for: init, compile, publish, install, eval, status, logs, destroy, pkg (package). Fix existing `run` and `dev` descriptions to match new semantics
- [x] T008 [US1] Add usage examples in `README.md` for undocumented commands: `init`, `compile` (with `--target`), `publish`, `install`, `eval` (with `--live`), `status`, `logs`, `destroy`, `pkg`
- [x] T009 [US1] Create CI validation test in `integration_tests/cli_doc_test.go` that extracts all subcommand names from `newRootCmd()` and asserts each appears in `README.md`. Test should parse the README CLI table and compare against registered cobra commands
- [x] T010 [US1] Update docs site CLI pages in `docs/user-guide/cli/run.md` and `docs/user-guide/cli/dev.md` to reflect the swapped semantics

**Checkpoint**: All 20 commands documented. CI test passes. README matches actual CLI.

---

## Phase 4: User Story 2 — Functional Compiler Output (Priority: P1)

**Goal**: All 4 framework compiler targets (CrewAI, LangGraph, LlamaIndex, LlamaStack) generate functional tool implementations for HTTP, command, and inline tool types instead of "not implemented" stubs.

**Independent Test**: Compile a sample agent with HTTP, command, and inline tools to each target. Grep generated code for "not implemented" — should find none.

### Implementation for User Story 2

- [x] T011 [US2] Fix inline tool generation in `internal/compiler/targets/crewai.go` — add `case "inline"` to the `generateTools()` switch that generates Python code to execute inline scripts (for Python: direct code embed; for other languages: `subprocess.run([interpreter, "-c", code])`)
- [x] T012 [P] [US2] Fix tool generation in `internal/compiler/targets/langgraph.go` — read `generateTools()` method, verify HTTP and command cases work correctly, add `case "inline"` for inline tool execution, change default case from `return "not implemented"` to `raise NotImplementedError("Tool type not supported")`
- [x] T013 [P] [US2] Fix tool generation in `internal/compiler/targets/llamaindex.go` — read `generateTools()` method, verify HTTP and command cases work correctly, add `case "inline"` for inline tool execution, change default case from `return "not implemented"` to `raise NotImplementedError("Tool type not supported")`
- [x] T014 [US2] Fix tool generation in `internal/compiler/targets/llamastack.go` — read `generateAgent()` method, add HTTP tool generation (`urllib.request`), add inline tool generation, change default case from stub to `raise NotImplementedError`
- [x] T015 [US2] Update all 4 targets to mark remaining customization points with `# TODO(agentspec):` comments instead of returning "not implemented" strings — only for cases where tool config is genuinely missing (no URL, no binary), not for unsupported types
- [x] T016 [US2] Add compiler tool generation integration tests in `integration_tests/compile_test.go` — test each target × tool type combination: compile a sample agent with HTTP/command/inline tools, assert generated code does not contain `"not implemented"` string, assert generated code contains expected patterns (urllib, subprocess, etc.)

**Checkpoint**: All 4 targets generate functional tool code. No "not implemented" stubs remain for configured tools.

---

## Phase 5: User Story 3 — Polished Frontend Experience (Priority: P2)

**Goal**: Frontend displays loading indicator during agent fetch, error banner with retry on failure, and welcome message with instructions on empty session.

**Independent Test**: Open frontend in three states: loading (spinner visible), connection failure (error banner + retry button), empty session (welcome card).

### Implementation for User Story 3

- [x] T017 [US3] Add CSS for loading spinner, error banner, and welcome card in `internal/frontend/web/index.html` — add styles for `.loading-overlay` (centered spinner), `.error-banner` (top banner with red accent, retry button), `.welcome-card` (centered card with instructions)
- [x] T018 [US3] Add HTML markup for loading overlay, error banner, and welcome card in `internal/frontend/web/index.html` — add hidden DOM elements that will be shown/hidden by JS state management
- [x] T019 [US3] Implement loading state in `internal/frontend/web/app.js` — show loading overlay before `fetchAgents()` call, hide on success or failure. Add `showLoading()` and `hideLoading()` functions
- [x] T020 [US3] Implement error state with retry in `internal/frontend/web/app.js` — on `fetchAgents()` failure, show error banner with message and retry button instead of just `addSystemMessage()`. Retry button calls `fetchAgents()` again. Hide error banner on successful fetch
- [x] T021 [US3] Implement welcome/empty state in `internal/frontend/web/app.js` — after successful agent fetch with no messages, show welcome card with "Welcome to AgentSpec" heading and "Select an agent and type a message to get started" instructions. Hide welcome card when first message is sent
- [x] T022 [US3] Verify markdown rendering edge cases in `internal/frontend/web/app.js` — test `renderMarkdown()` with nested lists, code blocks inside tables, consecutive headings, empty code blocks, and long unbroken lines. Fix any rendering issues found (FR-008)

**Checkpoint**: Frontend handles loading, error, and empty states gracefully with visible UI indicators. Markdown rendering works for all common patterns.

---

## Phase 6: User Story 4 — Live Agent Evaluation (Priority: P2)

**Goal**: `eval --live` flag invokes agents with real LLM client and compares responses against expected patterns.

**Independent Test**: Define eval cases in an .ias file, run `agentspec eval --live`, verify agent is invoked and results are compared.

### Implementation for User Story 4

- [x] T023 [US4] Create `liveInvoker` struct in `cmd/agentspec/eval.go` that implements the Invoker interface — accepts IR document and runtime config, creates `llm.NewClientForModel()` per agent, uses `loop.ReActStrategy` to execute invocations, returns the response output
- [x] T024 [US4] Add `--live` flag to eval command in `cmd/agentspec/eval.go` — register bool flag, when set use `liveInvoker` instead of `stubInvoker`, pass the parsed IR document and runtime config to the live invoker
- [x] T025 [US4] Update eval report format in `internal/evaluation/format.go` (or the file containing `FormatReport`) to include expected vs actual comparison — ensure the table/json/markdown formatters show the expected pattern and actual output side by side for failed test cases

**Checkpoint**: `agentspec eval sample.ias --live` invokes agents with real LLM and produces pass/fail report with expected vs actual comparison.

---

## Phase 7: User Story 6 — Fast Dev Mode File Watching (Priority: P3)

**Goal**: File changes detected within 500ms using event-based watching (fsnotify), with polling fallback.

**Independent Test**: Start server (`agentspec run`), save an .ias file change, measure time until reload — must be under 500ms.

### Implementation for User Story 6

- [x] T026 [US6] Replace polling file watcher with fsnotify in `cmd/agentspec/run.go` (now the server command) — remove `time.NewTicker(2*time.Second)` + `filepath.Walk` loop, create `fsnotify.NewWatcher()`, add watch on `.ias` files in the watch directory, filter events to only `Write` and `Create` operations on `.ias` files
- [x] T027 [US6] Add debounce timer (100ms) to file watcher in `cmd/agentspec/run.go` — use a `time.Timer` that resets on each file event, only trigger reload when timer fires (prevents rapid-fire reloads on save)
- [x] T028 [US6] Add polling fallback in `cmd/agentspec/run.go` — if `fsnotify.NewWatcher()` returns an error, log a warning message "fsnotify unavailable, falling back to polling (2s interval)" and use the old `time.NewTicker` + `filepath.Walk` approach

**Checkpoint**: File changes detected in <500ms on Linux/macOS/Windows. Polling fallback works on unsupported platforms.

---

## Phase 8: User Story 7 — Honest Feature Flags (Priority: P3)

**Goal**: Zero CLI flags exist that accept input but produce no effect. All discovered stubs are fixed.

**Independent Test**: Audit all commands for flags with no-op handlers — none should exist.

### Implementation for User Story 7

- [x] T029 [US7] Fix `publish --sign` in `cmd/agentspec/publish.go` — change from printing warning and continuing to returning `fmt.Errorf("package signing is not yet available; publish without --sign or wait for a future release")`. The command must exit with non-zero status and NOT publish
- [x] T030 [US7] Audit all CLI flags across all 20 commands in `cmd/agentspec/*.go` — for each command, check every registered flag and verify its value is used in the RunE function. Document findings. Known stubs to check: `plan --env`, `apply --env`, `apply --plan-file`, `export --env` (all have `_ = varName` placeholders)
- [x] T031 [US7] Fix all discovered stub flags from audit (T030) — for each flag that accepts input but has no effect: either implement the feature, remove the flag, or return an error explaining the feature is not yet available. Update help text accordingly

**Checkpoint**: No CLI flags accept input without producing an effect. `publish --sign` errors instead of silently continuing.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and cleanup across all user stories

- [x] T032 [P] Update `docs/user-guide/cli/` pages (excluding run.md and dev.md which were updated in T010) for any other commands whose help text or behavior changed during this feature
- [x] T033 [P] Update `CLAUDE.md` with any new commands, flags, or behavioral changes introduced by this feature
- [x] T034 Update golden fixtures and integration test expectations affected by the run↔dev rename — check `integration_tests/` for any tests that reference `run` or `dev` command names or help text, update accordingly (constitution: breaking changes require updated fixtures)
- [x] T035 Verify `--help` output consistency across all 20 commands — each command's Short and Long descriptions must clearly describe its behavior and differentiate from similar commands (FR-013). Spot-check at least 5 commands beyond run/dev
- [x] T036 Run all quickstart.md scenarios end-to-end to validate the feature works as documented
- [x] T037 Run full test suite (`go test ./... -count=1`), linter (`golangci-lint run ./...`), and formatter (`gofmt -l .`) — fix any issues

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational/US5 (Phase 2)**: Depends on Setup — BLOCKS US1 (documentation)
- **US1 Documentation (Phase 3)**: Depends on Phase 2 (command rename must be done before documenting)
- **US2 Compiler (Phase 4)**: Depends on Phase 1 only — can run in parallel with Phase 2/3
- **US3 Frontend (Phase 5)**: Depends on Phase 1 only — can run in parallel with Phase 2/3/4
- **US4 Eval (Phase 6)**: Depends on Phase 1 only — can run in parallel with Phase 2/3/4/5
- **US6 File Watching (Phase 7)**: Depends on Phase 1 (fsnotify) and Phase 2 (command rename, since watcher is in server command)
- **US7 Flags (Phase 8)**: Depends on Phase 2 (command rename may affect flag audit)
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

```
Phase 1 (Setup)
    │
    ├──> Phase 2 (US5: Command Rename) ──> Phase 3 (US1: Docs) ──> Phase 9 (Polish)
    │         │
    │         ├──> Phase 7 (US6: File Watching)
    │         └──> Phase 8 (US7: Flags)
    │
    ├──> Phase 4 (US2: Compiler) ──────────────────────────────────> Phase 9
    ├──> Phase 5 (US3: Frontend) ──────────────────────────────────> Phase 9
    └──> Phase 6 (US4: Eval) ──────────────────────────────────────> Phase 9
```

### Parallel Opportunities

- **After Phase 1**: US2 (compiler), US3 (frontend), and US4 (eval) can all run in parallel since they touch different files
- **After Phase 2**: US1 (docs), US6 (file watching), and US7 (flags) can proceed
- **Within US2**: T012 and T013 (LangGraph + LlamaIndex) can run in parallel (different files)

---

## Parallel Example: User Story 2 (Compiler)

```bash
# These can run in parallel (different target files):
Task: "Fix tool generation in internal/compiler/targets/langgraph.go"
Task: "Fix tool generation in internal/compiler/targets/llamaindex.go"

# This must run after both complete (integration):
Task: "Add compiler tool generation integration tests in integration_tests/compile_test.go"
```

## Parallel Example: Cross-Story

```bash
# After Phase 1 (Setup), these stories can all start in parallel:
Story US2: "Fix compiler tool generation" (internal/compiler/targets/*.go)
Story US3: "Polish frontend experience" (internal/frontend/web/*)
Story US4: "Live agent evaluation" (cmd/agentspec/eval.go)
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup (add fsnotify)
2. Complete Phase 2: Command Rename (US5)
3. Complete Phase 3: CLI Documentation (US1) — all commands documented
4. Complete Phase 4: Compiler Output (US2) — functional tool generation
5. **STOP and VALIDATE**: README complete, compiler produces working code
6. Run tests and lint

### Incremental Delivery

1. Setup + Command Rename → Foundation ready
2. Add US1 (Docs) + US2 (Compiler) → P1 stories complete → Demo
3. Add US3 (Frontend) + US4 (Eval) → P2 stories complete → Demo
4. Add US6 (File Watching) + US7 (Flags) → P3 stories complete → Demo
5. Polish → Feature complete

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- US5 (Command Rename) is placed in Foundational phase despite P3 priority because it blocks US1 documentation
- The command rename is a breaking change — deprecation aliases are required per constitution versioning policy
- Total tasks: 37 (T001-T037)
- Commit after each phase or logical group
