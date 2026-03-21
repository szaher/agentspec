# Data Model: Adoption & Developer Experience

**Feature**: 013-adoption-dev-experience
**Date**: 2026-03-18

## Entities

### Template

Represents a bundled project scaffold available via `agentspec init`.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Unique identifier (e.g., `basic-chatbot`, `support-bot`) |
| Description | string | One-line description shown in `--list-templates` |
| Category | string | Complexity tier: `beginner`, `intermediate`, `advanced` |
| Dir | string | Embedded directory path within `internal/templates/files/` |
| Files | []string | List of files in the template (`.ias`, `README.md`) |

**Relationships**: None (standalone entity, embedded in binary).

**Validation rules**:
- Name must be lowercase kebab-case (`^[a-z][a-z0-9-]*$`).
- Each template directory must contain at least one `.ias` file and one `README.md`.
- The `.ias` file must pass `agentspec validate` without errors.

### Release Artifact

Represents a platform-specific binary produced by GoReleaser.

| Field | Type | Description |
|-------|------|-------------|
| Version | string | Semver version (from git tag) |
| OS | string | Target OS: `linux`, `darwin`, `windows` |
| Arch | string | Target architecture: `amd64`, `arm64` |
| Format | string | Archive format: `tar.gz` or `zip` |
| Checksum | string | SHA256 checksum |

**Relationships**: Published to GitHub Releases. Referenced by Homebrew formula.

### Docker Image

Represents a container image published to ghcr.io.

| Field | Type | Description |
|-------|------|-------------|
| Registry | string | Always `ghcr.io` |
| Repository | string | `szaher/agentspec` |
| Tags | []string | Semver tags (`1.2.3`, `1.2`, `1`) + `latest` |
| BaseImage | string | Always `scratch` |
| BinaryPath | string | `/agentspec` |

**Relationships**: Built from Release Artifact. Tagged with same version.

### Example Project

Represents a runnable example in the `examples/` directory.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Directory name (e.g., `research-swarm`) |
| Description | string | One-line description |
| UseCase | string | Category: `single-agent`, `multi-agent`, `pipeline`, `batch` |
| IASFiles | []string | IntentLang files in the example |
| Prerequisites | []string | Required env vars or external services |
| HasREADME | bool | Must be `true` |

**Validation rules**:
- All `.ias` files must pass `agentspec validate`.
- README must exist and contain: Description, Prerequisites, Run Instructions.

## State Transitions

No new state transitions. This feature adds static artifacts (templates, examples, docs) and CLI commands that produce file output. No persistent state changes.

## Environment Variables

Templates use environment variable references resolved at runtime:

| Variable | Used By | Description |
|----------|---------|-------------|
| `ANTHROPIC_API_KEY` | All templates using Anthropic | API key for Anthropic LLM |
| `OPENAI_API_KEY` | Templates using OpenAI | API key for OpenAI LLM |
| `AGENTSPEC_MODEL` | Optional in templates | Override default model |
