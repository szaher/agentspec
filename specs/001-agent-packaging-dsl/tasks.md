# Tasks: Declarative Agent Packaging DSL

**Input**: Design documents from `/specs/001-agent-packaging-dsl/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Integration tests are included as the primary quality gate per constitution. Unit tests are omitted unless they unblock integration tests.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Go project initialization, directory structure, and tooling

- [x] T001 Create Go project directory structure per implementation plan (cmd/agentz/, internal/{parser,ast,formatter,ir,validate,plan,apply,state,plugins,adapters/{local,compose},sdk/{generator,python,typescript,go},events,policy}/, examples/, integration_tests/, spec/, plugins/monitor/, sdk/{python,typescript,go}/, DECISIONS/)
- [x] T002 Initialize Go 1.25+ module with pinned dependencies (cobra v1.10.2, wazero v1.11.0, go-cmp v0.7.0) in go.mod
- [x] T003 [P] Configure golangci-lint with project rules in .golangci.yml

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and interfaces that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 [P] Define AST node types for all resource kinds (Package, Agent, Prompt, Skill, MCPServer, MCPClient, Environment, Secret, Policy, Binding, Plugin) with source position tracking in internal/ast/ast.go
- [x] T005 [P] Define token types, keywords, and source position types for the lexer in internal/parser/token.go
- [x] T006 [P] Define IR document types (IRDocument, IRResource, IRPackage, IRPolicy, IRBinding) and deterministic JSON serializer with sorted keys and 2-space indentation in internal/ir/ir.go
- [x] T007 [P] Define state backend interface (Load, Save, Get, List) and state entry types (fqn, hash, status, last_applied, adapter, error) in internal/state/state.go
- [x] T008 [P] Implement local JSON state backend (read/write .agentz.state.json) in internal/state/local.go
- [x] T009 [P] Define adapter interface (Name, Validate, Plan, Apply, Export) with Action and Result types, and adapter registry with Register/Get functions in internal/adapters/adapter.go
- [x] T010 [P] Define structured event types (plan.started, apply.started, apply.resource, apply.completed, run.started, run.progress, run.completed, run.failed) with correlation ID support in internal/events/events.go
- [x] T011 [P] Define policy types and rule engine interface (Evaluate method taking resources, returning violations) in internal/policy/policy.go
- [x] T012 Set up CLI scaffold with cobra root command, global flags (--state-file, --verbose, --no-color, --correlation-id), and `agentz version` subcommand in cmd/agentz/main.go and cmd/agentz/version.go

**Checkpoint**: Foundation ready — user story implementation can now begin

---

## Phase 3: User Story 1 — Define and Validate Agent Configurations (Priority: P1) 🎯 MVP

**Goal**: Users can write .az definitions, format them canonically, and validate with actionable error messages including file:line:col + fix hints

**Independent Test**: Write a valid agent definition referencing a prompt and two skills → validate → zero errors. Introduce a typo in a skill reference → validate → error with location and "did you mean?" suggestion

### Implementation for User Story 1

- [x] T013 [US1] Implement lexer/tokenizer for .az syntax (keywords: package, version, lang, prompt, skill, agent, binding, uses, model, input, output, execution, description, content, default, adapter, secret, environment, policy, plugin, server, client, connects, exposes, env, store, command, require, deny, allow) in internal/parser/lexer.go
- [x] T014 [US1] Implement recursive descent parser producing AST from token stream, with error recovery and source-position-annotated error messages in internal/parser/parser.go
- [x] T015 [US1] Implement canonical formatter (AST → .az source) with deterministic output (consistent indentation, spacing, ordering) in internal/formatter/formatter.go
- [x] T016 [US1] Implement structural validator (required fields, type checks, schema conformance per data-model.md) in internal/validate/structural.go
- [x] T017 [US1] Implement semantic validator (reference resolution with "did you mean?" suggestions, plaintext secret rejection, duplicate name detection, import pin verification) in internal/validate/semantic.go
- [x] T017b [US1] Implement policy enforcement integration in validate pipeline (evaluate policy rules against IR resources, block unsafe configs per FR-031) in internal/policy/enforce.go
- [x] T018 [US1] Implement AST-to-IR lowering pass (resolve references, flatten to IRResource list, compute FQNs in package/kind/name format) in internal/ir/lower.go
- [x] T019 [US1] Implement IR content hash computation (SHA-256 of canonical JSON serialization of attributes with sorted keys, no whitespace) in internal/ir/hash.go
- [x] T020 [P] [US1] Implement `agentz fmt` command with --check and --diff flags in cmd/agentz/fmt.go
- [x] T021 [P] [US1] Implement `agentz validate` command with --format text|json output in cmd/agentz/validate.go
- [x] T022 [US1] Create golden fixture integration test for parse → validate → format round-trip (valid input, invalid input with error assertions, formatter idempotency) in integration_tests/validate_test.go

**Checkpoint**: At this point, User Story 1 should be fully functional — users can write, format, and validate .az definitions

---

## Phase 4: User Story 2 — Preview and Apply Changes (Priority: P2)

**Goal**: Users can preview changes with a deterministic plan, apply changes idempotently, detect drift, and handle partial failures gracefully

**Independent Test**: Create definition → plan (shows "create") → apply (resources created) → apply again ("no changes") → modify definition → plan (shows "update") → apply → apply again ("no changes"). Compare plan output from two runs for byte-identity

### Implementation for User Story 2

- [x] T023 [US2] Implement desired-state diff engine comparing IR resources against state entries (detect create/update/delete/noop actions using content hashes, resolve default binding per FR-043: implicit default for sole binding, error when ambiguous) in internal/plan/plan.go
- [x] T024 [US2] Implement deterministic plan serializer with text and JSON output formats (sorted by kind then name, sorted keys) in internal/plan/format.go
- [x] T025 [US2] Implement idempotent apply engine with mark-and-continue partial failure handling (per-resource success/failure results, accurate partial state recording) in internal/apply/apply.go
- [x] T026 [US2] Implement drift detection comparing state file against current adapter state in internal/plan/drift.go
- [x] T026b [P] [US2] Implement exportable run log and plan output persistence (write structured logs to file, support --out flag for plan export per FR-034) in internal/events/export.go
- [x] T027 [P] [US2] Implement `agentz plan` command with --target, --env, --format, --out flags (exit code 0 = no changes, 2 = changes pending) in cmd/agentz/plan.go
- [x] T028 [P] [US2] Implement `agentz apply` command with --target, --env, --auto-approve, --plan-file flags and structured event emission in cmd/agentz/apply.go
- [x] T029 [P] [US2] Implement `agentz diff` command with --target flag (exit code 0 = no drift, 2 = drift detected) in cmd/agentz/diff.go
- [x] T030 [US2] Create golden fixture integration test for plan → apply → apply(idempotency) → modify → plan → apply cycle with partial failure scenario in integration_tests/apply_test.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work — users can define, validate, plan, and apply configurations

---

## Phase 5: User Story 3 — Target Multiple Platforms (Priority: P3)

**Goal**: Users deploy one set of definitions to at least two target platforms (Local MCP and Docker Compose) via adapters, with exportable artifacts

**Independent Test**: Create agent definition → export to local-mcp (verify mcp-servers.json, mcp-clients.json, agents.json) → export same source to docker-compose (verify docker-compose.yml, config/, .env) → confirm DSL source unchanged

### Implementation for User Story 3

- [x] T031 [P] [US3] Implement Local MCP adapter (Validate, Plan, Apply, Export producing mcp-servers.json, mcp-clients.json, agents.json) in internal/adapters/local/local.go
- [x] T032 [P] [US3] Implement Docker Compose adapter (Validate, Plan, Apply, Export producing docker-compose.yml, config/, .env) in internal/adapters/compose/compose.go
- [x] T033 [US3] Implement `agentz export` command with --target, --env, --out-dir flags and deterministic artifact output in cmd/agentz/export.go
- [x] T034 [US3] Create golden fixture integration test for export to both adapters (byte-identical on re-export, distinct artifacts per adapter) in integration_tests/export_test.go

**Checkpoint**: At this point, User Stories 1–3 should work — definitions can be exported to two platforms from the same source

---

## Phase 6: User Story 4 — Manage Multi-Environment Configurations (Priority: P4)

**Goal**: Users maintain base definitions with environment-specific overlays (dev, staging, prod) that inherit unspecified attributes and reject conflicting values

**Independent Test**: Define base agent + dev overlay (change model) + prod overlay (change secret ref) → validate both → plan for dev (shows dev model) → plan for prod (shows prod values) → confirm conflict detection on ambiguous overlays

### Implementation for User Story 4

- [x] T035 [US4] Extend parser to handle environment blocks and override syntax (environment "name" { resource "ref" { attribute value } }) in internal/parser/parser.go
- [x] T036 [US4] Implement environment overlay merge logic (base attribute inheritance, override application, merge ordering) in internal/ir/environment.go
- [x] T037 [US4] Implement environment-aware validation (conflicting overlay detection, secret reference acceptance without value presence) in internal/validate/environment.go
- [x] T038 [US4] Wire --env flag through plan, apply, and export commands to select environment overlay before IR lowering in cmd/agentz/{plan,apply,export}.go
- [x] T039 [US4] Create golden fixture integration test for multi-environment plan/apply with conflict rejection scenario in integration_tests/environment_test.go

**Checkpoint**: At this point, User Stories 1–4 should work — users can manage environment-specific configurations

---

## Phase 7: User Story 5 — Extend with Plugins (Priority: P5)

**Goal**: Platform engineers create WASM plugins that add custom resource types, validators, transforms, and lifecycle hooks. Users install and reference plugins in their definitions

**Independent Test**: Load monitor plugin → write definition using Monitor resource type → validate (plugin validator runs) → plan (plugin transform runs) → apply (pre-apply hook executes) → verify plugin output in run log. Also test missing plugin error and duplicate type conflict

### Implementation for User Story 5

- [x] T040 [P] [US5] Implement WASM plugin host using wazero (memory management, execution timeout, WASI capabilities) in internal/plugins/host.go
- [x] T041 [P] [US5] Implement plugin manifest parser and loader (resolve from ./plugins/ and ~/.agentz/plugins/, parse JSON manifest, verify version pinning) in internal/plugins/loader.go
- [x] T042 [US5] Implement plugin validator dispatch (route validation to plugin based on applies_to resource types, aggregate errors) in internal/plugins/validate.go
- [x] T043 [US5] Implement plugin transform dispatch (call transform at compile stage, merge modified IR resources back) in internal/plugins/transform.go
- [x] T044 [US5] Implement lifecycle hook execution engine (pre-validate, post-validate, pre-plan, post-plan, pre-apply, post-apply, and runtime stages; explicit ordering enforcement; hook output capture) in internal/plugins/hooks.go
- [x] T045 [US5] Extend parser to handle plugin references (`plugin "name" version "x.y.z"`) and custom resource types from loaded plugins in internal/parser/parser.go
- [x] T046 [US5] Build example monitor plugin as standalone WASM module (manifest with Monitor resource type, threshold validator, alert transform, pre-apply hook) in plugins/monitor/
- [x] T047 [US5] Create golden fixture integration test for plugin load → validate → transform → hook execution, plus duplicate type conflict and missing plugin error in integration_tests/plugin_test.go

**Checkpoint**: At this point, User Stories 1–5 should work — the extension model is proven with one working plugin

---

## Phase 8: User Story 6 — Discover and Invoke Resources via SDKs (Priority: P6)

**Goal**: Developers use generated SDKs (Python, TypeScript, Go) to list agents, resolve endpoints, invoke runs, and stream events from applied configurations

**Independent Test**: Apply agent definition → generate Python SDK → use SDK to list agents (agent appears with name/status) → resolve endpoint → invoke run → stream events (start, progress, completed received)

### Implementation for User Story 6

- [x] T048 [US6] Implement type codegen engine reading IR JSON schema and generating language-specific type definitions in internal/sdk/generator/generator.go
- [x] T049 [P] [US6] Create Python SDK templates (AgentzClient, AsyncAgentzClient with list_*, get_*, resolve_endpoint, invoke, stream_events, typed errors) and generator in internal/sdk/python/
- [x] T050 [P] [US6] Create TypeScript SDK templates (@agentz/sdk with full TypeScript types, async/await, AsyncIterable streaming) and generator in internal/sdk/typescript/
- [x] T051 [P] [US6] Create Go SDK templates (sdk-go module with context-based cancellation, channel-based streaming) and generator in internal/sdk/go/
- [x] T052 [US6] Implement `agentz sdk generate` command with --lang python|typescript|go and --out-dir flags in cmd/agentz/sdk.go
- [x] T053 [P] [US6] Generate Python SDK output with minimal example (list agents, resolve endpoint) in sdk/python/
- [x] T054 [P] [US6] Generate TypeScript SDK output with minimal example in sdk/typescript/
- [x] T055 [P] [US6] Generate Go SDK output with minimal example in sdk/go/
- [x] T056 [US6] Create integration test for SDK generation and minimal example execution (Python, TypeScript, Go) in integration_tests/sdk_test.go

**Checkpoint**: At this point, all 6 user stories should be functional — SDKs provide programmatic access to defined resources

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Required constitution artifacts, examples, end-to-end tests, and documentation

- [x] T057 [P] Create at least 6 complete .az example configurations covering: basic agent, multi-skill agent, MCP server/client, multi-environment, plugin usage, multi-binding with export in examples/
- [x] T058 [P] Write normative language specification documenting all keywords, syntax rules, and semantics in spec/spec.md
- [x] T059 [P] Write IR JSON Schema (JSON Schema draft 2020-12) matching ir-schema.md contract in spec/ir.schema.json
- [x] T060 [P] Write ARCHITECTURE.md documenting components, boundaries, data flow (DSL → AST → IR → Adapter), and threat model at repository root
- [x] T061 [P] Create CHANGELOG.md with initial release entry at repository root
- [x] T062 [P] Create initial ADR documents (parser choice, plugin sandbox choice, state backend choice) in DECISIONS/
- [x] T062b [P] Implement language version migration guidance tooling (detect version mismatches, emit migration hints per FR-028) in cmd/agentz/migrate.go
- [x] T063 Create end-to-end golden fixture integration test suite covering full lifecycle: parse → validate → plan → apply → apply(idempotency) → export → adapter validation in integration_tests/golden_test.go
- [x] T064 Create cross-platform determinism golden fixture test (verify byte-identical IR, plan, and export output) in integration_tests/determinism_test.go
- [x] T065 Run quickstart.md validation — execute complete golden-path demo from fresh state

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Stories (Phase 3–8)**: All depend on Foundational phase completion
  - User stories can proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3 → P4 → P5 → P6)
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) — No dependencies on other stories
- **User Story 2 (P2)**: Depends on US1 (parser and IR lowering produce the inputs to plan/apply)
- **User Story 3 (P3)**: Depends on US2 (adapters need plan/apply infrastructure to operate)
- **User Story 4 (P4)**: Depends on US1 (parser extension for environment blocks) — Can run in parallel with US2/US3
- **User Story 5 (P5)**: Depends on US1 (parser extension for plugin refs) and US2 (hook execution depends on apply lifecycle) — Can start after US2
- **User Story 6 (P6)**: Depends on US2 (SDKs read state files produced by apply) — Can run in parallel with US3/US4/US5

### Within Each User Story

- Parser/lexer work before formatter/validator
- Core engine (plan, apply) before CLI commands
- Implementation before golden fixture integration test
- Story complete before moving to next priority (in sequential mode)

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (T004–T011)
- US1: T020 and T021 (fmt and validate CLI) can run in parallel after T013–T019
- US2: T027, T028, T029 (plan, apply, diff CLI) can run in parallel after T023–T026
- US3: T031 and T032 (local-mcp and compose adapters) can run in parallel
- US5: T040 and T041 (WASM host and plugin loader) can run in parallel
- US6: T049, T050, T051 (SDK templates per language) can run in parallel; T053, T054, T055 (SDK outputs) can run in parallel
- Phase 9: T057–T062 (examples, spec, architecture, changelog, ADRs) can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch CLI commands in parallel after parser/validator are ready:
Task: "Implement agentz fmt command in cmd/agentz/fmt.go"
Task: "Implement agentz validate command in cmd/agentz/validate.go"
```

## Parallel Example: User Story 3

```bash
# Launch both adapter implementations in parallel:
Task: "Implement Local MCP adapter in internal/adapters/local/local.go"
Task: "Implement Docker Compose adapter in internal/adapters/compose/compose.go"
```

## Parallel Example: User Story 6

```bash
# Launch all three SDK template implementations in parallel:
Task: "Create Python SDK templates and generator in internal/sdk/python/"
Task: "Create TypeScript SDK templates and generator in internal/sdk/typescript/"
Task: "Create Go SDK templates and generator in internal/sdk/go/"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test define → format → validate workflow independently
5. Demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Demo (MVP: define and validate!)
3. Add User Story 2 → Test independently → Demo (plan and apply!)
4. Add User Story 3 → Test independently → Demo (multi-platform export!)
5. Add User Story 4 → Test independently → Demo (multi-environment!)
6. Add User Story 5 → Test independently → Demo (plugin extensibility!)
7. Add User Story 6 → Test independently → Demo (SDK access!)
8. Complete Polish → Full golden-path demo and required artifacts

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (parser, formatter, validator)
   - Developer B: Foundational type refinements as A discovers needs
3. After US1 complete:
   - Developer A: User Story 2 (plan/apply engine)
   - Developer B: User Story 4 (environment overlays — only needs parser from US1)
4. After US2 complete:
   - Developer A: User Story 3 (adapters + export)
   - Developer B: User Story 5 (plugins)
   - Developer C: User Story 6 (SDKs)
5. Team completes Polish phase together

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Integration tests are the primary quality gate — golden fixtures verify determinism
- Constitution: unit tests allowed only when they unblock integration tests
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
