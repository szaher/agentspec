# Implementation Plan: AgentSpec Runtime Platform

**Branch**: `004-runtime-platform` | **Date**: 2026-02-23 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/004-runtime-platform/spec.md`

## Summary

Transform AgentSpec from a declarative state-tracking shell into a working agent deployment platform. The core gap is that `agentspec apply` currently does nothing — no agent process starts, no LLM is called, no tools execute. This plan adds a real runtime (agentic loop, LLM client, MCP tool execution), evolves IntentLang to 2.0 with richer constructs, implements deployment adapters (process, Docker, Kubernetes), and delivers SDKs and IDE tooling.

The implementation follows 5 phases: Language Foundation → Runtime Core → Deployment Targets → Advanced Features → Developer Experience.

## Technical Context

**Language/Version**: Go 1.25+ (existing)
**Primary Dependencies**:
- cobra v1.10.2 (existing — CLI framework)
- wazero v1.11.0 (existing — WASM plugin sandbox)
- `github.com/anthropics/anthropic-sdk-go` v1.26.0 (new — LLM client)
- `github.com/modelcontextprotocol/go-sdk/mcp` v1.3.1 (new — MCP client)
- `github.com/moby/moby/client` v29.x (new — Docker operations)
- `k8s.io/client-go` v0.35.1 (new — Kubernetes operations)

**Storage**: Local JSON state file (existing `.agentspec.state.json`). In-memory session store (new, default). Redis session store (new, opt-in).
**Testing**: Integration tests in `integration_tests/` (existing pattern). Golden file regression. `go test ./... -count=1`.
**Target Platform**: Linux/macOS CLI + runtime server
**Project Type**: CLI tool + runtime server (extending existing CLI)
**Performance Goals**: <2s streaming overhead, <5ms plan computation, <10ms parse+validate (per spec SC-005/006)
**Constraints**: Single runtime process per package. Graceful restart on apply with changes. State file locking for concurrent apply.
**Scale/Scope**: Single-node runtime (multi-node via Kubernetes adapter). Supports multiple agents per package.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Notes |
|---|-----------|--------|-------|
| I | Determinism | PASS | Parse→validate→plan→apply pipeline remains deterministic. Runtime agent behavior is inherently non-deterministic (LLM outputs) but the toolchain pipeline is not affected. |
| II | Idempotency | PASS | Apply twice with no changes = no restart. State file tracks deployed process PID/port. Drift detection checks running state. |
| III | Portability | PASS | New adapters (process, Docker, K8s) follow existing adapter pattern. Platform-specific logic isolated in `internal/adapters/`. |
| IV | Separation of Concerns | PASS | New language constructs are surface syntax. Runtime reads IR, not raw DSL. Tool execution is IR-driven. |
| V | Reproducibility | PASS | All new dependencies pinned in go.mod. Container images use pinned base images (distroless). |
| VI | Safe Defaults | PASS | Secrets remain references (`env()`/`store()`). Inline code sandboxed in subprocesses. API keys via environment variables. |
| VII | Minimal Surface Area | PASS | Each new keyword (`tool`, `deploy`, `pipeline`, `delegate`, `type`) has a concrete use case with example in spec. |
| VIII | English-Friendly Syntax | PASS | New constructs use natural English: `deploy "prod" target "kubernetes"`, `delegate to agent "billing" when "billing question"`. |
| IX | Canonical Formatting | PASS | Formatter extended for all new constructs. |
| X | Strict Validation | PASS | Validator extended for all new constructs with source location and fix hints. |
| XI | Explicit References | PASS | Import pinning unchanged. MCP server references are explicit in `.ias` file. |
| XII | No Hidden Behavior | PASS | Runtime behavior fully declared in `.ias` file. No undeclared transforms. |

**Contracts**:

| Contract | Status | Notes |
|----------|--------|-------|
| Desired-State Engine | PASS | Adds `run`, `dev`, `status`, `logs`, `destroy` commands. `plan` output remains stable and machine-diffable. |
| Adapter Contract | PASS | New adapters accept IR as input. Adapter outputs exportable. MVP delivers 3 adapters (process, Docker, K8s) — exceeds requirement of 2. |
| Plugin Contract | PASS | Implements currently-stubbed hooks/validators/transforms (FR-051). |
| SDK Contract | PASS | Runtime API contract (contracts/runtime-api.md) is the contract for SDK generation. Typed clients are generated from the contract definition and IR skill schemas. |

**MVP Definition of Done**:

| Criterion | How Met |
|-----------|---------|
| Golden-path demo | `agentspec apply my-bot.ias` → running agent → HTTP invoke → AI response with tool use |
| Two adapters | process + docker (+ K8s = 3 total) |
| One plugin demo | Existing monitor plugin with implemented hooks |
| SDKs in 3 languages | Python, TypeScript, Go clients generated from runtime API |

## Project Structure

### Documentation (this feature)

```text
specs/004-runtime-platform/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research findings
├── data-model.md        # Entity definitions and relationships
├── quickstart.md        # Getting started guide
├── contracts/
│   └── runtime-api.md   # Runtime HTTP API contract
└── tasks.md             # Task breakdown (generated by /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/agentspec/
├── main.go              # MODIFY — add new subcommands
├── validate.go          # MODIFY — IntentLang 2.0 support
├── fmt.go               # MODIFY — new constructs, remove resolveAZFiles
├── plan.go              # MODIFY — new resource types
├── apply.go             # MODIFY — real adapter dispatch
├── run.go               # NEW — one-shot agent invocation
├── dev.go               # NEW — development mode with hot reload
├── status.go            # NEW — deployment status
├── logs.go              # NEW — log streaming
├── destroy.go           # NEW — resource teardown
├── init.go              # NEW — project scaffolding from templates
└── migrate.go           # MODIFY — add --to-v2 language migration

internal/
├── parser/
│   ├── parser.go        # MODIFY — IntentLang 2.0 constructs
│   ├── lexer.go         # MODIFY — new tokens
│   └── token.go         # MODIFY — new token types
├── ast/
│   └── ast.go           # MODIFY — new node types (Tool, Deploy, Pipeline, Type, etc.)
├── formatter/
│   └── formatter.go     # MODIFY — format new constructs
├── validate/
│   ├── structural.go    # MODIFY — validate new constructs
│   └── semantic.go      # MODIFY — validate new references and types
├── ir/
│   ├── ir.go            # MODIFY — new resource types in IR
│   └── lower.go         # MODIFY — lower new AST nodes to IR
├── runtime/             # NEW — runtime lifecycle
│   ├── runtime.go       # Runtime start/stop/health
│   ├── config.go        # IR → runtime config conversion
│   └── server.go        # HTTP server (mux, middleware, auth)
├── loop/                # NEW — agentic loop strategies
│   ├── loop.go          # Strategy interface
│   ├── react.go         # ReAct strategy
│   ├── plan_execute.go  # Plan-and-Execute strategy
│   ├── reflexion.go     # Reflexion strategy
│   ├── router.go        # Router strategy
│   ├── map_reduce.go    # Map-Reduce strategy
│   └── delegation.go    # Agent delegation
├── llm/                 # NEW — LLM client abstraction
│   ├── client.go        # Client interface
│   ├── anthropic.go     # Anthropic Claude implementation
│   ├── mock.go          # Mock client for testing
│   └── tokens.go        # Token budget tracking
├── mcp/                 # NEW — MCP client for tool execution
│   ├── client.go        # MCP client wrapper
│   ├── pool.go          # Connection pooling
│   └── discovery.go     # Tool discovery from MCP servers
├── tools/               # NEW — tool execution registry
│   ├── registry.go      # Tool registry (MCP, HTTP, command, inline)
│   ├── http.go          # HTTP tool executor
│   ├── command.go       # Command tool executor
│   └── inline.go        # Inline code executor (sandboxed subprocess)
├── memory/              # NEW — conversation memory
│   ├── memory.go        # Memory interface
│   ├── sliding.go       # Sliding window strategy
│   └── summary.go       # Summarization strategy
├── session/             # NEW — session management
│   ├── session.go       # Session lifecycle
│   ├── store.go         # Store interface
│   └── memory_store.go  # In-memory session store
├── secrets/             # NEW — secret resolution
│   ├── resolver.go      # Resolver interface
│   ├── env.go           # Environment variable resolver
│   └── redact.go        # Secret redaction filter for logs/state
├── state/               # NEW — state file management
│   └── local.go         # State file locking (flock)
├── telemetry/           # NEW — observability
│   ├── metrics.go       # Prometheus metrics
│   └── logger.go        # Structured logging
├── adapters/
│   ├── adapter.go       # MODIFY — extended interface (Status, Logs, Destroy)
│   ├── process/         # NEW — local process adapter
│   │   ├── process.go   # Start/stop runtime process
│   │   └── health.go    # Health check polling
│   ├── docker/          # NEW — Docker adapter
│   │   ├── docker.go    # Build + run containers
│   │   └── dockerfile.go # Dockerfile generation
│   ├── kubernetes/      # NEW — Kubernetes adapter
│   │   ├── kubernetes.go # Apply manifests, wait for rollout
│   │   └── manifests.go # Manifest generation
│   ├── compose/
│   │   └── compose.go   # REWRITE — real Docker Compose management
│   └── local/
│       └── local.go     # DEPRECATE — replaced by process adapter
├── pipeline/            # NEW — multi-agent pipeline executor
│   ├── dag.go           # DAG builder and cycle detection
│   └── executor.go      # DAG-based pipeline execution
├── migrate/             # NEW — IntentLang 1.0→2.0 migration
│   └── v2.go            # AST rewriting for language migration
├── templates/           # NEW — project templates
│   ├── registry.go      # Template registry
│   └── templates/       # Embedded template files
├── plugins/
│   ├── hooks.go         # MODIFY — implement actual WASM hook execution
│   ├── validate.go      # MODIFY — implement actual WASM validation
│   └── transform.go     # MODIFY — implement actual WASM transforms
├── sdk/
│   └── generator/
│       └── generator.go # REWRITE — full SDK generation from runtime API contract
└── cli/
    └── deprecation.go   # DELETE — remove .az deprecation code

sdk/                     # REWRITE — full SDK client libraries
├── python/
│   └── agentspec/       # Python SDK package
├── typescript/
│   └── src/             # TypeScript SDK
└── go/
    └── agentspec/       # Go SDK package

vscode-agentspec/        # NEW — VSCode extension
├── package.json
├── syntaxes/
│   └── intentlang.tmLanguage.json
├── language-configuration.json
└── src/
    ├── extension.ts
    └── language-server/

templates/               # NEW — project scaffolding templates
├── customer-support.ias
├── rag-chatbot.ias
├── code-review-pipeline.ias
├── data-extraction.ias
└── research-assistant.ias
```

**Structure Decision**: Extends the existing Go project structure. All new runtime code goes under `internal/` following existing patterns. New CLI commands added to `cmd/agentspec/`. SDKs remain under `sdk/`. VSCode extension is a separate project under `vscode-agentspec/`.

## Implementation Phases

### Phase 1: Language Foundation

**Goal**: IntentLang 2.0 parser, AST, IR, formatter, and validator support all new constructs. Existing pipeline (parse → validate → plan) works with 2.0 files.

**Scope**:
- Add new tokens and parser rules for `tool`, `deploy`, `pipeline`, `delegate`, `type`, agent runtime config, prompt `variables`
- Add new AST node types for all 2.0 constructs
- Extend IR with new resource types (DeployTarget, Pipeline, Type)
- Extend formatter for canonical formatting of new constructs
- Extend structural and semantic validators
- Implement `agentspec migrate --to-v2` command
- Remove `.az` file support and deprecation code
- Remove stub `Plan()` methods from adapters
- Update all examples to IntentLang 2.0
- Update integration tests for new constructs

**Dependencies**: None (builds on existing parser/AST/IR)

**Constitution gates**: IX (formatter extended), X (validator extended), VII (each new keyword has example)

### Phase 2: Runtime Core

**Goal**: `agentspec apply` starts a local runtime process. Agents respond to HTTP requests. The ReAct agentic loop calls Claude and executes MCP tools.

**Scope**:
- Implement LLM client (Anthropic SDK wrapper with streaming, tool use, prompt caching)
- Implement MCP client (connection pooling, tool discovery, stdio transport)
- Implement tool execution registry (MCP, HTTP, command, inline with subprocess sandbox)
- Implement ReAct agentic loop strategy
- Implement runtime HTTP server (routes per API contract, SSE streaming, API key auth)
- Implement session management (in-memory store, sliding window memory)
- Implement secret resolution (env variables)
- Implement local process adapter (start/stop, health checks, PID tracking, state file locking)
- Implement `agentspec run` command (one-shot invocation)
- Implement `agentspec dev` command (file watcher + graceful restart)
- Replace stub `Apply()` in existing adapters
- Implement WASM plugin hooks/validators/transforms (FR-051)
- Add mock LLM client for testing
- Add integration tests for runtime lifecycle

**Dependencies**: Phase 1 (parser must support 2.0 constructs)

**New dependencies added to go.mod**:
- `github.com/anthropics/anthropic-sdk-go` v1.26.0
- `github.com/modelcontextprotocol/go-sdk/mcp` v1.3.1

**Constitution gates**: II (idempotent apply), VI (secrets as references, inline sandboxing), XII (no hidden behavior)

### Phase 3: Deployment Targets

**Goal**: Same `.ias` file deploys to Docker, Docker Compose, and Kubernetes. Operational commands (`status`, `logs`, `destroy`) work across targets.

**Scope**:
- Extend adapter interface with `Status()`, `Logs()`, `Destroy()` methods
- Implement Docker adapter (Dockerfile generation, image build, container lifecycle, health checks)
- Rewrite Docker Compose adapter (real compose files, stack management)
- Implement Kubernetes adapter (manifest generation via client-go, Server-Side Apply, rollout wait)
- Implement `agentspec status` command
- Implement `agentspec logs` command
- Implement `agentspec destroy` command
- Add integration tests for each adapter (Docker tests require Docker daemon, K8s tests require kind/minikube)

**Dependencies**: Phase 2 (runtime must work locally first)

**New dependencies added to go.mod**:
- `github.com/moby/moby/client` v29.x
- `k8s.io/client-go` v0.35.1
- `k8s.io/api`, `k8s.io/apimachinery`

**Constitution gates**: III (portability proven with 3 adapters), adapter contract (thin, IR-input)

### Phase 4: Advanced Features

**Goal**: Multi-agent pipelines, additional agentic strategies, and observability.

**Scope**:
- Implement pipeline executor (DAG-based, fail-fast cancellation, parallel step execution)
- Implement agent delegation (`delegate to agent "name" when "condition"`)
- Implement Plan-and-Execute strategy
- Implement Reflexion strategy
- Implement Router strategy
- Implement Map-Reduce strategy
- Implement Prometheus metrics exporter
- Implement structured logging with correlation IDs
- Add OpenTelemetry distributed tracing hooks

**Dependencies**: Phase 2 (core runtime)

**Constitution gates**: Observability requirements (structured events, correlation IDs, exportable logs)

### Phase 5: Developer Experience

**Goal**: SDK clients, VSCode extension, project templates, and testing tools.

**Scope**:
- Build Python SDK client (invoke, stream, session, typed responses)
- Build TypeScript SDK client (same capabilities)
- Build Go SDK client (same capabilities)
- Build VSCode extension (TextMate grammar, language configuration, snippet definitions)
- Add LSP server for diagnostics (shell out to `agentspec validate`)
- Add autocomplete provider
- Add go-to-definition provider
- Implement `agentspec init --template <name>` command
- Create 5+ project templates (customer-support, rag-chatbot, code-review-pipeline, data-extraction, research-assistant)
- Implement `agentspec test` command (mock LLM, conversation replay)

**Dependencies**: Phase 2 (runtime HTTP API must be stable for SDK generation)

**Constitution gates**: SDK contract (generated from stable API, supports listing/querying/invoking/streaming)

## Complexity Tracking

No constitution violations requiring justification. All gates pass.
