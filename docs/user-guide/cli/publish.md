# publish

Publish an AgentPack package to a Git remote.

## Usage

```bash
agentspec publish
```

## Description

The `publish` command reads the `agentpack.yaml` manifest in the current directory, validates the package, creates a Git version tag, and pushes it to the configured Git remote. This makes the package available for others to install with `agentspec install`.

Before publishing, the command checks that all exported files listed in the manifest exist and that the working directory has no uncommitted changes.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--sign` | `false` | Sign the package (not yet implemented) |

## Package Manifest

The `publish` command requires an `agentpack.yaml` file in the current directory. This manifest describes the package metadata, exported files, and dependencies:

```yaml
name: my-agents
version: 1.2.0
description: A collection of reusable agent specs
exports:
  - support-bot.ias
  - summarizer.ias
dependencies:
  - source: github.com/org/shared-tools
    version: ">=0.3.0"
```

| Field | Description |
|-------|-------------|
| `name` | Package name |
| `version` | Semantic version (used to create the Git tag `v<version>`) |
| `description` | Human-readable description |
| `exports` | List of `.ias` files included in the package |
| `dependencies` | Other AgentPack packages this package depends on |

## Git-Based Publishing

Publishing is Git-based. The command:

1. Validates that all exported files exist on disk.
2. Checks that the Git working directory is clean (no uncommitted changes).
3. Creates an annotated Git tag `v<version>` (e.g. `v1.2.0`).
4. Pushes the tag to the `origin` remote.

Consumers install published packages by referencing the Git repository URL and version tag.

## Examples

```bash
# Publish the package defined in agentpack.yaml
agentspec publish

# Attempt to publish with signing (not yet implemented)
agentspec publish --sign
```

## Output

```
Publishing my-agents@1.2.0
Published my-agents (tag: v1.2.0)
```

## Prerequisites

- An `agentpack.yaml` file must exist in the current directory.
- All files listed in `exports` must exist.
- The Git working directory must be clean (no uncommitted changes).
- The version tag must not already exist. Bump the version in `agentpack.yaml` if it does.
- A Git remote named `origin` must be configured.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Package published successfully |
| `1` | An error occurred (missing manifest, dirty working directory, tag conflict, etc.) |

## See Also

- [CLI: install](install.md) -- Install a published package
- [CLI: validate](validate.md) -- Validate .ias files before publishing
