# Feature Specification: CI Pipeline

**Feature Branch**: `002-ci-pipeline`
**Created**: 2026-02-22
**Status**: Draft
**Input**: User description: "Add CI for the repo and make it build the binaries, run basic examples to make sure everything works as expected"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Build and Test on Every Push (Priority: P1)

When a developer pushes code to the repository or opens a pull request, the CI pipeline automatically builds the project, runs all tests, and reports success or failure. This ensures that no broken code is merged into the main branch.

**Why this priority**: Without automated build and test verification, regressions can be introduced silently. This is the foundation of all CI — if code does not compile and tests do not pass, nothing else matters.

**Independent Test**: Push a commit to any branch. The CI pipeline triggers, builds the `agentz` binary, runs all tests, and reports pass/fail status on the commit or pull request.

**Acceptance Scenarios**:

1. **Given** a developer pushes a commit to any branch, **When** the CI pipeline runs, **Then** the Go project is compiled successfully and all tests pass, and the result is reported as a status check on the commit.
2. **Given** a developer opens a pull request against the main branch, **When** the CI pipeline runs, **Then** the build and test results are visible on the pull request before merging.
3. **Given** a commit introduces a compilation error, **When** the CI pipeline runs, **Then** the pipeline fails with a clear error message indicating the build failure.
4. **Given** a commit introduces a test failure, **When** the CI pipeline runs, **Then** the pipeline fails and reports which tests failed.

---

### User Story 2 - Validate Examples End-to-End (Priority: P2)

After the build succeeds, the CI pipeline runs the built binary against the example `.ias` files to verify that the toolchain works end-to-end: formatting, validation, planning, and applying all succeed on real definitions.

**Why this priority**: Unit and integration tests verify internal correctness, but running the actual CLI against real example files catches issues that tests might miss — broken CLI wiring, missing subcommands, incorrect output formatting, or regressions in the full pipeline.

**Independent Test**: The CI pipeline builds the binary, then runs `agentz validate`, `agentz fmt --check`, `agentz plan`, and `agentz apply --auto-approve` against example `.ias` files. All commands exit successfully.

**Acceptance Scenarios**:

1. **Given** the binary is built successfully, **When** the CI pipeline runs `agentz validate` against all example `.ias` files, **Then** every example validates without errors.
2. **Given** the binary is built successfully, **When** the CI pipeline runs `agentz fmt --check` against all example `.ias` files, **Then** every example is already in canonical format (no formatting changes needed).
3. **Given** the binary is built successfully, **When** the CI pipeline runs `agentz plan` and `agentz apply --auto-approve` against a representative example, **Then** the plan shows expected resources and the apply completes without errors.
4. **Given** the apply completes for an example, **When** the CI pipeline runs `agentz apply --auto-approve` again on the same example, **Then** the output indicates no changes (idempotency verification).

---

### User Story 3 - Lint Code Quality (Priority: P3)

The CI pipeline runs a linter on every push to enforce consistent code quality and catch common issues before review.

**Why this priority**: Linting catches style violations, potential bugs, and code smells early. It is lower priority than build and test but still valuable for maintaining code quality over time.

**Independent Test**: The CI pipeline runs the configured linter against the codebase and reports any violations.

**Acceptance Scenarios**:

1. **Given** a developer pushes code that passes all lint rules, **When** the CI pipeline runs the linter, **Then** the lint step passes.
2. **Given** a developer pushes code with lint violations, **When** the CI pipeline runs the linter, **Then** the lint step fails and reports the specific violations.

---

### Edge Cases

- What happens when the CI pipeline runs on a branch with no `.ias` example files? The example validation step should skip gracefully or report that no examples were found.
- What happens when a new Go dependency is added but `go.sum` is not updated? The build step should fail with a clear message about missing module checksums.
- What happens when the CI environment does not have the required Go version? The pipeline should fail early with a clear message about the version requirement.
- What happens when tests are flaky (pass sometimes, fail sometimes)? The pipeline should run tests with `-count=1` to disable test caching and surface genuine failures.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CI pipeline MUST trigger on every push to any branch.
- **FR-002**: The CI pipeline MUST trigger on every pull request targeting the main branch.
- **FR-003**: The CI pipeline MUST compile the Go project and produce the `agentz` binary.
- **FR-004**: The CI pipeline MUST run all Go tests with caching disabled (`-count=1`).
- **FR-005**: The CI pipeline MUST report build and test results as commit status checks visible on pull requests.
- **FR-006**: The CI pipeline MUST run `agentz validate` against all example `.ias` files and fail if any example has validation errors.
- **FR-007**: The CI pipeline MUST run `agentz fmt --check` against all example `.ias` files and fail if any example is not in canonical format.
- **FR-008**: The CI pipeline MUST run `agentz plan` and `agentz apply --auto-approve` against at least one representative example to verify the full lifecycle works.
- **FR-009**: The CI pipeline MUST verify idempotency by running `agentz apply --auto-approve` a second time and confirming no changes are reported.
- **FR-010**: The CI pipeline MUST run the configured Go linter and fail on lint violations.
- **FR-011**: The CI pipeline MUST use a Go version compatible with the project's `go.mod` requirement (Go 1.25+).
- **FR-012**: The CI pipeline MUST complete within 10 minutes to avoid blocking developer workflows.
- **FR-013**: The CI pipeline MUST produce clear, actionable output when any step fails, including the specific error and which step failed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every push and pull request receives automated build, test, and lint feedback without manual intervention.
- **SC-002**: All example `.ias` files validate and format-check successfully in CI on every run.
- **SC-003**: The full plan-apply-idempotency lifecycle passes in CI on every run for at least one example.
- **SC-004**: Developers can see CI results directly on their pull requests before merging.
- **SC-005**: A commit that breaks compilation, tests, linting, or example validation is caught by CI and prevented from being merged (when branch protection is enabled).

## Clarifications

### Session 2026-02-22

- Q: What is the maximum acceptable CI pipeline duration? → A: 10 minutes.
- Q: Should file extension references use `.ias` (per amended parent spec) or `.az` (current codebase)? → A: Use `.ias` to align with the amended parent spec.

## Assumptions

- The repository is hosted on GitHub, so the CI pipeline uses GitHub Actions.
- The Go version required (1.25+) is available as a GitHub Actions setup-go version.
- The `golangci-lint` tool (already configured in `.golangci.yml`) is used for linting.
- Example `.ias` files are located in `examples/*/` subdirectories.
- No external services or credentials are required to build, test, or run examples — everything runs locally with the `local-mcp` adapter.
- Branch protection rules (requiring CI to pass before merge) are configured separately by the repository owner and are outside the scope of this specification.
