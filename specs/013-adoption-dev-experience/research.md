# Research: Adoption & Developer Experience

**Feature**: 013-adoption-dev-experience
**Date**: 2026-03-18

## R-001: GoReleaser Homebrew Tap Integration

**Decision**: Use GoReleaser's built-in `brews` section to auto-publish a Homebrew formula to a dedicated tap repository (`szaher/homebrew-agentspec`).

**Rationale**: GoReleaser v2 natively supports generating and pushing Homebrew formulas on release. This avoids manual formula maintenance and keeps the tap in sync with releases automatically.

**Alternatives considered**:
- Manual formula maintenance in a separate repo — rejected due to ongoing maintenance burden and risk of version drift.
- Using `homebrew-core` — rejected because upstream acceptance requires significant adoption metrics and review cycles.

## R-002: Docker Image Build Strategy

**Decision**: Add a `dockers` section to `.goreleaser.yaml` to build and push multi-arch Docker images to `ghcr.io/szaher/agentspec` as part of the release pipeline. Use a `scratch` base image with only the static binary.

**Rationale**: GoReleaser v2 supports Docker builds and pushes natively, including multi-arch manifests. Using `scratch` as the base image produces the smallest possible image (the static binary is ~15-20 MB), well under the 50 MB compressed target. CGO_ENABLED=0 is already set, so the binary is fully static.

**Alternatives considered**:
- `alpine` base — rejected because the binary is static and needs no OS utilities, and alpine adds ~5 MB.
- `distroless` base — rejected for same reason; scratch is sufficient for a single static binary.
- Separate Dockerfile + CI workflow — rejected because GoReleaser handles this natively.

## R-003: Docker Image Tagging Strategy

**Decision**: Tag images with full semver (`1.2.3`), minor (`1.2`), major (`1`), and `latest`. GoReleaser's `dockers` section supports this via `image_templates`.

**Rationale**: Standard practice for CLI tool images. Allows CI/CD users to pin to exact versions while quick-start users can pull `latest`.

**Alternatives considered**:
- Full semver only — rejected because `latest` is expected by new users following quickstart guides.
- Adding `edge`/nightly — rejected as out of scope; pre-releases are handled by GoReleaser's prerelease detection.

## R-004: Template Embedding and Scaffolding Enhancement

**Decision**: Enhance the existing `internal/templates/` package to embed full project directories (`.ias` file + `README.md`) instead of just `.ias` files. Use `//go:embed` with directory patterns. Add interactive template selection using a simple numbered-list prompt on stdin.

**Rationale**: The current implementation embeds only `.ias` files. The spec requires each template to include a README with description, prerequisites, and run instructions. Embedding directories allows bundling multiple files per template while keeping the `embed` approach for offline use.

**Alternatives considered**:
- Tar/zip archives embedded and extracted — rejected because Go's `embed.FS` handles directory trees natively.
- External template registry with download — rejected because offline support is a requirement (FR-009, SC-007).

## R-005: Interactive Template Selector

**Decision**: Implement a simple numbered-list selector using `fmt.Print` + `bufio.Scanner` reading from stdin. No TUI library dependency.

**Rationale**: A numbered-list selector works in all terminals, requires no external dependencies, and keeps the binary size minimal. The template count (6) is small enough that arrow-key navigation adds no value.

**Alternatives considered**:
- TUI library (bubbletea, promptui) — rejected because it adds a dependency for a simple 6-item list, and may not work well in all CI/pipe contexts.
- fzf-style fuzzy finder — rejected for same reasons.

## R-006: Version Update Check

**Decision**: Add an HTTP call to the GitHub Releases API (`https://api.github.com/repos/szaher/agentspec/releases/latest`) in the `version` command. Compare the installed version against the latest release tag. Print a notice if outdated. Use a 2-second timeout to avoid blocking on network issues.

**Rationale**: The GitHub Releases API is free, unauthenticated, and returns the latest release tag. A short timeout ensures the version command remains fast even without network access.

**Alternatives considered**:
- Check on every CLI invocation — rejected per clarification (check only on `--version`).
- Cache the check result locally — rejected as over-engineering for a single-command check.

## R-007: Example Projects — Gap Analysis

**Decision**: Map the 6 required examples to existing examples where possible, create new ones where gaps exist.

| Required Example | Existing Match | Action |
|-----------------|---------------|--------|
| Support bot | `customer-support/` → rename to `support-bot/` | Rename, enhance README, verify validates |
| RAG assistant | `rag-chatbot/` → rename to `rag-assistant/` | Rename, enhance README, verify validates |
| Research swarm | None (research-assistant template exists but is single-agent) | Create new `research-swarm/` multi-agent example |
| Incident-response agent | None | Create new `incident-response/` example |
| GPU batch agent | None | Create new `gpu-batch/` example (dispatches to external API) |
| Multi-agent router | `control-flow-agent/` (has router pattern) | Enhance as `multi-agent-router/` or rename + enhance |

**Rationale**: Reusing and enhancing 2 existing examples (with rename to align with template names) reduces effort. 4 new examples need creation.

## R-008: Template Projects — Gap Analysis

**Decision**: Map the 6 required templates to existing templates where possible.

| Required Template | Existing Match | Action |
|------------------|---------------|--------|
| Basic chatbot | None (validated-agent is close but complex) | Create new minimal template |
| Support bot | `customer-support` template | Enhance with README |
| RAG assistant | `rag-chatbot` template | Enhance with README |
| Research swarm | None | Create new template |
| Incident-response agent | None | Create new template |
| Multi-agent router | None (code-review-pipeline is multi-agent but not router) | Create new template |

**Rationale**: 3 existing templates can be reused with enhancement. 3 new templates need creation. The `data-extraction` and `validated-agent` templates remain as bonus templates beyond the required 6.

## R-009: Quickstart Documentation

**Decision**: Add a `docs/quickstart.md` page to the existing MkDocs site. Include platform-tabbed install instructions, scaffolding walkthrough, and troubleshooting section. Reference existing `mkdocs.yml` configuration.

**Rationale**: The documentation site already uses MkDocs with GitHub Pages deployment. Adding a page follows the existing pattern.

**Alternatives considered**:
- Separate quickstart site — rejected; the MkDocs site already exists.
- README-only quickstart — rejected because the spec requires a documentation site page (FR-014).
