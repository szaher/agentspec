# Build from Source

This guide covers setting up a development environment, building the AgentSpec CLI, and running the test suite.

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25+ | Build and test |
| golangci-lint | v2.10+ | Linting |
| git | any | Source control |
| Docker | 20.10+ | (optional) Running adapter integration tests |

Install Go from [go.dev/dl](https://go.dev/dl/) and verify:

```bash
go version
# go version go1.25.0 ...
```

Install golangci-lint:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
golangci-lint version
# golangci-lint has version v2.10...
```

## Clone the Repository

```bash
git clone https://github.com/szaher/designs.git
cd designs/agentz
```

The Go module path is `github.com/szaher/designs/agentz`. All import paths use this prefix.

## Build

Build the CLI binary:

```bash
go build -o agentspec ./cmd/agentspec
```

Verify the build:

```bash
./agentspec --help
```

## Test

Run all tests:

```bash
go test ./... -count=1
```

The `-count=1` flag disables test caching, ensuring tests always execute fresh.

Run tests for a specific package:

```bash
go test ./internal/parser/ -count=1 -v
```

Run integration tests:

```bash
go test ./integration_tests/ -count=1 -v
```

## Lint

Run the linter:

```bash
golangci-lint run ./...
```

The linting configuration is in `.golangci.yml` at the repository root.

## Format

Check formatting:

```bash
gofmt -l .
```

Fix formatting:

```bash
gofmt -w .
```

## Validate Example Files

Validate all `.ias` example files:

```bash
./agentspec validate examples/*.ias
```

Format example files:

```bash
./agentspec fmt examples/*.ias
```

## Make Targets

If a Makefile is present, these targets are available:

| Target | Command | Purpose |
|--------|---------|---------|
| `build` | `go build -o agentspec ./cmd/agentspec` | Build the CLI |
| `test` | `go test ./... -count=1` | Run all tests |
| `lint` | `golangci-lint run ./...` | Run linter |
| `fmt` | `gofmt -w .` | Format all Go source |
| `validate` | `./agentspec validate examples/*.ias` | Validate example files |
| `clean` | `rm -f agentspec` | Remove build artifacts |

## Project Structure

```text
cmd/agentspec/         # CLI entrypoint (main package)
internal/              # Core library packages
  ast/                 # AST node types
  parser/              # Lexer and parser
  formatter/           # Canonical .ias formatter
  validate/            # Semantic validation
  ir/                  # Intermediate Representation
  plan/                # Plan engine (desired-state diff)
  apply/               # Action execution
  state/               # State persistence
  adapters/            # Adapter interface and implementations
    process/           #   Local process adapter
    local/             #   Local MCP adapter
    docker/            #   Docker adapter
    compose/           #   Docker Compose adapter
    kubernetes/        #   Kubernetes adapter
  plugins/             # WASM plugin host
  runtime/             # Agent runtime server
  loop/                # Agentic loop strategies
  llm/                 # LLM client abstraction
  tools/               # Tool registry and executors
  mcp/                 # MCP client and discovery
  session/             # Session management
  memory/              # Conversation memory
  secrets/             # Secret resolution
  policy/              # Policy enforcement
  events/              # Structured events
  migrate/             # Migration utilities
  pipeline/            # Pipeline execution
  templates/           # Code generation templates
  sdk/generator/       # SDK generator (Python, TS, Go)
  cli/                 # CLI command definitions
  telemetry/           # Telemetry collection
integration_tests/     # End-to-end integration tests
examples/              # Sample .ias files
docs/                  # Documentation (MkDocs site)
```

## IDE Setup

### VS Code

Install the Go extension (`golang.go`). Add to `.vscode/settings.json`:

```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"],
  "go.testFlags": ["-count=1"]
}
```

### GoLand / IntelliJ

The project should be detected automatically. Set the GOROOT to Go 1.25+ and configure golangci-lint as the external linter.

## Dependency Management

Dependencies are managed via Go modules. To add a dependency:

```bash
go get github.com/some/package@latest
go mod tidy
```

Key dependencies:

| Module | Version | Purpose |
|--------|---------|---------|
| `github.com/tetratelabs/wazero` | v1.11.0 | WASM plugin sandbox |
| `github.com/spf13/cobra` | v1.10.2 | CLI framework |
| `github.com/google/go-cmp` | v0.7.0 | Test comparison |
