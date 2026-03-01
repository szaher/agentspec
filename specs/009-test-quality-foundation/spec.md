# Feature Specification: Test Foundation & Quality Engineering

**Feature Branch**: `009-test-quality-foundation`
**Created**: 2026-03-01
**Status**: Draft
**Input**: Address quality engineering gaps from gap analysis: 33/34 internal packages with zero unit tests, missing CI security scanning, no coverage reporting, no race detection, insufficient linting, code duplication.

**Gap Analysis References**: QE-001 through QE-014, FEAT-001, FEAT-002, FEAT-003, FEAT-004, BUG-036, BUG-037

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Unit Test Safety Net for Security-Critical Packages (Priority: P1)

An engineer modifying authentication, secret management, or policy enforcement code needs confidence that their changes don't introduce regressions. Unit tests must exist for each security-critical package, testing individual functions and edge cases in isolation.

**Why this priority**: 33 of 34 internal packages have zero unit tests. The security-critical packages (auth, secrets, policy, tools, session, state) handle authentication, secret redaction, policy enforcement, and tool execution — bugs here have the highest impact. Without unit tests, refactoring is blind and regressions go undetected.

**Independent Test**: Can be tested by running `go test ./internal/auth/... ./internal/secrets/... ./internal/policy/... ./internal/tools/... ./internal/session/... ./internal/state/...` and verifying each package has meaningful test coverage.

**Acceptance Scenarios**:

1. **Given** the auth package, **When** unit tests are run, **Then** key validation, rate limiting, and middleware are tested for correct behavior, edge cases, and error conditions.
2. **Given** the secrets package, **When** unit tests are run, **Then** env resolution, Vault resolution, and redaction filtering are tested with both valid and invalid inputs.
3. **Given** the policy package, **When** unit tests are run, **Then** each deny/require rule type is tested for correct accept/reject behavior.
4. **Given** the tools package, **When** unit tests are run, **Then** command, HTTP, SSRF, and inline executors are tested for correct behavior, error handling, and input validation. (Note: MCP is a separate package tested independently under FR-011.)
5. **Given** each security-critical package, **When** coverage is measured, **Then** line coverage is at least 80%.

---

### User Story 2 - Automated Security Scanning in CI (Priority: P1)

An engineer opens a pull request that introduces a dependency with a known vulnerability. The CI pipeline must detect this and block the merge, providing a clear report of the vulnerability and remediation steps.

**Why this priority**: Without automated vulnerability scanning, known CVEs in dependencies go undetected until exploitation. This is a standard industry practice that is currently missing from the CI pipeline.

**Independent Test**: Can be tested by intentionally adding a dependency with a known CVE and verifying the CI job reports the vulnerability and fails.

**Acceptance Scenarios**:

1. **Given** a pull request with a vulnerable dependency, **When** CI runs, **Then** the security scanning job reports the vulnerability with CVE ID and severity.
2. **Given** a pull request with insecure code patterns, **When** CI runs, **Then** the static security analysis job reports the findings with file path and line number.
3. **Given** a clean pull request, **When** CI runs, **Then** both security scanning jobs pass successfully.

---

### User Story 3 - Test Coverage Visibility and Thresholds (Priority: P2)

An engineering lead wants to track test coverage trends across the project and ensure coverage doesn't regress below a minimum threshold. Coverage reports must be generated on every CI run and visible in pull request reviews.

**Why this priority**: Without coverage visibility, engineers don't know which code paths are untested. Coverage thresholds prevent regression and create a ratchet that improves quality over time.

**Independent Test**: Can be tested by reducing a test to lower coverage below the threshold and verifying CI fails with a coverage report.

**Acceptance Scenarios**:

1. **Given** a CI run, **When** tests complete, **Then** a coverage report is generated showing per-package line coverage.
2. **Given** a pull request that reduces coverage below the minimum threshold, **When** CI runs, **Then** the coverage job fails with a message showing current vs. required coverage.
3. **Given** a pull request review, **When** the reviewer checks coverage, **Then** the coverage report is available as a CI artifact or service integration.

---

### User Story 4 - Race Condition Detection in CI (Priority: P2)

An engineer introduces a data race in shared state code. The CI pipeline must detect this automatically and block the merge, as race conditions cause intermittent, hard-to-diagnose failures.

**Why this priority**: The codebase has multiple shared-state patterns (rate limiters, session stores, memory stores, MCP pools). Without race detection in CI, these bugs only surface intermittently in production.

**Independent Test**: Can be tested by intentionally introducing a data race (concurrent map access without sync) and verifying CI detects and reports it.

**Acceptance Scenarios**:

1. **Given** a pull request with a data race, **When** CI runs the race detection job, **Then** the race is detected and the job fails with a detailed report.
2. **Given** all existing code, **When** the race detector runs, **Then** zero races are found (baseline is clean).
3. **Given** a clean pull request, **When** CI runs, **Then** the race detection job passes.

---

### User Story 5 - Comprehensive Linting (Priority: P3)

An engineer writes code that has unclosed HTTP response bodies, missing context in HTTP requests, or other common Go anti-patterns. The linter must catch these issues before code review.

**Why this priority**: The current linter configuration only enables 5 linters, missing important checks for HTTP body closure, context usage, security patterns, and code correctness. More comprehensive linting catches bugs at the cheapest point in the development cycle.

**Independent Test**: Can be tested by verifying the linter catches a known anti-pattern (e.g., `resp.Body` not closed) that the current configuration misses.

**Acceptance Scenarios**:

1. **Given** code with an unclosed HTTP response body, **When** the linter runs, **Then** the issue is reported with file path and line number.
2. **Given** code making HTTP requests without context, **When** the linter runs, **Then** the issue is flagged.
3. **Given** existing code with new linters enabled, **When** the linter runs, **Then** zero new violations exist (existing issues resolved or explicitly suppressed).

---

### User Story 6 - Code Deduplication (Priority: P3)

An engineer fixing a bug in the rate limiter discovers there are two nearly identical implementations. Consolidation into a single implementation reduces maintenance burden and prevents divergent behavior.

**Why this priority**: Code duplication means bug fixes and improvements must be applied in multiple places. The duplicate rate limiter and session ID generator implementations are low-risk consolidation targets.

**Independent Test**: Can be tested by verifying only one rate limiter implementation exists and both use cases (per-IP and per-agent) use the same underlying code.

**Acceptance Scenarios**:

1. **Given** the rate limiting functionality, **When** reviewing the codebase, **Then** there is exactly one rate limiter implementation used by all consumers.
2. **Given** the session ID generation functionality, **When** reviewing the codebase, **Then** there is exactly one session ID generator used by all session stores.
3. **Given** agent/pipeline lookup, **When** the server receives a request, **Then** lookup is done via indexed data structure (not linear scan).

---

### Edge Cases

- What happens when enabling new linters produces hundreds of existing violations? Violations are resolved incrementally with explicit suppressions for known-safe cases; CI does not block on pre-existing issues.
- What happens when race detection causes test timeouts? The race detection job has a longer timeout than normal tests (3x).
- What happens when coverage measurement significantly slows CI? Coverage is run as a separate job that does not block the main test job.

## Clarifications

### Session 2026-03-01

- Q: Constitution says "unit tests are permitted only when they unblock integration test development" and lists "unit-test-first development" as a Non-Goal. This spec proposes comprehensive unit testing across all packages. How should this conflict be resolved? → A: Amend constitution — add unit tests as a complementary quality gate alongside integration tests, reflecting project maturity beyond MVP.
- Q: How does the coverage threshold ratchet from 50% to 70%? Automated high-water mark, manual config updates, or time-based schedule? → A: Manual — engineer updates threshold in CI config when coverage improves (e.g., 50% → 60% → 70%).
- Q: SC-001 says all 34 packages need tests but only 15 are explicitly named (6 security-critical at 80% + 9 in FR-011 at 60%). Should all 34 get tests in this feature? → A: 15 named packages only; remaining packages get tests incrementally as they are touched. Update SC-001 accordingly.
- Q: Which tools should be used for CI security scanning (FR-002 dependency vulnerabilities, FR-003 static analysis)? → A: govulncheck (dependency vulnerability scanning) + gosec (static security analysis) — standard Go ecosystem tools.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Security-critical packages (auth, secrets, policy, tools, session, state) MUST have unit tests achieving at least 80% line coverage.
- **FR-002**: CI pipeline MUST include dependency vulnerability scanning using `govulncheck` that blocks merges for known vulnerabilities.
- **FR-003**: CI pipeline MUST include static security analysis using `gosec` that reports insecure code patterns.
- **FR-004**: CI pipeline MUST generate test coverage reports on every run.
- **FR-005**: CI pipeline MUST enforce a minimum overall coverage threshold (starting at 50%, ratcheting to 70%). The threshold is a manually maintained value in CI configuration; engineers update it when project coverage improves sufficiently.
- **FR-006**: CI pipeline MUST include a race detection job that runs all tests with the race detector enabled.
- **FR-007**: Linter configuration MUST include checks for unclosed HTTP bodies, missing context, and security-sensitive patterns.
- **FR-008**: Duplicate rate limiter implementations MUST be consolidated into a single reusable component.
- **FR-009**: Duplicate session ID generators MUST be consolidated into a single function.
- **FR-010**: Agent and pipeline lookups MUST use indexed data structures instead of linear scans.
- **FR-011**: Unit tests for remaining internal packages (loop, pipeline, llm, expr, memory, mcp, ir, validate, compiler) MUST be added achieving at least 60% line coverage.
- **FR-012**: The project constitution's Testing Strategy section MUST be amended to recognize unit tests as a complementary quality gate alongside integration tests, reflecting the project's maturity beyond MVP.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The 15 explicitly targeted internal packages (6 security-critical + 9 from FR-011) have unit test files with meaningful tests. Remaining packages receive tests incrementally as they are modified.
- **SC-002**: Security-critical packages achieve at least 80% line coverage.
- **SC-003**: Overall project test coverage is at least 50%.
- **SC-004**: CI security scanning catches 100% of known CVEs in dependencies.
- **SC-005**: CI race detection passes with zero reported data races.
- **SC-006**: Pull requests with coverage regression below threshold are automatically blocked.
- **SC-007**: Linter catches unclosed HTTP bodies and missing context in new code.
- **SC-008**: Zero duplicate implementations of rate limiting or session ID generation exist.

## Assumptions

- The existing integration test suite (31 files) will continue to run alongside the new unit tests.
- Coverage thresholds will be ratcheted upward manually (50% initial, then 60%, then 70%) by updating the threshold value in CI configuration when project coverage improves.
- New linter rules will be enabled incrementally to avoid blocking existing work.
- The race detection job may take longer than normal tests and should have its own CI timeout.
- Test coverage for compiler targets and SDK generators may be lower due to code generation complexity.
- The project constitution's Testing Strategy will be amended to add unit tests as a complementary quality gate (FR-012). The Non-Goals entry "Unit-test-first development" remains — this feature adds unit tests retroactively, not as a TDD mandate.
