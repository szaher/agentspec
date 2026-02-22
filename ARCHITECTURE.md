# Architecture: Agentz Toolchain

## Overview

Agentz is a declarative agent packaging and deployment toolchain.
It parses `.az` definition files through a pipeline that produces
platform-neutral artifacts via pluggable adapters.

## Data Flow

```
.az source → Lexer → Tokens → Parser → AST → Validator → IR → Adapter → Artifacts
                                                              ↓
                                                         State File
```

## Components

### Parser (`internal/parser/`)
Hand-written recursive descent parser producing AST from `.az` source.
Includes lexer/tokenizer with keyword recognition and source position tracking.

### AST (`internal/ast/`)
Abstract syntax tree node types for all resource kinds with source position
tracking for error reporting.

### Formatter (`internal/formatter/`)
Canonical formatter producing deterministic `.az` output from AST.
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
Platform-specific artifact generators. Two built-in:
- **local-mcp**: Generates MCP server/client configuration files
- **docker-compose**: Generates Docker Compose services and config

### Plugins (`internal/plugins/`)
WASM-based plugin system using wazero. Supports custom resource types,
validators, transforms, and lifecycle hooks.

### SDK Generator (`internal/sdk/generator/`)
Generates typed SDKs for Python, TypeScript, and Go from IR schema.

### Events (`internal/events/`)
Structured event emitter for toolchain operations with correlation ID support.

### Policy (`internal/policy/`)
Policy engine for enforcing security constraints (deny/allow/require rules).

## Threat Model

- **Secrets**: Never stored in plaintext; only `env()` and `store()` references
- **Plugins**: Sandboxed in WASM with no filesystem/network access by default
- **State**: Local JSON file; no remote transmission in MVP
- **Imports**: Must be pinned to version/SHA; floating refs require explicit policy
