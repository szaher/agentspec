# Tasks: AgentSpec Documentation Site

**Input**: Design documents from `/specs/005-docs-site/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/site-structure.md, quickstart.md

**Tests**: No test tasks generated — tests were not explicitly requested. Example validation is part of the build infrastructure (Phase 2 foundational).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize MkDocs project structure, custom Pygments lexer, CI workflow, and site foundation.

- [X] T001 Create documentation directory structure per plan.md (`docs/`, `docs/user-guide/`, `docs/developer-guide/`, `docs/examples/`, `docs/stylesheets/`, `docs-tools/`, `docs-tools/pygments_intentlang/`)
- [X] T002 Create MkDocs configuration with Material theme, navigation tabs, search, Mermaid support, and full nav structure in `mkdocs.yml`
- [X] T003 [P] Create pinned Python dependencies file in `docs-tools/requirements.txt` (mkdocs-material v9.x, pymdownx-extensions)
- [X] T004 [P] Create custom CSS overrides in `docs/stylesheets/extra.css`
- [X] T005 Create custom Pygments lexer `__init__.py` in `docs-tools/pygments_intentlang/__init__.py`
- [X] T006 Create custom Pygments lexer with IntentLang token definitions (~50 keywords, strings, comments, numbers, booleans, operators) in `docs-tools/pygments_intentlang/lexer.py`
- [X] T007 Create Pygments lexer setup.py with entry point registration in `docs-tools/pygments_intentlang/setup.py`
- [X] T008 Create site homepage with project overview, audience links (User Guide / Developer Guide), and feature highlights in `docs/index.md`
- [X] T009 Create GitHub Actions docs workflow (build Go binary, validate examples, install MkDocs, build site, deploy to GitHub Pages) in `.github/workflows/docs.yml` (note: `--strict` omitted due to upstream mkdocs-material MkDocs 2.0 warning)

**Checkpoint**: Site builds locally with `mkdocs serve` and CI workflow is defined.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create shared example `.ias` files and the Go integration test for example validation. These are used across multiple user stories.

**⚠️ CRITICAL**: No user story content work can begin until this phase is complete — examples and validation infrastructure must exist first.

- [X] T010 [P] Create validated example `docs/examples/basic-agent.ias` (simple agent with prompt, tool, deploy — used in getting started and language reference)
- [X] T011 [P] Create validated example `docs/examples/customer-support.ias` (multi-skill customer support agent — used in getting started and use cases)
- [X] T012 [P] Create validated example `docs/examples/code-review-pipeline.ias` (pipeline with multiple steps — used in pipeline reference and use cases)
- [X] T013 [P] Create validated example `docs/examples/rag-chatbot.ias` (RAG pattern with tool and prompt — used in getting started)
- [X] T014 [P] Create validated example `docs/examples/data-pipeline.ias` (data processing pipeline — used in use cases)
- [X] T015 [P] Create validated example `docs/examples/react-agent.ias` (ReAct strategy agent — used in use cases)
- [X] T016 [P] Create validated example `docs/examples/plan-execute-agent.ias` (plan-and-execute strategy — used in use cases)
- [X] T017 [P] Create validated example `docs/examples/reflexion-agent.ias` (reflexion strategy — used in use cases)
- [X] T018 [P] Create validated example `docs/examples/router-agent.ias` (router/triage pattern — used in use cases)
- [X] T019 [P] Create validated example `docs/examples/map-reduce-agent.ias` (map-reduce pattern — used in use cases)
- [X] T020 [P] Create validated example `docs/examples/delegation-agent.ias` (agent delegation pattern — used in use cases)
- [X] T021 Create Go integration test that extracts and validates fenced `.ias` code blocks from all docs Markdown files (supports `ias`, `ias fragment`, `ias invalid`, `ias novalidate` tags) in `integration_tests/docexample_test.go`

**Checkpoint**: All 11 example files pass `agentspec validate`. Integration test framework is ready.

---

## Phase 3: User Story 1 — IntentLang Language Reference (Priority: P1) 🎯 MVP

**Goal**: Complete IntentLang 2.0 language reference with all 13 resource types documented. Each page has syntax definition, attribute table, valid values, and annotated examples.

**Independent Test**: Every IntentLang 2.0 keyword in `spec/spec.md` has a reference page with syntax, attributes, and at least one working example. Run `mkdocs build` and verify all language pages render.

### Implementation for User Story 1

- [X] T022 [P] [US1] Create language reference overview page with keyword index and navigation in `docs/user-guide/language/index.md`
- [X] T023 [P] [US1] Create `agent` block reference page (syntax, attributes: model, strategy, max_turns, timeout, token_budget, temperature, stream, on_error, max_retries, fallback, delegates, skills, tools, prompts) in `docs/user-guide/language/agent.md`
- [X] T024 [P] [US1] Create `prompt` block reference page (syntax, attributes: content, variables with required/default, role) in `docs/user-guide/language/prompt.md`
- [X] T025 [P] [US1] Create `skill` block reference page (syntax, attributes: prompt, tools, description) in `docs/user-guide/language/skill.md`
- [X] T026 [P] [US1] Create `tool` block reference page covering all 4 variants (mcp, http, command, inline) with syntax, attributes, and examples for each variant in `docs/user-guide/language/tool.md`
- [X] T027 [P] [US1] Create `deploy` block reference page (syntax, attributes: target, replicas, cpu, memory, port, health_check, env, volumes, scaling) in `docs/user-guide/language/deploy.md`
- [X] T028 [P] [US1] Create `pipeline` block reference page (syntax, attributes: steps, step, depends_on, parallel, agent, input, output) in `docs/user-guide/language/pipeline.md`
- [X] T029 [P] [US1] Create `type` definition reference page (syntax, fields, required/default modifiers, nested types, enums) in `docs/user-guide/language/type.md`
- [X] T030 [P] [US1] Create `server` block reference page (syntax, attributes: transport, command, args, url, auth, port, exposes) in `docs/user-guide/language/server.md`
- [X] T031 [P] [US1] Create `client` block reference page (syntax, attributes: transport, url, command, auth, connects, headers) in `docs/user-guide/language/client.md`
- [X] T032 [P] [US1] Create `secret` block reference page (syntax, attributes: from, provider, key, path) in `docs/user-guide/language/secret.md`
- [X] T033 [P] [US1] Create `environment` block reference page (syntax, attributes: overlays, overrides for agent config) in `docs/user-guide/language/environment.md`
- [X] T034 [P] [US1] Create `policy` block reference page (syntax, attributes: max_tokens, rate_limit, allowed_tools, denied_tools, require_approval) in `docs/user-guide/language/policy.md`
- [X] T035 [P] [US1] Create `plugin` block reference page (syntax, attributes: source, hash, hooks — validator, transform, pre_deploy) in `docs/user-guide/language/plugin.md`
- [X] T036 [P] [US1] Create agent runtime configuration page (model, strategy, max_turns, timeout, token_budget, temperature, stream, on_error, max_retries, fallback) in `docs/user-guide/configuration/runtime.md`
- [X] T037 [P] [US1] Create prompt template variables page ({{variable}} syntax, variables block, required/default modifiers, examples) in `docs/user-guide/configuration/prompt-variables.md`
- [X] T038 [P] [US1] Create error handling page (on_error strategies: stop, retry, fallback, ignore; max_retries; fallback agent) in `docs/user-guide/configuration/error-handling.md`
- [X] T039 [P] [US1] Create agent delegation page (delegates block, delegate-to pattern, inter-agent communication) in `docs/user-guide/configuration/delegation.md`

**Checkpoint**: All 18 language reference and configuration pages exist with syntax, attributes, and examples. `mkdocs build` passes.

---

## Phase 4: User Story 2 — Getting Started and Tutorials (Priority: P1)

**Goal**: New user onboarding guide from installation through first working agent.

**Independent Test**: A new user can follow the getting-started guide end-to-end and successfully run `agentspec validate` on a `.ias` file they created.

### Implementation for User Story 2

- [X] T040 [P] [US2] Create getting started index page with installation instructions (Go binary, package managers), system requirements, and section overview in `docs/user-guide/getting-started/index.md`
- [X] T041 [P] [US2] Create quickstart tutorial page — walk user through creating a `.ias` file, running `agentspec validate`, `agentspec plan`, and `agentspec apply` with a basic agent in `docs/user-guide/getting-started/quickstart.md`
- [X] T042 [US2] Create core concepts page — explain agents, prompts, skills, tools, pipelines, deploy targets, desired-state model, and how they relate in `docs/user-guide/getting-started/concepts.md`
- [X] T043 [US2] Add troubleshooting section to quickstart page — document common errors (invalid syntax, missing package header, unknown keyword, validation failures), their error messages, explanations, and fixes in `docs/user-guide/getting-started/quickstart.md`

**Checkpoint**: Getting started section complete. A new user can follow quickstart, understand core concepts, and troubleshoot common errors.

---

## Phase 5: User Story 3 — Agentic Architecture Use Cases (Priority: P2)

**Goal**: Use-case catalog with 7+ agentic architectures, each with Mermaid diagram, complete `.ias` example, and deployment instructions. Architecture comparison table.

**Independent Test**: Each use-case page has a Mermaid diagram, a complete `.ias` example that passes validation, and deployment instructions. The overview page has a comparison table.

### Implementation for User Story 3

- [X] T044 [US3] Create use-case catalog overview page with architecture comparison table (strategy, complexity, latency, cost, recommended use case) and navigation to individual patterns in `docs/user-guide/use-cases/index.md`
- [X] T045 [P] [US3] Create ReAct agent use-case page (problem statement, Mermaid diagram, link to `docs/examples/react-agent.ias`, when to use, trade-offs, deployment config) in `docs/user-guide/use-cases/react.md`
- [X] T046 [P] [US3] Create Plan-and-Execute use-case page (problem statement, Mermaid diagram, link to `docs/examples/plan-execute-agent.ias`, when to use, trade-offs, deployment config) in `docs/user-guide/use-cases/plan-execute.md`
- [X] T047 [P] [US3] Create Reflexion use-case page (problem statement, Mermaid diagram, link to `docs/examples/reflexion-agent.ias`, when to use, trade-offs, deployment config) in `docs/user-guide/use-cases/reflexion.md`
- [X] T048 [P] [US3] Create Router/Triage use-case page (problem statement, Mermaid diagram, link to `docs/examples/router-agent.ias`, when to use, trade-offs, deployment config) in `docs/user-guide/use-cases/router.md`
- [X] T049 [P] [US3] Create Map-Reduce use-case page (problem statement, Mermaid diagram, link to `docs/examples/map-reduce-agent.ias`, when to use, trade-offs, deployment config) in `docs/user-guide/use-cases/map-reduce.md`
- [X] T050 [P] [US3] Create Multi-Agent Pipeline use-case page (problem statement, Mermaid diagram, link to `docs/examples/code-review-pipeline.ias`, when to use, trade-offs, deployment config) in `docs/user-guide/use-cases/pipeline.md`
- [X] T051 [P] [US3] Create Agent Delegation use-case page (problem statement, Mermaid diagram, link to `docs/examples/delegation-agent.ias`, when to use, trade-offs, deployment config) in `docs/user-guide/use-cases/delegation.md`

**Checkpoint**: All 8 use-case pages exist with diagrams, validated examples, and deployment configs. Comparison table on overview page.

---

## Phase 6: User Story 4 — Deployment Guide (Priority: P2)

**Goal**: Deployment guides for all 4 supported targets with production best practices.

**Independent Test**: Each deployment target has a complete guide with prerequisites, `deploy` block example, health check config, and verification steps.

### Implementation for User Story 4

- [X] T052 [P] [US4] Create deployment overview page with target comparison, decision guide, and deployment workflow in `docs/user-guide/deployment/index.md`
- [X] T053 [P] [US4] Create local process deployment guide (prerequisites, deploy block, running, health check, troubleshooting) in `docs/user-guide/deployment/process.md`
- [X] T054 [P] [US4] Create Docker deployment guide (Dockerfile generation, deploy block, building, running, health check, volumes) in `docs/user-guide/deployment/docker.md`
- [X] T055 [P] [US4] Create Docker Compose deployment guide (compose generation, deploy block, multi-agent setup, networking, health checks) in `docs/user-guide/deployment/compose.md`
- [X] T056 [P] [US4] Create Kubernetes deployment guide (manifest generation, deploy block, resource limits, autoscaling, ingress, secret management, monitoring) in `docs/user-guide/deployment/kubernetes.md`
- [X] T057 [US4] Create production best practices page (secret management, monitoring, scaling, CI/CD integration, logging, resource planning) in `docs/user-guide/deployment/best-practices.md`

**Checkpoint**: All 6 deployment pages exist. Each target has prerequisites, config reference, and working examples.

---

## Phase 7: User Story 5 — CLI Command Reference (Priority: P2)

**Goal**: Complete CLI reference with every `agentspec` subcommand documented.

**Independent Test**: Every CLI subcommand from `cmd/agentspec/*.go` has a corresponding page with usage syntax, flags, and at least one example.

### Implementation for User Story 5

- [X] T058 [US5] Create CLI reference overview page with command summary table and common usage patterns in `docs/user-guide/cli/index.md`
- [X] T059 [P] [US5] Create `validate` command reference (usage, flags: --strict/--format/--quiet, examples, success/error output) in `docs/user-guide/cli/validate.md`
- [X] T060 [P] [US5] Create `fmt` command reference (usage, flags: --check/--write/--diff, examples, output) in `docs/user-guide/cli/fmt.md`
- [X] T061 [P] [US5] Create `plan` command reference (usage, flags: --out/--format/--target, examples, output showing planned changes) in `docs/user-guide/cli/plan.md`
- [X] T062 [P] [US5] Create `apply` command reference (usage, flags: --auto-approve/--plan-file/--target, examples, output) in `docs/user-guide/cli/apply.md`
- [X] T063 [P] [US5] Create `run` command reference (usage, flags: --input/--stream/--session/--verbose, examples, output) in `docs/user-guide/cli/run.md`
- [X] T064 [P] [US5] Create `dev` command reference (usage, flags: --watch/--port/--hot-reload, examples, output) in `docs/user-guide/cli/dev.md`
- [X] T065 [P] [US5] Create `status` command reference (usage, flags: --format/--watch, examples, output) in `docs/user-guide/cli/status.md`
- [X] T066 [P] [US5] Create `logs` command reference (usage, flags: --follow/--since/--tail/--format, examples, output) in `docs/user-guide/cli/logs.md`
- [X] T067 [P] [US5] Create `destroy` command reference (usage, flags: --force/--target, examples, output) in `docs/user-guide/cli/destroy.md`
- [X] T068 [P] [US5] Create `init` command reference (usage, flags: --template/--name, examples, output showing project scaffolding) in `docs/user-guide/cli/init.md`
- [X] T069 [P] [US5] Create `migrate` command reference (usage, flags: --to-v2/--dry-run, examples, output) in `docs/user-guide/cli/migrate.md`
- [X] T070 [P] [US5] Create `export` command reference (usage, flags: --format/--output, examples, output) in `docs/user-guide/cli/export.md`
- [X] T071 [P] [US5] Create `diff` command reference (usage, flags: --format/--color, examples, output) in `docs/user-guide/cli/diff.md`
- [X] T072 [P] [US5] Create `sdk` command reference (usage, flags: --language/--output, examples, output) in `docs/user-guide/cli/sdk.md`
- [X] T073 [P] [US5] Create `version` command reference (usage, output format) in `docs/user-guide/cli/version.md`

**Checkpoint**: All 16 CLI reference pages exist. Each command has usage syntax, flags table, and examples.

---

## Phase 8: User Story 6 — SDK & HTTP API Documentation (Priority: P3)

**Goal**: SDK documentation for Python, TypeScript, and Go plus HTTP API reference for all runtime endpoints.

**Independent Test**: Each SDK has installation, client initialization, and code examples. HTTP API reference covers all endpoints with curl examples.

### Implementation for User Story 6

- [X] T074 [P] [US6] Create HTTP API overview page with authentication (X-API-Key, Bearer token), base URL, error response format, and endpoint index in `docs/user-guide/api/index.md`
- [X] T075 [P] [US6] Create Agent API endpoints page (/v1/agents, /v1/agents/{name}/invoke, /v1/agents/{name}/stream — request/response schemas, curl examples) in `docs/user-guide/api/agents.md`
- [X] T076 [P] [US6] Create Session API endpoints page (/v1/agents/{name}/sessions — create, continue, list — request/response schemas, curl examples) in `docs/user-guide/api/sessions.md`
- [X] T077 [P] [US6] Create Pipeline API endpoints page (/v1/pipelines/{name}/run — request/response schemas, curl examples) in `docs/user-guide/api/pipelines.md`
- [X] T078 [P] [US6] Create Health & Metrics endpoints page (/healthz, /v1/metrics — response schemas, monitoring integration) in `docs/user-guide/api/health-metrics.md`
- [X] T079 [P] [US6] Create Python SDK documentation (pip install, client init, sync invoke, async streaming, session management, error handling, type hints) in `docs/user-guide/sdks/python.md`
- [X] T080 [P] [US6] Create TypeScript SDK documentation (npm install, client init, invoke, streaming, session management, error handling, TypeScript types) in `docs/user-guide/sdks/typescript.md`
- [X] T081 [P] [US6] Create Go SDK documentation (go get, client init, invoke, streaming, session management, error handling, Go types) in `docs/user-guide/sdks/go.md`

**Checkpoint**: All 8 API/SDK pages exist. Each endpoint has curl examples. Each SDK has install + code examples.

---

## Phase 9: User Story 7 — Developer/Contributor Documentation (Priority: P3)

**Goal**: Complete developer documentation with architecture docs, build guide, extensibility guides, and internals reference.

**Independent Test**: A developer can clone the repo, follow the build guide, run tests, and find architecture documentation for each major internal package.

### Implementation for User Story 7

- [X] T082 [P] [US7] Create architecture overview page with system component diagram (Mermaid), data flow, and package index in `docs/developer-guide/architecture/index.md`
- [X] T083 [P] [US7] Create parser pipeline page (lexer, tokens, parser, AST construction, error recovery, source from `internal/parser/`) in `docs/developer-guide/architecture/parser.md`
- [X] T084 [P] [US7] Create IR and plan engine page (IR structure, plan generation, diff algorithm, state reconciliation, source from `internal/ir/` and `internal/planner/`) in `docs/developer-guide/architecture/ir.md`
- [X] T085 [P] [US7] Create runtime and agentic loop page (agent lifecycle, tool dispatch, streaming, session management, source from `internal/runtime/`) in `docs/developer-guide/architecture/runtime.md`
- [X] T086 [P] [US7] Create adapter system page (adapter interface, built-in adapters: process, docker, compose, kubernetes, adapter lifecycle, source from `internal/adapters/`) in `docs/developer-guide/architecture/adapters.md`
- [X] T087 [P] [US7] Create plugin host page (WASM sandbox, plugin contract, hook types: validator, transform, pre_deploy, source from `internal/plugins/`) in `docs/developer-guide/architecture/plugins.md`
- [X] T088 [P] [US7] Create build-from-source page (prerequisites, clone, build, binary location, dev setup) in `docs/developer-guide/contributing/index.md`
- [X] T089 [P] [US7] Create testing guide page (running unit tests, integration tests, test structure, testdata conventions) in `docs/developer-guide/contributing/testing.md`
- [X] T090 [P] [US7] Create code style page (Go conventions, linting config, naming, error handling patterns) in `docs/developer-guide/contributing/code-style.md`
- [X] T091 [P] [US7] Create PR guidelines page (branch naming, commit messages, review process, CI checks) in `docs/developer-guide/contributing/pull-requests.md`
- [X] T092 [P] [US7] Create custom adapter guide (adapter interface definition, step-by-step implementation, registration, testing, working example) in `docs/developer-guide/extending/adapters.md`
- [X] T093 [P] [US7] Create WASM plugin guide (plugin contract, hook interfaces, building with TinyGo/Rust, testing, deployment, working example) in `docs/developer-guide/extending/plugins.md`
- [X] T094 [P] [US7] Create state management internals page (`.agentspec.state.json` format, state transitions, reconciliation, source from `internal/state/`) in `docs/developer-guide/internals/state.md`
- [X] T095 [P] [US7] Create secret resolution internals page (secret providers, resolution order, env/vault/file sources, source from `internal/secrets/`) in `docs/developer-guide/internals/secrets.md`
- [X] T096 [P] [US7] Create telemetry/observability internals page (logging, metrics, tracing, OpenTelemetry integration, source from `internal/telemetry/`) in `docs/developer-guide/internals/telemetry.md`
- [X] T097 [P] [US7] Create SDK code generation internals page (template system, language targets, generated code structure, source from `internal/sdkgen/`) in `docs/developer-guide/internals/sdk-generation.md`

**Checkpoint**: All 16 developer guide pages exist. A contributor can find architecture docs and build/test/contribute.

---

## Phase 10: User Story 8 — Site Navigation, Search, and Publishing (Priority: P3)

**Goal**: Migration guide, changelog, sitemap, SEO metadata, and final navigation/search verification.

**Independent Test**: Site builds and deploys to GitHub Pages. Navigation separates user/developer docs. Search returns relevant results for IntentLang keywords.

### Implementation for User Story 8

- [X] T098 [P] [US8] Create IntentLang 1.0 to 2.0 migration guide (deprecated constructs, renamed keywords, `agentspec migrate --to-v2` usage, before/after examples) in `docs/user-guide/migration.md`
- [X] T099 [P] [US8] Create changelog page (sourced from existing CHANGELOG.md — v0.1.0, v0.2.0, v0.3.0 release notes) in `docs/user-guide/changelog.md`
- [X] T100 [US8] Add sitemap plugin configuration and SEO meta descriptions to `mkdocs.yml`
- [X] T101 [US8] Update `mkdocs.yml` nav section to include all pages from US1–US7 and verify complete navigation tree

**Checkpoint**: Full navigation tree defined. Site builds with `mkdocs build`. Search and sitemap functional.

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: Validation pass, cross-linking, consistency review, and final build verification.

- [X] T102 Run `mkdocs build` and fix all warnings (broken links, missing pages, invalid references) (note: `--strict` omitted due to upstream mkdocs-material MkDocs 2.0 warning)
- [X] T103 Run `go test ./integration_tests/ -run TestDocExamples -v` and fix all example validation failures
- [X] T104 [P] Add cross-links between related pages (language reference ↔ use cases, CLI ↔ API, getting started → language reference, deployment → use cases)
- [X] T105 [P] Review all pages for consistency in tone, formatting, heading structure, and depth
- [X] T106 Verify search returns relevant results for key terms: "agent", "pipeline", "deploy", "tool", "prompt", "kubernetes", "execution command" (deprecated)
- [X] T107 Verify GitHub Pages deployment workflow runs end-to-end (build binary → validate examples → build site → deploy)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user story content
- **US1 Language Reference (Phase 3)**: Depends on Phase 2 — P1 priority, start first
- **US2 Getting Started (Phase 4)**: Depends on Phase 2 — P1 priority, can run in parallel with US1
- **US3 Use Cases (Phase 5)**: Depends on Phase 2 — P2 priority, benefits from US1 for cross-linking
- **US4 Deployment (Phase 6)**: Depends on Phase 2 — P2 priority, can run in parallel with US3
- **US5 CLI Reference (Phase 7)**: Depends on Phase 2 — P2 priority, can run in parallel with US3/US4
- **US6 SDK & API (Phase 8)**: Depends on Phase 2 — P3 priority, can run in parallel with US3–US5
- **US7 Developer Docs (Phase 9)**: Depends on Phase 2 — P3 priority, can run in parallel with US3–US6
- **US8 Site Nav/Search/Publishing (Phase 10)**: Depends on US1–US7 (all pages must exist for nav tree)
- **Polish (Phase 11)**: Depends on all phases (all content must be written)

### User Story Dependencies

- **US1 (P1)**: Can start after Phase 2 — No dependencies on other stories
- **US2 (P1)**: Can start after Phase 2 — No dependencies on other stories (references examples from Phase 2)
- **US3 (P2)**: Can start after Phase 2 — Benefits from US1 cross-links but not blocked
- **US4 (P2)**: Can start after Phase 2 — Independent of other stories
- **US5 (P2)**: Can start after Phase 2 — Independent of other stories
- **US6 (P3)**: Can start after Phase 2 — Independent of other stories
- **US7 (P3)**: Can start after Phase 2 — Independent of other stories
- **US8 (P3)**: Requires US1–US7 content for complete navigation tree

### Within Each User Story

- Pages marked [P] within a story can be written in parallel
- Overview/index pages should be written first within each section (they define structure)
- Content pages reference shared examples from `docs/examples/` (created in Phase 2)

### Parallel Opportunities

- **Phase 1**: T003, T004 can run in parallel with each other (and after T001/T002)
- **Phase 2**: All example files (T010–T020) can be created in parallel
- **Phase 3 (US1)**: All 18 language/config pages (T022–T039) can be written in parallel
- **Phase 4 (US2)**: T040, T041 can run in parallel; T042, T043 sequential
- **Phase 5 (US3)**: All 7 use-case pages (T045–T051) can run in parallel; T044 (overview) first
- **Phase 6 (US4)**: All target pages (T052–T056) can run in parallel; T057 after
- **Phase 7 (US5)**: All 15 command pages (T059–T073) can run in parallel; T058 (overview) first
- **Phase 8 (US6)**: All 8 API/SDK pages (T074–T081) can run in parallel
- **Phase 9 (US7)**: All 16 developer pages (T082–T097) can run in parallel
- **Phase 10 (US8)**: T098, T099 can run in parallel; T100, T101 sequential
- **US1–US7 can all run in parallel** once Phase 2 is complete

---

## Parallel Example: User Story 1 (Language Reference)

```bash
# Launch all language reference pages in parallel:
Task: "Create agent block reference in docs/user-guide/language/agent.md"
Task: "Create prompt block reference in docs/user-guide/language/prompt.md"
Task: "Create skill block reference in docs/user-guide/language/skill.md"
Task: "Create tool block reference in docs/user-guide/language/tool.md"
Task: "Create deploy block reference in docs/user-guide/language/deploy.md"
Task: "Create pipeline block reference in docs/user-guide/language/pipeline.md"
# ... (all 13 resource type pages + 4 configuration pages)
```

## Parallel Example: User Story 3 (Use Cases)

```bash
# After overview page (T044), launch all use-case pages in parallel:
Task: "Create ReAct agent use-case in docs/user-guide/use-cases/react.md"
Task: "Create Plan-and-Execute use-case in docs/user-guide/use-cases/plan-execute.md"
Task: "Create Reflexion use-case in docs/user-guide/use-cases/reflexion.md"
Task: "Create Router/Triage use-case in docs/user-guide/use-cases/router.md"
Task: "Create Map-Reduce use-case in docs/user-guide/use-cases/map-reduce.md"
Task: "Create Multi-Agent Pipeline use-case in docs/user-guide/use-cases/pipeline.md"
Task: "Create Agent Delegation use-case in docs/user-guide/use-cases/delegation.md"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2 Only)

1. Complete Phase 1: Setup (MkDocs project, Pygments lexer, CI workflow)
2. Complete Phase 2: Foundational (example files, validation test)
3. Complete Phase 3: User Story 1 — Language Reference (18 pages)
4. Complete Phase 4: User Story 2 — Getting Started (4 pages)
5. **STOP and VALIDATE**: Run `mkdocs build`, verify examples, test search
6. Deploy MVP — Users can learn IntentLang and get started

### Incremental Delivery

1. Setup + Foundational → Infrastructure ready
2. US1 + US2 → Language Reference + Getting Started → Deploy MVP (22 content pages)
3. US3 + US4 + US5 → Use Cases + Deployment + CLI → Deploy (30 more pages)
4. US6 + US7 → SDKs/API + Developer Docs → Deploy (24 more pages)
5. US8 + Polish → Navigation, search, cross-linking → Final deployment
6. Each increment adds user value without breaking previous content

### Parallel Team Strategy

With multiple contributors:

1. Team completes Setup + Foundational together
2. Once Phase 2 is done:
   - Contributor A: US1 (Language Reference — 18 pages)
   - Contributor B: US2 (Getting Started — 4 pages) then US3 (Use Cases — 8 pages)
   - Contributor C: US5 (CLI Reference — 16 pages)
   - Contributor D: US4 (Deployment — 6 pages) then US7 (Developer Docs — 16 pages)
3. After all content: US8 (navigation) + Polish

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Content source references: `spec/spec.md` (language spec), `README.md` (quick start), `ARCHITECTURE.md` (system design), `CHANGELOG.md` (releases), `examples/` (10 working examples), `cmd/agentspec/*.go` (CLI flags)
- Code examples in pages use fence tags: `ias` (complete), `ias fragment` (pedagogical), `ias invalid` (error docs), `ias novalidate` (conceptual)
- Commit after each task or logical group
- Stop at any checkpoint to validate independently
