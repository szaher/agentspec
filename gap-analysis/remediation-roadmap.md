# Remediation Roadmap

**Last updated:** 2026-03-01

## Now (0–2 weeks): Must-Fix Blockers, Security, Data Integrity

### Week 1: Critical Security & Data Integrity

| # | Action | IDs | Effort | Risk |
|---|--------|-----|--------|------|
| 1 | Fix session ID generation with `crypto/rand` | BUG-002, GAP-003, SEC-001 | S (2h) | Low — drop-in replacement |
| 2 | Use constant-time API key comparison in server | BUG-018, GAP-006, SEC-002 | S (30min) | Low — 1 line change |
| 3 | Make state file writes atomic (temp + rename) | BUG-005, GAP-002 | S (2h) | Low — standard pattern |
| 4 | Implement state file locking (flock) | BUG-004 | S (4h) | Low — prevents concurrent corruption |
| 5 | Add HTTP server timeouts | BUG-025, GAP-005, PERF-007 | S (1h) | Low — standard Go pattern |
| 6 | Add request body size limits | BUG-024, GAP-009, SEC-014 | S (2h) | Low — `MaxBytesReader` wrapper |
| 7 | Log warning when no API key configured | GAP-007, SEC-003 | S (1h) | Low — log statement |
| 8 | Fix swallowed JSON errors in LLM clients | BUG-003 | S (2h) | Medium — may surface previously hidden errors |

### Week 2: Security Hardening

| # | Action | IDs | Effort | Risk |
|---|--------|-----|--------|------|
| 9 | Add SSRF protection to HTTP tool | GAP-008, SEC-006 | M (4h) | Medium — need private IP range list |
| 10 | Add HTTP response body size limit | GAP-010, BUG-023, SEC-007 | S (1h) | Low |
| 11 | Implement policy `checkRequirement()` | GAP-001, SEC-016 | M (8h) | Medium — needs design for each requirement type |
| 12 | Add command tool binary allowlist | GAP-012, SEC-005 | M (4h) | Medium — need to define default allowlist |
| 13 | Replace Redis `KEYS` with `SCAN` | GAP-011, PERF-009 | S (2h) | Low — API-compatible |
| 14 | Make CORS configurable | GAP-019, BUG-019, SEC-012 | S (2h) | Low |
| 15 | Add CI security scanning (govulncheck + gosec) | FEAT-002 | S (2h) | Low — CI config only |
| 16 | Add CI race detection job | FEAT-004 | S (1h) | Low — CI config only |

**Sequencing rationale:** Security fixes first because they have the highest blast radius. State integrity next because data loss is irreversible. CI improvements enable catching regressions going forward.

**Quick wins:** Items 1, 2, 5, 6, 7, 10, 13, 14, 15, 16 are all < 2 hours each and can be done in a single day.

## Next (2–6 weeks): Missing Core Features, Reliability, Performance

### Weeks 3–4: Reliability & Testing Foundation

| # | Action | IDs | Effort | Risk |
|---|--------|-----|--------|------|
| 17 | Unit tests for auth, secrets, tools packages | FEAT-001, QE-001 | L (3d) | Low — additive |
| 18 | Unit tests for session, state, policy packages | FEAT-001, QE-001 | L (3d) | Low — additive |
| 19 | Fix rate limiter bucket eviction | BUG-007, GAP-024 | S (4h) | Low — background goroutine |
| 20 | Fix session memory store eviction | BUG-008, GAP-017 | S (4h) | Low — background goroutine |
| 21 | Fix Redis SaveMessages race condition | BUG-022, GAP-018 | M (4h) | Medium — Redis data structure change |
| 22 | Fix command/inline tool environment | BUG-006, SEC-011 | S (2h) | Medium — changing env affects tool behavior |
| 23 | Add CI coverage reporting | FEAT-003, QE-002 | S (2h) | Low — CI config only |
| 24 | Enable additional linters | QE-004 | M (4h) | Low — may surface many existing warnings |

### Weeks 5–6: Product Completeness

| # | Action | IDs | Effort | Risk |
|---|--------|-----|--------|------|
| 25 | Update README with all 19 CLI commands | GAP-013, UX-001 | S (2h) | Low |
| 26 | Fix compiler targets — generate functional stubs | GAP-014, UX-007 | L (2d) | Medium — need to understand each framework |
| 27 | Add frontend loading, error, empty states | GAP-015, UX-004 | M (4h) | Low |
| 28 | Consolidate duplicate rate limiter implementations | BUG-036, QE-008 | M (4h) | Low |
| 29 | Consolidate duplicate session ID generators | QE-009 | S (1h) | Low |
| 30 | Fix tool result-action correlation by ID | BUG-010 | M (4h) | Medium — behavior change |
| 31 | Add structured logging with correlation IDs | FEAT-010 | M (8h) | Low — additive |
| 32 | Unit tests for loop, pipeline, llm packages | FEAT-001, QE-001 | L (3d) | Low — additive |

**Sequencing rationale:** Testing foundation first to catch regressions from subsequent changes. Product completeness builds on the stable base.

**Risk mitigation:** Each reliability fix should include corresponding unit tests. Run race detector on all changes. Review Redis changes in staging before production.

## Later (6+ weeks): New Features, Refactors, Tech Debt

### Phase 1: Developer Experience (Weeks 7–10)

| # | Action | IDs | Effort | Risk |
|---|--------|-----|--------|------|
| 33 | Add TLS support to HTTP server | GAP-023, SEC-013 | M (8h) | Low |
| 34 | Implement `eval` with live LLM invocation | GAP-016, UX-003 | L (2d) | Medium |
| 35 | Replace dev mode polling with fsnotify | GAP-021, BUG-028, UX-008 | M (4h) | Low |
| 36 | Implement `publish --sign` or remove flag | GAP-020, UX-002 | L (3d) if implementing; S (1h) if removing | Medium |
| 37 | Sandbox inline tool execution | GAP-004, SEC-004 | XL (2w) | High — major architecture change |
| 38 | LLM contract tests (recorded responses) | QE-007, FEAT-016 | L (3d) | Low |

### Phase 2: Production Readiness (Weeks 11–14)

| # | Action | IDs | Effort | Risk |
|---|--------|-----|--------|------|
| 39 | RBAC / multi-user access control | FEAT-005 | XL (3w) | High — major feature |
| 40 | Agent versioning and rollback | FEAT-007 | L (1w) | Medium |
| 41 | Agent observability dashboard | FEAT-006 | L (1w) | Low |
| 42 | Cost tracking and budgets | FEAT-015 | M (3d) | Low |
| 43 | OpenTelemetry integration | FEAT-018 | L (1w) | Medium |
| 44 | Release automation (goreleaser) | QE-010 | M (4h) | Low |

### Phase 3: Ecosystem (Weeks 15+)

| # | Action | IDs | Effort | Risk |
|---|--------|-----|--------|------|
| 45 | Multi-model fallback chains | FEAT-009 | L (1w) | Medium |
| 46 | Agent guardrails / content filtering | FEAT-011 | L (1w) | Medium |
| 47 | Plugin marketplace | FEAT-013 | XL (3w) | High |
| 48 | Interactive agent testing / playground | FEAT-008 | L (2w) | Medium |
| 49 | Conversation export and replay | FEAT-014 | M (3d) | Low |
| 50 | Agent-to-agent communication protocol | FEAT-017 | XL (3w) | High |

**Sequencing rationale:** Developer experience improvements reduce friction for adopters. Production readiness is needed before scaling usage. Ecosystem features build on a stable, well-tested platform.

## Risk Summary

| Risk | Mitigation |
|------|-----------|
| Inline tool sandboxing is a major undertaking | Consider short-term: document risk, add `--allow-inline` flag, default to deny. Long-term: WASM-based sandbox |
| Redis data structure change may break existing sessions | Provide migration script; version the storage format |
| Enabling more linters may surface hundreds of warnings | Enable incrementally; use `//nolint` for known-safe cases |
| Policy enforcement may break existing valid `.ias` files | Add `--policy=warn` mode that reports violations without blocking |
| Multi-user auth is a major scope increase | Start with team-oriented auth (shared API keys with names); full RBAC later |
