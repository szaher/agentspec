# Architecture: AgentSpec Toolchain

## Overview

AgentSpec is a declarative agent packaging, runtime, and deployment toolchain.
It parses `.ias` IntentLang definition files through a pipeline that produces
platform-neutral artifacts via pluggable adapters. At runtime, agents are served
via an HTTP API with an agentic loop that orchestrates LLM calls and tool execution.

## Data Flow

### Build-Time Pipeline

```
.ias source → Lexer → Tokens → Parser → AST → Validator → IR → Adapter → Artifacts
                                                              ↓
                                                         State File
```

### Runtime Request Flow

```
HTTP Request → Auth Middleware → Rate Limiter → Handler → Agentic Loop → LLM Client
                                                              ↑   ↓
                                                          Tool Registry
                                                              ↓
                                                     Session / Memory Store
                                                              ↓
                                                        Metrics / Traces
```

## Components

### Parser (`internal/parser/`)
Hand-written recursive descent parser producing AST from `.ias` source.
Includes lexer/tokenizer with keyword recognition and source position tracking.

### AST (`internal/ast/`)
Abstract syntax tree node types for all resource kinds with source position
tracking for error reporting.

### Formatter (`internal/formatter/`)
Canonical formatter producing deterministic `.ias` output from AST.
Zero configuration options — one canonical style.

### Validator (`internal/validate/`)
Two-phase validation: structural (required fields, types) and semantic
(reference resolution, duplicate detection, "did you mean?" suggestions).

### IR (`internal/ir/`)
Intermediate Representation — the canonical, platform-neutral data format.
Features deterministic JSON serialization with sorted keys and content-addressed
hashing via SHA-256.

### Plan (`internal/plan/`)
Desired-state diff engine comparing IR resources against state entries.
Produces deterministic action lists (create/update/delete/noop).

### Apply (`internal/apply/`)
Idempotent applier with mark-and-continue partial failure handling.
Records per-resource results and updates state atomically.

### State (`internal/state/`)
Pluggable state backend interface with local JSON implementation.
Tracks resource lifecycle (applied/failed) with content hashes.

### Adapters (`internal/adapters/`)
Platform-specific artifact generators:
- **local/process**: Starts agents as local HTTP servers
- **docker**: Generates Docker images and Compose services
- **kubernetes**: Generates Kubernetes Deployment, Service, and ConfigMap manifests
- **compose**: Generates Docker Compose services with networking

### LLM Client (`internal/llm/`)
Abstracted LLM client interface supporting multiple providers (Anthropic, OpenAI).
Handles message formatting, token counting, streaming, and mock clients for testing.

### Agentic Loop (`internal/loop/`)
Orchestrates the turn-based interaction between user input, LLM responses, and
tool execution. Implements the ReAct strategy with configurable max turns and
token budgets. Supports both synchronous invocation and SSE streaming.

### Runtime Server (`internal/runtime/`)
HTTP server hosting agent endpoints:
- `POST /v1/agents/{name}/invoke` — synchronous invocation
- `POST /v1/agents/{name}/stream` — SSE streaming
- `POST /v1/agents/{name}/sessions` — session management
- `POST /v1/pipelines/{name}/run` — multi-agent pipeline execution
- `GET /v1/metrics` — Prometheus metrics endpoint
- `GET /healthz` — health check

Includes authentication middleware (API key / Bearer token), per-agent rate
limiting (token bucket), and metrics recording.

### Session Management (`internal/session/`)
Manages conversational sessions with message history persistence.
Two store implementations:
- **MemoryStore**: In-memory with configurable expiry (default for development)
- **RedisStore**: Redis-backed with TTL and key prefix (production)

### Memory (`internal/memory/`)
Manages conversation memory strategies:
- **SlidingWindow**: Fixed-size FIFO message history
- **Summary**: LLM-based conversation summarization above threshold

### Tool Registry (`internal/tools/`)
Registry for agent tools (skills). Maps tool names to executable handlers,
handles input validation and output formatting.

### Pipeline (`internal/pipeline/`)
Multi-agent pipeline execution with step dependencies and data flow.
Steps execute sequentially with output from one step feeding the next.

### Telemetry (`internal/telemetry/`)
Observability stack:
- **Metrics**: Prometheus-format counters (invocations, tokens, tool calls) and
  histograms (invocation duration) with per-agent labels
- **Logger**: Structured JSON logging via slog with correlation IDs
- **Traces**: Span-based tracing with parent-child relationships and exportable
  via the SpanExporter interface

### Secrets (`internal/secrets/`)
Secret resolution from external stores. Vault-compatible resolver with KV v2
HTTP API, token authentication, and in-memory caching with configurable TTL.

### Plugins (`internal/plugins/`)
WASM-based plugin system using wazero. Supports custom resource types,
validators, transforms, and lifecycle hooks.

### SDK Generator (`internal/sdk/generator/`)
Template-based code generator producing typed runtime API clients for Python,
TypeScript, and Go from RuntimeConfig. Generated clients include invoke, stream,
session management, and pipeline execution methods.

### Project Templates (`internal/templates/`)
Embedded `.ias` template files for `agentspec init`. Includes starter templates
for customer support, RAG chatbot, code review pipeline, data extraction, and
research assistant patterns.

### Events (`internal/events/`)
Structured event emitter for toolchain operations with correlation ID support.

### Policy (`internal/policy/`)
Policy engine for enforcing security constraints (deny/allow/require rules).

### MCP (`internal/mcp/`)
Model Context Protocol client for connecting to MCP servers via stdio transport.

## Threat Model

- **Secrets**: Never stored in plaintext; only `env()` and `store()` references.
  Vault resolver uses token-based auth with response caching.
- **Plugins**: Sandboxed in WASM with no filesystem/network access by default
- **State**: Local JSON file; no remote transmission in MVP
- **Imports**: Must be pinned to version/SHA; floating refs require explicit policy
- **Auth**: API key authentication on all endpoints except health check.
  Rate limiting per agent prevents abuse.
- **Sessions**: Redis store uses key prefixes and configurable TTL for isolation
