<!--
Sync Impact Report
===================
Version change: N/A (template placeholders) → 1.0.0
Bump rationale: MAJOR — initial ratification of project constitution.

Modified principles: N/A (all new)

Added sections:
  - Mission
  - Non-Goals
  - Core Principles (12): Determinism, Idempotency, Portability,
    Separation of Concerns, Reproducibility, Safe Defaults,
    Minimal Surface Area, English-Friendly Syntax,
    Canonical Formatting, Strict Validation,
    Explicit References, No Hidden Behavior
  - Contracts & Standards: Desired-State Engine, Adapter Contract,
    Plugin/Extension Contract, SDK Contract
  - Development Workflow: Spec-First Workflow, Required Artifacts,
    Testing Strategy, Review Gates
  - Operational Requirements: Security & Supply-Chain, Observability
  - Governance: Amendment Process, Versioning Policy,
    Definition of Done (MVP)

Removed sections: N/A

Templates requiring updates:
  - .specify/templates/plan-template.md: ✅ compatible
    (Constitution Check is a runtime placeholder filled by /speckit.plan)
  - .specify/templates/spec-template.md: ✅ compatible
    (no constitution-specific constraints in template structure)
  - .specify/templates/tasks-template.md: ⚠ pending
    (template says "Tests are OPTIONAL"; constitution mandates
    integration tests as the primary quality gate — reconciled at
    runtime by /speckit.tasks reading the constitution, but template
    wording may mislead; consider adding a constitution-override note)
  - .specify/templates/commands/*.md: N/A (no command files found)
  - README.md: N/A (not yet created)
  - ARCHITECTURE.md: N/A (not yet created; listed as required artifact)

Follow-up TODOs:
  - Create ARCHITECTURE.md (required by section "Required Artifacts")
  - Create spec/spec.md (required by section "Required Artifacts")
  - Create spec/ir.schema.json (required by section "Required Artifacts")
  - Create examples/ with at least 6 configs (required artifact)
  - Create integration_tests/ directory (required artifact)
  - Create CHANGELOG.md (required artifact)
  - Create DECISIONS/ directory for ADRs (required artifact)
-->
# Agentz Constitution

## Mission

Build an English-friendly declarative language that expresses desired
state for agents, prompts, skills/tools, MCP servers/clients, and
platform bindings; compiles to a canonical IR; applies idempotently
across runs; generates SDKs (Python, TypeScript, Go) for discovery
and invocation.

## Non-Goals

The following are explicitly out of scope until promoted by an
amendment to this constitution:

- Building a full hosted SaaS control plane.
- Complex UI.
- Exhaustive provider support.
- Unit-test-first development.

## Core Principles

### I. Determinism

Same inputs MUST produce the same AST, IR, plan, and export bytes.

**Rationale:** Determinism is the foundation of trust in a
desired-state system. Without it, drift detection, caching, and
reproducible builds are impossible.

### II. Idempotency

Running `apply` twice with no intervening changes MUST produce no
mutations on the second run.

**Rationale:** Idempotency guarantees safe re-execution and enables
retry-based recovery without side effects.

### III. Portability

The source DSL MUST be platform-neutral. Platform-specific behavior
MUST be isolated in adapters.

**Rationale:** Portability ensures the DSL serves as a single source
of truth regardless of deployment target.

### IV. Separation of Concerns

Surface syntax MUST NOT encode semantics. All semantic meaning MUST
reside in the IR.

**Rationale:** Decoupling syntax from semantics enables multiple
surface syntaxes, tooling interoperability, and independent evolution
of parser and engine.

### V. Reproducibility

All builds and exports MUST be pinned and verifiable. Dependencies
MUST be locked to specific versions or content hashes.

**Rationale:** Reproducibility is required for auditing, rollback,
and supply-chain integrity.

### VI. Safe Defaults

Secrets MUST NOT appear as plaintext literals in DSL source. The
system MUST enforce least-privilege defaults for all resource
definitions.

**Rationale:** Security misconfigurations are the most common class
of production incidents; safe defaults reduce the blast radius of
mistakes.

### VII. Minimal Surface Area

New keywords and constructs MUST be justified by a concrete use case
accompanied by at least one example. Convention MUST be preferred
over configuration.

**Rationale:** A small language surface reduces learning cost, parser
complexity, and maintenance burden.

### VIII. English-Friendly Syntax

DSL source MUST be readable by non-programmers while remaining
machine-parseable and unambiguous.

**Rationale:** Agent definitions are authored collaboratively by
engineers and domain experts; readability lowers the barrier to
participation.

### IX. Canonical Formatting

A single formatter MUST exist. All source MUST conform to its
output. No stylistic variance is permitted.

**Rationale:** Canonical formatting eliminates bikeshedding and
produces clean diffs.

### X. Strict Validation

The toolchain MUST perform both schema and semantic validation.
Every error MUST include source location and an actionable fix hint.

**Rationale:** Early, precise validation shortens feedback loops and
prevents invalid state from reaching downstream systems.

### XI. Explicit References

All external imports MUST be pinned to a version or git SHA.
Intentionally floating references MUST declare a policy flag.

**Rationale:** Implicit dependency resolution is a primary vector
for supply-chain attacks and nondeterministic builds.

### XII. No Hidden Behavior

All transforms MUST be declared and discoverable. The system MUST
NOT apply undeclared mutations to the IR or output artifacts.

**Rationale:** Hidden behavior erodes trust and makes debugging
impossible at scale.

## Contracts & Standards

### Desired-State Engine

The CLI MUST provide the following commands: `fmt`, `validate`,
`plan`, `apply`, `diff`, `export`.

- `plan` output MUST be stable and machine-diffable.
- The state backend MUST be pluggable. MVP MUST support a local
  state file.
- Drift detection MUST be explicit: the engine MUST report drift
  and MUST NOT silently reconcile.

### Adapter Contract

- Adapters MUST accept IR as input, never raw DSL.
- Adapters MUST be thin: mapping and deployment only, no business
  logic.
- MVP MUST include at least two adapters to prove portability.
- Adapter outputs MUST be exportable as artifacts (manifests or
  config bundles).

### Plugin/Extension Contract

- Plugins MUST declare their capabilities: resource types,
  validators, transforms, and hooks.
- Plugins MUST be versioned and pinned.
- Default isolation MUST be a separate process or WASM sandbox
  (WASM preferred).
- Hooks MUST be lifecycle-scoped: `pre-validate`, `post-validate`,
  `pre-plan`, `post-plan`, `pre-apply`, `post-apply`, `runtime`.

### SDK Contract

SDKs MUST be generated from a single source of truth:

- If an API exists: a formal API contract (OpenAPI, protocol
  definition, or equivalent structured specification) MUST drive
  SDK generation.
- If no API exists: a stable library API over state and artifacts
  MUST be provided.

SDKs MUST support:

- Listing and querying resources (agents, prompts, skills, servers,
  clients).
- Resolving endpoints and URIs.
- Invoking an agent/run and streaming events (where the target
  platform supports it).

## Development Workflow

### Spec-First Workflow

Every change MUST follow this sequence:

1. Write or update the specification before writing code.
2. Add or adjust examples that demonstrate the change.
3. Update the IR schema and adapter/plugin contracts if impacted.
4. Add or adjust integration tests that exercise end-to-end
   behavior.
5. Implement.
6. Run formatter, validate, plan/apply, and export fixture checks.

### Required Artifacts

The repository MUST maintain the following artifacts:

- `spec/spec.md` — normative language and semantics.
- `spec/ir.schema.json` (or protobuf) — canonical IR schema.
- `ARCHITECTURE.md` — component boundaries and threat model.
- `examples/` — at least 6 complete configuration examples.
- `integration_tests/` — end-to-end tests with fixture-based
  assertions.
- `CHANGELOG.md` — release history.
- `DECISIONS/` — Architecture Decision Records for breaking
  choices.

### Testing Strategy

Integration tests are the primary quality gate. The test pipeline
MUST exercise:

- parse, validate, plan, apply, apply (idempotency), export,
  adapter validation.
- Plugin load, hook execution, and lifecycle ordering.
- Cross-platform determinism via golden fixtures.

Unit tests are permitted only when they unblock integration test
development.

### Review Gates

A change MUST NOT be merged unless:

- The specification is updated (or "no spec change" is explicitly
  justified).
- Examples are updated or added.
- Integration tests are updated or added.
- The formatter produces stable output on all changed files.
- Plan/apply/export fixtures are updated deterministically.

## Operational Requirements

### Security & Supply-Chain

- Secrets MUST be references (environment variable or secret store
  path), never literal values.
- Signed packages and provenance MUST be designed as first-class
  concepts. MVP MAY stub the signing implementation.
- Dependency pinning is REQUIRED for all reproducible exports.
- A policy layer MUST be capable of blocking unsafe configurations
  (network access, file system access, exec permissions, unpinned
  imports).

### Observability

- The system MUST emit structured events for `plan`, `apply`, and
  `run` operations.
- Every operation MUST carry a correlation ID.
- Run logs and plans MUST be exportable.
- Core MUST expose minimal metrics and tracing hooks. Richer
  observability MUST be delegable to plugins.

## Governance

### Amendment Process

This constitution supersedes all other development practices for
the Agentz project. Amendments require:

1. A pull request modifying this file with rationale in the PR
   description.
2. At least one approval from a project maintainer.
3. Migration guidance for any backward-incompatible changes.
4. Updated golden fixtures reflecting the amendment's impact.

### Versioning Policy

- The language version and IR version MUST be explicit and tracked
  independently.
- Backward-compatible changes are the default expectation.
- Breaking changes MUST include:
  - Migration guidance.
  - A deprecation window of at least one minor release.
  - Updated golden fixtures.

### Definition of Done (MVP)

The MVP is complete when all of the following are satisfied:

- One golden-path demo runs from a fresh clone to a working
  deployment.
- Two adapters produce correct output from the same DSL/IR.
- One plugin demonstrates a custom resource type with validation
  and transform.
- SDKs in Python, TypeScript, and Go compile and execute a
  minimal example.

**Version**: 1.0.0 | **Ratified**: 2026-02-22 | **Last Amended**: 2026-02-22
