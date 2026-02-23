# ADR-001: Hand-Written Recursive Descent Parser

## Status
Accepted

## Context
The IntentLang DSL requires parsing with high-quality error messages
including file:line:col positions and contextual fix suggestions.

## Decision
Use a hand-written recursive descent parser in Go, following the
approach used by Go's own parser and HCL.

## Consequences
- Full control over error messages and recovery
- Deterministic AST construction
- Round-trip formatting is straightforward
- Zero external dependencies
- Maintenance cost is manageable for ~10 keywords and ~6 resource types
