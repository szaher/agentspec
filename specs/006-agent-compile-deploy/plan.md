# Implementation Plan: Agent Compilation & Deployment Framework

**Branch**: `006-agent-compile-deploy` | **Date**: 2026-02-28 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/006-agent-compile-deploy/spec.md`

## Summary

Transform AgentSpec from an infrastructure-as-code tool into an agent compilation platform. Users write declarative `.ias` files defining agents with reasoning loops, tools, validation, evaluation, and control flow. The compiler produces standalone executable agents (via `go:embed`), framework-specific source code (via WASM plugins targeting CrewAI/LangGraph/LlamaStack/LlamaIndex), and deployment packages (Docker/K8s/binary). IntentLang 3.0 adds imports, `if/else`, `for each`, config declarations, validation rules, and eval test cases. A built-in vanilla JS frontend enables real-time agent interaction. The package ecosystem uses Git-based resolution (MVP) with Minimal Version Selection, evolving to an HTTP registry.

## Technical Context

**Language/Version**: Go 1.25+ (existing project, all new code in Go)
**Primary Dependencies**: wazero v1.11.0 (WASM plugin sandbox — existing), cobra v1.10.2 (CLI — existing), expr-lang/expr (expression evaluation — new), go:embed (frontend + config bundling — stdlib)
**Storage**: Local JSON state file `.agentspec.state.json` (existing), local package cache `~/.agentspec/cache/` (new), in-memory session store (existing)
**Testing**: `go test` with integration test fixtures (existing pattern), golden file assertions for deterministic compilation output
**Target Platform**: CLI runs on macOS/Linux/Windows. Compiled agents target linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64.
**Project Type**: CLI tool + compiler + runtime + embedded web frontend
**Performance Goals**: Compilation <10s for 500-line .ias, agent startup <3s, container images <100MB
**Constraints**: Deterministic compilation output (FR-004), WASM sandbox for plugins, API key auth by default, no secrets in artifacts

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Determinism | ✅ Pass | FR-004 requires byte-identical compilation output. Go 1.21+ guarantees reproducible builds with `-trimpath`. |
| II. Idempotency | ✅ Pass | Compilation is pure: same inputs → same outputs. Existing `apply` idempotency preserved. |
| III. Portability | ✅ Pass | `.ias` DSL is platform-neutral. Platform specifics isolated in adapters and compilation plugins. |
| IV. Separation of Concerns | ✅ Pass | Surface syntax → IR → compilation targets. All semantic meaning in IR. Compilation plugins receive IR, never raw DSL. |
| V. Reproducibility | ✅ Pass | Lock file (`.agentspec.lock`) pins dependency versions. Package checksums verified. |
| VI. Safe Defaults | ✅ Pass | API key auth required by default (FR-042). Secrets never baked into artifacts (FR-043). Config params with `secret` modifier cannot have defaults. |
| VII. Minimal Surface Area | ⚠️ Justified | Adding 6 new constructs: `import`, `if/else`, `for each`, `config`, `validate`, `eval`. Each justified by concrete use cases in spec and quickstart.md. |
| VIII. English-Friendly Syntax | ✅ Pass | New syntax uses natural English: `for each source in input.data_sources`, `validate { rule no_pii error ... }`. Readable by non-programmers. |
| IX. Canonical Formatting | ✅ Pass | Formatter must be extended for new constructs. Single canonical output. |
| X. Strict Validation | ✅ Pass | Compile-time validation with source locations (FR-002). Expr expressions type-checked at compile time (FR-048). |
| XI. Explicit References | ✅ Pass | All imports pinned to version or SHA (FR-007). Lock file records resolved versions. |
| XII. No Hidden Behavior | ✅ Pass | Compilation transforms are declared plugins. Validation rules explicitly defined. No undeclared mutations. |
| Adapter Contract | ✅ Pass | Adapters accept IR, not raw DSL. Refactored to consume compiled artifacts while maintaining the contract. |
| Plugin Contract | ✅ Pass | Compilation plugins declare capabilities, are versioned, and run in WASM sandbox. |
| SDK Contract | ✅ Pass | Existing SDK generation unchanged. Compiled agents expose the same HTTP API. |
| Non-Goal: Complex UI | ⚠️ Justified | Built-in frontend is a minimal chat interface (~5-10KB vanilla JS), not a complex control plane. It is a developer tool, not a SaaS product. Constitution says "until promoted by amendment" — this is the promotion via spec. |

**Post-Phase 1 re-check**: All gates pass. The `on input` block and expression syntax stay within English-friendly bounds. Minimal Surface Area is addressed by each construct having concrete examples in quickstart.md.

## Project Structure

### Documentation (this feature)

```text
specs/006-agent-compile-deploy/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model
├── quickstart.md        # Phase 1 quickstart guide
├── contracts/
│   ├── cli-contract.md              # New CLI commands
│   ├── compiler-plugin-contract.md  # WASM compilation plugin interface
│   ├── agent-runtime-contract.md    # Compiled agent HTTP API
│   ├── intentlang-extensions-contract.md  # IntentLang 3.0 syntax
│   └── package-registry-contract.md # Package registry HTTP API
├── checklists/
│   └── requirements.md             # Spec quality checklist
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/agentspec/
├── main.go              # Entry point (existing)
├── compile.go           # NEW: agentspec compile command
├── eval.go              # NEW: agentspec eval command
├── pkg.go               # NEW: agentspec package command
├── publish.go           # NEW: agentspec publish command
├── install.go           # NEW: agentspec install command
└── ... (existing commands)

internal/
├── ast/
│   └── ast.go           # MODIFIED: Add import, if/else, for-each, config, validate, eval nodes
├── parser/
│   ├── lexer.go         # MODIFIED: Add new keywords (import, if, else, for, each, config, validate, eval, rule, case, when)
│   └── parser.go        # MODIFIED: Parse new constructs
├── formatter/
│   └── formatter.go     # MODIFIED: Format new constructs
├── validate/
│   └── validate.go      # MODIFIED: Validate imports, control flow, config params, validation rules, eval cases
├── ir/
│   ├── ir.go            # MODIFIED: Extend IR with import graph, control flow, config, validation, eval
│   └── lower.go         # MODIFIED: Lower new AST nodes to IR
├── compiler/            # NEW: Compilation framework
│   ├── compiler.go      # Core compilation orchestrator
│   ├── embed.go         # Standalone target: go:embed + go build
│   ├── template.go      # Template for compiled agent main.go
│   ├── cross.go         # Cross-compilation support
│   ├── configref.go     # Config reference document generator
│   └── safezone.go      # Safe zone detection and preservation for recompilation
├── imports/             # NEW: Import resolution
│   ├── resolver.go      # Import resolution and caching
│   ├── graph.go         # Dependency graph + cycle detection (Tarjan's SCC)
│   ├── lock.go          # Lock file read/write
│   └── mvs.go           # Minimal Version Selection algorithm
├── expr/                # NEW: Expression evaluation
│   ├── evaluator.go     # Runtime expression evaluator (wraps expr-lang/expr)
│   └── compiler.go      # Compile-time expression validation
├── controlflow/         # NEW: Control flow execution
│   ├── executor.go      # If/else and for-each runtime executor
│   └── context.go       # Runtime context (input, session, steps, config)
├── validation/          # NEW: Agent output validation
│   ├── validator.go     # Rule execution engine
│   └── retry.go         # Retry logic for failed validations
├── evaluation/          # NEW: Agent quality evaluation
│   ├── runner.go        # Batch eval runner
│   ├── scoring.go       # Scoring methods (exact, contains, semantic, custom)
│   └── report.go        # Eval report generation
├── frontend/            # NEW: Built-in web frontend
│   ├── handler.go       # HTTP handler serving embedded SPA
│   ├── sse.go           # SSE streaming handler
│   └── web/             # Static frontend files
│       ├── index.html   # Chat UI (vanilla JS + CSS)
│       └── app.js       # Frontend logic
├── registry/            # NEW: Package registry client
│   ├── client.go        # Registry HTTP client
│   ├── git.go           # Git-based resolution (Phase 0)
│   ├── cache.go         # Local package cache
│   └── manifest.go      # agentpack.yaml parser
├── auth/                # NEW: API key authentication
│   ├── middleware.go     # HTTP auth middleware
│   ├── key.go           # API key validation
│   └── ratelimit.go     # Configurable rate limiting middleware
├── runtime/             # MODIFIED: Enhanced for compiled agents
│   ├── runtime.go       # Add control flow, validation, config resolution
│   └── server.go        # Add frontend serving, SSE streaming, auth middleware
├── plugins/             # MODIFIED: Add compilation plugin support
│   └── compile.go       # Compilation plugin host functions
├── adapters/            # MODIFIED: Consume compiled artifacts
│   ├── docker/
│   ├── kubernetes/
│   └── process/
└── tools/               # Existing tool executors

examples/
├── compiled-agent/              # NEW: Basic compiled agent example
│   └── hello-agent.ias
├── multi-file-agent/            # NEW: Import system example
│   ├── main.ias
│   └── skills/
│       ├── search.ias
│       └── respond.ias
├── control-flow-agent/          # NEW: If/else + for-each example
│   └── router-agent.ias
├── validated-agent/             # NEW: Validation + eval example
│   └── support-agent.ias
└── ... (existing examples)

integration_tests/
├── compile_test.go              # NEW: Compilation integration tests
├── import_test.go               # NEW: Import resolution tests
├── controlflow_test.go          # NEW: Control flow execution tests
├── validation_test.go           # NEW: Agent validation tests
├── eval_test.go                 # NEW: Agent evaluation tests
├── frontend_test.go             # NEW: Frontend serving tests
├── auth_test.go                 # NEW: Authentication tests
├── registry_test.go             # NEW: Package registry tests
├── framework_compile_test.go    # NEW: Framework code generation tests
├── packaging_test.go            # NEW: Deployment packaging tests
├── benchmark_test.go            # NEW: Performance benchmark tests
└── fixtures/
    ├── compile/                 # Golden fixtures for compilation
    ├── import/                  # Import resolution fixtures
    └── eval/                    # Eval scoring fixtures
```

**Structure Decision**: Single Go module, extending the existing `internal/` package layout. New packages (`compiler`, `imports`, `expr`, `controlflow`, `validation`, `evaluation`, `frontend`, `registry`, `auth`) follow the established pattern. The frontend is embedded via `go:embed` from `internal/frontend/web/`. No separate frontend build process — vanilla JS files are committed directly.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 6 new language constructs (Principle VII) | Each serves a distinct, essential purpose: `import` (composability), `if/else` (routing), `for each` (iteration), `config` (runtime portability), `validate` (quality), `eval` (testing). All have concrete examples in quickstart.md. | Fewer constructs would leave the language unable to express real agent workflows. Each has been validated against user stories P1-P6. |
| Built-in frontend (Non-Goal: Complex UI) | The frontend is a ~5-10KB vanilla JS chat interface, not a complex control plane. It serves as a developer tool and basic interaction surface. The constitution allows promotion via amendment — this spec is the amendment. | External-only UI would require users to build custom frontends for basic agent testing. Every comparable tool (LangGraph Studio, CrewAI Playground) ships a built-in UI. |
| Adapter contract: compiled binary input (Adapter Contract) | Deployment adapters (docker, kubernetes, process) are refactored to accept compiled agent binaries instead of IR. The constitutional principle "Adapters MUST accept IR as input, never raw DSL" is preserved in spirit — adapters never process raw DSL. Compiled binaries contain embedded IR and are a post-processing artifact. | Requiring adapters to accept only raw IR would force deployment packaging to re-lower from IR on every deploy, defeating the purpose of compilation. The compiled binary is a sealed, verified artifact. A constitution amendment should formalize this evolution. |
