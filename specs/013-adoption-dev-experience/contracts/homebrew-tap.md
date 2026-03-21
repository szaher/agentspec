# Homebrew Tap Contract

## Tap Repository

`szaher/homebrew-agentspec` on GitHub.

## Install Command

```bash
brew install szaher/agentspec/agentspec
```

Or with tap add:

```bash
brew tap szaher/agentspec
brew install agentspec
```

## Formula Behavior

| Property | Value |
|----------|-------|
| Binary name | `agentspec` |
| Platforms | macOS (Intel amd64, Apple Silicon arm64) |
| Dependencies | None (static binary) |
| Test | `system bin/"agentspec", "version"` |

## Automation

GoReleaser's `brews` section auto-generates and pushes the formula to the tap repository on each tagged release. No manual formula maintenance required.

## GoReleaser Configuration (to add)

```yaml
brews:
  - name: agentspec
    repository:
      owner: szaher
      name: homebrew-agentspec
    homepage: "https://github.com/szaher/agentspec"
    description: "Declarative language for defining and deploying AI agents"
    license: "Apache-2.0"
    test: |
      system "#{bin}/agentspec", "version"
    install: |
      bin.install "agentspec"
```
