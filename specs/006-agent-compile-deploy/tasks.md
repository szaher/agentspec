# Tasks: Agent Compilation & Deployment Framework

**Input**: Design documents from `/specs/006-agent-compile-deploy/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- CLI commands: `cmd/agentspec/`
- Core packages: `internal/`
- Examples: `examples/`
- Integration tests: `integration_tests/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, new dependencies, and package directory structure

- [x] T001 Add `expr-lang/expr` dependency to `go.mod` and run `go mod tidy`
- [x] T002 [P] Create package directory structure for all new packages: `internal/compiler/`, `internal/imports/`, `internal/expr/`, `internal/controlflow/`, `internal/validation/`, `internal/evaluation/`, `internal/frontend/`, `internal/frontend/web/`, `internal/registry/`, `internal/auth/`
- [x] T003 [P] Create integration test fixture directories: `integration_tests/fixtures/compile/`, `integration_tests/fixtures/import/`, `integration_tests/fixtures/eval/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: IntentLang 3.0 language extensions and shared infrastructure that ALL user stories depend on

**CRITICAL**: No user story work can begin until this phase is complete

### AST & Parser Extensions

- [x] T004 Add AST node types for `import`, `config`, `validate`, `eval`, `on input`, `if/else`, `for each`, `rule`, `case`, `use skill`, `delegate`, `respond` in `internal/ast/ast.go`
- [x] T005 Add new keywords to the lexer: `import`, `as`, `if`, `else`, `for`, `each`, `in`, `config`, `validate`, `eval`, `rule`, `case`, `when`, `on`, `input`, `use`, `skill`, `with`, `delegate`, `to`, `respond`, `required`, `secret`, `default`, `scoring`, `threshold`, `tags`, `max_retries` in `internal/parser/lexer.go`
- [x] T006 Implement parser for `import` statements (local path and versioned package) in `internal/parser/parser.go`
- [x] T007 Implement parser for `config` block within agent definitions in `internal/parser/parser.go`
- [x] T008 Implement parser for `validate` block with `rule` entries in `internal/parser/parser.go`
- [x] T009 Implement parser for `eval` block with `case` entries in `internal/parser/parser.go`
- [x] T010 Implement parser for `on input` block with `use skill`, `delegate`, `respond` statements in `internal/parser/parser.go`
- [x] T011 Implement parser for `if`/`else if`/`else` conditional blocks in `internal/parser/parser.go`
- [x] T012 Implement parser for `for each` loop construct in `internal/parser/parser.go`

### Formatter, Validator, IR

- [x] T013 Extend formatter for all new constructs (import, config, validate, eval, on input, if/else, for each) in `internal/formatter/formatter.go`
- [x] T014 Extend semantic validator for import references, config param constraints (secret cannot have default), validation rule expressions, eval case thresholds, and control flow completeness (else required or warning) in `internal/validate/validate.go`
- [x] T015 Extend IR types to include ImportGraph, ConfigParams, ValidationRules, EvalCases, ControlFlowBlocks, and OnInputBlock in `internal/ir/ir.go`
- [x] T016 Implement lowering of all new AST nodes to IR in `internal/ir/lower.go`

### Expression Engine

- [x] T017 Implement expression compiler that wraps `expr-lang/expr` with compile-time type checking and validation of expression syntax in `internal/expr/compiler.go`
- [x] T018 Implement runtime expression evaluator that evaluates compiled expressions against a context (input, session, steps, config, output) in `internal/expr/evaluator.go`

### Auth Middleware

- [x] T019 Implement API key validation logic (compare against `AGENTSPEC_API_KEY` env var, timing-safe comparison) in `internal/auth/key.go`
- [x] T020 Implement HTTP auth middleware that checks `Authorization: Bearer <key>` header, skips `/healthz` and static assets, returns 401 JSON error for unauthenticated requests, and supports `--no-auth` bypass in `internal/auth/middleware.go`

**Checkpoint**: Foundation ready — IntentLang 3.0 syntax fully parseable, IR represents all new constructs, expression engine operational, auth middleware available. User story implementation can now begin.

---

## Phase 3: User Story 1 — Compile Agent to Standalone Service (Priority: P1) MVP

**Goal**: A developer writes an `.ias` file and compiles it into a self-contained executable that runs as an agent service with health checks, config resolution, validation, and eval support.

**Independent Test**: Compile `examples/compiled-agent/hello-agent.ias`, run the binary with `--no-auth`, send a request to `/v1/agents/hello/invoke`, verify a valid response.

### Implementation for User Story 1

- [x] T021 [US1] Implement runtime config resolver that reads declared config params from environment variables and config files, with fail-fast on missing required params at startup in `internal/runtime/config.go`
- [x] T022 [US1] Implement config reference generator that outputs a markdown file listing all required and optional params for a compiled agent in `internal/compiler/configref.go`
- [x] T023 [US1] Implement output validation rule executor that runs declared validation rules against agent responses using the expression engine, with retry logic for `error` severity in `internal/validation/validator.go`
- [x] T024 [US1] Implement validation retry logic that re-invokes the agent with validation feedback appended to the prompt, respecting `max_retries` per rule in `internal/validation/retry.go`
- [x] T025 [US1] Integrate config resolution, auth middleware, and validation into the runtime server: add config loading at startup, auth middleware on all API routes, validation after each agent response in `internal/runtime/runtime.go` and `internal/runtime/server.go`
- [x] T026 [US1] Create the compiled agent `main.go` template that embeds config JSON via `go:embed`, initializes RuntimeConfig, starts HTTP server with auth middleware, health endpoint, and agent routes in `internal/compiler/template.go`
- [x] T027 [US1] Implement the standalone compilation target: generate temp dir, write config JSON + main.go from template, invoke `go build -trimpath -ldflags="-s -w"`, report output path and size in `internal/compiler/embed.go`
- [x] T028 [US1] Implement cross-compilation support that sets `GOOS`/`GOARCH` based on `--platform` flag for the 5 supported targets in `internal/compiler/cross.go`
- [x] T029 [US1] Implement the core compilation orchestrator: parse .ias → validate → lower to IR → select target → invoke target → produce artifact → generate config reference in `internal/compiler/compiler.go`
- [x] T030 [US1] Implement the `agentspec compile` CLI command per cli-contract.md with `--target`, `--output`, `--platform`, `--name`, `--embed-frontend`, `--verbose` flags and JSON output mode in `cmd/agentspec/compile.go`
- [x] T031 [US1] Implement the batch evaluation runner that loads eval cases from IR, invokes the agent for each input, scores outputs using the configured scoring method, and produces a quality report in `internal/evaluation/runner.go`
- [x] T032 [P] [US1] Implement scoring methods: `exact` (string match), `contains` (substring), `semantic` (embedding similarity with threshold), `custom` (expr expression) in `internal/evaluation/scoring.go`
- [x] T033 [P] [US1] Implement eval report generation in table, JSON, and markdown formats with per-case results, overall score, and comparison against previous runs in `internal/evaluation/report.go`
- [x] T034 [US1] Implement the `agentspec eval` CLI command per cli-contract.md with `--agent`, `--tags`, `--output`, `--format`, `--compare` flags in `cmd/agentspec/eval.go`
- [x] T035 [P] [US1] Create example `.ias` file `examples/compiled-agent/hello-agent.ias` demonstrating a basic agent with prompt, model, config params, validation rules, and eval cases
- [x] T036 [US1] Create golden fixture for compilation determinism test and write integration test in `integration_tests/compile_test.go` that compiles hello-agent.ias, verifies binary is produced, runs it, and checks `/healthz` and `/v1/agents/hello/invoke` endpoints
- [x] T037 [US1] Write integration test for validation rule execution in `integration_tests/validation_test.go` that compiles an agent with validation rules, invokes it, and verifies rules are enforced
- [x] T038 [US1] Write integration test for eval command in `integration_tests/eval_test.go` that compiles an agent with eval cases, runs `agentspec eval`, and verifies the report output

**Checkpoint**: At this point, a developer can write an `.ias` file, compile it to a standalone binary, run it, send requests, and evaluate agent quality. This is the MVP.

---

## Phase 4: User Story 2 — Enhanced IntentLang with Imports and Control Flow (Priority: P2)

**Goal**: A developer splits agent definitions across multiple `.ias` files using imports, and uses `if/else` and `for each` within agent definitions for runtime branching.

**Independent Test**: Create a multi-file project with `main.ias` importing `skills/search.ias`, compile it, run the agent, send input matching each conditional branch, verify correct skill is invoked per branch.

### Implementation for User Story 2

- [x] T039 [US2] Implement local file import resolver that resolves relative paths, reads imported files, and returns parsed AST nodes in `internal/imports/resolver.go`
- [x] T040 [US2] Implement dependency graph builder with circular dependency detection using Tarjan's SCC algorithm in `internal/imports/graph.go`
- [x] T041 [US2] Implement lock file read/write (`.agentspec.lock`) recording resolved versions and content hashes for reproducible builds in `internal/imports/lock.go`
- [x] T042 [US2] Implement Minimal Version Selection algorithm for resolving package version constraints across transitive dependencies in `internal/imports/mvs.go`
- [x] T043 [US2] Implement runtime context object that provides `input`, `session`, `steps`, `config`, and `output` variables to control flow expressions in `internal/controlflow/context.go`
- [x] T044 [US2] Implement `if/else` and `for each` runtime executor that evaluates conditions via the expression engine and dispatches to the appropriate skill/delegate/respond action in `internal/controlflow/executor.go`
- [x] T045 [US2] Integrate import resolution into the compile pipeline: resolve all imports before validation, merge imported definitions into the IR document, and report import errors with dependency chains in `internal/compiler/compiler.go`
- [x] T046 [US2] Integrate control flow executor into the runtime loop: when an agent has an `on input` block, execute it instead of the default prompt-and-respond flow in `internal/runtime/runtime.go`
- [x] T047 [P] [US2] Create multi-file example `examples/multi-file-agent/main.ias` with `examples/multi-file-agent/skills/search.ias` and `examples/multi-file-agent/skills/respond.ias` demonstrating import and reuse
- [x] T048 [P] [US2] Create control flow example `examples/control-flow-agent/router-agent.ias` demonstrating `if/else` routing and `for each` iteration
- [x] T049 [US2] Write integration test in `integration_tests/import_test.go` that compiles the multi-file example, verifies import resolution, tests circular dependency detection, and tests missing import error reporting
- [x] T050 [US2] Write integration test in `integration_tests/controlflow_test.go` that compiles the router-agent example, runs it, sends inputs matching each branch, and verifies correct routing and loop execution

**Checkpoint**: At this point, developers can build multi-file agent projects with imports, conditionals, and loops. Agents can route requests dynamically based on input.

---

## Phase 5: User Story 3 — Framework Code Generation via Compilation Plugins (Priority: P3)

**Goal**: A developer compiles their `.ias` file to framework-specific source code (CrewAI, LangGraph, LlamaStack, LlamaIndex) using compilation plugins.

**Independent Test**: Compile a standard `.ias` agent file with `--target crewai`, inspect the generated project for correct structure and idiomatic Python code, verify `pyproject.toml` and `main.py` are present.

### Implementation for User Story 3

- [x] T051 [US3] Implement compilation plugin host functions: `compile(ir_json) -> CompileResult`, `feature_support() -> FeatureMap`, `version() -> string` as WASM exports in `internal/plugins/compile.go`
- [x] T052 [US3] Extend plugin loader to detect `compile` capability in plugin manifests and register compilation targets in `internal/plugins/loader.go`
- [x] T053 [US3] Implement safe zone detection and preservation logic: parse `AGENTSPEC GENERATED START/END` and `USER CODE START/END` markers, preserve user code sections during recompilation in `internal/compiler/safezone.go`
- [x] T054 [US3] Implement feature support gap analysis: compare agent IR features against target's `feature_support()` map, generate warnings for `partial`/`none` features, include workaround suggestions in compilation output in `internal/compiler/compiler.go`
- [x] T055 [US3] Build the CrewAI compilation target (reference implementation) as built-in target that generates `pyproject.toml`, `main.py`, `crew.py`, `config/agents.yaml`, `config/tasks.yaml`, `tools/__init__.py` from AgentSpec IR per research.md mappings in `internal/compiler/targets/crewai.go`
- [x] T056 [P] [US3] Build the LangGraph compilation target as built-in target that generates `requirements.txt`, `graph.py`, `tools.py`, `main.py` with StateGraph, conditional edges, and ToolNode mappings per research.md in `internal/compiler/targets/langgraph.go`
- [x] T057 [P] [US3] Build the LlamaStack compilation target as built-in target that generates `requirements.txt`, `agent.py` using the high-level Agent API per research.md in `internal/compiler/targets/llamastack.go`
- [x] T058 [P] [US3] Build the LlamaIndex compilation target as built-in target that generates `requirements.txt`, `tools.py`, `agent.py`, `main.py` with ReActAgent and FunctionTool mappings per research.md in `internal/compiler/targets/llamaindex.go`
- [x] T059 [US3] Write integration test in `integration_tests/framework_compile_test.go` that compiles a standard `.ias` file to each of the 4 framework targets, verifies generated project structure, and checks for feature gap warnings

**Checkpoint**: At this point, developers can compile `.ias` files into code for 4 agentic frameworks. AgentSpec is a universal agent definition language.

---

## Phase 6: User Story 4 — Multi-Format Deployment Packaging (Priority: P4)

**Goal**: A developer packages compiled agents as Docker images, Kubernetes manifests, or standalone cross-platform binaries ready for deployment.

**Independent Test**: Compile an agent, run `agentspec package --format docker --tag test:1.0`, verify the Docker image builds and runs correctly.

### Implementation for User Story 4

- [x] T060 [US4] Implement the `agentspec package` CLI command per cli-contract.md with `--format` (docker, kubernetes, helm, binary), `--output`, `--tag`, `--registry`, `--push` flags in `cmd/agentspec/pkg.go`
- [x] T061 [US4] Refactor Docker adapter to accept a compiled agent binary as input (instead of IR), generate a minimal Dockerfile (`FROM alpine`), build the image with the binary + health check in `internal/adapters/docker/dockerfile.go`
- [x] T062 [US4] Kubernetes adapter generates deployment manifests (Deployment, Service, ConfigMap) referencing compiled agent container image with readiness/liveness probes in `internal/adapters/kubernetes/manifests.go`
- [x] T063 [P] [US4] Add Helm chart generation to the Kubernetes adapter: generate `Chart.yaml`, `values.yaml`, `templates/deployment.yaml`, `templates/service.yaml` in `internal/adapters/kubernetes/helm.go`
- [x] T064 [US4] Implement multi-agent pipeline packaging that bundles all agents in a pipeline into a single Docker Compose manifest with inter-agent networking in `internal/adapters/compose/compose.go`
- [x] T065 [US4] Write integration test for packaging: Dockerfile from binary, K8s manifests, Helm charts, Docker Compose in `integration_tests/packaging_test.go`

**Checkpoint**: At this point, compiled agents can be deployed anywhere — locally, in containers, or on Kubernetes.

---

## Phase 7: User Story 5 — Built-in Agent Frontend (Priority: P5)

**Goal**: Users interact with deployed agents through a built-in chat web interface with real-time streaming, activity trace, and structured input controls.

**Independent Test**: Start a compiled agent with `--ui`, open `http://localhost:8080/` in a browser, send a chat message, verify the response streams in with visible reasoning trace.

### Implementation for User Story 5

- [x] T066 [US5] Implement SSE streaming handler that wraps the existing agent stream endpoint, emitting `thought`, `tool_call`, `tool_result`, `token`, `validation`, and `done` events per agent-runtime-contract.md in `internal/frontend/sse.go`
- [x] T067 [US5] Implement the frontend HTTP handler that serves embedded static files via `go:embed`, handles SPA routing (fallback to `index.html`), and mounts alongside agent API routes in `internal/frontend/handler.go`
- [x] T068 [US5] Build the chat frontend `internal/frontend/web/index.html`: HTML structure with agent selector, chat message list, input field, collapsible activity panel, API key prompt modal, and inline CSS styling
- [x] T069 [US5] Build the frontend logic `internal/frontend/web/app.js`: EventSource SSE connection, message rendering with streaming tokens, activity panel updates, agent switching via `GET /v1/agents`, session management via sessionStorage, and API key injection into Authorization header
- [x] T070 [US5] Integrate dynamic input controls: read agent input schema from `GET /v1/agents/{name}`, render appropriate form fields (text, dropdown, file upload) based on schema type in `internal/frontend/web/app.js`
- [x] T071 [US5] Integrate frontend handler into the runtime server: mount on `/` and `/ui/*` when `--ui` flag is set (default true for compiled agents), skip auth for static assets in `internal/runtime/server.go`
- [x] T072 [US5] Add `--ui` and `--ui-port` flags to `agentspec dev` command to serve the frontend during development in `cmd/agentspec/dev.go`
- [x] T073 [US5] Write integration test in `integration_tests/frontend_test.go` that starts a compiled agent with frontend enabled, fetches `index.html`, verifies SSE streaming endpoint works, and tests API key auth flow
- [x] T074 [US5] Write integration test in `integration_tests/auth_test.go` that verifies: requests without API key return 401, requests with valid key succeed, `--no-auth` bypasses auth, `/healthz` is always accessible

**Checkpoint**: At this point, any compiled agent ships with a built-in web UI for chat, activity viewing, and structured input.

---

## Phase 8: User Story 6 — AgentSpec Ecosystem & Package Registry (Priority: P6)

**Goal**: Developers publish and share reusable `.ias` packages via a registry, and import them by name and version in their projects.

**Independent Test**: Publish a skills package to a local Git repo, import it by URL and version in another `.ias` file, compile successfully, verify the imported skill is available.

### Implementation for User Story 6

- [x] T075 [US6] Implement `agentpack.yaml` manifest parser and writer for package metadata (name, version, description, author, dependencies, exports) in `internal/registry/manifest.go`
- [x] T076 [US6] Implement Git-based package resolver that clones/fetches repos by URL, checks out version tags, reads manifests, and verifies checksums in `internal/registry/git.go`
- [x] T077 [US6] Implement local package cache with `~/.agentspec/cache/<host>/<path>/@v/<version>/` directory structure, cache lookup, and cache invalidation in `internal/registry/cache.go`
- [x] T078 [US6] Implement registry client with fallback chain: check local cache → check lock file → resolve from Git → verify checksum → cache locally in `internal/registry/client.go`
- [x] T079 [US6] Integrate registry client into import resolver: when an import has `kind: package`, delegate to registry client for resolution, then parse the resolved package files in `internal/imports/resolver.go`
- [x] T080 [US6] Implement version conflict detection: when two transitive dependencies require incompatible versions of the same package, report both dependency chains and suggest resolution options in `internal/imports/mvs.go`
- [x] T081 [US6] Implement the `agentspec publish` CLI command that reads `agentpack.yaml`, validates the package, creates a version tag, and pushes to the configured Git remote in `cmd/agentspec/publish.go`
- [x] T082 [US6] Implement the `agentspec install` CLI command that resolves a package, downloads it to the local cache, and updates the lock file in `cmd/agentspec/install.go`
- [x] T083 [US6] Write integration test in `integration_tests/registry_test.go` that creates a test Git repo, publishes a package, imports it in another `.ias` file, compiles, and verifies the imported definitions are available. Also tests version conflict detection.
- [x] T089 [US6] Design and stub package signing: add `signature`, `signer`, and `provenance` fields to Package entity, update `agentpack.yaml` manifest to include signature placeholder, add `--sign` flag to `agentspec publish` (prints "not yet implemented"), emit info message during unsigned package resolution in `internal/registry/manifest.go` and `internal/registry/client.go`

**Checkpoint**: At this point, the AgentSpec ecosystem is operational — developers can share packages via Git repositories and import them by name and version.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that span multiple user stories, documentation, and final validation

- [x] T084 [P] Create validated-agent example `examples/validated-agent/support-agent.ias` demonstrating the full quickstart.md workflow: config, validation, eval, imports, and control flow combined
- [x] T085 Update `cmd/agentspec/main.go` to register all new commands (compile, eval, package, publish, install) and bump version number
- [x] T086 [P] Extend `agentspec init` to scaffold new projects with IntentLang 3.0 template including config, validate, and eval blocks in `cmd/agentspec/init.go`
- [x] T087 Run end-to-end quickstart.md validation: execute all 8 quickstart steps from scratch and verify each produces expected output
- [x] T088 Determinism verification: compile the same `.ias` file twice on the same platform and assert byte-identical output binaries in `integration_tests/compile_test.go` (extend existing)
- [x] T090 [P] Performance benchmark test: measure compilation time (<10s for 500-line .ias per SC-002), agent startup time (<3s per SC-004), and package resolution time (<5s per SC-007) in `integration_tests/benchmark_test.go`
- [x] T091 Integrate compilation events into existing telemetry system: emit structured events with correlation IDs for `compile`, `eval`, `package`, and `publish` operations in `internal/telemetry/` (extend existing)
- [x] T092 [P] Implement configurable rate limiting middleware for compiled agent API endpoints with default limits overridable via `AGENTSPEC_RATE_LIMIT` env var in `internal/auth/ratelimit.go`
- [x] T093 Verify process adapter works with compiled agent binaries: ensure `agentspec dev` and direct binary execution work as local processes without external dependencies in `integration_tests/compile_test.go` (extend existing)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Stories (Phases 3-8)**: All depend on Foundational phase completion
  - US1 (Phase 3): No dependencies on other stories
  - US2 (Phase 4): No dependencies on other stories (imports/control flow are language features independent of compilation)
  - US3 (Phase 5): Depends on US1 (compilation framework must exist to add targets)
  - US4 (Phase 6): Depends on US1 (must have compiled artifacts to package)
  - US5 (Phase 7): Depends on US1 (must have compiled agents to serve frontend on)
  - US6 (Phase 8): Depends on US2 (import system must exist for registry to feed into)
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

```text
                    ┌──────────┐
                    │  Setup   │
                    │ Phase 1  │
                    └────┬─────┘
                         │
                    ┌────▼─────┐
                    │Foundation│
                    │ Phase 2  │
                    └────┬─────┘
                         │
              ┌──────────┴──────────┐
              │                     │
        ┌─────▼────┐          ┌────▼───┐
        │  US1:P1  │          │ US2:P2 │
        │ Compile  │          │ Import │
        │ Phase 3  │          │ Phase4 │
        └──┬──┬──┬─┘          └───┬────┘
           │  │  │                │
     ┌─────┘  │  └─────┐         │
     │        │        │         │
┌────▼───┐┌───▼──┐┌────▼────┐┌──▼──────┐
│ US3:P3 ││US4:P4││ US5:P5  ││ US6:P6  │
│Codegen ││Deploy││Frontend ││Registry │
│Phase 5 ││Phse 6││ Phase 7 ││ Phase 8 │
└────────┘└──────┘└─────────┘└─────────┘
```

### Within Each User Story

- Models/entities before services
- Core implementation before integration
- Integration before CLI commands
- CLI commands before examples
- Examples before integration tests

### Parallel Opportunities

**Within Phase 2 (Foundational)**:
```
T004 (AST) → T005 (lexer) → T006-T012 (parser, can parallelize across constructs)
T017 + T018 (expression engine, parallel with parser work after T004)
T019 + T020 (auth middleware, parallel with everything in Phase 2)
T013 (formatter) + T014 (validator) after T004-T012 complete
T015 (IR) + T016 (lower) after T004-T012 complete
```

**Within Phase 3 (US1)**:
```
T021 (config resolver) | T023+T024 (validation) — parallel, different packages
T031 (eval runner) | T032 (scoring) | T033 (report) — parallel, different files
T035 (example) — parallel with any implementation task
```

**Across User Stories (after Phase 2)**:
```
US1 (Phase 3) | US2 (Phase 4) — fully parallel, no shared code
US3 (Phase 5) — starts after US1 core (T029) completes
US4 (Phase 6) — starts after US1 core (T029) completes
US5 (Phase 7) — can start after US1 T025+T026 (runtime + template)
US6 (Phase 8) — starts after US2 T039-T042 (import system) completes
```

---

## Parallel Example: User Story 1

```bash
# Launch config + validation in parallel (different packages):
Task T021: "Implement runtime config resolver in internal/runtime/config.go"
Task T023: "Implement output validation rule executor in internal/validation/validator.go"

# Launch eval scoring methods in parallel (different files):
Task T032: "Implement scoring methods in internal/evaluation/scoring.go"
Task T033: "Implement eval report generation in internal/evaluation/report.go"
```

## Parallel Example: User Story 3

```bash
# Launch all 4 framework plugins in parallel (completely independent WASM modules):
Task T055: "Build CrewAI compilation plugin"
Task T056: "Build LangGraph compilation plugin"
Task T057: "Build LlamaStack compilation plugin"
Task T058: "Build LlamaIndex compilation plugin"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (~3 tasks)
2. Complete Phase 2: Foundational (~17 tasks) — CRITICAL, blocks all stories
3. Complete Phase 3: User Story 1 (~18 tasks)
4. **STOP and VALIDATE**: Compile hello-agent.ias → run binary → send request → verify response → run eval
5. Deploy/demo if ready — this is the MVP

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add US1 (Compile) → Test independently → **MVP!**
3. Add US2 (Imports + Control Flow) → Test independently → Language complete
4. Add US3 (Framework Codegen) → Test independently → Universal agent language
5. Add US4 (Deployment) → Test independently → Production-ready
6. Add US5 (Frontend) → Test independently → Developer experience
7. Add US6 (Registry) → Test independently → Ecosystem
8. Polish → End-to-end validation

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: US1 (Compile) — MVP path
   - Developer B: US2 (Imports + Control Flow) — language features
3. After US1 completes:
   - Developer A: US3 (Framework Codegen) or US4 (Deploy)
   - Developer C: US5 (Frontend)
4. After US2 completes:
   - Developer B: US6 (Registry)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Constitution requires integration tests as the primary quality gate — every phase includes integration tests
- All compilation output must be deterministic (same input → byte-identical output)
- All compiled agents default to API key auth (secure by default)
