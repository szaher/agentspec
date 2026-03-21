# CLI Contract: `agentspec version` (enhanced)

## Synopsis

```
agentspec version
agentspec --version
```

## Current Behavior (unchanged)

```
agentspec 0.5.0 (commit abc1234, built 2026-03-18, lang 3.0, ir 2.0)
```

## New Behavior (with update check)

### Up to date

```
$ agentspec version
agentspec 0.5.0 (commit abc1234, built 2026-03-18, lang 3.0, ir 2.0)
```

### Update available

```
$ agentspec version
agentspec 0.5.0 (commit abc1234, built 2026-03-18, lang 3.0, ir 2.0)

Update available: 0.6.0 → https://github.com/szaher/agentspec/releases/tag/v0.6.0
  brew upgrade agentspec    (macOS)
  docker pull ghcr.io/szaher/agentspec:latest
```

### Network unavailable (silent fallback)

```
$ agentspec version
agentspec 0.5.0 (commit abc1234, built 2026-03-18, lang 3.0, ir 2.0)
```

## Implementation Notes

- HTTP GET to `https://api.github.com/repos/szaher/agentspec/releases/latest`
- 2-second timeout; on error or timeout, silently skip update notice
- Compare `tag_name` (strip leading `v`) against compiled `version`
- Only print notice if remote version is strictly newer (semver comparison)
