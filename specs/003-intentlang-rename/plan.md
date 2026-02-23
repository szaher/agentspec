# Implementation Plan: IntentLang Rename

**Branch**: `003-intentlang-rename` | **Date**: 2026-02-23 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-intentlang-rename/spec.md`

## Summary

Rename the project's language to IntentLang, file extension from `.az` to `.ias`, CLI binary from `agentz` to `agentspec`, and update all internal references, documentation, examples, tests, CI, state files, and plugin paths accordingly. Maintain backward compatibility for `.az` files with deprecation warnings.

## Technical Context

**Language/Version**: Go 1.25+ (existing codebase)
**Primary Dependencies**: No new dependencies — this is a rename/refactor of existing code
**Storage**: State file rename (`.agentz.state.json` → `.agentspec.state.json`)
**Testing**: Existing integration tests updated with new extension/paths
**Target Platform**: Same as existing (CLI tool, cross-platform)
**Project Type**: CLI refactor
**Performance Goals**: N/A — rename has no performance impact
**Constraints**: Backward compatibility required for `.az` files during transition
**Scale/Scope**: 50+ files affected across source, tests, docs, examples, CI

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Determinism | PASS | Rename does not affect determinism — same inputs produce same outputs |
| II. Idempotency | PASS | Apply behavior unchanged; only file paths and naming change |
| III. Portability | PASS | DSL remains platform-neutral; rename is cosmetic |
| IV. Separation of Concerns | PASS | No semantic changes to IR or AST |
| V. Reproducibility | PASS | Pinned versions unchanged |
| VI. Safe Defaults | PASS | No security changes |
| VII. Minimal Surface Area | PASS | No new keywords or constructs |
| VIII. English-Friendly Syntax | PASS | IntentLang name is more descriptive than "az" |
| IX. Canonical Formatting | PASS | Formatter behavior unchanged |
| X. Strict Validation | PASS | Validator behavior unchanged; deprecation warning added |
| XI. Explicit References | PASS | No dependency changes |
| XII. No Hidden Behavior | PASS | Deprecation warnings are explicit and visible |

**Review Gates**:
- Spec updated: Yes (this spec)
- Examples updated: Yes (renamed from .az to .ias)
- Integration tests updated: Yes (updated extensions and paths)
- Formatter stable: Yes (formatter behavior unchanged)

All gates pass. No violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/003-intentlang-rename/
├── plan.md              # This file
├── research.md          # Phase 0 output (minimal — no unknowns)
├── quickstart.md        # How to verify the rename
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code Changes

```text
Changes across existing structure (no new directories):

cmd/agentz/ → cmd/agentspec/     # Directory rename
  ├── main.go                     # Update binary name, help text
  ├── fmt.go                      # Update deprecation warning for .az
  ├── validate.go                 # Update deprecation warning for .az
  ├── plan.go                     # Update deprecation warning for .az
  ├── apply.go                    # Update deprecation warning for .az
  ├── diff.go                     # Update deprecation warning for .az
  ├── export.go                   # Update deprecation warning for .az
  ├── sdk.go                      # Update help text
  ├── version.go                  # Update version output text
  └── migrate.go                  # Update to handle .az → .ias rename

internal/
  ├── cli/deprecation.go          # NEW: Shared deprecation warning helper + .az/.ias conflict detection
  ├── state/local.go              # .agentz.state.json → .agentspec.state.json + migration
  ├── plugins/loader.go           # ~/.agentz/plugins/ → ~/.agentspec/plugins/ + fallback
  ├── plugins/host.go             # Comment update
  ├── plugins/validate.go         # Update "agentz" string references
  ├── plugins/transform.go        # Update "agentz" string references
  ├── sdk/generator/generator.go  # Update generated SDK naming
  ├── adapters/adapter.go         # Update "agentz" string references
  ├── adapters/compose/compose.go # Update generated file comments
  ├── adapters/local/local.go     # Update "agentz" string references
  ├── apply/apply.go              # Update "agentz" string references
  ├── plan/{plan,format,drift}.go # Update "agentz" string references
  ├── policy/{enforce,policy}.go  # Update "agentz" string references
  ├── validate/{environment,semantic,structural}.go # Update "agentz" references
  ├── ir/lower.go                 # Update "agentz" string references
  ├── parser/{lexer,parser,token}.go # Update .az references in comments/error messages
  └── ast/ast.go                  # Update .az references in comments/error messages

examples/*/*.az → examples/*/*.ias  # 10 file renames
examples/*/README.md                # 10 README updates
examples/README.md                  # Top-level README update

.github/workflows/ci.yml           # Update globs and binary name
.gitignore                          # Update binary pattern
ARCHITECTURE.md                     # Update references
CHANGELOG.md                        # Update references
spec/spec.md                        # Update references

integration_tests/*.go              # 8 test files: extension + state refs
```

**Structure Decision**: No new directories. This is a rename/refactor across the existing structure. The `cmd/agentz/` directory is renamed to `cmd/agentspec/`. The Go module path remains `github.com/szaher/designs/agentz`.

## Rename Strategy

### Phase Approach

1. **Extension first** (P1): Update parser/CLI to accept `.ias`, add `.az` deprecation warnings, rename example files
2. **Documentation second** (P2): Update all READMEs, docs, specs
3. **Binary last** (P3): Rename `cmd/agentz/` → `cmd/agentspec/`, update CI, state file, plugin paths

### Key Decisions

1. **Backward compatibility**: `.az` files continue to work with a deprecation warning to stderr. This is implemented in the CLI command layer, not the parser — the parser accepts any file content regardless of extension.

2. **State file migration**: On startup, if `.agentspec.state.json` doesn't exist but `.agentz.state.json` does, auto-rename it and print a migration notice.

3. **Plugin directory fallback**: Check `~/.agentspec/plugins/` first, then fall back to `~/.agentz/plugins/` with a deprecation warning. No file moving — just dual-path resolution.

4. **CI workflow**: Update glob patterns from `examples/*/*.az` to `examples/*/*.ias` and binary name from `agentz` to `agentspec`.

5. **Go module path**: Stays as `github.com/szaher/designs/agentz`. A module path rename would be a separate, more disruptive change requiring a major version bump.

6. **Migrate command**: The existing `agentz migrate` command is updated to handle `.az` → `.ias` file renames in addition to its existing version migration functionality.

## Files Affected Summary

| Category | Count | Changes |
|----------|-------|---------|
| Example .az files | 10 | Rename to .ias |
| Example READMEs | 11 | Update extension/binary refs |
| Go source (CLI) | 10 | Help text, deprecation warnings, binary name |
| Go source (internal) | 19 | Deprecation helper (new), state file, plugin path, SDK templates, string refs, comments |
| Integration tests | 11 | 8 test files (extension + state refs) + 3 testdata file renames |
| CI/Config | 3 | .gitignore, ci.yml, .golangci.yml |
| Documentation | 4 | ARCHITECTURE.md, CHANGELOG.md, spec/spec.md, init-spec.md |
| Spec contracts | 3 | cli.md, plugin-manifest.md, sdk-api.md |
| Other | 3 | CLAUDE.md, DECISIONS/, plugins/monitor/manifest.json |
| **Total** | **74** | |
