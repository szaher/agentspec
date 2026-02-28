# Package Registry Contract

**Feature**: 006-agent-compile-deploy

## Overview

The package registry enables publishing, discovering, and resolving AgentSpec packages. This contract defines the HTTP API for registry interactions.

## Phase 0: Git-Based Resolution (MVP)

No registry server required. The CLI resolves packages directly from Git repositories.

### Package Reference Format

```
github.com/user/agentpack@v1.2.3
```

Components:
- **Host**: Git hosting provider (github.com, gitlab.com, etc.)
- **Path**: Repository path
- **Version**: Git tag (must be valid semver with `v` prefix)

### Resolution Algorithm

1. Check local cache: `~/.agentspec/cache/<host>/<path>/@v/<version>/`
2. If cache miss: `git clone --depth 1 --branch <version> https://<host>/<path>.git`
3. Read package manifest: `agentpack.yaml` in repository root
4. Verify checksum against `.agentspec.lock` (if exists)
5. Cache resolved package locally

### Package Manifest (`agentpack.yaml`)

```yaml
name: web-tools
version: 1.2.3
description: Common web interaction tools for AgentSpec agents
author: AgentSpec Community
license: MIT
agentspec: ">=0.3.0"
dependencies:
  github.com/agentspec/core-skills: "^1.0.0"
exports:
  - skills/search.ias
  - skills/scrape.ias
  - prompts/web-assistant.ias
```

### Lock File (`.agentspec.lock`)

```yaml
version: 1
packages:
  github.com/user/agentpack:
    version: v1.2.3
    hash: sha256:a1b2c3d4e5f6...
    signature: unsigned
    resolved_at: 2026-02-28T12:00:00Z
  github.com/agentspec/core-skills:
    version: v1.0.2
    hash: sha256:f6e5d4c3b2a1...
    signature: unsigned
    resolved_at: 2026-02-28T12:00:00Z
```

---

## Phase 1: HTTP Registry API

### Base URL

```
https://registry.agentspec.dev/v1
```

Configurable via `AGENTSPEC_REGISTRY` environment variable or `.agentspec.yaml`:
```yaml
registry: https://registry.agentspec.dev/v1
```

### Endpoints

#### List Versions

```
GET /v1/packages/{namespace}/{name}/@v/list
```

**Response** (200, `text/plain`):
```
v1.0.0
v1.1.0
v1.2.0
v1.2.3
```

#### Version Info

```
GET /v1/packages/{namespace}/{name}/@v/{version}.info
```

**Response** (200, `application/json`):
```json
{
  "version": "v1.2.3",
  "time": "2026-02-28T12:00:00Z",
  "deprecated": false
}
```

#### Package Manifest

```
GET /v1/packages/{namespace}/{name}/@v/{version}.manifest
```

**Response** (200, `application/yaml`):
```yaml
name: web-tools
version: 1.2.3
# ... same as agentpack.yaml
```

#### Download Package

```
GET /v1/packages/{namespace}/{name}/@v/{version}.zip
```

**Response** (200, `application/zip`): ZIP archive of package contents.

#### Latest Version

```
GET /v1/packages/{namespace}/{name}/@latest
```

**Response** (200, `application/json`):
```json
{
  "version": "v1.2.3",
  "time": "2026-02-28T12:00:00Z"
}
```

#### Publish Package

```
PUT /v1/packages/{namespace}/{name}/@v/{version}
Content-Type: multipart/form-data
Authorization: Bearer <token>
```

**Form fields**:
- `manifest`: Package manifest (YAML)
- `archive`: Package archive (ZIP)
- `checksum`: SHA-256 of archive

**Response** (201):
```json
{
  "status": "published",
  "package": "namespace/name",
  "version": "v1.2.3",
  "checksum": "sha256:a1b2c3d4..."
}
```

#### Deprecate Version

```
POST /v1/packages/{namespace}/{name}/@v/{version}/deprecate
Authorization: Bearer <token>
```

**Request**:
```json
{
  "message": "Security vulnerability in search tool. Use v1.2.4 instead."
}
```

---

### Authentication

**Publishing** requires authentication. **Reading** is public by default.

Authentication methods (Phase 1):
- **Token-based**: `Authorization: Bearer <token>` header
- **GitHub OIDC**: Automatic authentication in GitHub Actions workflows

### Checksum Verification

Every package download is verified against its published checksum:

```
GET /v1/packages/{namespace}/{name}/@v/{version}.checksum
```

**Response** (200, `text/plain`):
```
sha256:a1b2c3d4e5f6789...
```

The CLI verifies this checksum after download and before using the package. Checksum mismatches abort compilation with a clear error.

---

## Package Signing & Provenance (FR-049)

Package signing is designed as a first-class concept per the project constitution. The MVP stubs the signing implementation but establishes the data structures and verification flow.

### Signature Data Model

Each published package version has an associated signature record:

```json
{
  "package": "namespace/name",
  "version": "v1.2.3",
  "checksum": "sha256:a1b2c3d4...",
  "signature": "unsigned",
  "signer": "",
  "signed_at": null,
  "provenance": {
    "build_system": "",
    "source_repo": "",
    "commit_sha": ""
  }
}
```

### MVP Behavior

- `signature` field is always `"unsigned"` in MVP
- CLI emits an informational message: `"Package is unsigned. Signature verification will be available in a future release."`
- Lock file records `signature: "unsigned"` for each dependency
- The `agentspec publish` command accepts a `--sign` flag that prints `"Signing not yet implemented"` and publishes without a signature

### Phase 1 Signing (Post-MVP)

```
GET /v1/packages/{namespace}/{name}/@v/{version}.sig
```

**Response** (200, `application/json`):
```json
{
  "signature": "base64-encoded-signature",
  "signer": "keyid:abc123",
  "algorithm": "ed25519",
  "signed_at": "2026-03-15T12:00:00Z"
}
```

Verification: CLI downloads `.sig` alongside `.zip`, verifies signature against a configured trust store (`~/.agentspec/keys/`).

---

## Dependency Resolution: Minimal Version Selection

The registry client implements MVS (Minimal Version Selection):

1. Build dependency graph from all transitive `agentpack.yaml` manifests
2. For each package, collect all required versions across the graph
3. Select the minimum version that satisfies all constraints
4. If no version satisfies all constraints, report the conflict with both dependency chains

**Example conflict report**:
```
Version conflict for github.com/agentspec/core-skills:
  → your-project requires ^1.0.0
    → github.com/user/agentpack@v1.2.3 requires ^1.2.0
  → github.com/other/tools@v2.0.0 requires ^2.0.0

  No version satisfies both ^1.2.0 and ^2.0.0.

Suggestions:
  1. Update github.com/user/agentpack to a version compatible with core-skills ^2.0.0
  2. Pin github.com/other/tools to a version compatible with core-skills ^1.x
```
