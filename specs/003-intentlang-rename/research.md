# Research: IntentLang Rename

**Feature**: 003-intentlang-rename
**Date**: 2026-02-23

## Summary

Minimal research required — this is a rename/refactor with no technical unknowns. All decisions are user-specified (naming conventions, extension choice) or follow established patterns (backward compatibility, deprecation warnings).

## Decisions

### 1. File Extension Choice

- **Decision**: `.ias` (IntentLang AgentSpec)
- **Rationale**: User-specified. Mnemonic maps to the naming hierarchy: IntentLang (language) + AgentSpec (file type).
- **Alternatives considered**: `.il` (too short, conflicts with IL assembly), `.ilang` (verbose), `.as` (conflicts with ActionScript)

### 2. Naming Hierarchy

- **Decision**: IntentLang (language) → AgentSpec (file) → AgentPack (bundle)
- **Rationale**: User-specified. Clear three-level hierarchy: language name, individual file, distributable package.
- **Alternatives considered**: None — user explicitly defined all three terms.

### 3. Backward Compatibility Strategy

- **Decision**: `.az` files continue to work with stderr deprecation warning
- **Rationale**: Breaking existing workflows immediately would cause user friction. Deprecation period allows gradual migration.
- **Alternatives considered**: Hard break (reject `.az` immediately) — too disruptive for a rename.

### 4. Go Module Path

- **Decision**: Keep `github.com/szaher/designs/agentz` unchanged
- **Rationale**: Renaming Go module paths requires a major version bump and breaks all import references. The module path is internal infrastructure, not user-facing.
- **Alternatives considered**: Rename to `github.com/szaher/designs/agentspec` — deferred to separate feature if ever needed.

### 5. State File Migration Approach

- **Decision**: Auto-rename on first run (rename, not copy)
- **Rationale**: State file is a local artifact. Renaming is atomic and simple. Copying would leave orphaned files.
- **Alternatives considered**: Manual migration via `migrate` command — too much friction for a single file rename.

### 6. Plugin Directory Strategy

- **Decision**: Dual-path resolution with deprecation warning (no file moving)
- **Rationale**: Moving plugin files could break running systems. Read-only fallback is safe and non-destructive.
- **Alternatives considered**: Auto-move plugins to new directory — risky if plugins are symlinked or shared.

## No Unknowns Remaining

All technical decisions are resolved. No external dependencies, no API integrations, no performance concerns. Proceed directly to task generation.
