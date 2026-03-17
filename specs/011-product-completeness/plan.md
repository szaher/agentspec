# Implementation Plan: Product Completeness & UX

**Branch**: `011-product-completeness` | **Date**: 2026-03-17 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/011-product-completeness/spec.md`

## Summary

Address product/UX gaps: fix compiler tool generation stubs across 4 targets, add live eval command, swap run/dev command semantics with deprecation aliases, add fsnotify-based file watching, improve frontend states (loading/error/empty), complete README documentation with CI validation, and audit all CLI flags for stubs.

## Technical Context

**Language/Version**: Go 1.25+ (backend), Vanilla JS (frontend)
**Primary Dependencies**: cobra v1.10.2 (CLI), fsnotify (new — file watching), existing llm/loop/runtime packages
**Storage**: N/A (no schema changes)
**Testing**: `go test ./...`, integration tests in `integration_tests/`
**Target Platform**: Linux, macOS, Windows (CLI + embedded web UI)
**Project Type**: CLI tool with embedded HTTP server and web frontend
**Performance Goals**: File change detection <500ms, frontend state transitions <200ms
**Constraints**: No new external service dependencies; frontend must remain vanilla JS (no framework)
**Scale/Scope**: 20 CLI commands, 4 compiler targets, 1 frontend app

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Determinism | PASS | Compiler output changes are deterministic (same IR → same code) |
| II. Idempotency | PASS | No changes to plan/apply idempotency |
| III. Portability | PASS | fsnotify supports Linux/macOS/Windows; polling fallback for others |
| VI. Safe Defaults | PASS | No new secret handling |
| VII. Minimal Surface Area | PASS | `--live` flag justified by eval use case; fsnotify replaces polling (simpler) |
| IX. Canonical Formatting | N/A | No DSL syntax changes |
| X. Strict Validation | PASS | Stub flags now error instead of silently continuing |
| XII. No Hidden Behavior | PASS | Deprecation warnings are explicit; stub flags fail loudly |
| Breaking Changes | JUSTIFIED | run↔dev rename has deprecation aliases for one release cycle per constitution versioning policy |

**Post-Phase 1 re-check**: All gates still pass. No new entities or abstractions introduced.

## Project Structure

### Documentation (this feature)

```text
specs/011-product-completeness/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model
├── quickstart.md        # Phase 1 integration scenarios
├── contracts/
│   ├── cli-commands.md  # CLI command registry contract
│   ├── compiler-tools.md # Compiler tool generation contract
│   └── frontend-states.md # Frontend state machine contract
└── tasks.md             # Phase 2 task breakdown (via /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/agentspec/
├── main.go              # MODIFY: swap run/dev registration, add deprecation aliases
├── run.go               # MODIFY: swap to server behavior (current dev.go logic)
├── dev.go               # MODIFY: swap to one-shot behavior (current run.go logic)
├── eval.go              # MODIFY: add --live flag + liveInvoker
└── publish.go           # MODIFY: --sign returns error

internal/compiler/targets/
├── crewai.go            # MODIFY: add inline tool support, improve stubs
├── langgraph.go         # MODIFY: add HTTP/command/inline tool generation
├── llamaindex.go        # MODIFY: add HTTP/command/inline tool generation
└── llamastack.go        # MODIFY: add HTTP/command/inline tool generation

internal/frontend/web/
├── index.html           # MODIFY: add loading/error/empty state markup + CSS
└── app.js               # MODIFY: add loading/error/empty state logic

README.md                # MODIFY: document all 20 commands, fix run/dev descriptions

integration_tests/
├── compile_test.go      # MODIFY: add tool generation tests
└── cli_doc_test.go      # NEW: CI validation that all commands are documented
```

**Structure Decision**: Modifications to existing files only, except for one new test file for CI documentation validation. No new packages or directories needed.

## Implementation Phases

### Phase A: CLI Command Rename (FR-014, FR-015)

**Priority**: P3 but foundational — must be done first since it affects documentation and help text.

1. **Swap run.go ↔ dev.go logic**:
   - `run.go`: Move current `dev.go` server logic here. Update `Use`, `Short`, `Long` descriptions.
   - `dev.go`: Move current `run.go` one-shot logic here. Update `Use`, `Short`, `Long` descriptions.
   - Alternative: keep filenames, just swap the function bodies and command metadata.

2. **Add deprecation aliases in main.go**:
   ```go
   // Deprecation alias: old 'run' behavior (one-shot) is now 'dev'
   root.AddCommand(newDeprecatedAlias("run-oneshot", "run", "dev", newDevCmd))
   ```
   Implementation: Create helper that wraps a command with a deprecation warning on stderr.

3. **Update file watching** (part of server command, now `run`):
   - Add `fsnotify/fsnotify` dependency
   - Replace polling loop in server command with fsnotify watcher
   - Filter to `.ias` file events only
   - Add 100ms debounce timer
   - Add polling fallback with log message if fsnotify init fails

### Phase B: Compiler Tool Generation (FR-003, FR-004)

**Priority**: P1

1. **Audit all 4 targets** for tool generation gaps:
   - Read `generateTools()` in each target file
   - Identify which tool types produce "not implemented" stubs

2. **Fix each target**:
   - Add/fix `case "http"` with URL + method
   - Add/fix `case "command"` with binary + args
   - Add `case "inline"` with script execution
   - Change default case from silent stub to TODO + error

3. **Add integration tests** for each target × tool type combination.

### Phase C: Eval --live (FR-009, FR-010)

**Priority**: P2

1. **Create `liveInvoker`** in `cmd/agentspec/eval.go`:
   - Accept IR document and runtime config
   - For each agent, create `llm.NewClientForModel()` + `loop.ReActStrategy`
   - Implement `Invoke()` that calls `strategy.Execute()`

2. **Add `--live` flag** to eval command
3. **Update eval runner** to use liveInvoker when `--live` is set
4. **Ensure report format** includes expected vs actual comparison

### Phase D: Frontend Polish (FR-005, FR-006, FR-007, FR-008)

**Priority**: P2

1. **Add CSS** for loading spinner, error banner, welcome card to `index.html`
2. **Modify `fetchAgents()`** in `app.js`:
   - Show loading state before fetch
   - Show error banner with retry button on failure
   - Show welcome card on success with no messages
3. **Markdown rendering**: Already implemented — verify edge cases

### Phase E: Documentation & Flags (FR-001, FR-002, FR-012, FR-013)

**Priority**: P1 (docs), P3 (flags)

1. **Update README.md**:
   - Add all 20 commands to CLI table
   - Fix run/dev descriptions to match new semantics
   - Add usage examples for new commands

2. **Create CI validation test** (`integration_tests/cli_doc_test.go`):
   - Extract command names from `newRootCmd()` subcommands
   - Parse README.md for documented commands
   - Assert all registered commands are documented

3. **Fix publish --sign**:
   - Return `fmt.Errorf("package signing is not yet available")` instead of warning
   - Do not proceed with publish

4. **Audit remaining flags**:
   - Check each of the 20 commands for flags that accept input but have no effect
   - Fix each discovered stub

## Complexity Tracking

No constitution violations requiring justification.

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/fsnotify/fsnotify` | latest stable | Event-based file watching |

All other dependencies are already in `go.mod`.
