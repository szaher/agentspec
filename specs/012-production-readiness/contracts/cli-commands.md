# Contract: New CLI Commands

## `agentspec rollback`

Restores the previous agent version from state history.

```
Usage: agentspec rollback [flags]

Flags:
  --agent string     Agent name to roll back (required if multiple agents exist)
  --state-file       Path to state file (default: .agentspec.state.json)
```

**Behavior**:
- Reads version history from state file
- Restores the IR snapshot from the previous version
- Creates a new version entry (rollback is itself versioned)
- Prints: `Rolled back "support-agent" from version 3 to version 2`
- Exit code 0 on success, 1 if no previous version exists

## `agentspec history`

Lists version history for an agent.

```
Usage: agentspec history [flags]

Flags:
  --agent string     Agent name (required if multiple agents exist)
  --state-file       Path to state file (default: .agentspec.state.json)
```

**Output format**:
```
Version  Timestamp                 Summary
3        2026-03-17T10:00:00Z      Updated model to claude-sonnet-4-20250514
2        2026-03-16T15:30:00Z      Added web-search skill
1        2026-03-15T09:00:00Z      Initial deployment
```

## DSL Extensions

### `user` block

```
user "alice" {
  key secret("ALICE_API_KEY")
  agents ["support-agent", "search-agent"]
  role "invoke"
}
```

### `guardrail` block

```
guardrail "safety" {
  mode "block"
  keywords ["password", "credit card", "SSN"]
  fallback "I cannot provide that information."
}
```

### Agent block extensions

```
agent "support" {
  uses prompt "system"
  uses skill "lookup"
  uses guardrail "safety"
  models ["claude-sonnet-4-20250514", "gpt-4o-mini"]
  budget daily 10.0
  budget monthly 200.0
}
```

## Release Automation

### GoReleaser Configuration (`.goreleaser.yaml`)

Triggered by GitHub Actions on `v*.*.*` tag push.

**Build targets**:
- `linux/amd64`, `linux/arm64`
- `darwin/amd64`, `darwin/arm64`
- `windows/amd64`, `windows/arm64`

**Artifacts**:
- Compressed binaries (`.tar.gz` for Linux/macOS, `.zip` for Windows)
- `checksums.txt` (SHA-256)
- GitHub Release with auto-generated changelog

**Version injection**:
```yaml
ldflags:
  - -s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}
```
