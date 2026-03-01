# Implementation Plan: Security Hardening & Compliance

**Branch**: `007-security-hardening` | **Date**: 2026-03-01 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/007-security-hardening/spec.md`

## Summary

Harden the AgentSpec runtime server, tool executors, session management, and policy engine for internet-facing deployment with untrusted .ias files. Changes span 12 packages in `internal/` — fixing predictable session IDs, timing-vulnerable auth, unsandboxed inline tools, stub policy enforcement, missing server timeouts/limits, SSRF-vulnerable HTTP tools, unprotected command tools, race conditions, swallowed errors, CORS misconfiguration, and plugin output leaks.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: wazero v1.11.0 (WASM sandbox), cobra v1.10.2 (CLI), anthropic-sdk-go, go-mcp-sdk
**Storage**: Local JSON state file (`.agentspec.state.json`), in-memory session store, Redis session store
**Testing**: `go test` with integration tests in `integration_tests/`; unit tests to be added per-package
**Target Platform**: Linux/macOS/Windows (CLI + HTTP server)
**Project Type**: CLI tool + HTTP runtime server
**Performance Goals**: Session ID generation <1ms; auth check <1ms; sandbox startup <5s cold, <100ms warm
**Constraints**: No CGo dependency (wazero is pure Go); backward-compatible CLI flags
**Scale/Scope**: ~22K LOC Go; 12 packages modified; 20 functional requirements

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Determinism | PASS | Security changes don't affect AST/IR/plan determinism |
| II. Idempotency | PASS | Apply behavior unchanged; policy checks are read-only |
| III. Portability | PASS | All changes are platform-neutral (pure Go, no CGo) |
| IV. Separation of Concerns | PASS | Security logic stays in dedicated packages (auth, policy, tools) |
| V. Reproducibility | PASS | Signed packages requirement aligns with reproducibility |
| VI. Safe Defaults | PASS | Core goal: secure-by-default (block command tools, require auth) |
| VII. Minimal Surface Area | PASS | No new DSL keywords; config changes only |
| VIII. English-Friendly Syntax | N/A | No syntax changes |
| IX. Canonical Formatting | N/A | No formatter changes |
| X. Strict Validation | PASS | Policy engine will produce actionable error messages |
| XI. Explicit References | PASS | `signed packages` requirement enforces pinned+signed imports |
| XII. No Hidden Behavior | PASS | All security enforcement is logged and discoverable |

**Security & Supply-Chain (Operational)**: PASS — This feature directly implements the policy layer, secret protection, and signed packages requirements.

**Observability (Operational)**: PASS — Error transparency (FR-016/017) and auth failure logging align with observability requirements.

**Gate result: ALL PASS. No violations to justify.**

## Project Structure

### Documentation (this feature)

```text
specs/007-security-hardening/
├── plan.md              # This file
├── research.md          # Phase 0: technology decisions
├── data-model.md        # Phase 1: entity and config schemas
├── quickstart.md        # Phase 1: verification guide
├── contracts/           # Phase 1: interface contracts
│   ├── auth-contract.md
│   ├── policy-contract.md
│   ├── sandbox-contract.md
│   └── tool-security-contract.md
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── auth/
│   ├── key.go              # Existing: ValidateKey (no changes)
│   ├── middleware.go        # Existing: auth middleware (add auth failure rate limiting)
│   └── ratelimit.go         # Existing: rate limiter (add eviction, auth failure tracking)
├── session/
│   ├── store.go             # Existing: Store interface (no changes)
│   ├── memory_store.go      # Modify: replace generateID with crypto/rand
│   ├── redis_store.go       # Modify: replace generateSessionID with shared crypto/rand
│   └── id.go                # NEW: shared generateSecureID function
├── runtime/
│   └── server.go            # Modify: use auth.ValidateKey, add timeouts, MaxBytesReader, CORS config
├── tools/
│   ├── command.go           # Modify: add allowlist validation, safe env
│   ├── http.go              # Modify: add SSRF check, response limit, safe body rendering
│   ├── inline.go            # Modify: add sandbox wrapper, enforce memory limit
│   ├── env.go               # NEW: shared SafeEnv() utility for command and inline tools
│   ├── ssrf.go              # NEW: private IP detection and URL validation
│   └── allowlist.go         # NEW: binary allowlist validation
├── policy/
│   └── policy.go            # Modify: implement checkRequirement for 4 types
├── mcp/
│   └── pool.go              # Modify: fix TOCTOU race with singleflight or per-key lock
├── secrets/
│   └── redact.go            # Modify: fix WithAttrs/WithGroup mutex sharing
├── llm/
│   ├── anthropic.go         # Modify: handle JSON errors instead of discarding
│   └── openai.go            # Modify: handle JSON errors instead of discarding
├── frontend/
│   └── sse.go               # Modify: configurable CORS origin
├── plugins/
│   └── host.go              # Modify: capture stdout/stderr to buffers
└── sandbox/                 # NEW package: inline tool sandbox using wazero or OS-level isolation
    ├── sandbox.go           # Sandbox interface
    ├── process.go           # OS-level process isolation (cgroups/ulimit)
    └── noop.go              # No-op sandbox for testing

cmd/agentspec/
├── run.go                   # Modify: add --no-auth flag
└── dev.go                   # Modify: add --cors-origin flag, auto-localhost CORS

integration_tests/
├── auth_test.go             # Existing: extend with timing, rate limiting, no-auth tests
├── tools_test.go            # Existing: extend with allowlist, SSRF tests
├── policy_test.go           # NEW: policy enforcement tests
└── security_test.go         # NEW: comprehensive security integration tests
```

**Structure Decision**: This feature modifies existing packages in-place (no new top-level directories). One new package `internal/sandbox/` for inline tool isolation. Two new files in `internal/tools/` for SSRF and allowlist validation. Changes follow existing Go package conventions.

## Complexity Tracking

> No constitution violations — no complexity justification needed.
