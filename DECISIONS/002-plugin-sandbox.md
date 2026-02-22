# ADR-002: WASM Plugin Sandboxing via wazero

## Status
Accepted

## Context
Plugins must run in a sandboxed environment to prevent unauthorized
filesystem/network access while maintaining single-binary distribution.

## Decision
Use wazero (pure-Go WASM runtime) for plugin sandboxing with
capability-based security.

## Consequences
- No CGo dependency; maintains single-binary distribution
- Plugins can be authored in any WASM-targeting language
- JSON I/O over WASI for IR exchange
- Memory and execution time limits enforced by host
