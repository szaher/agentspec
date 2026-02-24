# Changelog

## [0.3.0] - 2026-02-23

### Added
- Runtime HTTP server with agent invocation, streaming, and session endpoints
- Agentic loop with ReAct strategy, configurable max turns, and token budgets
- LLM client abstraction with Anthropic and OpenAI provider support
- SSE streaming for real-time agent responses
- Session management with in-memory and Redis-backed stores
- Conversation memory strategies: sliding window and LLM-based summarization
- Multi-agent pipeline execution with step dependencies
- Multi-target deployment adapters: process, Docker, Kubernetes, Compose
- Deployment lifecycle commands: `agentspec dev`, `agentspec run`, `agentspec status`, `agentspec logs`, `agentspec destroy`
- Tool registry with MCP, HTTP, command, and inline tool types
- IntentLang 2.0 constructs: `skill`, `deploy`, `pipeline`, `type`, `server` blocks
- Agent delegation with `delegate to` and `fallback` support
- Error handling strategies: `on_error` with retry, fallback, and abort modes
- Prometheus-style metrics endpoint (`/v1/metrics`) with invocation, token, and tool call counters
- Structured JSON logging with slog and correlation IDs
- OpenTelemetry-style tracing with span hierarchies
- Per-agent rate limiting middleware (token bucket)
- Vault-compatible secret resolver with caching
- SDK client libraries for Python, TypeScript, and Go
- Template-based SDK generator producing typed clients from RuntimeConfig
- `agentspec init --template` command with 5 starter templates
- VSCode extension with syntax highlighting, snippets, diagnostics, autocomplete, and go-to-definition
- Health check endpoint (`/healthz`)
- API key authentication middleware (X-API-Key / Bearer token)

## [0.2.0] - 2026-02-23

### Changed
- Renamed DSL from "Agentz DSL" to "IntentLang"
- Renamed file extension from `.az` to `.ias`
- Renamed CLI command from `agentz` to `agentspec`
- Renamed definition files to "AgentSpec" and bundles to "AgentPack"
- Go module path (`agentz`) remains unchanged

## [0.1.0] - 2026-02-22

### Added
- Initial release of the AgentSpec toolchain
- Custom `.ias` IntentLang DSL with English-friendly syntax
- Hand-written recursive descent parser with source position error reporting
- Canonical formatter (`agentspec fmt`) with idempotent output
- Two-phase validation (structural + semantic) with "did you mean?" suggestions
- Intermediate Representation (IR) with deterministic JSON serialization
- Content-addressed hashing (SHA-256) for change detection
- Desired-state diff engine with plan/apply lifecycle
- Idempotent apply with partial failure handling
- Drift detection (`agentspec diff`)
- Two adapters: Local MCP and Docker Compose
- Export command for generating platform-specific artifacts
- Multi-environment configuration with overlay merging
- WASM plugin system (wazero) for custom resource types and hooks
- SDK generation for Python, TypeScript, and Go
- Policy engine for security constraints
- Structured event emission with correlation IDs
- Golden fixture integration tests for determinism validation
