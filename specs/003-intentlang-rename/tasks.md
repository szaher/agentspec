# Tasks: IntentLang Rename

**Input**: Design documents from `/specs/003-intentlang-rename/`
**Prerequisites**: plan.md (required), spec.md (required for user stories)
**Dependencies**: Features 001-agent-packaging-dsl and 002-ci-pipeline MUST be merged to main before implementation

**Tests**: No new test files — existing integration tests are updated in-place with new extensions and paths.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Preparation)

**Purpose**: Merge dependencies and verify baseline before any rename work begins

- [x] T001 Merge 001-agent-packaging-dsl and 002-ci-pipeline to main, then rebase 003-intentlang-rename onto main
- [x] T002 Verify all existing tests pass and CI is green before starting rename work (`go test ./... -count=1`)

---

## Phase 2: Foundational (Deprecation Warning Infrastructure)

**Purpose**: Add the deprecation warning helper and `.az`/`.ias` conflict detection that all CLI commands will use. MUST complete before user story work begins.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T003 Add deprecation warning helper function in internal/cli/deprecation.go — accepts a file path, checks extension, emits stderr warning if `.az`, returns error if both `.az` and `.ias` versions exist in the same directory
- [x] T004 Update `.az` references in parser/AST comments, error messages, and file-type descriptions to include `.ias` in internal/parser/lexer.go, internal/parser/parser.go, internal/parser/token.go, and internal/ast/ast.go — the parser is extension-agnostic (accepts any content regardless of extension), so no parsing logic changes are needed; only update human-readable strings

**Checkpoint**: Deprecation infrastructure ready — user story implementation can now begin

---

## Phase 3: User Story 1 — Rename File Extension from .az to .ias (Priority: P1) 🎯 MVP

**Goal**: CLI accepts `.ias` files, emits deprecation warnings for `.az`, and `migrate` command renames `.az` → `.ias`

**Independent Test**: Rename an existing `.az` file to `.ias`. Run `agentz validate`, `agentz fmt`, `agentz plan`, and `agentz apply` against it — all succeed. Run against a `.az` file — deprecation warning appears on stderr.

### Implementation for User Story 1

- [x] T005 [P] [US1] Wire deprecation warning into `validate` command in cmd/agentz/validate.go — call deprecation helper before processing, emit warning to stderr for `.az` files
- [x] T006 [P] [US1] Wire deprecation warning into `fmt` command in cmd/agentz/fmt.go
- [x] T007 [P] [US1] Wire deprecation warning into `plan` command in cmd/agentz/plan.go
- [x] T008 [P] [US1] Wire deprecation warning into `apply` command in cmd/agentz/apply.go
- [x] T009 [P] [US1] Wire deprecation warning into `diff` command in cmd/agentz/diff.go
- [x] T010 [P] [US1] Wire deprecation warning into `export` command in cmd/agentz/export.go
- [x] T011 [US1] Update `migrate` command in cmd/agentz/migrate.go — add `.az` → `.ias` file rename logic: walk current directory and subdirectories, rename all `.az` files to `.ias`, update internal references if `.az` appears in file content, print summary of renamed files
- [x] T012 [US1] Add conflict detection in cmd/agentz/migrate.go — if both `foo.az` and `foo.ias` exist, report error and skip that file
- [x] T013 [US1] Update integration tests to verify `.ias` files are accepted in integration_tests/validate_test.go — add test case that uses `.ias` extension
- [x] T014 [P] [US1] Add integration test for `.az` deprecation warning in integration_tests/validate_test.go — verify stderr contains deprecation message when processing `.az` file
- [x] T015 [P] [US1] Rename integration_tests/testdata/valid.az → integration_tests/testdata/valid.ias
- [x] T016 [P] [US1] Rename integration_tests/testdata/multi_env.az → integration_tests/testdata/multi_env.ias
- [x] T017 [P] [US1] Rename integration_tests/testdata/invalid_ref.az → integration_tests/testdata/invalid_ref.ias
- [x] T018 [US1] Update all integration test files that reference `.az` testdata paths to use `.ias` — files: integration_tests/apply_test.go, integration_tests/determinism_test.go, integration_tests/environment_test.go, integration_tests/export_test.go, integration_tests/golden_test.go, integration_tests/plugin_test.go, integration_tests/sdk_test.go, integration_tests/validate_test.go
- [x] T019 [US1] Verify: run `go test ./... -count=1` — all tests pass with new `.ias` extension and deprecation warnings

**Checkpoint**: `.ias` files are fully supported, `.az` files emit deprecation warnings, migrate command works

---

## Phase 4: User Story 2 — Rename All Example Files and Documentation (Priority: P2)

**Goal**: All example files use `.ias` extension, all documentation uses IntentLang/AgentSpec/AgentPack naming

**Independent Test**: All files in `examples/*/` use `.ias` extension. All READMEs reference IntentLang. Run `agentz validate` and `agentz fmt --check` against all examples — all pass.

### Implementation for User Story 2

#### Example File Renames (all parallel — different files)

- [x] T020 [P] [US2] Rename examples/basic-agent/basic-agent.az → examples/basic-agent/basic-agent.ias
- [x] T021 [P] [US2] Rename examples/code-review-pipeline/code-review-pipeline.az → examples/code-review-pipeline/code-review-pipeline.ias
- [x] T022 [P] [US2] Rename examples/customer-support/customer-support.az → examples/customer-support/customer-support.ias
- [x] T023 [P] [US2] Rename examples/data-pipeline/data-pipeline.az → examples/data-pipeline/data-pipeline.ias
- [x] T024 [P] [US2] Rename examples/mcp-server-client/mcp-server-client.az → examples/mcp-server-client/mcp-server-client.ias
- [x] T025 [P] [US2] Rename examples/multi-binding/multi-binding.az → examples/multi-binding/multi-binding.ias
- [x] T026 [P] [US2] Rename examples/multi-environment/multi-environment.az → examples/multi-environment/multi-environment.ias
- [x] T027 [P] [US2] Rename examples/multi-skill-agent/multi-skill-agent.az → examples/multi-skill-agent/multi-skill-agent.ias
- [x] T028 [P] [US2] Rename examples/plugin-usage/plugin-usage.az → examples/plugin-usage/plugin-usage.ias
- [x] T029 [P] [US2] Rename examples/rag-chatbot/rag-chatbot.az → examples/rag-chatbot/rag-chatbot.ias

#### README Updates (all parallel — different files)

- [x] T030 [P] [US2] Update examples/README.md — replace all `.az` references with `.ias`, replace "agentz" CLI references with "agentspec" where appropriate, add IntentLang/AgentSpec/AgentPack naming
- [x] T031 [P] [US2] Update examples/basic-agent/README.md — replace `.az` → `.ias`, update CLI command examples, use IntentLang naming
- [x] T032 [P] [US2] Update examples/code-review-pipeline/README.md — same as T031
- [x] T033 [P] [US2] Update examples/customer-support/README.md — same as T031
- [x] T034 [P] [US2] Update examples/data-pipeline/README.md — same as T031
- [x] T035 [P] [US2] Update examples/mcp-server-client/README.md — same as T031
- [x] T036 [P] [US2] Update examples/multi-binding/README.md — same as T031
- [x] T037 [P] [US2] Update examples/multi-environment/README.md — same as T031
- [x] T038 [P] [US2] Update examples/multi-skill-agent/README.md — same as T031
- [x] T039 [P] [US2] Update examples/plugin-usage/README.md — same as T031
- [x] T040 [P] [US2] Update examples/rag-chatbot/README.md — same as T031

#### Top-Level Documentation Updates (all parallel — different files)

- [x] T041 [P] [US2] Update ARCHITECTURE.md — replace all `.az` references with `.ias`, use IntentLang/AgentSpec/AgentPack naming throughout
- [x] T042 [P] [US2] Update CHANGELOG.md — add entry for IntentLang rename, update any existing `.az` references to `.ias`
- [x] T043 [P] [US2] Update spec/spec.md — replace `.az` references with `.ias`, update language naming to IntentLang
- [x] T044 [P] [US2] Update init-spec.md — replace `.az` references with `.ias`, use IntentLang naming

#### Spec Contract Updates (all parallel — different files)

- [x] T045 [P] [US2] Update specs/001-agent-packaging-dsl/contracts/cli.md — replace `.az` → `.ias`, update CLI binary references from `agentz` to `agentspec`
- [x] T046 [P] [US2] Update specs/001-agent-packaging-dsl/contracts/plugin-manifest.md — update any `.az` or `agentz` references
- [x] T047 [P] [US2] Update specs/001-agent-packaging-dsl/contracts/sdk-api.md — update any `.az` or `agentz` references

- [x] T048 [US2] Verify: run `agentz validate` and `agentz fmt --check` against all renamed `.ias` example files — all pass

**Checkpoint**: All examples use `.ias`, all documentation uses IntentLang/AgentSpec/AgentPack naming

---

## Phase 5: User Story 3 — Update CLI Binary Name and Internal References (Priority: P3)

**Goal**: CLI binary renamed to `agentspec`, state file uses `.agentspec.state.json`, plugin directory uses `~/.agentspec/plugins/`, all help text updated

**Independent Test**: Build the CLI — binary is `agentspec`. Run `agentspec version` — shows new name. State file is `.agentspec.state.json`. Old state files auto-migrate.

### Implementation for User Story 3

#### State File Migration

- [x] T049 [US3] Update internal/state/local.go — change state file name constant from `.agentz.state.json` to `.agentspec.state.json`, add auto-migration logic: on load, if `.agentspec.state.json` not found but `.agentz.state.json` exists, rename it and print migration notice to stderr

#### Plugin Directory Migration

- [x] T050 [US3] Update internal/plugins/loader.go — change primary plugin directory from `~/.agentz/plugins/` to `~/.agentspec/plugins/`, add fallback: if primary dir not found, check `~/.agentz/plugins/` and emit deprecation warning to stderr if plugins found there

#### CLI Help Text and Binary Rename

- [x] T051 [US3] Update cmd/agentz/main.go — change binary name references in help text from "agentz" to "agentspec", update root command Use/Short/Long descriptions to reference IntentLang and AgentSpec
- [x] T052 [P] [US3] Update cmd/agentz/version.go — change version output to show "agentspec" as the program name
- [x] T053 [P] [US3] Update cmd/agentz/sdk.go — update help text to reference IntentLang/AgentSpec naming
- [x] T054 [US3] Rename directory cmd/agentz/ → cmd/agentspec/ using `git mv cmd/agentz cmd/agentspec`

#### Internal Source Reference Updates (all parallel — different files)

- [x] T055 [P] [US3] Update internal/plugins/host.go — update any "agentz" comment references
- [x] T056 [P] [US3] Update internal/adapters/compose/compose.go — update generated file comments from "agentz" to "agentspec"
- [x] T057 [P] [US3] Update internal/sdk/generator/generator.go — update generated SDK naming and references from "agentz" to "agentspec"
- [x] T058 [P] [US3] Update internal/plugins/validate.go — update any "agentz" string references
- [x] T059 [P] [US3] Update internal/plugins/transform.go — update any "agentz" string references
- [x] T060 [P] [US3] Update internal/plan/format.go — update any "agentz" string references in plan output
- [x] T061 [P] [US3] Update internal/plan/plan.go — update any "agentz" string references
- [x] T062 [P] [US3] Update internal/plan/drift.go — update any "agentz" string references
- [x] T063 [P] [US3] Update internal/apply/apply.go — update any "agentz" string references
- [x] T064 [P] [US3] Update internal/policy/enforce.go — update any "agentz" string references
- [x] T065 [P] [US3] Update internal/policy/policy.go — update any "agentz" string references
- [x] T066 [P] [US3] Update internal/validate/environment.go — update any "agentz" string references
- [x] T067 [P] [US3] Update internal/validate/semantic.go — update any "agentz" string references
- [x] T068 [P] [US3] Update internal/validate/structural.go — update any "agentz" string references
- [x] T069 [P] [US3] Update internal/ir/lower.go — update any "agentz" string references
- [x] T070 [P] [US3] Update internal/adapters/adapter.go — update any "agentz" string references
- [x] T071 [P] [US3] Update internal/adapters/local/local.go — update any "agentz" string references

#### CI and Config Updates

- [x] T072 [US3] Update .github/workflows/ci.yml — change glob patterns from `examples/*/*.az` to `examples/*/*.ias`, change binary build target from `./cmd/agentz` to `./cmd/agentspec`, change binary name from `agentz` to `agentspec` in all step commands
- [x] T073 [P] [US3] Update .gitignore — replace `/agentz` binary pattern with `/agentspec`
- [x] T074 [P] [US3] Update .golangci.yml — no structural changes expected, but verify no hardcoded `.az` or `agentz` references
- [x] T075 [P] [US3] Update plugins/monitor/manifest.json — update any "agentz" references to "agentspec"

#### Integration Test Updates for US3

- [x] T076 [US3] Update integration test files that reference `.agentz.state.json` to use `.agentspec.state.json` — check all files in integration_tests/
- [x] T077 [US3] Update integration test files that reference `~/.agentz/plugins/` to use `~/.agentspec/plugins/` — check integration_tests/plugin_test.go
- [x] T078 [US3] Verify: run `go build -o agentspec ./cmd/agentspec` — binary builds successfully
- [x] T079 [US3] Verify: run `go test ./... -count=1` — all tests pass with renamed binary, state file, and plugin paths

**Checkpoint**: Binary is `agentspec`, state files auto-migrate, plugin paths use new directory with fallback

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, CLAUDE.md update, and overall consistency checks

- [x] T080 [P] Update CLAUDE.md — update binary name references from `agentz` to `agentspec`, add note about IntentLang rename
- [x] T081 [P] Update DECISIONS/ documents if any reference `.az` or `agentz` CLI — DECISIONS/001-parser-choice.md, DECISIONS/002-plugin-sandbox.md, DECISIONS/003-state-backend.md
- [x] T082 Run full test suite: `go test ./... -count=1 -v` — verify zero failures
- [x] T083 Run format check: `agentspec fmt --check` against all `.ias` examples — verify all pass
- [x] T084 Run validation: `agentspec validate` against all `.ias` examples — verify all pass
- [x] T085 Verify no remaining `.az` references in Go source (excluding test files that explicitly test backward compatibility): `grep -r '\.az' --include='*.go' internal/ cmd/` — only deprecation-related references should remain
- [x] T086 Verify no remaining bare `agentz` references in Go source (excluding Go module path): `grep -r 'agentz' --include='*.go' internal/ cmd/` — only import paths should contain `agentz`
- [x] T087 Run quickstart.md validation scenarios (once quickstart.md is written)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — merge and verify baseline
- **Foundational (Phase 2)**: Depends on Phase 1 — creates deprecation infrastructure
- **User Story 1 (Phase 3)**: Depends on Phase 2 — uses deprecation helpers
- **User Story 2 (Phase 4)**: Depends on Phase 3 — examples must work with `.ias` before renaming
- **User Story 3 (Phase 5)**: Depends on Phase 4 — binary rename is last to avoid breaking CI during transition
- **Polish (Phase 6)**: Depends on all user stories complete

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Foundational (Phase 2) — adds `.ias` support and deprecation warnings
- **User Story 2 (P2)**: Depends on US1 — examples are renamed after `.ias` is supported
- **User Story 3 (P3)**: Depends on US2 — binary rename is last, most disruptive change

**Note**: Unlike many features, these user stories are **sequential** by design — each builds on the previous rename step to avoid breaking the build mid-rename.

### Within Each User Story

- CLI command updates can run in parallel (different files)
- File renames within examples can run in parallel
- README updates can run in parallel
- Internal source updates can run in parallel
- Verification tasks must run after all changes in their phase

### Parallel Opportunities

#### Phase 3 (US1): CLI deprecation wiring
```bash
# These 5 commands can run in parallel (different files):
T006: fmt.go deprecation warning
T007: plan.go deprecation warning
T008: apply.go deprecation warning
T009: diff.go deprecation warning
T010: export.go deprecation warning
```

#### Phase 4 (US2): File renames and README updates
```bash
# All 10 example renames can run in parallel:
T020-T029: Rename examples/*/*.az → *.ias

# All 11 README updates can run in parallel:
T030-T040: Update examples/*/README.md

# All doc updates can run in parallel:
T041-T047: Update ARCHITECTURE.md, CHANGELOG.md, spec/spec.md, contracts/
```

#### Phase 5 (US3): Internal source updates
```bash
# All 17 internal/ updates can run in parallel:
T055-T071: Update internal/**/*.go references

# Config updates can run in parallel:
T073-T075: .gitignore, .golangci.yml, manifest.json
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (merge dependencies)
2. Complete Phase 2: Foundational (deprecation helpers)
3. Complete Phase 3: User Story 1 (`.ias` support + deprecation)
4. **STOP and VALIDATE**: `.ias` files work, `.az` files emit warnings
5. Binary still named `agentz` — existing workflows unbroken

### Incremental Delivery

1. Setup + Foundational → Infrastructure ready
2. Add User Story 1 → `.ias` accepted, `.az` deprecated → Test independently
3. Add User Story 2 → All examples and docs updated → Test independently
4. Add User Story 3 → Binary renamed, state/plugins migrated → Test independently
5. Polish → Final consistency verification

### Sequential Requirement

Unlike typical features, this rename MUST be done sequentially (US1 → US2 → US3) to avoid breaking the build:
- US1 adds `.ias` support while keeping `agentz` binary → safe
- US2 renames examples (now that `.ias` is supported) → safe
- US3 renames binary and CI (now that examples use `.ias`) → safe

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- User stories are **sequential** for this rename (unlike typical independent stories)
- Go module path `github.com/szaher/designs/agentz` remains unchanged — only user-facing names change
- Internal `"agentz"` in import paths is expected and correct — do NOT rename Go import paths
- Deprecation warnings go to stderr, not stdout
- State file migration is silent rename + stderr notice (not a copy)
- Plugin directory fallback is read-only (check old path, don't move files)
