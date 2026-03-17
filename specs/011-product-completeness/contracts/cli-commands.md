# CLI Command Contract

**Feature**: 011-product-completeness
**Date**: 2026-03-17

## Command Registry (Post-Rename)

All commands MUST be registered in `cmd/agentspec/main.go` and documented in `README.md`.

| Command | Short Description | Behavior |
|---------|-------------------|----------|
| `version` | Display version information | Print version, lang version, IR version |
| `fmt <file>` | Format to canonical style | Deterministic formatting |
| `validate <file>` | Check syntax and semantic correctness | Structural + semantic validation |
| `plan <file>` | Preview changes without applying | Desired-state diff |
| `apply <file>` | Apply changes idempotently | Execute plan |
| `diff <file>` | Show detailed resource differences | Resource-level diff |
| `export <file>` | Generate platform-specific artifacts | Adapter output |
| `sdk <file>` | Generate client SDKs | Python/TypeScript/Go |
| `migrate [path]` | Rename/rewrite IntentLang files | v1→v2, .az→.ias |
| `init` | Initialize a new AgentSpec project | Scaffold .ias file |
| `compile <file>` | Compile agent to target framework | CrewAI/LangGraph/LlamaIndex/LlamaStack |
| `publish` | Publish AgentPack to Git remote | Tag + push |
| `install <pkg>` | Install an AgentPack package | Git clone + validate |
| `eval <file>` | Run evaluation test cases | Expression eval + `--live` for LLM |
| `run <file>` | Start agent runtime server | **NEW**: HTTP server with hot reload |
| `dev <file>` | One-shot agent invocation | **NEW**: Invoke agent, print, exit |
| `status` | Show runtime status | Agent/session state |
| `logs` | Show runtime logs | Structured log output |
| `destroy` | Tear down deployed resources | Reverse of apply |
| `pkg` | Package management | List/info subcommands |

## Deprecation Aliases

| Old Command | New Command | Warning Message |
|-------------|-------------|-----------------|
| `run <file> --input "..."` (one-shot) | `dev <file> --input "..."` | "Warning: 'run' for one-shot invocation is deprecated. Use 'dev' instead." |
| `dev <file>` (server) | `run <file>` | "Warning: 'dev' for server mode is deprecated. Use 'run' instead." |

Aliases MUST:
- Print deprecation warning to stderr
- Execute the correct (new) behavior
- Be removed after one release cycle

## Eval Command Contract

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--agent` | string | "" | Evaluate specific agent by name |
| `--tags` | string | "" | Filter eval cases by tags (comma-separated) |
| `--output` | string | "" | Write report to file |
| `--format` | string | "table" | Output format: table, json, markdown |
| `--compare` | string | "" | Path to previous eval report |
| `--live` | bool | false | **NEW**: Invoke with real LLM client |

### Behavior

- Without `--live`: Use expression-based evaluation (existing)
- With `--live`: Create LLM client per agent, invoke via ReAct strategy, compare output against expected patterns

### Report Format

```
Agent: <name>
  PASS  case-1: <description>
  FAIL  case-2: <description>
    Expected: <pattern>
    Actual:   <output>

Summary: 1/2 passed (50%)
```

## Stub Flag Contract

After this feature, zero CLI flags should exist that:
1. Accept input (via `--flag value` or `--flag`)
2. Produce no side effect
3. Allow the command to continue as if the flag worked

Known stubs to fix:
- `publish --sign`: Must return error instead of warning
