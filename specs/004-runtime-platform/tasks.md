# Tasks: AgentSpec Runtime Platform

**Input**: Design documents from `/specs/004-runtime-platform/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/runtime-api.md

**Tests**: Integration tests included per constitution mandate (integration tests are the primary quality gate).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add new dependencies and create package directory structure

- [x] T001 Add new dependencies to go.mod: `github.com/anthropics/anthropic-sdk-go` v1.26.0, `github.com/modelcontextprotocol/go-sdk/mcp` v1.3.1
- [x] T002 Create new package directories: `internal/runtime/`, `internal/loop/`, `internal/llm/`, `internal/mcp/`, `internal/tools/`, `internal/memory/`, `internal/session/`, `internal/secrets/`, `internal/telemetry/`, `internal/pipeline/`, `internal/migrate/`, `internal/adapters/process/`, `internal/adapters/docker/`, `internal/adapters/kubernetes/`
- [x] T003 [P] Add test fixture directory `integration_tests/testdata/v2/` with sample IntentLang 2.0 files for parser testing
- [x] T004 [P] Update .gitignore to include runtime artifacts (PID files, runtime config JSON, container build context)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core IntentLang 2.0 language changes required before runtime implementation. Parser, AST, IR, formatter, and validator must support new constructs.

**CRITICAL**: No user story work can begin until this phase is complete

### Lexer & Token Changes

- [x] T005 Add new token types for IntentLang 2.0 keywords in internal/parser/token.go: `tool`, `deploy`, `target`, `pipeline`, `step`, `delegate`, `type`, `strategy`, `max_turns`, `timeout`, `token_budget`, `temperature`, `stream`, `parallel`, `depends_on`, `from`, `when`, `to`, `health`, `autoscale`, `resources`, `memory`, `on_error`, `fallback`, `variables`, `default`, `required`, `enum`, `list`
- [x] T006 Add lexer rules for new tokens in internal/parser/lexer.go, including recognition of `{{variable}}` template syntax in string literals

### Parser Rules

- [x] T007 Add parser rules for `tool` block variants (mcp, http, command, inline) inside `skill` blocks in internal/parser/parser.go
- [x] T008 Add parser rules for agent runtime config attributes (`strategy`, `max_turns`, `timeout`, `token_budget`, `temperature`, `stream`, `on_error`, `max_retries`, `fallback`, `memory`) in internal/parser/parser.go
- [x] T009 Add parser rules for `deploy "name" target "type" { ... }` block replacing `binding` in internal/parser/parser.go

### AST Nodes

- [x] T010 Add new AST node types in internal/ast/ast.go: `ToolConfig` (with MCP/HTTP/Command/Inline variants), `DeployTarget` (replacing Binding), `AgentRuntimeConfig`, `MemoryConfig`, `HealthConfig`, `AutoscaleConfig`, `ResourceLimits`

### IR Extensions

- [x] T011 Add IR resource types for `DeployTarget` and `ToolConfig` in internal/ir/ir.go, extending the existing Resource model
- [x] T012 Extend AST→IR lowering for `tool`, `deploy`, agent runtime config in internal/ir/lower.go, computing FQNs for new resource types

### Validation Extensions

- [x] T013 [P] Extend structural validator for new constructs in internal/validate/structural.go: validate tool block has exactly one variant, deploy target type is valid, agent runtime config values are in range
- [x] T014 [P] Extend semantic validator for new constructs in internal/validate/semantic.go: validate MCP tool references resolve to declared servers, deploy target names are unique, fallback agent references exist

### Formatter Extension

- [x] T015 Extend formatter for canonical formatting of `tool`, `deploy`, and agent runtime config blocks in internal/formatter/formatter.go

### Deprecation Removal

- [x] T016 [P] Delete .az deprecation code: remove internal/cli/deprecation.go entirely
- [x] T017 [P] Remove .az glob from resolveAZFiles() in cmd/agentspec/fmt.go, rename function to resolveFiles(), reject .az files with migration message
- [x] T018 [P] Remove stub Plan() methods from internal/adapters/local/local.go and internal/adapters/compose/compose.go

### Integration Tests

- [x] T019 Add integration tests for IntentLang 2.0 parsing of `tool`, `deploy`, agent runtime config in integration_tests/validate_test.go using testdata/v2/ fixtures
- [x] T020 [P] Add golden file tests for formatter output of new constructs in integration_tests/golden_test.go

**Checkpoint**: IntentLang 2.0 core constructs parse, validate, lower to IR, and format correctly. All existing tests pass. `.az` files rejected.

---

## Phase 3: User Story 1 — Run an Agent Locally (Priority: P1) + User Story 2 — Executable Skills (Priority: P1) MVP

**Goal**: `agentspec apply` starts a local runtime process serving agents via HTTP. The ReAct agentic loop calls Claude and dispatches tools via MCP, HTTP, command, and inline execution. Sessions maintain conversation context.

**Independent Test**: Write a minimal `.ias` file with one agent, one prompt, and one MCP-backed skill. Run `agentspec apply`. Send an HTTP request. Verify AI-generated response with tool use.

### LLM Client

- [x] T021 [P] [US1] Implement LLM client interface (`Chat`, `ChatStream` methods, `ChatRequest`/`ChatResponse` types) in internal/llm/client.go
- [x] T022 [US1] Implement Anthropic Claude client wrapping `anthropic-sdk-go` (Messages API, streaming, tool use, prompt caching) in internal/llm/anthropic.go
- [x] T023 [P] [US1] Implement mock LLM client with configurable responses and tool call sequences for testing in internal/llm/mock.go
- [x] T024 [P] [US1] Implement token usage tracking and budget enforcement in internal/llm/tokens.go

### MCP Client

- [x] T025 [P] [US2] Implement MCP client wrapper around official SDK (stdio transport, tool call, tool listing) in internal/mcp/client.go
- [x] T026 [US2] Implement MCP connection pool (get-or-create connections by server name, auto-start stdio servers) in internal/mcp/pool.go
- [x] T027 [P] [US2] Implement MCP tool discovery (list tools from connected servers, map to LLM tool definitions) in internal/mcp/discovery.go

### Tool Execution

- [x] T028 [US2] Implement tool execution registry (register MCP/HTTP/command/inline executors, dispatch by tool type, concurrent execution of multiple calls) in internal/tools/registry.go
- [x] T029 [P] [US2] Implement HTTP tool executor (configurable method, URL, headers, body template, response parsing) in internal/tools/http.go
- [x] T030 [P] [US2] Implement command tool executor (subprocess via os/exec with context timeout, stdin/stdout capture, env injection) in internal/tools/command.go
- [x] T031 [P] [US2] Implement inline code executor (sandboxed subprocess with resource limits, env/secret pass-through, timeout enforcement) in internal/tools/inline.go

### Agentic Loop

- [x] T032 [US1] Implement strategy interface (`Strategy.Execute`, `Invocation`, `Response`, `ToolCallRecord`, `TokenUsage` types) in internal/loop/loop.go
- [x] T033 [US1] Implement ReAct strategy (reason→act→observe loop, concurrent tool calls, max turns enforcement, token budget check, streaming support) in internal/loop/react.go

### Conversation Memory

- [x] T034 [P] [US1] Implement conversation memory interface (`Load`, `Save`, `Clear` methods) in internal/memory/memory.go
- [x] T035 [US1] Implement sliding window memory strategy (fixed message count, FIFO eviction) in internal/memory/sliding.go
- [x] T035a [P] [US1] Implement summarization memory strategy (LLM-based conversation summary when message count exceeds threshold) in internal/memory/summary.go

### Session Management

- [x] T036 [P] [US1] Implement session store interface (`Create`, `Get`, `Delete`, `List` methods) and session types in internal/session/store.go
- [x] T037 [US1] Implement in-memory session store with session expiry in internal/session/memory_store.go
- [x] T038 [US1] Implement session lifecycle (create, send message, close, expire) in internal/session/session.go

### Secret Resolution

- [x] T039 [P] [US1] Implement secret resolver interface (`Resolve(ref) → value` method) in internal/secrets/resolver.go
- [x] T040 [US1] Implement environment variable secret resolver (reads `env(VAR)` references from OS environment) in internal/secrets/env.go
- [x] T040a [US1] Implement secret redaction filter (wrap slog handler to scrub resolved secret values from log output, mask secrets in state file serialization) in internal/secrets/redact.go

### Runtime Server

- [x] T041 [US1] Implement IR→runtime config conversion (parse IR resources into runtime config struct with agents, tools, MCP servers, secrets) in internal/runtime/config.go
- [x] T042 [US1] Implement runtime HTTP server per contracts/runtime-api.md: routes for `/healthz`, `/v1/agents`, `/v1/agents/{name}/invoke`, `/v1/agents/{name}/stream` (SSE), `/v1/agents/{name}/sessions`, API key auth middleware in internal/runtime/server.go
- [x] T043 [US1] Implement runtime lifecycle (start all MCP servers, initialize tool registry, start HTTP server, health check, graceful shutdown) in internal/runtime/runtime.go

### Local Process Adapter

- [x] T044 [US1] Implement local process adapter (start runtime subprocess, pass config, wait for health check, record PID/port in state) in internal/adapters/process/process.go
- [x] T045 [P] [US1] Implement health check polling (HTTP GET /healthz with retry, configurable interval/timeout) in internal/adapters/process/health.go
- [x] T046 [US1] Implement state file locking (flock-based lock on state file, fail with error if locked) in internal/state/local.go

### CLI Commands

- [x] T047 [US1] Update `agentspec apply` to dispatch to real adapter Apply(), resolve secrets, start runtime process in cmd/agentspec/apply.go
- [x] T048 [P] [US1] Implement `agentspec run <agent> --input "message"` command (one-shot invocation: parse, validate, start runtime, invoke, print response, shutdown) in cmd/agentspec/run.go
- [x] T049 [P] [US1] Implement `agentspec dev` command (file watcher on .ias files, graceful restart on change, colored log output) in cmd/agentspec/dev.go

### Plugin Implementation

- [x] T050 [P] [US1] Implement WASM plugin hook execution (call pre/post hook exports with serialized context) in internal/plugins/hooks.go
- [x] T051 [P] [US1] Implement WASM plugin resource validation (call validator exports, collect errors) in internal/plugins/validate.go
- [x] T052 [P] [US1] Implement WASM plugin transforms (call transform exports, apply mutations to IR) in internal/plugins/transform.go
- [x] T052a [US1] Create plugin demo: update existing monitor plugin to use implemented WASM hooks (pre-invoke, post-invoke), verify hook execution with integration test in integration_tests/plugin_test.go

### Integration Tests

- [x] T053 [US1] Add integration test for runtime lifecycle (start→health→invoke→stop) using mock LLM in integration_tests/runtime_test.go
- [x] T054 [US1] Add integration test for ReAct agentic loop (multi-turn with tool calls) using mock LLM in integration_tests/loop_test.go
- [x] T054a [US1] Add integration test for idempotent apply (apply twice with no changes → verify process PID unchanged, no restart) in integration_tests/idempotent_test.go
- [x] T055 [P] [US2] Add integration test for tool execution (MCP, HTTP, command tool types) in integration_tests/tools_test.go
- [x] T055a [P] [US1] Add integration test verifying secrets are redacted from logs and state files in integration_tests/secrets_test.go
- [x] T056 [US1] Update examples/basic-agent/basic-agent.ias to IntentLang 2.0 with `tool` and `deploy` blocks

**Checkpoint**: `agentspec apply` starts a local process. Agents respond to HTTP requests with real Claude responses. MCP tools execute. Sessions maintain context. `agentspec run` and `agentspec dev` work. All existing tests pass.

---

## Phase 4: User Story 3 — IntentLang 2.0 Language Constructs (Priority: P2)

**Goal**: Complete IntentLang 2.0 with prompt variables, type definitions, pipeline blocks, delegation, and the `migrate --to-v2` command.

**Independent Test**: Write `.ias` files using all 2.0 constructs, run `agentspec validate`, verify acceptance. Run `agentspec migrate --to-v2` on a 1.0 file and verify correct rewrite.

### Parser Extensions

- [x] T057 [US3] Add parser rules for prompt `variables` block with type, required, and default declarations in internal/parser/parser.go
- [x] T058 [US3] Add parser rules for `type` definitions with fields, enum, list, and nesting in internal/parser/parser.go
- [x] T059 [US3] Add parser rules for `pipeline` block with `step`, `input`, `output`, `parallel`, `depends_on` in internal/parser/parser.go
- [x] T060 [US3] Add parser rules for `delegate to agent "name" when "condition"` inside agent blocks in internal/parser/parser.go

### AST & IR Extensions

- [x] T061 [US3] Add AST nodes for `Variable`, `TypeDef`, `TypeField`, `Pipeline`, `PipelineStep`, `StepInput`, `Delegate` in internal/ast/ast.go
- [x] T062 [US3] Add IR resource types for Pipeline, Type and extend Agent IR with delegation rules in internal/ir/ir.go
- [x] T063 [US3] Extend AST→IR lowering for pipeline steps, type definitions, delegation in internal/ir/lower.go

### Validation & Formatting

- [x] T064 [P] [US3] Extend validators for pipeline step dependencies (no circular deps), type field references, prompt variable usage in internal/validate/semantic.go
- [x] T065 [P] [US3] Extend formatter for pipeline, type, delegate, and prompt variables constructs in internal/formatter/formatter.go

### Migration Command

- [x] T066 [US3] Implement IntentLang 1.0→2.0 AST rewriter (replace `execution command` with `tool command`, replace `binding` with `deploy`, set `lang "2.0"`) in internal/migrate/v2.go
- [x] T067 [US3] Update migrate command to support `--to-v2` flag, call AST rewriter and write output in cmd/agentspec/migrate.go
- [x] T068 [US3] Update parser to reject `lang "1.0"` files with actionable error directing to `agentspec migrate --to-v2` in internal/parser/parser.go

### Examples & Tests

- [x] T069 [US3] Update all examples to IntentLang 2.0 syntax in examples/ (10 example files)
- [x] T070 [US3] Add integration tests for prompt variables, type definitions, pipeline parsing, delegation, and migrate command in integration_tests/validate_test.go
- [x] T071 [P] [US3] Add golden file fixtures for all new 2.0 constructs in integration_tests/testdata/

**Checkpoint**: All IntentLang 2.0 constructs parse, validate, lower, and format. `agentspec migrate --to-v2` converts 1.0 files. All examples use 2.0 syntax. Parser rejects 1.0 files with migration guidance.

---

## Phase 5: User Story 4 — Multi-Target Deployment (Priority: P3)

**Goal**: Deploy agents to Docker, Docker Compose, and Kubernetes. Operational commands (`status`, `logs`, `destroy`) work across targets.

**Independent Test**: Write an `.ias` file with `deploy "staging" target "docker"`, run `agentspec apply --target staging`, verify container starts and agent responds.

### Adapter Interface Extension

- [x] T072 [US4] Extend adapter interface with `Status()`, `Logs()`, and `Destroy()` methods in internal/adapters/adapter.go
- [x] T073 [US4] Update local process adapter to implement new interface methods in internal/adapters/process/process.go

### Docker Adapter

- [x] T074 [P] [US4] Implement Dockerfile generation (distroless base image, copy runtime binary + config, expose port, health check) in internal/adapters/docker/dockerfile.go
- [x] T075 [US4] Implement Docker adapter (build image via moby/moby client, create+start container, port mapping, health check, record container ID in state) in internal/adapters/docker/docker.go

### Docker Compose Adapter

- [x] T076 [US4] Rewrite Docker Compose adapter with real compose file generation (services from agents, health checks, networking, volumes, env vars) and stack management via `docker compose up/down` in internal/adapters/compose/compose.go

### Kubernetes Adapter

- [x] T077 [P] [US4] Implement K8s manifest generation (Deployment, Service, ConfigMap, Secret, HPA, Ingress) from deploy block config using client-go types in internal/adapters/kubernetes/manifests.go
- [x] T078 [US4] Implement Kubernetes adapter (apply manifests via Server-Side Apply, rollout status polling, record resource UIDs in state) in internal/adapters/kubernetes/kubernetes.go

### CLI Commands

- [x] T079 [P] [US4] Implement `agentspec status` command (query adapter Status() for all deployed resources, display health/endpoint/utilization table) in cmd/agentspec/status.go
- [x] T080 [P] [US4] Implement `agentspec logs` command (query adapter Logs(), stream to stdout with `--follow` support) in cmd/agentspec/logs.go
- [x] T081 [P] [US4] Implement `agentspec destroy` command (prompt confirmation, call adapter Destroy(), update state) in cmd/agentspec/destroy.go

### Go Module Update

- [x] T082 [US4] Add deployment dependencies to go.mod: adapters use CLI-based approach (docker/kubectl) instead of Go client libraries for portability

### Integration Tests

- [x] T083 [US4] Add integration test for Docker adapter (build+start+invoke+destroy cycle) in integration_tests/docker_test.go (requires Docker daemon)
- [x] T084 [P] [US4] Add integration test for Kubernetes adapter (apply+status+destroy cycle) in integration_tests/kubernetes_test.go (requires kind/minikube)

**Checkpoint**: Same `.ias` file deploys to local process, Docker, and Kubernetes. `agentspec status`, `logs`, and `destroy` work across all targets.

---

## Phase 6: User Story 5 — Multi-Agent Coordination (Priority: P4)

**Goal**: Pipeline executor runs multi-agent workflows with parallel steps, dependency ordering, and fail-fast cancellation. Agent delegation routes conversations.

**Independent Test**: Define a pipeline with 3 agents (2 parallel, 1 dependent), invoke the pipeline, verify correct execution order and data flow.

### Pipeline Executor

- [x] T085 [P] [US5] Implement DAG builder for pipeline steps (parse dependencies, detect cycles, compute topological order) in internal/pipeline/dag.go
- [x] T086 [US5] Implement pipeline executor (execute steps respecting DAG order, run parallel steps concurrently via errgroup, fail-fast cancellation on any step failure, collect results) in internal/pipeline/executor.go

### Agent Delegation

- [x] T087 [US5] Implement agent delegation (LLM-evaluated condition matching, conversation handoff to delegate agent, response routing) in internal/loop/delegation.go

### Additional Strategies

- [x] T088 [P] [US5] Implement Plan-and-Execute strategy (LLM creates plan, execute steps sequentially, re-plan on failure) in internal/loop/plan_execute.go
- [x] T089 [P] [US5] Implement Reflexion strategy (execute, self-critique, iterate until satisfactory) in internal/loop/reflexion.go
- [x] T089a [P] [US5] Implement Router strategy (classify input, dispatch to specialized sub-agent, return result) in internal/loop/router.go
- [x] T089b [P] [US5] Implement Map-Reduce strategy (split input into chunks, fan-out to parallel agent calls, merge results) in internal/loop/map_reduce.go

### Runtime Integration

- [x] T090 [US5] Wire pipeline execution endpoint `/v1/pipelines/{name}/run` to runtime HTTP server per contracts/runtime-api.md in internal/runtime/server.go

### Integration Tests

- [x] T091 [US5] Add integration test for pipeline execution (parallel steps, dependency ordering, fail-fast, data passing) using mock LLM in integration_tests/pipeline_exec_test.go
- [x] T092 [P] [US5] Add integration test for agent delegation using mock LLM in integration_tests/delegation_test.go

**Checkpoint**: Pipelines execute with correct ordering, parallelism, and fail-fast. Delegation routes conversations. All additional strategies work: Plan-and-Execute, Reflexion, Router, and Map-Reduce.

---

## Phase 7: User Story 6 — Developer SDKs and Programmatic Access (Priority: P5)

**Goal**: Python, TypeScript, and Go SDK clients can invoke agents, stream responses, and manage sessions.

**Independent Test**: Install Python SDK, point at a running agent, verify `client.invoke()` returns a response and `client.stream()` yields chunks.

### Python SDK

- [x] T093 [P] [US6] Implement Python SDK client (`AgentSpecClient` with `invoke`, `stream`, `session` methods, typed response objects) in sdk/python/agentspec/client.py
- [x] T094 [P] [US6] Implement Python SDK streaming (async generator yielding SSE events) in sdk/python/agentspec/streaming.py
- [x] T095 [P] [US6] Create Python SDK package config (pyproject.toml, __init__.py, type stubs) in sdk/python/

### TypeScript SDK

- [x] T096 [P] [US6] Implement TypeScript SDK client (AgentSpecClient class with invoke, stream, session methods) in sdk/typescript/src/client.ts
- [x] T097 [P] [US6] Implement TypeScript SDK streaming (EventSource-based SSE consumer) in sdk/typescript/src/streaming.ts
- [x] T098 [P] [US6] Create TypeScript SDK package config (package.json, tsconfig.json, index.ts) in sdk/typescript/

### Go SDK

- [x] T099 [P] [US6] Implement Go SDK client (Client struct with Invoke, Stream, Session methods) in sdk/go/agentspec/client.go
- [x] T100 [P] [US6] Create Go SDK module (go.mod, exported types) in sdk/go/

### SDK Generator

- [x] T101 [US6] Rewrite SDK generator to produce typed clients from runtime API contract and IR skill schemas in internal/sdk/generator/generator.go

### Project Templates

- [x] T102 [P] [US6] Implement `agentspec init --template <name>` command (prompt for config values, render template, write files) in cmd/agentspec/init.go
- [x] T103 [US6] Create 5 project templates (customer-support, rag-chatbot, code-review-pipeline, data-extraction, research-assistant) as embedded `.ias` files in internal/templates/

**Checkpoint**: SDKs in Python, TypeScript, and Go can invoke agents, stream responses, and manage sessions. `agentspec init --template` scaffolds new projects.

---

## Phase 8: User Story 7 — IDE Support for IntentLang (Priority: P5)

**Goal**: VSCode extension with syntax highlighting, bracket matching, code folding, snippets, format-on-save, and inline diagnostics.

**Independent Test**: Install extension, open an `.ias` file, verify keywords are highlighted and validation errors appear inline on save.

- [x] T104 [P] [US7] Create VSCode extension manifest with IntentLang language registration, activation events, and contribution points in vscode-agentspec/package.json
- [x] T105 [P] [US7] Create TextMate grammar for IntentLang 2.0 (keywords, strings, numbers, comments, block structure, template variables) in vscode-agentspec/syntaxes/intentlang.tmLanguage.json
- [x] T106 [P] [US7] Create language configuration (bracket pairs, comment tokens, auto-closing pairs, folding markers) in vscode-agentspec/language-configuration.json
- [x] T107 [P] [US7] Create snippet definitions for agent, prompt, skill, deploy, pipeline blocks in vscode-agentspec/snippets/intentlang.json
- [x] T108 [US7] Implement extension entry point (activate/deactivate, register providers, format-on-save via `agentspec fmt`) in vscode-agentspec/src/extension.ts
- [x] T109 [US7] Implement LSP diagnostics provider (run `agentspec validate` on save, parse errors, map to VS Code diagnostics) in vscode-agentspec/src/language-server/diagnostics.ts
- [x] T110 [US7] Implement autocomplete provider (keyword completion, resource type completion, cross-reference completion for prompt/skill/agent names) in vscode-agentspec/src/language-server/completion.ts
- [x] T111 [US7] Implement go-to-definition provider (resolve `uses prompt "name"` to prompt block location) in vscode-agentspec/src/language-server/definition.ts

**Checkpoint**: VSCode extension provides syntax highlighting, snippets, format-on-save, inline diagnostics, autocomplete, and go-to-definition for `.ias` files.

---

## Phase 9: User Story 8 — Production Observability and Operations (Priority: P6)

**Goal**: Deployed agents emit Prometheus metrics, support distributed tracing, enforce token budgets and rate limits, and resolve secrets from secure stores.

**Independent Test**: Deploy an agent, send requests, verify metrics at `/v1/metrics` endpoint and that secrets from `env(VAR)` are resolved.

- [x] T112 [P] [US8] Implement Prometheus metrics collector (request counter, latency histogram, token counter, tool call counter by agent/tool/status) in internal/telemetry/metrics.go
- [x] T113 [P] [US8] Implement structured logging with slog (request-scoped fields, correlation IDs, JSON output) in internal/telemetry/logger.go
- [x] T114 [P] [US8] Add OpenTelemetry tracing hooks (span per invocation, child spans for LLM calls and tool calls) in internal/telemetry/traces.go
- [x] T115 [US8] Implement per-agent rate limiting middleware (token bucket, configurable rate/burst, 429 response) in internal/runtime/server.go
- [x] T116 [P] [US8] Implement Vault-style secret resolver (HTTP API to key-value secret store, token auth, caching) in internal/secrets/vault.go
- [x] T117 [P] [US8] Implement persistent session store (Redis-backed, configurable TTL, session serialization) in internal/session/redis_store.go
- [x] T118 [US8] Wire metrics endpoint `/v1/metrics` and tracing middleware into runtime server in internal/runtime/server.go
- [x] T119 [US8] Add integration test for metrics emission (invoke agent, scrape metrics endpoint, verify counters) in integration_tests/telemetry_test.go

**Checkpoint**: Deployed agents emit Prometheus metrics, support distributed tracing, enforce rate limits and token budgets, and resolve secrets from environment variables and secure stores.

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, documentation, and cross-cutting improvements

- [x] T120 Update ARCHITECTURE.md with runtime components, agentic loop, adapter architecture, and data flow diagram
- [x] T121 [P] Update CHANGELOG.md with all changes for this feature
- [x] T122 Run quickstart.md validation end-to-end (build CLI, create agent, apply, invoke, verify response)
- [x] T123 Update golden file fixtures for all new IntentLang 2.0 constructs in integration_tests/testdata/
- [x] T124 [P] Add determinism tests for new IR resource types (tool, deploy, pipeline, type) in integration_tests/determinism_test.go
- [x] T125 Verify all 10 examples in examples/ are runnable with `agentspec dev` using mock LLM
- [x] T126 [P] Delete init-spec.md from repository root (superseded by spec/ directory)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories
- **US1+US2 (Phase 3)**: Depends on Foundational — core runtime, marks MVP
- **US3 (Phase 4)**: Depends on Foundational — remaining language features (can parallelize with Phase 3)
- **US4 (Phase 5)**: Depends on US1+US2 — deployment targets need a working runtime
- **US5 (Phase 6)**: Depends on US1+US2 and US3 — pipelines need runtime + pipeline language constructs
- **US6 (Phase 7)**: Depends on US1+US2 — SDKs need a stable runtime HTTP API
- **US7 (Phase 8)**: Depends on US3 — IDE needs full 2.0 language support
- **US8 (Phase 9)**: Depends on US1+US2 — observability hooks into the runtime
- **Polish (Phase 10)**: Depends on all desired stories being complete

### User Story Dependencies

```
Phase 1 (Setup)
    │
    ▼
Phase 2 (Foundational)
    │
    ├──────────────────────┐
    ▼                      ▼
Phase 3 (US1+US2) ◄── Phase 4 (US3)
    │                      │
    ├──────┬──────┬────────┤
    ▼      ▼      ▼        ▼
  US4    US6    US8      US5
(Phase5)(Phase7)(Phase9)(Phase6)
                          │
                          ▼
                        US7
                      (Phase8)
    │      │      │        │
    ▼      ▼      ▼        ▼
         Phase 10 (Polish)
```

### Within Each User Story

- Interfaces/types before implementations
- Implementations before CLI commands
- CLI commands before integration tests
- Core functionality before convenience features

### Parallel Opportunities

**Phase 2 (Foundational)**:
- T005+T006 (tokens/lexer) must complete before T007-T009 (parser)
- T013+T014 (validators) are parallel with each other
- T016+T017+T018 (deprecation removal) are parallel with all other Phase 2 work

**Phase 3 (US1+US2)**:
- T021-T024 (LLM client) are parallel with T025-T027 (MCP client) and T029-T031 (tool executors)
- T034-T035 (memory) parallel with T036-T038 (sessions) parallel with T039-T040 (secrets)
- T050-T052 (plugin implementation) parallel with all other Phase 3 work

**Cross-phase parallelism**:
- Phase 4 (US3) can start in parallel with Phase 3 after Phase 2 completes
- Phase 7 (US6 SDKs) can start in parallel with Phase 5 (US4 deployment) after Phase 3 completes
- Phase 8 (US7 IDE) can start in parallel with Phase 6 (US5 pipelines) after Phase 4 completes

---

## Parallel Example: Phase 3 (US1+US2)

```bash
# Launch LLM client tasks in parallel:
Task: "T021 [US1] Implement LLM client interface in internal/llm/client.go"
Task: "T023 [US1] Implement mock LLM client in internal/llm/mock.go"
Task: "T024 [US1] Implement token usage tracking in internal/llm/tokens.go"

# Launch MCP + tool executor tasks in parallel (different packages):
Task: "T025 [US2] Implement MCP client wrapper in internal/mcp/client.go"
Task: "T029 [US2] Implement HTTP tool executor in internal/tools/http.go"
Task: "T030 [US2] Implement command tool executor in internal/tools/command.go"
Task: "T031 [US2] Implement inline code executor in internal/tools/inline.go"

# Launch memory + session + secrets in parallel (different packages):
Task: "T034 [US1] Implement memory interface in internal/memory/memory.go"
Task: "T036 [US1] Implement session store interface in internal/session/store.go"
Task: "T039 [US1] Implement secret resolver interface in internal/secrets/resolver.go"
```

---

## Implementation Strategy

### MVP First (US1 + US2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: US1 + US2 (Runtime Core)
4. **STOP and VALIDATE**: Run quickstart.md end-to-end. Agent responds to requests with Claude. Tools execute via MCP.
5. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Language parses 2.0 syntax
2. US1+US2 → Agents run locally with tools → **MVP**
3. US3 → Full IntentLang 2.0 with migration
4. US4 → Deploy to Docker/Kubernetes
5. US5 → Multi-agent pipelines
6. US6 → SDK clients
7. US7 → VSCode extension
8. US8 → Production observability
9. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: US1+US2 (runtime core) — blocks most other work
   - Developer B: US3 (remaining language) — can proceed in parallel
3. Once US1+US2 complete:
   - Developer A: US4 (deployment targets)
   - Developer B: US5 (pipelines, needs US3 done)
   - Developer C: US6 (SDKs)
4. Independent streams:
   - Developer D: US7 (IDE, needs US3 done)
   - Developer E: US8 (observability, needs US1+US2 done)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Constitution mandates integration tests as primary quality gate
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
