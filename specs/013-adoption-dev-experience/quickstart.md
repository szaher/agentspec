# Quickstart: Developer Setup

**Feature**: 013-adoption-dev-experience

## Prerequisites

- Go 1.25+ (for development/building from source only)
- Git (for cloning the repository)

## Build from Source (Development)

```bash
git clone https://github.com/szaher/agentspec.git
cd agentspec
go build -o agentspec ./cmd/agentspec
./agentspec version
```

## Work Areas for This Feature

### 1. GoReleaser Enhancement

**Files to modify**:
- `.goreleaser.yaml` — add `brews` and `dockers` sections

**How to test**:
```bash
goreleaser check                    # Validate config syntax
goreleaser release --snapshot --clean  # Local dry-run (no publish)
```

### 2. Template Enhancement

**Files to modify**:
- `internal/templates/templates.go` — update Template struct, add new templates
- `internal/templates/files/` — restructure to directories with README.md per template

**Files to create**:
- `internal/templates/files/basic-chatbot/` — new template
- `internal/templates/files/incident-response/` — new template
- `internal/templates/files/research-swarm/` — new template
- `internal/templates/files/multi-agent-router/` — new template

**How to test**:
```bash
go test ./internal/templates/... -count=1
go build -o agentspec ./cmd/agentspec
./agentspec init --list-templates
./agentspec init --template basic-chatbot --output-dir /tmp/test-scaffold
./agentspec validate /tmp/test-scaffold/basic-chatbot/basic-chatbot.ias
```

### 3. Init Command Enhancement

**Files to modify**:
- `cmd/agentspec/init.go` — add interactive selector, overwrite confirmation, README scaffolding

**How to test**:
```bash
go test ./cmd/agentspec/... -count=1
echo "1" | ./agentspec init                           # Interactive mode
./agentspec init --template support-bot --output-dir /tmp/test
```

### 4. Version Command Enhancement

**Files to modify**:
- `cmd/agentspec/version.go` — add GitHub Releases API check

**How to test**:
```bash
./agentspec version        # Should show version + update notice if outdated
```

### 5. Example Projects

**Files to create/enhance**:
- `examples/research-swarm/` — new multi-agent example
- `examples/incident-response/` — new example
- `examples/gpu-batch/` — new example (external API dispatch)
- `examples/multi-agent-router/` — new or enhanced from control-flow-agent
- `examples/customer-support/README.md` — enhance existing
- `examples/rag-chatbot/README.md` — enhance existing

**How to test**:
```bash
./agentspec validate examples/research-swarm/research-swarm.ias
./agentspec validate examples/incident-response/incident-response.ias
./agentspec validate examples/gpu-batch/gpu-batch.ias
./agentspec validate examples/multi-agent-router/multi-agent-router.ias
```

### 6. Documentation

**Files to create**:
- `docs/quickstart.md` — quickstart guide page for MkDocs site

**How to test**:
```bash
cd docs-tools && pip install -r requirements.txt
mkdocs build   # Verify site builds
```

## Validation Checklist

```bash
# Pre-commit checks
gofmt -l .
go build ./...
go test ./... -count=1

# Feature-specific checks
./agentspec init --list-templates          # Shows 6+ templates
./agentspec init --template basic-chatbot  # Scaffolds project
./agentspec validate examples/*/*.ias      # All examples validate
./agentspec version                        # Shows version info
```
