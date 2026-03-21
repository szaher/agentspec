# Implementation Plan: Adoption & Developer Experience

**Branch**: `013-adoption-dev-experience` | **Date**: 2026-03-18 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/013-adoption-dev-experience/spec.md`

## Summary

Make AgentSpec installable and usable in minutes by adding packaged releases (Homebrew tap, Docker image on ghcr.io), enhancing the `agentspec init` command with interactive template selection and 6 bundled starter templates, creating 6 curated example projects, adding version update checking, and publishing a quickstart guide on the documentation site.

## Technical Context

**Language/Version**: Go 1.25+ (existing)
**Primary Dependencies**: cobra v1.10.2 (existing CLI), GoReleaser v2 (existing release tooling), `embed` (stdlib, existing for templates)
**Storage**: N/A (no new state; templates embedded in binary, examples are static files)
**Testing**: `go test` (existing), CI validation of all examples via `agentspec validate`
**Target Platform**: macOS (amd64, arm64), Linux (amd64, arm64), Windows (amd64) — binary releases; Linux (amd64, arm64) — Docker image
**Project Type**: CLI tool enhancement + content (templates, examples, docs)
**Performance Goals**: `agentspec init` completes in < 3 seconds with no network access (SC-007)
**Constraints**: Docker image < 50 MB compressed (FR-004); templates work offline (embedded in binary)
**Scale/Scope**: 6 starter templates, 6 example projects, 1 quickstart doc page, 3 GoReleaser config additions (brews, dockers, docker_manifests)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Determinism | PASS | Templates are static embedded files; scaffolding is deterministic |
| II. Idempotency | PASS | No `apply` changes; `init` detects existing files and prompts |
| III. Portability | PASS | Cross-platform binaries; Docker image for containerized use |
| IV. Separation of Concerns | PASS | Templates are content, not semantic logic |
| V. Reproducibility | PASS | GoReleaser builds are pinned; Docker tags are versioned |
| VI. Safe Defaults | PASS | Secrets use env var references, never plaintext literals |
| VII. Minimal Surface Area | PASS | Adds `init` enhancements + `version` update check only; no new keywords |
| VIII. English-Friendly Syntax | PASS | Template `.ias` files follow existing DSL conventions |
| IX. Canonical Formatting | PASS | All template `.ias` files must pass `agentspec fmt --check` |
| X. Strict Validation | PASS | All templates must pass `agentspec validate` |
| XI. Explicit References | PASS | No floating imports in templates |
| XII. No Hidden Behavior | PASS | Version check is visible output; no silent mutations |

**Gate result**: PASS — no violations.

## Project Structure

### Documentation (this feature)

```text
specs/013-adoption-dev-experience/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   ├── cli-init.md      # Init command contract
│   ├── cli-version.md   # Version command contract
│   ├── docker-image.md  # Docker image contract
│   └── homebrew-tap.md  # Homebrew tap contract
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
# GoReleaser config (modify existing)
.goreleaser.yaml                    # Add brews, dockers, docker_manifests sections

# Dockerfile (new)
Dockerfile                          # Multi-stage: scratch + static binary

# Template system (modify + extend)
internal/templates/
├── templates.go                    # Enhanced: directory-based templates, categories
└── files/
    ├── basic-chatbot/
    │   ├── basic-chatbot.ias
    │   └── README.md
    ├── support-bot/
    │   ├── support-bot.ias         # Renamed from customer-support.ias
    │   └── README.md
    ├── rag-assistant/
    │   ├── rag-assistant.ias       # Renamed from rag-chatbot.ias
    │   └── README.md
    ├── research-swarm/
    │   ├── research-swarm.ias
    │   └── README.md
    ├── incident-response/
    │   ├── incident-response.ias
    │   └── README.md
    └── multi-agent-router/
        ├── multi-agent-router.ias
        └── README.md

# CLI commands (modify existing)
cmd/agentspec/
├── init.go                         # Enhanced: interactive selector, directory scaffolding
├── version.go                      # Enhanced: GitHub Releases API update check
└── main.go                         # Update root command: no-args help message

# Example projects (new + enhance existing)
examples/
├── support-bot/                    # Existing (renamed from customer-support/) — enhance README
│   ├── support-bot.ias
│   └── README.md
├── rag-assistant/                  # Existing (renamed from rag-chatbot/) — enhance README
│   ├── rag-assistant.ias
│   └── README.md
├── research-swarm/                 # New
│   ├── research-swarm.ias
│   └── README.md
├── incident-response/              # New
│   ├── incident-response.ias
│   └── README.md
├── gpu-batch/                      # New
│   ├── gpu-batch.ias
│   └── README.md
└── multi-agent-router/             # New (or enhanced from control-flow-agent)
    ├── multi-agent-router.ias
    └── README.md

# Documentation (new page)
docs/
└── quickstart.md                   # Quickstart guide for MkDocs site

# CI workflows (modify existing)
.github/workflows/
├── release.yaml                    # Already triggers GoReleaser; no changes needed
│                                   # (brews + dockers configs in .goreleaser.yaml)
└── ci.yml                          # Add template validation step
```

**Structure Decision**: This feature primarily adds content (templates, examples, docs) and enhances existing CLI commands. No new internal packages are required. The `internal/templates/` package is restructured from flat files to directories to support README bundling.

## Complexity Tracking

No constitution violations to justify.

## Implementation Phases

### Phase A: Release & Distribution Infrastructure

**Goal**: Enable one-command install on all platforms.

1. **GoReleaser: Homebrew tap** — Add `brews` section to `.goreleaser.yaml` targeting `szaher/homebrew-agentspec`. GoReleaser auto-pushes the formula on tagged release.

2. **GoReleaser: Docker image** — Add `dockers` and `docker_manifests` sections to `.goreleaser.yaml`. Build `linux/amd64` and `linux/arm64` images from `scratch` base. Push to `ghcr.io/szaher/agentspec` with semver + `latest` tags.

3. **Dockerfile** — Create a minimal Dockerfile at repo root for GoReleaser to use. Contents: `FROM scratch`, `COPY agentspec /agentspec`, `ENTRYPOINT ["/agentspec"]`.

4. **Release workflow permissions** — Ensure `.github/workflows/release.yaml` has `packages: write` permission for ghcr.io push.

**Covers**: FR-001, FR-002, FR-003, FR-004, FR-017, SC-004, SC-005

### Phase B: Template System Enhancement

**Goal**: 6 starter templates with READMEs, interactive selection, offline scaffolding.

1. **Restructure embedded templates** — Move from flat `files/*.ias` to `files/<name>/<name>.ias` + `files/<name>/README.md`. Update `//go:embed` directive to `files/*`.

2. **Update Template struct** — Add `Category` field. Update `All()` to return 6 required templates with categories (beginner/intermediate/advanced).

3. **Create new templates** — `basic-chatbot`, `research-swarm`, `incident-response`, `multi-agent-router`. Rename `customer-support` → `support-bot`, `rag-chatbot` → `rag-assistant`.

4. **Template content** — Each `.ias` file uses env var references (`${ANTHROPIC_API_KEY}`), inline comments explaining sections. Each README covers description, prerequisites, config, and run instructions.

5. **Enhance init command** — Interactive numbered-list selector (when no `--template` flag), overwrite confirmation (FR-016), directory scaffolding (creates `<name>/` with `.ias` + README), improved "next steps" output showing env vars needed.

6. **Template validation in CI** — Add step to `ci.yml` that scaffolds each template to a temp dir and runs `agentspec validate` on the output.

**Covers**: FR-005 through FR-011, FR-016, SC-002, SC-007

### Phase C: Example Projects

**Goal**: 6 curated, runnable examples with comprehensive READMEs.

1. **Rename and enhance existing examples** — Rename `customer-support/` → `support-bot/` and `rag-chatbot/` → `rag-assistant/` to align with template names. Update READMEs with standardized format: Description, Architecture Overview, Prerequisites, Step-by-Step Run Instructions, Customization Tips.

2. **Create research-swarm example** — Multi-agent example using existing multi-agent DSL features. Demonstrates coordinated research across multiple agents.

3. **Create incident-response example** — Single agent with escalation logic, runbook execution, and alert triage patterns.

4. **Create gpu-batch example** — Agent that dispatches batch inference tasks to an external API endpoint. Demonstrates async task dispatch pattern without requiring local GPU.

5. **Create multi-agent-router example** — Router agent that delegates to specialized sub-agents based on input classification.

6. **Validate all examples in CI** — Existing CI already validates `examples/*/*.ias`; verify new examples are picked up.

**Covers**: FR-012, FR-013, SC-003

### Phase D: CLI Polish & Version Check

**Goal**: Helpful CLI output for new users + version update notification.

1. **Root command no-args message** — When `agentspec` is run with no arguments, display a welcome message pointing to `--help`, `init`, and the quickstart guide URL.

2. **Version update check** — Add HTTP call to GitHub Releases API in `version` command. 2-second timeout. Print update notice if outdated. Silent fallback on network error.

**Covers**: FR-015, FR-018

### Phase E: Quickstart Documentation

**Goal**: Documentation site quickstart guide.

1. **Create `docs/quickstart.md`** — Platform-tabbed install instructions (macOS/Homebrew, Linux/binary, Windows/binary, Docker). Scaffold + configure + run walkthrough. Troubleshooting section (missing API key, wrong architecture, port conflicts).

2. **Update `mkdocs.yml`** — Add quickstart page to navigation.

**Covers**: FR-014, SC-006

## Dependencies Between Phases

```
Phase A (Release Infrastructure) ──┐
                                    ├── Phase E (Quickstart Docs — references install commands)
Phase B (Templates) ───────────────┤
                                    │
Phase C (Examples) ────────────────┘

Phase D (CLI Polish) ── independent, can run in parallel with any phase
```

Phases A, B, C, D can proceed in parallel. Phase E depends on A and B being complete (to reference final install commands and template names in the docs).
