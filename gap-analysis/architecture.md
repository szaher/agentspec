# Architecture Summary

**Last updated:** 2026-03-01

## Context Diagram (C4 Level 1)

```
┌─────────────────────────────────────────────────────────────────┐
│                        Users / CI/CD                            │
│   (AI/ML engineers, DevOps, GitHub Actions)                     │
└────────────┬────────────────────────┬───────────────────────────┘
             │ CLI commands           │ HTTP API
             ▼                        ▼
┌────────────────────────┐  ┌─────────────────────────────────────┐
│   agentspec CLI        │  │   Agent Runtime Server              │
│   (cmd/agentspec/)     │  │   (internal/runtime/)               │
│                        │  │                                     │
│  validate, fmt, plan,  │  │  /v1/agents/:name/invoke            │
│  apply, compile, run,  │  │  /v1/agents/:name/stream            │
│  dev, export, pkg,     │  │  /v1/sessions, /v1/pipelines        │
│  publish, install,     │  │  /healthz, /v1/metrics              │
│  eval, init, migrate   │  │                                     │
└────────────┬───────────┘  └──────┬────────────┬─────────────────┘
             │                     │            │
             ▼                     ▼            ▼
┌──────────────────────────────────────────────────────────────────┐
│                     Core Engine                                  │
│                                                                  │
│  ┌─────────┐  ┌─────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │ Parser  │→ │   AST   │→ │ Validate │→ │  IR (Lower)      │  │
│  │ + Lexer │  │         │  │ struct + │  │  Deterministic   │  │
│  │         │  │         │  │ semantic │  │  JSON + SHA-256  │  │
│  └─────────┘  └─────────┘  └──────────┘  └────────┬─────────┘  │
│                                                     │            │
│  ┌──────────┐  ┌───────────┐  ┌──────────────────┐ │            │
│  │ Plan     │← │ State     │  │ Apply            │←┘            │
│  │ (diff)   │  │ (JSON     │  │ (idempotent)     │              │
│  │ create/  │  │  backend) │  │ partial-failure  │              │
│  │ update/  │  │           │  │ handling         │              │
│  │ delete   │  │           │  │                  │              │
│  └──────────┘  └───────────┘  └──────────────────┘              │
└──────────────────────────────────────────────────────────────────┘
             │                              │
             ▼                              ▼
┌──────────────────────────┐  ┌────────────────────────────────────┐
│  Deployment Adapters     │  │  External Integrations             │
│                          │  │                                    │
│  - process (local HTTP)  │  │  - LLM: Anthropic API, OpenAI API │
│  - docker (Dockerfile)   │  │  - MCP: stdio transport           │
│  - compose (docker-      │  │  - Vault: KV v2 secrets           │
│    compose.yml)          │  │  - Redis: session storage          │
│  - kubernetes (k8s       │  │  - WASM: plugin sandbox (wazero)  │
│    manifests)            │  │                                    │
│  - local (filesystem)    │  │                                    │
└──────────────────────────┘  └────────────────────────────────────┘
```

## Container Diagram (C4 Level 2)

### Build-Time Pipeline (CLI)

```
.ias file(s) → Lexer → Parser → AST → Validator → IR → Plan → Apply → State
                                   ↓                              ↓
                              Formatter                    Deployment Adapter
                                   ↓                              ↓
                           Formatted .ias              Docker/K8s/Compose artifacts
```

### Runtime Pipeline (Server)

```
HTTP Request → Auth Middleware → Rate Limiter → Route Handler
                                                      │
                     ┌────────────────────────────────┤
                     ▼                                ▼
              Agent Invoke                     Pipeline Execute
                     │                                │
                     ▼                                ▼
              Session Load                     DAG Topological Sort
                     │                                │
                     ▼                                ▼
              Memory Strategy                  Parallel Step Execution
              (Sliding/Summary)                       │
                     │                                ▼
                     ▼                         Per-step Agent Loop
              Agentic Loop (ReAct)
                     │
           ┌─────────┼──────────┐
           ▼         ▼          ▼
        LLM Call  Tool Call  Delegation
           │         │          │
           ▼         ▼          ▼
      Anthropic  Command/   Agent-to-
      / OpenAI   HTTP/MCP/  Agent call
                 Inline
```

## Component Inventory

| Package | LOC | Purpose | Dependencies |
|---------|-----|---------|-------------|
| `parser` | ~2,000 | Hand-written recursive descent parser + lexer | ast |
| `ast` | ~200 | AST node definitions | — |
| `validate` | ~800 | Structural + semantic validation | ast, ir |
| `ir` | ~1,000 | Intermediate representation, deterministic JSON, SHA-256 hashing | ast |
| `plan` | ~180 | Desired-state diff engine (create/update/delete/noop) | ir, state |
| `apply` | ~130 | Idempotent resource application | ir, state, adapters |
| `state` | ~200 | Pluggable state backend (JSON file default) | — |
| `formatter` | ~300 | Canonical .ias file formatting | ast |
| `runtime` | ~750 | HTTP server, endpoints, middleware | loop, session, pipeline, auth, telemetry |
| `loop` | ~700 | ReAct agentic loop, streaming, tool dispatch | llm, tools, memory |
| `llm` | ~600 | LLM client abstraction (Anthropic, OpenAI, Mock) | — |
| `tools` | ~400 | Tool registry + executors (command, HTTP, MCP, inline) | — |
| `session` | ~400 | Session management (memory, Redis stores) | llm |
| `memory` | ~250 | Conversation memory strategies | llm |
| `pipeline` | ~300 | Multi-agent DAG execution | loop |
| `auth` | ~250 | API key validation, rate limiting, middleware | — |
| `secrets` | ~300 | Secret resolution (env, Vault) + redaction | — |
| `plugins` | ~500 | WASM plugin loading, validation, lifecycle hooks | wazero |
| `compiler` | ~1,500 | Framework compilation targets (CrewAI, LangGraph, LlamaIndex, LlamaStack) | ast, ir |
| `policy` | ~100 | Security policy enforcement (deny/require rules) | ir |
| `telemetry` | ~300 | Prometheus metrics, structured logging, tracing | — |
| `imports` | ~400 | Import resolution, version management | parser |
| `mcp` | ~200 | Model Context Protocol client | — |
| `expr` | ~200 | Expression language integration (expr-lang) | — |
| `controlflow` | ~150 | If/else routing | expr |
| `evaluation` | ~250 | Configuration and expression evaluation | expr |
| `events` | ~150 | Structured event emission | — |
| `frontend` | ~150 | Embedded web UI (Vanilla JS + SSE) | — |
| `registry` | ~300 | Agent/skill discovery and registration | — |
| `migrate` | ~100 | DSL v1→v2 migration | parser |
| `sdk` | ~200 | SDK generator (Go, Python, TypeScript) | — |
| `templates` | ~50 | Embedded .ias templates | — |
| `adapters` | ~500 | Deployment adapters (process, docker, compose, k8s, local) | — |

## Data Stores

| Store | Type | Location | Purpose |
|-------|------|----------|---------|
| `.agentspec.state.json` | Local JSON file | Working directory | Infrastructure state tracking |
| In-memory map | Ephemeral | Process memory | Session storage (dev) |
| Redis | External | Configurable | Session storage (production) |

## Key Design Decisions

1. **Hand-written parser** over ANTLR — for better error messages and Go-native control (DECISIONS/001)
2. **WASM sandbox** via wazero — for plugin isolation without CGo dependency (DECISIONS/002)
3. **Pluggable state backend** — JSON file default, extensible to remote stores (DECISIONS/003)
4. **Deterministic IR** — sorted keys + SHA-256 hashing for reliable change detection
5. **ReAct loop** — standard observe-think-act pattern for agent execution

## Top 10 Technical Risks

1. State file corruption due to non-atomic writes
2. Inline tool execution without sandboxing
3. Policy engine `checkRequirement()` is a no-op stub
4. Predictable session IDs from timestamp
5. No unit tests for 33/34 internal packages
6. Race conditions in MCP connection pool and session stores
7. Unbounded memory growth in rate limiters and session stores
8. HTTP server has no timeouts (slow-loris vulnerable)
9. Redis KEYS command blocks under load
10. OpenAI streaming client may be broken (incomplete SSE parsing)

## Top 10 Product Risks

1. Compiler targets generate "not implemented" stubs — users get non-functional code
2. `eval` command uses stub invoker — cannot actually evaluate agent behavior
3. `publish --sign` flag does nothing — false security signal
4. README missing 8 of 19 CLI commands — poor discoverability
5. Frontend lacks loading, error, empty states — poor UX
6. `run` vs `dev` naming confusion — different behaviors not obvious
7. No TLS support — API keys transmitted in cleartext
8. CORS wildcard on SSE — any website can invoke agents
9. Documentation may drift from implementation — no automated validation
10. No release automation — manual version bumps and binary distribution
