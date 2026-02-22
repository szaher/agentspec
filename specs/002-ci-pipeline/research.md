# Research: CI Pipeline

**Feature**: 002-ci-pipeline
**Date**: 2026-02-22

## Research Summary

No significant unknowns or NEEDS CLARIFICATION items existed in the technical context. The following decisions were confirmed through standard practice review.

## Decision 1: CI Platform — GitHub Actions

**Decision**: Use GitHub Actions as the CI platform.
**Rationale**: The repository is hosted on GitHub. GitHub Actions is the native CI/CD solution with zero additional setup, free for public repositories, and integrates directly with pull request status checks.
**Alternatives considered**:
- CircleCI: Requires external account setup and webhook configuration.
- Jenkins: Self-hosted, requires infrastructure management.
- GitLab CI: Repository is not on GitLab.

## Decision 2: Single Job vs Multi-Job Pipeline

**Decision**: Use a single job with sequential steps.
**Rationale**: The total expected runtime is well under 10 minutes. A single job avoids the overhead of artifact passing between jobs, simplifies the workflow file, and reduces GitHub Actions minutes consumption. The build step takes ~5 seconds, so parallelizing build and lint provides negligible benefit.
**Alternatives considered**:
- Separate jobs for build, test, lint, and examples: Adds complexity (artifact upload/download), increases total wall time due to job scheduling overhead, and is unnecessary for a project of this size.
- Matrix strategy for multiple Go versions: Not needed — the project targets only Go 1.25+ and does not need backward compatibility testing.

## Decision 3: Linter Integration

**Decision**: Use the official `golangci/golangci-lint-action` GitHub Action.
**Rationale**: It handles golangci-lint installation, caching, and provides clean annotations on PRs. The project already has `.golangci.yml` configured.
**Alternatives considered**:
- Running `golangci-lint run` manually after installing it: Works but loses PR annotation integration and caching benefits.
- `reviewdog` with golangci-lint: More complex setup, not needed for a single-language project.

## Decision 4: Example File Extension

**Decision**: Use `.ias` as the file extension for example files in CI validation.
**Rationale**: The parent spec (001-agent-packaging-dsl) was amended to use `.ias` as the AgentSpec file extension. CI should validate against the canonical extension.
**Note**: The current codebase implementation still uses `.az`. The CI pipeline should use whichever extension the example files actually use at the time of implementation. The spec is forward-looking; the implementation may need to handle both during a transition period.

## Decision 5: Smoke Test Example

**Decision**: Use `examples/basic-agent/` as the representative example for the full plan-apply-idempotency smoke test.
**Rationale**: It's the simplest example with the fewest resources (1 prompt, 1 agent, 1 binding), making it the fastest and most reliable choice for CI smoke testing. More complex examples (multi-environment, plugin-usage) are still validated via `agentz validate` but don't need the full apply cycle in CI.
