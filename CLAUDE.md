# AgentSpec Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-02-23

## Active Technologies
- Go 1.25+ + wazero v1.11.0 (WASM plugin sandbox), cobra v1.10.2, go-cmp v0.7.0 (001-agent-packaging-dsl)
- YAML (GitHub Actions workflow syntax) + GitHub Actions, actions/checkout, actions/setup-go, golangci/golangci-lint-action (002-ci-pipeline)
- Go 1.25+ (existing) (004-runtime-platform)
- Local JSON state file (existing `.agentspec.state.json`). In-memory session store (new, default). Redis session store (new, opt-in). (004-runtime-platform)
- Python 3.9+ (MkDocs), Go 1.25+ (example validation), Markdown (content) (005-docs-site)
- N/A (static site, no database) (005-docs-site)
- Go 1.25+ + wazero v1.11.0 (WASM sandbox), cobra v1.10.2 (CLI), anthropic-sdk-go, go-mcp-sdk (007-security-hardening)
- Local JSON state file (`.agentspec.state.json`), in-memory session store, Redis session store (007-security-hardening)
- Go 1.25+ (existing) + `syscall` (flock, existing), `go-redis` (existing), `crypto/rand` (existing) (008-state-data-integrity)
- Local JSON state file (`.agentspec.state.json`), Redis (session messages) (008-state-data-integrity)
- Go 1.25+ (existing) + golangci-lint v2.10.1 (existing), govulncheck (new), gosec (new via golangci-lint) (009-test-quality-foundation)
- N/A (testing/CI infrastructure only) (009-test-quality-foundation)

## Project Structure

```text
cmd/agentspec/     # CLI binary (formerly cmd/agentz/)
internal/          # Core packages
integration_tests/ # Integration test suite
examples/          # IntentLang (.ias) example files
```

## Commands

- `go build -o agentspec ./cmd/agentspec` — build the CLI binary
- `go test ./... -count=1` — run all tests
- `./agentspec validate <file.ias>` — validate an IntentLang file
- `./agentspec fmt <file.ias>` — format an IntentLang file
- `./agentspec plan <file.ias>` — show planned changes
- `./agentspec apply <file.ias>` — apply changes

## Code Style

Go 1.25+: Follow standard conventions

## Naming

- **IntentLang** (ilang): The declarative language for defining agent specifications
- **AgentSpec**: An individual definition file written in IntentLang (file extension: `.ias`)
- **AgentPack**: A distributable bundle of one or more AgentSpec files
- CLI binary: `agentspec` (formerly `agentz`)
- State file: `.agentspec.state.json` (auto-migrates from `.agentz.state.json`)
- Plugin directory: `~/.agentspec/plugins/` (fallback: `~/.agentz/plugins/`)
- Go module path: `github.com/szaher/designs/agentz` (unchanged)

## Recent Changes
- 009-test-quality-foundation: Added Go 1.25+ (existing) + golangci-lint v2.10.1 (existing), govulncheck (new), gosec (new via golangci-lint)
- 008-state-data-integrity: Added Go 1.25+ (existing) + `syscall` (flock, existing), `go-redis` (existing), `crypto/rand` (existing)
- 007-security-hardening: Added Go 1.25+ + wazero v1.11.0 (WASM sandbox), cobra v1.10.2 (CLI), anthropic-sdk-go, go-mcp-sdk

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
