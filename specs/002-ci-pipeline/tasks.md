# Tasks: CI Pipeline

**Input**: Design documents from `/specs/002-ci-pipeline/`
**Prerequisites**: plan.md (required), spec.md (required for user stories)

**Tests**: No test tasks included — the CI pipeline itself is the test infrastructure. Verification is done by pushing a commit and observing the pipeline run.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: Create the GitHub Actions workflow directory structure

- [x] T001 Create .github/workflows/ directory structure at repository root

---

## Phase 2: User Story 1 — Build and Test on Every Push (Priority: P1) 🎯 MVP

**Goal**: CI pipeline triggers on push/PR, builds the Go binary, runs all tests, and reports results as commit status checks

**Independent Test**: Push a commit to any branch. The CI run should build the binary and run `go test ./... -count=1 -v`. The result appears as a status check on the commit.

### Implementation for User Story 1

- [x] T002 [US1] Create GitHub Actions workflow file with trigger configuration (push to all branches, PR to main), Go 1.25+ setup via actions/setup-go@v5, Go module caching, dependency download (`go mod download`), binary build (`go build -o agentz ./cmd/agentz`), and test execution (`go test ./... -count=1 -v`) in .github/workflows/ci.yml
- [x] T003 [US1] Set job timeout to 10 minutes (`timeout-minutes: 10`) and configure job to run on `ubuntu-latest` in .github/workflows/ci.yml

**Checkpoint**: At this point, pushing a commit should trigger CI that builds and tests the project

---

## Phase 3: User Story 2 — Validate Examples End-to-End (Priority: P2)

**Goal**: CI validates all example files (fmt --check, validate) and runs the full plan-apply-idempotency smoke test on a representative example

**Independent Test**: After US1 steps pass, the pipeline runs `agentz validate` and `agentz fmt --check` on all examples, then runs `agentz plan`, `agentz apply --auto-approve`, and a second `agentz apply --auto-approve` (expecting no changes) on the basic-agent example.

### Implementation for User Story 2

- [x] T004 [US2] Add step to validate all example files by running `./agentz validate` against each file matching `examples/*/*.az` in .github/workflows/ci.yml. If no matching files are found, the step should exit with a warning rather than failing silently on an empty glob
- [x] T005 [US2] Add step to format-check all example files by running `./agentz fmt --check` against each file matching `examples/*/*.az` in .github/workflows/ci.yml. If no matching files are found, the step should exit with a warning rather than failing silently on an empty glob
- [x] T006 [US2] Add smoke test steps: run `./agentz plan examples/basic-agent/basic-agent.az`, then `./agentz apply examples/basic-agent/basic-agent.az --auto-approve`, then run apply again and verify output contains "No changes" (idempotency check) in .github/workflows/ci.yml

**Checkpoint**: At this point, the CI pipeline validates all examples and verifies the full lifecycle

---

## Phase 4: User Story 3 — Lint Code Quality (Priority: P3)

**Goal**: CI runs golangci-lint on every push using the existing `.golangci.yml` configuration

**Independent Test**: Push code with a lint violation (e.g., unused variable). The lint step should fail and report the violation.

### Implementation for User Story 3

- [x] T007 [US3] Add golangci-lint step using golangci/golangci-lint-action@v6 with the existing .golangci.yml configuration in .github/workflows/ci.yml

**Checkpoint**: At this point, the full CI pipeline is complete — build, test, examples, and lint

---

## Phase 5: Polish & Validation

**Purpose**: Verify the complete pipeline works end-to-end

- [x] T008 Run quickstart.md validation — push a commit to the branch and verify all CI steps pass in the GitHub Actions UI

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **User Story 1 (Phase 2)**: Depends on Setup — creates the workflow file with build/test steps
- **User Story 2 (Phase 3)**: Depends on US1 — adds example validation steps to the existing workflow file
- **User Story 3 (Phase 4)**: Depends on US1 — adds lint step to the existing workflow file
- **Polish (Phase 5)**: Depends on all user stories being complete

### Important Note

All user story tasks modify the **same file** (`.github/workflows/ci.yml`), so they MUST be executed sequentially. There are no parallel opportunities within this feature — each task appends steps to the workflow file.

### Execution Order

```
T001 (setup) → T002-T003 (build/test) → T004-T006 (examples) → T007 (lint) → T008 (validate)
```

US2 (T004-T006) and US3 (T007) both depend on US1 but are independent of each other — they could theoretically be reordered. However, since they all modify the same file, sequential execution is required.

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001)
2. Complete Phase 2: User Story 1 (T002-T003)
3. **STOP and VALIDATE**: Push to branch, verify CI triggers and passes
4. Proceed to US2 and US3

### Full Delivery

1. T001: Create directory → T002-T003: Build and test workflow
2. T004-T006: Add example validation and smoke test
3. T007: Add linting
4. T008: Push and verify everything passes

---

## Notes

- All tasks modify `.github/workflows/ci.yml` — no parallelization possible
- The workflow uses pinned action versions: actions/checkout@v4, actions/setup-go@v5, golangci/golangci-lint-action@v6
- Example files currently use `.az` extension (codebase reality); the spec references `.ias` (forward-looking). Tasks use `.az` to match what exists in the repository today
- The smoke test uses `examples/basic-agent/basic-agent.az` as the representative example
- Total: 8 tasks across 5 phases
