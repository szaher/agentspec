# Implementation Plan: CI Pipeline

**Branch**: `002-ci-pipeline` | **Date**: 2026-02-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-ci-pipeline/spec.md`

## Summary

Add a GitHub Actions CI pipeline that builds the Go project, runs all tests, validates example `.az` files end-to-end using the built binary, and lints the codebase. The pipeline triggers on every push and pull request, completing within 10 minutes.

## Technical Context

**Language/Version**: YAML (GitHub Actions workflow syntax)
**Primary Dependencies**: GitHub Actions, actions/checkout, actions/setup-go, golangci/golangci-lint-action
**Storage**: N/A
**Testing**: Go test (`go test ./... -count=1`), CLI smoke tests (built binary against examples)
**Target Platform**: GitHub Actions runners (ubuntu-latest)
**Project Type**: CI/CD configuration
**Performance Goals**: Pipeline completes within 10 minutes
**Constraints**: Go 1.25+ required (must be available in setup-go action), no external services or credentials needed
**Scale/Scope**: Single workflow file, ~100 lines of YAML

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Determinism | PASS | CI runs tests that verify determinism (golden fixtures) |
| II. Idempotency | PASS | CI verifies idempotency via double-apply test |
| III. Portability | N/A | CI is infrastructure, not DSL semantics |
| IV. Separation of Concerns | PASS | CI is separate from application code |
| V. Reproducibility | PASS | Pinned Go version and action versions |
| VI. Safe Defaults | PASS | No secrets needed for CI |
| VII. Minimal Surface Area | PASS | Single workflow file |
| VIII. English-Friendly Syntax | N/A | CI config, not DSL |
| IX. Canonical Formatting | PASS | CI runs `fmt --check` to enforce formatting |
| X. Strict Validation | PASS | CI runs `validate` on all examples |
| XI. Explicit References | PASS | All action versions pinned |
| XII. No Hidden Behavior | PASS | All CI steps are visible in workflow file |

**Review Gates**:
- Spec updated: Yes (this spec)
- Examples updated: N/A (CI doesn't change examples)
- Integration tests updated: N/A (CI runs existing tests)
- Formatter stable: N/A (CI verifies formatter stability)

**Testing Strategy**:
- Integration tests are exercised by the CI pipeline itself (the pipeline runs `go test ./...`)
- Example validation is an additional end-to-end check beyond the test suite

All gates pass. No violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/002-ci-pipeline/
├── plan.md              # This file
├── research.md          # Phase 0 output (minimal — no unknowns)
├── quickstart.md        # How to verify CI is working
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
.github/
└── workflows/
    └── ci.yml           # The CI workflow file (single file)
```

**Structure Decision**: This feature adds a single GitHub Actions workflow file at `.github/workflows/ci.yml`. No application code changes are needed. The `data-model.md` and `contracts/` artifacts are not applicable for a CI configuration feature.

## Workflow Design

### Pipeline Steps

The CI workflow runs as a single job with sequential steps:

```
Trigger (push/PR)
  │
  ├── 1. Checkout code
  ├── 2. Set up Go 1.25+
  ├── 3. Cache Go modules
  ├── 4. Download dependencies (go mod download)
  ├── 5. Build binary (go build -o agentz ./cmd/agentz)
  ├── 6. Run tests (go test ./... -count=1 -v)
  ├── 7. Lint (golangci-lint)
  ├── 8. Validate all examples (agentz validate examples/*/*.az)
  ├── 9. Format-check all examples (agentz fmt --check examples/*/*.az)
  ├── 10. Smoke test: plan + apply + idempotency on basic-agent example
  │     ├── agentz plan examples/basic-agent/basic-agent.az
  │     ├── agentz apply examples/basic-agent/basic-agent.az --auto-approve
  │     └── agentz apply examples/basic-agent/basic-agent.az --auto-approve (no changes)
  └── Done
```

### Trigger Configuration

```yaml
on:
  push:
    branches: ['**']     # All branches (FR-001)
  pull_request:
    branches: [main]     # PRs targeting main (FR-002)
```

### Key Decisions

1. **Single job vs multiple jobs**: Single job is simpler, avoids artifact passing overhead, and fits within the 10-minute constraint. The build step is fast (~5s) so parallelizing build and lint is unnecessary.

2. **Go module caching**: Use `actions/setup-go` built-in caching to avoid re-downloading dependencies on every run.

3. **golangci-lint-action**: Use the official `golangci/golangci-lint-action` for caching and performance. Pin to a specific version.

4. **Example validation loop**: Use shell glob `examples/*/*.az` to iterate all examples. If no files match, the step should exit with a clear warning rather than failing silently on an empty glob.

5. **Smoke test target**: Use `examples/basic-agent/basic-agent.az` as the representative example for the full plan-apply-idempotency cycle. It's the simplest example and least likely to have external dependencies.

6. **Idempotency verification**: The second `apply` run should produce output containing "No changes" or exit with code 0 and no mutations. The CI step checks for this.

7. **Timeout**: Set `timeout-minutes: 10` on the job to enforce FR-012.

8. **File extension**: The CI pipeline uses `.az` to match current codebase reality. When the codebase migrates to `.ias`, update the glob patterns accordingly.

## Action Version Pins

| Action | Version | Purpose |
|--------|---------|---------|
| actions/checkout | v4 | Clone repository |
| actions/setup-go | v5 | Install Go 1.25+ |
| golangci/golangci-lint-action | v6 | Run golangci-lint |
