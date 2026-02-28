# Research: Agent Compilation & Deployment Framework

**Feature**: 006-agent-compile-deploy
**Date**: 2026-02-28

## R1: Compilation Strategy

**Decision**: Go binary embedding with `go:embed`

**Rationale**: Embed the agent's IR/RuntimeConfig as JSON into a Go binary that contains the full AgentSpec runtime. The compilation pipeline is: `.ias` → parse → IR → JSON → `go:embed` into template `main.go` → `go build` → standalone binary. This is the simplest approach with excellent performance and deterministic output.

**Alternatives considered**:
- **Go code generation + compilation**: Generates Go source from IR, then compiles. Produces marginally smaller binaries (~1-2% less) but significantly more complex to implement and maintain. Code generation is fragile (escaping, type mismatches). Rejected: complexity not justified by marginal gains.
- **Template-based binary shell**: Pre-build a universal runtime binary, patch in config at compile time using tools like `ember`. Fastest build (<1s) and doesn't require Go toolchain on build machine. Rejected: binary patching is fragile and platform-specific; maintaining a matrix of pre-built shells for all OS/arch combinations is a maintenance burden.

**Key details**:
- Build time: 2-5 seconds (standard Go compilation)
- Binary size: ~15-16MB (15MB runtime + <1MB embedded config)
- Cross-compilation: Native via `GOOS`/`GOARCH` environment variables
- Deterministic: Yes, with `-trimpath -ldflags="-s -w"` (Go 1.21+ guarantees reproducible builds)
- Requires Go toolchain on build machine (not on target machine)
- Startup overhead: ~1ms for JSON unmarshal of embedded config

## R2: Framework Code Generation Targets

**Decision**: Template-based code generation for 4 frameworks, each producing a minimum viable project

**Rationale**: Each framework has distinct concepts and project structures. The compiler's IR provides a common input; framework-specific templates produce idiomatic output code. The generated code is "ejectable" — users own it after generation.

### CrewAI (Python)
- **Mapping**: Agent → CrewAI Agent, Skill → Tool (`@tool` decorator), Pipeline → Crew, Prompt → role/goal/backstory
- **Min files**: `pyproject.toml`, `main.py`, `crew.py`, `config/agents.yaml`, `config/tasks.yaml`, `tools/__init__.py` (~100-150 LOC)
- **Gaps**: No max_turns, no token budget, no per-agent timeout in CrewAI. Loop strategies partially map to Process.sequential/hierarchical.

### LangGraph (Python)
- **Mapping**: Agent → Node function, Pipeline → StateGraph, Skill → Tool (bind_tools), Control flow → conditional edges
- **Min files**: `requirements.txt`, `graph.py`, `tools.py`, `main.py` (~80-120 LOC)
- **Strengths**: Best mapping for AgentSpec's control flow (conditional edges map directly). State management via TypedDict + reducers.
- **Gaps**: max_turns/retry require custom state logic; token budget not natively tracked.

### LlamaStack (Python)
- **Mapping**: Agent → Agent (high-level API), Skill → custom tool functions, Prompt → instructions parameter
- **Min files**: `requirements.txt`, `agent.py` (~30-50 LOC)
- **Gaps**: No native multi-agent orchestration (manual coordination needed). No pipelines, no conditional flows. Simplest framework = fewest features.

### LlamaIndex (Python)
- **Mapping**: Agent → ReActAgent, Skill → FunctionTool, Pipeline → Workflow (event-driven), Query → QueryEngine
- **Min files**: `requirements.txt`, `tools.py`, `agent.py`, `main.py` (~60-100 LOC)
- **Strengths**: Workflow pattern (event-driven) maps well to AgentSpec pipelines. Strong RAG support.
- **Gaps**: max_turns not exposed, retry requires custom logic.

### Common gaps across all frameworks
- Token budgets, per-agent timeouts, and retry strategies must be implemented as generated application logic rather than framework features.
- AgentSpec validation rules have no framework equivalent — must be generated as wrapper functions around agent responses.
- Evaluation test cases are AgentSpec-specific — generated as separate test scripts.

## R3: Import System & Dependency Resolution

**Decision**: Go-style Minimal Version Selection (MVS) with Terraform-like source flexibility

**Rationale**: MVS is deterministic without lock files, simple to implement (no SAT solver), and proven at scale in the Go ecosystem. Combined with support for local paths and Git-based sources, it provides the flexibility users expect.

**Import syntax**:
```
import "./skills/search.ias"                          // local relative
import "agentspec/web-tools" version "1.2.0"          // versioned package
import "github.com/user/agents" version "^2.1.0"      // Git-based
```

**Resolution algorithm**: MVS — when multiple versions of the same dependency are required, select the minimum version that satisfies all constraints. No SAT solver needed.

**Circular dependency detection**: Tarjan's Strongly Connected Components (SCC) algorithm — O(V+E), finds all cycles in one pass, produces clear error messages showing the full cycle path.

**Lock file**: `.agentspec.lock` — records resolved versions for reproducible builds. Constitution Principle XI (Explicit References) requires all external imports be pinned.

**Alternatives considered**:
- npm-style resolution (latest by default): Rejected — non-deterministic, requires lock file to be reproducible.
- Cargo-style resolution (SAT solver): Rejected — more complex to implement, MVS is sufficient for this ecosystem size.

## R4: Expression Evaluation for Control Flow

**Decision**: `expr-lang/expr` library

**Rationale**: Fastest Go expression evaluator (70 ns/op vs 91 ns/op for CEL), sandboxed (no side effects, always terminates), statically typed at compile time, readable syntax for non-programmers. Production-proven at Uber, Google Cloud, Alibaba, ByteDance.

**Expression syntax examples**:
```
input.category == "support"
input.priority >= 5 and input.status != "resolved"
input.metadata.type in ["urgent", "critical"]
steps.validate.status == "success"
```

**Supported operations**: Property access, comparisons (==, !=, >, <, >=, <=), boolean logic (and, or, not), type checks, `in` operator for collections. No arbitrary function calls, no I/O, no loops within expressions.

**Alternatives considered**:
- `google/cel-go` (Common Expression Language): Excellent choice, battle-tested at Google, but 23% slower than expr and heavier dependency. Better suited if Kubernetes/protobuf integration needed. Rejected: performance and dependency weight.
- Custom recursive descent evaluator: Maximum control but requires significant testing and maintenance for security. Rejected: reinventing the wheel.

## R5: Built-in Frontend

**Decision**: Vanilla JavaScript + SSE, embedded via `go:embed`

**Rationale**: Zero build step, ~5-10KB bundle, native browser SSE API for streaming, industry standard for AI chat (ChatGPT, Claude use SSE). No frontend build toolchain required — HTML/CSS/JS files are directly embedded into the Go binary.

**Streaming protocol**: Server-Sent Events (SSE) — unidirectional server→client streaming over HTTP. User sends requests via HTTP POST, receives streaming responses via SSE. Automatic reconnection built into browser EventSource API.

**Bundle composition**: Single `index.html` with inline CSS and JavaScript, or a small directory of static files. Total: 5-10KB embedded into the Go binary (negligible size impact on a 15MB binary).

**Precedent**: Prometheus, CockroachDB, and Gitea all embed web UIs into Go binaries using `go:embed`.

**Alternatives considered**:
- VanJS (1KB reactive framework): Good future upgrade path if UI complexity grows. No build step required. Defer to Phase 2.
- Preact/Svelte: Excellent frameworks but require build toolchain (webpack/vite). Overkill for a supplementary chat UI. Rejected: unnecessary complexity.
- HTMX + Go templates: Good for traditional web apps but not ideal for real-time streaming chat. 14KB library overhead. Rejected: SSE + vanilla JS is simpler and better suited.
- WebSocket: Bidirectional but overkill for server→client streaming. No automatic reconnection. Rejected: SSE is simpler and sufficient.

**Constitution note**: The constitution lists "Complex UI" as a non-goal. This frontend is a minimal chat interface (~5KB), not a complex control plane. Justified as a developer tool and basic interaction surface, not a full UI product.

## R6: Package Registry

**Decision**: Phased approach — Git-based MVP → GOPROXY-style HTTP server → OCI registry

**Rationale**: Start with zero infrastructure (Git repos as packages), add a dedicated registry when the ecosystem needs caching and speed, then support enterprise OCI registries for organizations with existing container infrastructure.

### Phase 0 (MVP): Git-based registry
- Packages are Git repositories with version tags
- `agentspec install github.com/user/agentpack@v1.2.3`
- Resolution: clone/fetch → checkout tag → read manifest
- Local cache: `~/.agentspec/cache/`
- Zero infrastructure, instant availability

### Phase 1: GOPROXY-style HTTP server
- Simple HTTP API: `GET /<package>/@v/list`, `GET /<package>/@v/<version>.zip`
- Storage: local disk (MVP), S3/GCS (later)
- Proxy to Git for cache misses
- `agentspec publish` command for package authors
- Namespace authentication (GitHub OIDC)

### Phase 2: OCI registry support
- Use ORAS client library to push/pull AgentPacks as OCI artifacts
- Custom media type: `application/vnd.agentspec.pack.v1+tar+gzip`
- Compatible with Docker Hub, GHCR, ACR, GCR, Harbor
- Leverage existing enterprise auth and CDN

**Alternatives considered**:
- Start with OCI immediately: Rejected — too complex for MVP, OCI 1.1 features not universally supported.
- npm-style centralized registry: Rejected — requires hosting and maintenance from day one.
- Simple file-based directory: Rejected — no versioning intelligence, doesn't scale.
