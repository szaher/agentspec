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
- Go 1.25+ + `log/slog` (structured logging), `sync` (RWMutex), `oklog/ulid` (correlation IDs), existing Redis client interface, `encoding/json` (state file) (010-memory-performance)
- Local JSON state file (`.agentspec.state.json`), Redis (session store, opt-in), in-memory maps (rate limiter, session store default, conversation memory) (010-memory-performance)
- Go 1.25+ (backend), Vanilla JS (frontend) + cobra v1.10.2 (CLI), fsnotify (new — file watching), existing llm/loop/runtime packages (011-product-completeness)
- N/A (no schema changes) (011-product-completeness)
- Go 1.25+ (existing) + `crypto/tls` (stdlib), `log/slog` (stdlib), `regexp` (stdlib), cobra v1.10.2 (existing), fsnotify (existing), GoReleaser (CI only — not a Go dependency) (012-production-readiness)
- Local JSON state file (`.agentspec.state.json`) extended with `budgets` and `agent_versions` sections. Separate `agentspec-audit.log` file for audit entries. (012-production-readiness)
- Go 1.25+ (existing) + cobra v1.10.2 (existing CLI), GoReleaser v2 (existing release tooling), `embed` (stdlib, existing for templates) (013-adoption-dev-experience)
- N/A (no new state; templates embedded in binary, examples are static files) (013-adoption-dev-experience)
- Go 1.25+ (existing project language) + controller-runtime (kubebuilder framework), client-go, apimachinery, cobra v1.10.2 (existing CLI), sigs.k8s.io/controller-tools (CRD generation) (014-k8s-operator-control-plane)
- Kubernetes etcd (via CRDs), existing AgentSpec state file (`.agentspec.state.json`) for CLI bridge (014-k8s-operator-control-plane)
- Local JSON file (existing), Kubernetes CRDs, etcd, PostgreSQL, S3-compatible object storage (015-distributed-state-reconciliation)
- Go 1.25+ (existing project language) + cobra v1.10.2 (existing CLI), `embed` (stdlib, existing for web assets), `net/http` (stdlib, web server), `html/template` (stdlib, template rendering), `os/exec` (stdlib, browser launch), `encoding/json` (stdlib, API endpoint) (016-graph-visualization)
- N/A (read-only command, no state) (016-graph-visualization)

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
- `./agentspec run <file.ias>` — start runtime server with hot reload and web UI
- `./agentspec dev <file.ias> --input "msg"` — one-shot agent invocation
- `./agentspec eval <file.ias> --live` — evaluate agent with real LLM
- `./agentspec compile <file.ias> --target crewai` — compile to framework target

## Pre-Commit Checks (REQUIRED)

Before every commit, ALL of these must pass:
1. `gofmt -l .` — must produce no output (all files formatted)
2. `go build ./...` — must succeed with zero errors
3. `go test ./... -count=1` — must pass with zero failures

Do NOT create a commit if any check fails. Fix issues first.

## Code Style

Go 1.25+: Follow standard conventions

## Naming

- **IntentLang** (ilang): The declarative language for defining agent specifications
- **AgentSpec**: An individual definition file written in IntentLang (file extension: `.ias`)
- **AgentPack**: A distributable bundle of one or more AgentSpec files
- CLI binary: `agentspec` (formerly `agentz`)
- State file: `.agentspec.state.json` (auto-migrates from `.agentz.state.json`)
- Plugin directory: `~/.agentspec/plugins/` (fallback: `~/.agentz/plugins/`)
- Go module path: `github.com/szaher/agentspec` (unchanged)

## Recent Changes
- 016-graph-visualization: Added Go 1.25+ (existing project language) + cobra v1.10.2 (existing CLI), `embed` (stdlib, existing for web assets), `net/http` (stdlib, web server), `html/template` (stdlib, template rendering), `os/exec` (stdlib, browser launch), `encoding/json` (stdlib, API endpoint)
- 015-distributed-state-reconciliation: Added Go 1.25+ (existing)
- 014-k8s-operator-control-plane: Added Go 1.25+ (existing project language) + controller-runtime (kubebuilder framework), client-go, apimachinery, cobra v1.10.2 (existing CLI), sigs.k8s.io/controller-tools (CRD generation)

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
