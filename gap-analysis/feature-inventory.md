# Feature Inventory

**Last updated:** 2026-03-01

## Feature Areas

| Feature Area | User-Facing Behavior | Entry Points | Core Modules | Data Entities | Config/Flags | Observability | Test Files |
|---|---|---|---|---|---|---|---|
| **DSL Parsing** | Write `.ias` files with IntentLang syntax; get syntax errors with line/col info | `agentspec validate`, `agentspec fmt` | parser, lexer, ast, formatter | AST nodes | — | Error messages with positions | parser_v3_test.go, validate_test.go, validation_test.go |
| **Validation** | Structural + semantic validation of `.ias` files | `agentspec validate` | validate (structural, semantic) | AST, IR resources | — | Validation error list | validate_test.go, validation_test.go |
| **Planning** | Preview create/update/delete changes before applying | `agentspec plan` | plan, ir, state | IR resources, state entries | `--env`, `--target` | Diff output | golden_test.go, determinism_test.go |
| **Applying** | Idempotent deployment of agent infrastructure | `agentspec apply` | apply, state, adapters | State entries | `--env`, `--target` | Status per resource | apply_test.go, idempotent_test.go |
| **Formatting** | Canonical formatting of `.ias` files | `agentspec fmt` | formatter | AST | `--check` | — | (via integration) |
| **Export** | Generate platform artifacts (Docker, K8s, Compose) | `agentspec export` | adapters (docker, k8s, compose, local) | IR resources | `--format` | — | export_test.go, docker_test.go, kubernetes_test.go |
| **Compilation** | Compile `.ias` to framework code (CrewAI, LangGraph, etc.) | `agentspec compile` | compiler, targets | AST, IR | `--target`, `--output` | — | compile_test.go, framework_compile_test.go |
| **Runtime Server** | HTTP API for agent invocation, streaming, sessions | `agentspec run`, `agentspec dev` | runtime, loop, session, memory, auth | Sessions, messages | `--port`, `--ui`, `--api-key` | `/v1/metrics`, `/healthz` | runtime_test.go, loop_test.go, auth_test.go |
| **Agent Loop** | ReAct pattern: LLM reasoning + tool calling + delegation | `/v1/agents/:name/invoke` | loop (react, reflexion, streaming) | Messages, tool results | Agent config | Telemetry events | loop_test.go |
| **Pipeline Execution** | Multi-agent DAG orchestration | `/v1/pipelines/:name/run` | pipeline (dag, executor) | Pipeline state | Pipeline config | Step-level events | pipeline_exec_test.go |
| **Session Management** | Persistent conversation history per session | `/v1/sessions/*` | session (memory, redis stores) | Sessions, messages | Store type, TTL, Redis config | — | (via runtime_test.go) |
| **Tool Execution** | Execute tools (command, HTTP, MCP, inline) during agent loop | Internal (via loop) | tools (command, http, mcp, inline) | Tool configs | Tool type + config | — | tools_test.go |
| **LLM Integration** | Call Anthropic/OpenAI APIs for agent reasoning | Internal (via loop) | llm (anthropic, openai, mock) | Messages, tool calls | Provider config, API keys | Token metrics | provider_test.go |
| **Secret Management** | Resolve secrets from env vars or Vault | Internal (via runtime) | secrets (env, vault, redact) | Secret refs | Vault addr/token/path | Redacted logging | secrets_test.go |
| **Plugin System** | Load/execute WASM plugins for lifecycle hooks | `agentspec apply` hooks | plugins (host, loader, validate) | Plugin binaries | Plugin path | Plugin stdout/stderr | plugin_test.go |
| **Import System** | Import and compose `.ias` files with versioning | `agentspec validate/plan/apply` | imports (resolver, version) | Import declarations | — | — | import_test.go |
| **Package Management** | Package, publish, install AgentPack bundles | `agentspec pkg`, `publish`, `install` | registry | Package metadata | Registry URL | — | packaging_test.go, registry_test.go |
| **SDK Generation** | Generate typed client SDKs (Go, Python, TypeScript) | `agentspec sdk` | sdk, templates | SDK templates | `--lang`, `--output` | — | sdk_test.go |
| **Expression Eval** | Evaluate expressions in configs and control flow | `agentspec eval`, runtime | expr, evaluation, controlflow | Expressions | — | — | eval_test.go |
| **Migration** | Migrate `.az` → `.ias` file format and syntax | `agentspec migrate` | migrate | File renames | `--dry-run` | — | (via integration) |
| **Dev Mode** | File watching + auto-reload server | `agentspec dev` | (cmd-level) | — | `--port`, `--ui` | Console logs | (via runtime_test.go) |
| **Frontend UI** | Browser-based agent chat interface | `agentspec dev --ui` | frontend (web assets, SSE) | — | — | — | frontend_test.go |
| **Telemetry** | Prometheus metrics, structured logs, tracing | `/v1/metrics` | telemetry | Metric counters | — | Self-referential | telemetry_test.go |
| **MCP Client** | Connect to MCP-compatible tool servers | Internal (via tools) | mcp | MCP tool list | Server command + args | — | (via tools_test.go) |
| **Policy Engine** | Deny/require security policy rules | `agentspec apply` | policy | Policy rules | — | — | (via integration) |
| **Init/Scaffold** | Create new AgentSpec projects from templates | `agentspec init` | templates | Template files | `--name` | — | — |

## Roles/Permissions Model

| Role | Scope | Enforcement |
|------|-------|-------------|
| API Key Bearer | Full access to all runtime endpoints | `internal/auth/middleware.go` + `internal/runtime/server.go` |
| Unauthenticated | Full access when no API key configured | `internal/runtime/server.go:145-148` (silent open access) |
| Rate-Limited Client | Per-IP token bucket | `internal/auth/ratelimit.go` |
| Rate-Limited Agent | Per-agent-name token bucket | `internal/runtime/server.go:694-746` |

**Note:** There is no user/role model beyond API key presence. No RBAC, no multi-user support, no agent-level access control.

## Critical Workflows

### 1. Author → Validate → Plan → Apply
```
Engineer writes .ias → validate syntax/semantics → plan changes → apply to infrastructure
                                                                       ↓
                                                              Update state file
```

### 2. Deploy → Run → Invoke
```
Apply creates deployment → runtime server starts → client sends HTTP request
                                                          ↓
                                                    Auth + rate limit
                                                          ↓
                                                    Load/create session
                                                          ↓
                                                    ReAct loop (LLM + tools)
                                                          ↓
                                                    Return response / stream SSE
```

### 3. Package → Publish → Install
```
Engineer packages .ias files → publishes to registry → others install and use
```

### 4. Compile → Deploy to Framework
```
Engineer compiles .ias → generates CrewAI/LangGraph/etc. code → deploys using framework tooling
```

## Non-Functional Expectations (Inferred)

| Aspect | Current State | Evidence |
|--------|--------------|---------|
| **Reliability** | Partial failure handling in apply; idempotency tested | apply_test.go, idempotent_test.go |
| **Performance** | Benchmark tests exist; no SLOs defined | benchmark_test.go |
| **Security** | API key auth, rate limiting, policy engine, secret redaction, WASM sandbox | auth/, secrets/, plugins/, policy/ |
| **Privacy** | Secret redaction filter for logs | secrets/redact.go |
| **Scalability** | Redis session store option; no horizontal scaling of runtime | session/redis_store.go |
| **Observability** | Prometheus metrics, structured logging, tracing stubs | telemetry/ |
| **Compliance** | None implemented | — |
