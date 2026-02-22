# ADR-003: Local JSON State Backend

## Status
Accepted

## Context
The toolchain needs to track applied resource state for idempotent
operations and drift detection.

## Decision
Use a local JSON file as the state backend for MVP.

## Consequences
- Simplest implementation for pluggable state interface
- Human-inspectable for debugging
- Deterministic serialization with sorted keys
- Sufficient for single-user local operation
- State interface enables future backends (SQLite, remote store)
