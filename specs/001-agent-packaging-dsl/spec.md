# Feature Specification: Declarative Agent Packaging DSL

**Feature Branch**: `001-agent-packaging-dsl`
**Created**: 2026-02-22
**Status**: Draft
**Input**: User description from init-spec.md

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Define and Validate Agent Configurations (Priority: P1)

An agent developer writes declarative definitions for their agents,
prompts, skills/tools, and MCP server connections using an
English-friendly syntax. They run the toolchain's validation command
and receive immediate, actionable feedback on any errors — including
the exact location in the source file and a suggested fix.

**Why this priority**: Without the ability to author and validate
definitions, no other workflow is possible. This is the foundation
of the entire system.

**Independent Test**: Can be fully tested by writing a valid agent
definition referencing a prompt and two skills, running the
validator, and confirming zero errors. Then introducing an error
(e.g., referencing a nonexistent skill) and confirming the error
message includes the file, line number, and a fix suggestion.

**Acceptance Scenarios**:

1. **Given** a syntactically correct agent definition referencing
   one prompt and two skills, **When** the user runs validate,
   **Then** the system reports no errors.
2. **Given** a definition with a reference to a nonexistent skill
   name, **When** the user runs validate, **Then** the system
   reports an error with the exact source location (file, line,
   column) and suggests the closest matching name.
3. **Given** a definition containing a plaintext secret value,
   **When** the user runs validate, **Then** the system rejects the
   definition with an error indicating that secrets must be
   references to environment variables or secret stores.
4. **Given** a valid definition, **When** the user runs the
   formatter followed by validate, **Then** the formatted output is
   identical to re-formatting the already-formatted file.

---

### User Story 2 - Preview and Apply Changes (Priority: P2)

An agent developer previews what changes will occur before
committing them. They run a plan command that shows a stable,
machine-diffable summary of pending changes. They then apply
changes, and the system executes them idempotently — running apply
again with no source changes produces zero mutations.

**Why this priority**: The desired-state engine (plan/apply) is the
core value proposition. Without it, the DSL is a config format with
no operational power.

**Independent Test**: Can be tested by creating a definition,
running plan (verifying output describes creation), running apply
(verifying resources created), then running apply again (verifying
"no changes" output). Then modify the definition, run plan
(verifying output describes the diff), apply (verifying mutation),
and apply again (verifying no changes).

**Acceptance Scenarios**:

1. **Given** a new agent definition and no prior state, **When** the
   user runs plan, **Then** the output shows all resources as "to be
   created" in a stable, deterministic format.
2. **Given** the plan from scenario 1, **When** the user runs apply,
   **Then** all resources are created and the state is recorded.
3. **Given** the state from scenario 2 and no source changes,
   **When** the user runs apply again, **Then** the system reports
   "no changes" and performs zero mutations.
4. **Given** an applied state, **When** the user modifies one skill
   definition and runs plan, **Then** only the modified skill
   appears as "to be updated."
5. **Given** the same inputs run on two separate machines, **When**
   plan output is compared, **Then** the outputs are byte-identical.

---

### User Story 3 - Target Multiple Platforms (Priority: P3)

An agent developer writes one set of definitions and deploys to at
least two different target platforms. The DSL source remains
identical; only the target binding changes. Each target produces its
own exportable artifacts.

**Why this priority**: Portability is a core principle. Proving it
with at least two adapters validates the separation between DSL
semantics and platform-specific behavior.

**Independent Test**: Can be tested by creating a single agent
definition, exporting it to adapter A (verifying correct artifacts),
then exporting the same source to adapter B (verifying correct
artifacts), and confirming the DSL source was not modified for
either target.

**Acceptance Scenarios**:

1. **Given** one agent definition and two configured target
   bindings, **When** the user runs export for each target, **Then**
   two distinct sets of platform-specific artifacts are produced.
2. **Given** the same definition and target, **When** export is run
   twice, **Then** the output artifacts are byte-identical.
3. **Given** an agent definition, **When** the user runs plan
   against target A, **Then** the plan references only adapter-A
   concepts. **When** run against target B, **Then** the plan
   references only adapter-B concepts.

---

### User Story 4 - Manage Multi-Environment Configurations (Priority: P4)

An agent developer maintains a base definition and layers
environment-specific overrides (e.g., dev, staging, production).
Each environment can override variables, secret references, and
resource attributes while inheriting everything else from the base.

**Why this priority**: Multi-environment support is standard
practice for deployment tools. Without it, users must duplicate
entire definitions per environment.

**Independent Test**: Can be tested by defining a base agent with a
dev and prod overlay, validating both resolve correctly, and
confirming each environment produces distinct plan output reflecting
its overrides.

**Acceptance Scenarios**:

1. **Given** a base agent definition and a "dev" environment overlay
   that changes the model parameter, **When** the user runs plan for
   the "dev" environment, **Then** the plan reflects the dev model
   parameter.
2. **Given** the same base with a "prod" overlay, **When** the user
   runs plan for "prod," **Then** the plan reflects the
   prod-specific values.
3. **Given** an environment overlay that references a secret,
   **When** the user runs validate, **Then** the secret reference is
   accepted without requiring the actual secret value to be present.

---

### User Story 5 - Extend with Plugins (Priority: P5)

A platform engineer creates a plugin that introduces a custom
resource type, a validation rule for that type, and a compile-time
transform. The plugin integrates with the standard toolchain
lifecycle through declared hooks. Other users can install and use
the plugin in their definitions.

**Why this priority**: Extensibility enables the ecosystem to grow
beyond built-in resource types. One working plugin proves the
extension model is viable.

**Independent Test**: Can be tested by loading a plugin that defines
a "Monitor" resource type, writing a definition that uses it,
validating (confirming the plugin's validator runs), and planning
(confirming the plugin's transform runs and produces expected
output).

**Acceptance Scenarios**:

1. **Given** a plugin that declares a custom resource type
   "Monitor," **When** a user writes a definition using `Monitor`,
   **Then** the validator accepts it and applies the plugin's
   validation rules.
2. **Given** the same plugin with a compile-time transform, **When**
   the user runs plan, **Then** the transform modifies the
   intermediate representation as declared and the modification is
   visible in the plan output.
3. **Given** a plugin with a `pre-apply` hook, **When** the user
   runs apply, **Then** the hook executes before the apply operation
   and its output is included in the run log.
4. **Given** a plugin that is not installed, **When** a definition
   references its custom resource type, **Then** the validator
   reports the missing plugin and suggests how to install it.

---

### User Story 6 - Discover and Invoke Resources via SDKs (Priority: P6)

A developer uses a generated SDK (in Python, TypeScript, or Go) to
programmatically list available agents, resolve an agent's endpoint,
invoke a run, and stream events from the execution. The SDK operates
against local state and compiled artifacts.

**Why this priority**: SDKs make the DSL consumable by application
code. Without them, the DSL's value is limited to configuration
management.

**Independent Test**: Can be tested by applying a definition
locally, then using the SDK to list agents (confirming the defined
agent appears), resolving its endpoint, triggering an invocation,
and receiving at least one streamed event.

**Acceptance Scenarios**:

1. **Given** an applied agent definition, **When** a developer calls
   "list agents" via the SDK, **Then** the defined agent appears
   with its name, version, and status.
2. **Given** a listed agent, **When** the developer calls "resolve
   endpoint," **Then** the SDK returns the correct address for
   invocation.
3. **Given** a resolved agent, **When** the developer invokes a run
   with input parameters, **Then** the SDK returns a run identifier
   and the run begins execution.
4. **Given** an active run, **When** the developer streams events,
   **Then** at least start, progress, and completion events are
   received in order.

---

### Edge Cases

- What happens when a definition references a resource from a
  package pinned to a version that is no longer available?
- What happens when two plugins register the same custom resource
  type name? The system rejects the configuration at validation
  time with an error identifying both plugins and the conflicting
  type name.
- What happens when an apply fails midway through creating multiple
  resources? The system records partial state accurately, reports
  which resources succeeded and which failed, and allows the user to
  re-run apply to retry only the failed resources.
- What happens when the local state file is corrupted or deleted
  between plan and apply?
- What happens when a definition file contains syntax valid in
  language version N but deprecated in version N+1?
- What happens when two environment overlays define conflicting
  values for the same attribute? The system rejects the
  configuration at validation time. Each environment MUST resolve
  to a single unambiguous value for every attribute.
- What happens when two plugins declare hooks at the same lifecycle
  stage? The user MUST declare explicit ordering; the system
  rejects configurations where same-stage hooks lack a declared
  execution order.
- What happens when the user runs apply with no target binding
  specified and multiple bindings are defined? The system uses the
  binding marked as default. If only one binding exists, it is
  implicitly default. If multiple bindings exist and none is marked
  default, the system errors and lists available targets.

## Clarifications

### Session 2026-02-22

- Q: How are resources uniquely identified? → A: By type + name within a package scope; cross-package references use fully-qualified paths (package/type/name).
- Q: What happens when apply fails midway through multiple resources? → A: Mark-and-continue — record partial state accurately, report which resources succeeded and which failed, allow re-running apply to retry only failed resources.
- Q: How are plugin conflicts handled (duplicate type names, same-stage hooks)? → A: Fail-fast — reject duplicate resource type names at validation time; require explicit user-declared ordering for hooks registered at the same lifecycle stage.
- Q: How are environment overlay conflicts handled? → A: Fail-fast — reject conflicting attribute values within the same environment as a validation error. Each environment MUST resolve to a single unambiguous value for every attribute.
- Q: What happens when apply is run with no target and multiple bindings exist? → A: One binding may be marked as default; a sole binding is implicitly default; the system errors and lists available targets if no default is set with multiple bindings.

## Requirements *(mandatory)*

### Functional Requirements

**DSL Authoring & Formatting**

- **FR-001**: The system MUST provide a declarative syntax for
  defining agents, prompts, skills/tools, MCP servers, MCP clients,
  and platform bindings.
- **FR-002**: The syntax MUST be readable by non-programmers while
  remaining machine-parseable and unambiguous.
- **FR-003**: The system MUST provide a canonical formatter that
  produces a single deterministic output for any valid input.
- **FR-004**: Running the formatter on already-formatted source MUST
  produce byte-identical output.

**Validation**

- **FR-005**: The system MUST perform both structural (schema) and
  semantic validation on definitions.
- **FR-006**: Every validation error MUST include the source file
  path, line number, column number, and an actionable fix
  suggestion.
- **FR-007**: The system MUST reject definitions containing
  plaintext secret values and direct the user to use secret
  references instead.

**Desired-State Engine**

- **FR-008**: The system MUST provide plan, apply, diff, and export
  commands.
- **FR-009**: The plan command MUST produce stable, machine-diffable
  output that is byte-identical when run against unchanged inputs.
- **FR-010**: The apply command MUST be idempotent — running it
  twice with no intervening source changes MUST produce zero
  mutations on the second run.
- **FR-011**: The system MUST persist state after each apply and
  detect drift between the desired state and actual state on
  subsequent runs.
- **FR-012**: The state backend MUST be replaceable. The initial
  release MUST support a local file-based state backend.
- **FR-039**: When apply fails partway through, the system MUST
  record partial state accurately, report which resources succeeded
  and which failed, and allow re-running apply to retry only the
  failed resources without re-applying already-succeeded ones.

**Portability & Adapters**

- **FR-013**: The system MUST separate platform-neutral semantics
  from platform-specific behavior through an adapter mechanism.
- **FR-014**: Adapters MUST operate on an intermediate
  representation, not on the raw DSL source.
- **FR-015**: The initial release MUST include at least two working
  adapters to demonstrate portability.
- **FR-016**: Each adapter MUST produce exportable artifacts
  (configuration bundles or manifests).
- **FR-043**: A binding MAY be marked as default. When only one
  binding is defined, it MUST be treated as implicitly default.
  When multiple bindings exist and no default is marked, the system
  MUST error and list available targets rather than choosing one
  implicitly.

**Environments & Overrides**

- **FR-017**: The system MUST support named environments (e.g., dev,
  staging, production) that override base definitions.
- **FR-018**: Environment overrides MUST inherit all unspecified
  attributes from the base definition.
- **FR-042**: Each environment MUST resolve to a single unambiguous
  value for every attribute. The system MUST reject at validation
  time any configuration where multiple overlays for the same
  environment define conflicting values for the same attribute.

**Plugins & Extensions**

- **FR-019**: The system MUST support plugins that declare
  capabilities: custom resource types, validators, transforms, and
  lifecycle hooks.
- **FR-020**: Plugins MUST be versioned and pinned to specific
  versions in definitions.
- **FR-021**: Plugins MUST execute in isolation (separate process or
  sandbox).
- **FR-022**: The system MUST support lifecycle hooks at:
  pre-validate, post-validate, pre-plan, post-plan, pre-apply,
  post-apply, and runtime.
- **FR-040**: The system MUST reject at validation time any
  configuration where two plugins declare the same custom resource
  type name.
- **FR-041**: When multiple plugins register hooks at the same
  lifecycle stage, the user MUST declare explicit execution
  ordering. The system MUST reject configurations where same-stage
  hooks lack a declared order.

**SDK Generation**

- **FR-023**: The system MUST generate SDKs for Python, TypeScript,
  and Go.
- **FR-024**: SDKs MUST support listing and querying defined
  resources (agents, prompts, skills, servers, clients).
- **FR-025**: SDKs MUST support resolving resource endpoints and
  addresses.
- **FR-026**: SDKs MUST support invoking agent runs and streaming
  execution events.

**Resource Identity**

- **FR-036**: Each resource MUST be uniquely identified by the
  combination of its type and name within a package scope.
- **FR-037**: Cross-package references MUST use fully-qualified
  paths in the form `package/type/name`.
- **FR-038**: The system MUST reject definitions that declare two
  resources of the same type with the same name within a single
  package.

**Versioning & Compatibility**

- **FR-027**: The DSL language version MUST be explicitly declared in
  every definition file or package.
- **FR-028**: The system MUST provide migration guidance for any
  breaking changes between language versions.
- **FR-029**: All external package references MUST be pinned to
  specific versions, content hashes, or git SHAs.

**Security**

- **FR-030**: Secrets MUST be expressed as references to environment
  variables or external secret stores, never as literal values.
- **FR-031**: The system MUST support a policy mechanism capable of
  blocking configurations that violate security constraints (e.g.,
  disallowing unrestricted network access or unpinned imports).

**Observability**

- **FR-032**: The system MUST emit structured events for plan,
  apply, and run operations.
- **FR-033**: Every operation MUST include a correlation identifier
  for traceability.
- **FR-034**: Run logs and plan outputs MUST be exportable for
  external analysis.

**Determinism**

- **FR-035**: Given identical inputs, the system MUST produce
  byte-identical AST, intermediate representation, plan output, and
  exported artifacts.

### Key Entities

- **Agent**: A declarative definition of an AI agent, including its
  model configuration, associated prompt, available skills, and
  behavioral parameters.
- **Prompt**: A reusable prompt template with variable placeholders,
  versioned independently.
- **Skill**: A capability (tool) an agent can invoke, defined by
  name, input/output schema, and execution binding.
- **MCP Server**: A server exposing capabilities via the Model
  Context Protocol, with transport configuration and authentication
  references.
- **MCP Client**: A client that connects to one or more MCP servers
  and makes their capabilities available to agents.
- **Package**: A distributable, versioned unit containing related
  definitions with dependency metadata. Resources within a package
  are uniquely identified by type + name. Cross-package references
  use fully-qualified paths (package/type/name).
- **Environment**: A named configuration scope (e.g., dev, staging,
  prod) providing variable overrides layered on a base definition.
- **Secret**: A reference to a sensitive value stored outside DSL
  source (environment variable path or secret store key).
- **Policy**: A set of rules governing permitted configurations,
  enforced during validation and apply.
- **Binding**: A mapping from a platform-neutral definition to a
  specific target platform adapter.
- **Plugin**: An extension package that adds custom resource types,
  validators, transforms, or lifecycle hooks.
- **State**: A persisted record of the last-applied desired state,
  used for drift detection and idempotency.

## Assumptions

- The initial release targets local CLI operation with no hosted
  control plane or API server. SDKs operate against local state and
  compiled artifacts.
- The two MVP adapters will target a local runtime and one external
  agentic platform (specific platform to be decided during
  planning).
- Plugin isolation uses sandboxing where feasible; strict process
  isolation is acceptable as a fallback for MVP.
- Package registry support is local-only for MVP (local file system
  paths); remote registry support is deferred.
- The system is a developer tool targeting engineers and technical
  domain experts, not end-users of the agents themselves.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new user can clone the repository, follow the
  documentation, and run the golden-path demo (define, validate,
  plan, apply, export) from scratch in under 15 minutes.
- **SC-002**: The same definition applied to two different target
  platforms produces correct, distinct artifacts for each without
  any source modification.
- **SC-003**: Running apply twice with no source changes produces
  zero mutations on the second run, 100% of the time.
- **SC-004**: Running plan on identical inputs on two separate
  machines produces byte-identical output.
- **SC-005**: Every validation error includes the source location
  and an actionable fix suggestion, with zero "unknown error" or
  location-less messages.
- **SC-006**: At least one plugin successfully adds a custom
  resource type, validates instances of it, and transforms the
  intermediate representation — all exercised by an integration
  test.
- **SC-007**: SDKs in all three target languages compile and
  successfully execute a minimal example that lists resources,
  resolves an endpoint, and invokes a run.
- **SC-008**: The formatter produces byte-identical output when run
  on already-formatted source, 100% of the time.
- **SC-009**: 100% of integration tests pass, covering the full
  lifecycle: parse, validate, plan, apply, apply (idempotency),
  export, and adapter validation.
