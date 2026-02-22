# Implementation Plan: Declarative Agent Packaging DSL

**Branch**: `001-agent-packaging-dsl` | **Date**: 2026-02-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-agent-packaging-dsl/spec.md`

## Summary

Build the Agentz toolchain: a CLI (`agentz`) that parses an
English-friendly declarative DSL (`.az` files), validates
definitions, compiles to a canonical IR, plans and applies desired
state idempotently via pluggable adapters, and generates SDKs for
Python, TypeScript, and Go. The MVP delivers a hand-written
recursive descent parser, two adapters (Local MCP and Docker
Compose), one example plugin (WASM via wazero), and integration
tests with golden fixtures.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: wazero v1.11.0 (WASM plugin sandbox),
  go-cmp v0.7.0 (test diffing), cobra v1.10.2 (CLI framework)
**Storage**: Local JSON file (state backend)
**Testing**: `go test` with golden fixture comparison
**Target Platform**: Cross-platform CLI (Linux, macOS, Windows)
**Project Type**: CLI tool + compiler + SDK generator
**Performance Goals**: Parse/validate/plan 500-resource definitions
  in under 5 seconds
**Constraints**: Single static binary, offline-capable, no external
  runtime dependencies for core operations
**Scale/Scope**: Single-user local operation; definitions up to
  hundreds of resources

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1
design.*

| #    | Principle            | Status  | Evidence                                    |
|------|----------------------|---------|---------------------------------------------|
| I    | Determinism          | ✅ PASS | Sorted map keys in IR serialization; content-addressed hashing for exports; golden fixture tests verify byte-identical outputs |
| II   | Idempotency          | ✅ PASS | State file diff drives plan/apply; no mutations when desired == actual; double-apply integration tests |
| III  | Portability          | ✅ PASS | DSL is platform-neutral; two adapters (Local MCP, Docker Compose) consume only IR |
| IV   | Separation           | ✅ PASS | Parser → AST → IR pipeline; adapters never see DSL source; syntax isolated in parser package |
| V    | Reproducibility      | ✅ PASS | Go modules with checksums; package refs pinned to version/SHA; exports content-hashed |
| VI   | Safe Defaults        | ✅ PASS | Validator rejects plaintext secrets; resources default to least-privilege; policy layer blocks unsafe configs |
| VII  | Minimal Surface      | ✅ PASS | ~6 built-in resource types; new keywords require example in `examples/` |
| VIII | English-Friendly     | ✅ PASS | Custom grammar with natural-language keywords (`uses`, `connects to`, `exposes`) |
| IX   | Canonical Formatting | ✅ PASS | `agentz fmt` round-trips through AST; single formatter, zero options |
| X    | Strict Validation    | ✅ PASS | Two-phase validation (structural + semantic); errors include file:line:col + fix hint |
| XI   | Explicit References  | ✅ PASS | Import statements require version/SHA; floating refs require `allow floating` policy flag |
| XII  | No Hidden Behavior   | ✅ PASS | Plugin transforms declared in manifest; plan output shows all mutations; no implicit transforms |

All 12 principles pass. No violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/001-agent-packaging-dsl/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── cli.md
│   ├── ir-schema.md
│   ├── adapter-interface.md
│   ├── plugin-manifest.md
│   └── sdk-api.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/agentz/                    # CLI entrypoint (cobra)
internal/
├── parser/                    # Lexer + recursive descent parser
├── ast/                       # AST node types
├── formatter/                 # Canonical formatter (AST → .az)
├── ir/                        # IR types + deterministic serializer
├── validate/                  # Schema + semantic validators
├── plan/                      # Desired-state diff engine
├── apply/                     # Idempotent applier
├── state/                     # State backend interface + local JSON
├── plugins/                   # Plugin host + wazero WASM runtime
├── adapters/                  # Adapter registry + interface
│   ├── local/                 # Local MCP adapter
│   └── compose/               # Docker Compose adapter
├── sdk/                       # SDK generator
│   ├── generator/             # Template engine for type codegen
│   ├── python/                # Python SDK templates
│   ├── typescript/            # TypeScript SDK templates
│   └── go/                    # Go SDK templates
├── events/                    # Structured event emitter
└── policy/                    # Policy engine (security constraints)

examples/                      # At least 6 .az example configs
integration_tests/             # End-to-end golden fixture tests
spec/
├── spec.md                    # Normative language specification
└── ir.schema.json             # Canonical IR JSON Schema

plugins/
└── monitor/                   # Example plugin (WASM)

sdk/
├── python/                    # Generated Python SDK
├── typescript/                # Generated TypeScript SDK
└── go/                        # Generated Go SDK

ARCHITECTURE.md
CHANGELOG.md
DECISIONS/
```

**Structure Decision**: Single-project Go layout following the
standard `cmd/` + `internal/` convention. SDK outputs are generated
into top-level `sdk/` directories per language. The `spec/`
directory contains the normative language spec and IR schema
(required constitution artifacts). Adapters are internal packages
with thin implementations. The example plugin lives under
`plugins/monitor/` as a standalone WASM module.

## Complexity Tracking

No constitution violations detected. Table intentionally empty.
