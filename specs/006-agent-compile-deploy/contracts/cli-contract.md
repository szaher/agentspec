# CLI Contract: Compilation & Deployment Commands

**Feature**: 006-agent-compile-deploy

## New Commands

### `agentspec compile`

Compile `.ias` files into a deployable agent artifact.

```
agentspec compile [flags] <file.ias | directory>
```

**Flags**:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--target` | string | `standalone` | Compilation target: `standalone`, `crewai`, `langgraph`, `llamastack`, `llamaindex` |
| `--output` | string | `./build/` | Output directory for compiled artifacts |
| `--platform` | string | current OS/arch | Target platform: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64` |
| `--name` | string | package name | Output binary/project name |
| `--embed-frontend` | bool | `true` | Embed the built-in frontend in the compiled binary |
| `--verbose` | bool | `false` | Show detailed compilation progress |

**Exit codes**:

| Code | Meaning |
|------|---------|
| 0 | Compilation succeeded |
| 1 | Compilation failed (syntax/semantic errors) |
| 2 | Import resolution failed |
| 3 | Plugin error (framework target unavailable) |

**Output (human)**:
```
Compiling customer-support.ias...
  ✓ Parsed 3 files (2 imports resolved)
  ✓ Validated 2 agents, 5 skills, 3 validation rules
  ✓ Lowered to IR (hash: a1b2c3d4)
  ✓ Compiled to standalone binary

Output: ./build/customer-support (15.2 MB)
Platform: darwin/arm64
Agents: support-agent, escalation-agent
Config: ./build/customer-support.config.md
```

**Output (JSON, `--json` flag)**:
```json
{
  "status": "success",
  "artifact": {
    "path": "./build/customer-support",
    "size_bytes": 15938560,
    "content_hash": "sha256:a1b2c3d4...",
    "platform": "darwin/arm64",
    "target": "standalone",
    "agents": ["support-agent", "escalation-agent"]
  },
  "config_ref": "./build/customer-support.config.md",
  "warnings": [],
  "compilation_time_ms": 3200
}
```

---

### `agentspec eval`

Run evaluation test cases against a compiled or running agent.

```
agentspec eval [flags] <file.ias | compiled-binary>
```

**Flags**:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--agent` | string | all | Evaluate specific agent by name |
| `--tags` | string[] | all | Filter eval cases by tags |
| `--output` | string | stdout | Write report to file |
| `--format` | string | `table` | Output format: `table`, `json`, `markdown` |
| `--compare` | string | none | Path to previous eval report for comparison |

**Output (human)**:
```
Evaluating support-agent (5 test cases)...

  ✓ greeting-test          score: 0.95  (threshold: 0.80)
  ✓ refund-request         score: 0.88  (threshold: 0.80)
  ✗ edge-case-empty-input  score: 0.62  (threshold: 0.80)
  ✓ multi-turn-context     score: 0.91  (threshold: 0.80)
  ✓ tool-usage-test        score: 0.87  (threshold: 0.80)

Results: 4/5 passed (80%)
Overall score: 0.85

Compared to previous run:
  Overall: 0.85 → 0.85 (no change)
  Regressions: 0
  Improvements: 1 (refund-request: 0.82 → 0.88)
```

---

### `agentspec package`

Package a compiled agent for deployment.

```
agentspec package [flags] <compiled-artifact>
```

**Flags**:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | required | `docker`, `kubernetes`, `helm`, `binary` |
| `--output` | string | `./dist/` | Output directory |
| `--tag` | string | `latest` | Version tag for container images |
| `--registry` | string | none | Container registry to push to |
| `--push` | bool | `false` | Push image to registry after build |

---

### `agentspec publish`

Publish a package to the AgentSpec registry.

```
agentspec publish [flags] <directory>
```

**Flags**:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--registry` | string | default | Registry URL |
| `--version` | string | from package declaration | Override version |
| `--dry-run` | bool | `false` | Validate without publishing |

---

### `agentspec install`

Install a package from the registry.

```
agentspec install [flags] <package@version>
```

**Flags**:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--registry` | string | default | Registry URL |
| `--cache-dir` | string | `~/.agentspec/cache/` | Local cache directory |

---

## Modified Commands

### `agentspec dev` (enhanced)

Add `--ui` flag to serve the built-in frontend during development.

```
agentspec dev --ui [flags] <file.ias>
```

New flag:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--ui` | bool | `false` | Serve built-in frontend on agent port |
| `--ui-port` | int | same as agent | Separate port for frontend (optional) |
| `--no-auth` | bool | `false` | Disable API key auth for local dev |

---

## Compiled Agent Runtime CLI

The compiled agent binary itself accepts runtime flags:

```
./my-agent [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--port` | int | `8080` | HTTP server port |
| `--host` | string | `0.0.0.0` | Bind address |
| `--ui` | bool | `true` | Serve built-in frontend |
| `--no-auth` | bool | `false` | Disable API key authentication |
| `--config` | string | none | Path to config file (alternative to env vars) |
| `--log-level` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `--log-format` | string | `text` | Log format: `text`, `json` |

**Environment variables** (always take precedence over config file):

| Variable | Description |
|----------|-------------|
| `AGENTSPEC_API_KEY` | API key for endpoint authentication |
| `AGENTSPEC_PORT` | Override `--port` |
| `AGENTSPEC_LOG_LEVEL` | Override `--log-level` |
| Agent-specific vars | `AGENTSPEC_<AGENT>_<PARAM>` pattern |
