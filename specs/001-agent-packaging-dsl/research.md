# Research: Declarative Agent Packaging DSL

**Date**: 2026-02-22
**Branch**: `001-agent-packaging-dsl`

## Decision 1: Implementation Language

**Decision**: Go 1.25+ (latest stable: Go 1.26, Feb 2026)

**Rationale**:
- Repository lives in a Go workspace; team familiarity assumed.
- Go compiles to a single static binary (cross-platform).
- Excellent CLI ecosystem (cobra v1.10.2, pflag).
- Strong JSON marshaling with sorted keys for determinism.
- wazero v1.11.0 provides pure-Go WASM runtime (no CGo) for plugin
  sandboxing. Requires Go 1.24+ minimum.
- go-cmp v0.7.0 for test diffing. Requires Go 1.21+ minimum.
- Simpler than Rust for the scope of this MVP; faster iteration.
- Go 1.25+ chosen as minimum to stay within Go's supported release
  window (1.25 and 1.26) and satisfy wazero's Go 1.24+ requirement.

**Alternatives considered**:
- **Rust**: Stronger type system and performance guarantees, but
  steeper learning curve, slower compilation, and no clear benefit
  for a config-DSL toolchain at MVP scale. Could revisit for
  performance-critical paths post-MVP.

## Decision 2: Parser Technology

**Decision**: Hand-written recursive descent parser

**Rationale**:
- Full control over error messages with file, line, column, and
  contextual fix suggestions (FR-006). Go's own parser and HCL
  both use this approach for the same reason.
- Deterministic AST construction by design — no generator
  nondeterminism.
- Round-trip formatting (AST → canonical output) is straightforward
  with a hand-written printer.
- Zero external dependencies; pure Go; no build-time tooling.
- Maintenance cost is manageable for a DSL with ~10 keywords and
  ~6 resource types.

**Alternatives considered**:
- **PEG (pigeon)**: Requires code generation step; error messages
  are generic ("expected X"); harder to produce fix suggestions.
- **tree-sitter**: Requires CGo or WASM shim; designed for editors
  not compilers; no formatting support.
- **ANTLR4**: Java build dependency; Go target has ecosystem
  fragmentation; runtime is heavyweight for a config DSL.

## Decision 3: DSL Syntax Style

**Decision**: Custom sentence-like grammar (`.az` files)

**Rationale**:
- More readable to non-programmers than YAML/JSON (Principle VIII:
  English-Friendly Syntax).
- Avoids YAML pitfalls (indentation sensitivity, implicit type
  coercion of "yes"/"no", Norway problem).
- Enables strict canonical formatting with no stylistic variance
  (Principle IX).
- Proves separation of concerns (Principle IV): custom parser
  produces AST, semantics live in IR, adapters never see syntax.
- Natural-language keywords (`uses`, `connects to`, `exposes`)
  lower the barrier for domain experts.

**Alternatives considered**:
- **YAML-compatible with English-y keys**: Familiar to developers
  but doesn't satisfy English-friendly for non-programmers; YAML
  formatters allow stylistic variance; doesn't prove syntax/
  semantics separation as strongly.

## Decision 4: Plugin Sandboxing

**Decision**: WASM via wazero (pure Go), with process isolation
as fallback

**Rationale**:
- wazero is a pure-Go WASM runtime — no CGo, maintains single-
  binary distribution.
- Capability-based security: plugins cannot access filesystem or
  network unless explicitly granted.
- Plugins can be authored in any WASM-targeting language (Rust,
  TinyGo, AssemblyScript, C).
- JSON I/O over WASI for IR exchange.
- Plugin manifest can be embedded in WASM custom sections.
- Constitution prefers WASM sandboxing.

**Alternatives considered**:
- **wasmtime-go**: Requires CGo, breaks single-binary goal; limited
  to x86_64.
- **Process isolation only**: OS-level isolation is strong but heavy
  (process spawn overhead, IPC serialization cost, requires
  distributing separate plugin binaries).

## Decision 5: MVP Adapter Targets

**Decision**: Local MCP adapter + Docker Compose adapter

**Rationale**:
- **Local MCP**: The natural "native" target. Generates MCP server
  and client configurations for local runtimes (Claude Desktop,
  Cursor, etc.). Writes config files to disk.
- **Docker Compose**: Generates `docker-compose.yml` with services,
  volumes, and environment variables from the same IR. Proves
  portability — same IR, completely different deployment model.
- Both are thin mappers (Adapter Contract: no business logic).
- Both produce exportable artifacts (config files / compose files).
- Both are testable without external infrastructure.

**Alternatives considered**:
- **Kubernetes**: Too complex for MVP; would require cluster access
  for integration tests.
- **Claude Desktop config only**: Too narrow; doesn't prove
  portability sufficiently.

## Decision 6: SDK Generation

**Decision**: Hand-written thin SDK libraries per language, with
IR type definitions generated from JSON Schema

**Rationale**:
- No API server in MVP — OpenAPI/gRPC codegen would create
  impedance mismatch (assumes HTTP/RPC transport).
- Each SDK is small: read local state file, parse IR JSON, provide
  typed query methods, stream events from local runs.
- IR schema (JSON Schema) is the single source of truth for type
  generation.
- Hand-written query logic is ~200 lines per language; templates
  for types keep schema in sync.

**Alternatives considered**:
- **Full template-based codegen**: Higher maintenance for 3
  languages; type generation alone is sufficient.
- **OpenAPI codegen**: Requires API server; poor fit for file-based
  local state.
- **Protobuf/gRPC**: Same network-transport assumption; overkill
  for local file access.

## Decision 7: State Backend Format

**Decision**: Local JSON file

**Rationale**:
- Simplest implementation for pluggable state interface.
- Human-inspectable for debugging.
- Deterministic serialization with sorted keys.
- Sufficient for MVP single-user local operation.
- State interface is an internal Go interface; swapping backends
  later (SQLite, remote store) requires only a new implementation.

**Alternatives considered**:
- **SQLite**: More complex; unnecessary for MVP scale; adds CGo
  dependency (or pure-Go driver overhead).
- **BoltDB**: Append-only; binary format less inspectable; no clear
  benefit over JSON at MVP scale.
