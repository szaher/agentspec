# AgentSpec

A declarative toolchain for defining, validating, and deploying AI agent systems. AgentSpec uses **IntentLang**, a purpose-built DSL, to describe agents, prompts, skills, MCP servers, and deployment targets in a single source of truth.

## Quick Start

```bash
# Build
make build

# Write an agent spec
cat > hello.ias <<'EOF'
package "hello" version "0.1.0" lang "2.0"

prompt "system" {
  content "You are a helpful assistant."
}

skill "greet" {
  description "Greet the user"
  input { name string required }
  output { message string }
  tool command { binary "greet-tool" }
}

agent "assistant" {
  uses prompt "system"
  uses skill "greet"
  model "claude-sonnet-4-20250514"
}

deploy "local" target "process" {
  default true
}
EOF

# Validate, plan, and apply
./agentspec validate hello.ias
./agentspec plan hello.ias
./agentspec apply hello.ias --auto-approve
```

## Installation

Requires Go 1.25+.

```bash
git clone https://github.com/szaher/designs.git
cd designs/agentz
make build
```

The binary is built as `./agentspec`.

## CLI Commands

| Command | Description |
|---------|-------------|
| `validate <file>` | Check syntax and semantic correctness |
| `fmt <file>` | Format to canonical style |
| `plan <file>` | Preview changes without applying |
| `apply <file>` | Apply changes idempotently |
| `export <file>` | Generate platform-specific artifacts |
| `diff <file>` | Show detailed resource differences |
| `sdk <file>` | Generate client SDKs (Python, TypeScript, Go) |
| `migrate [path]` | Rename `.az` files to `.ias` |
| `migrate --to-v2` | Rewrite IntentLang 1.0 files to 2.0 syntax |
| `run <file>` | Start the agent runtime server |
| `dev <file>` | Development mode with file watching |
| `version` | Display version information |

### Common Flags

```
--state-file <path>    State file path (default: .agentspec.state.json)
--verbose              Enable verbose output
--no-color             Disable colored output
--correlation-id <id>  Set a correlation ID for tracing
```

### Usage Examples

```bash
# Format and check
./agentspec fmt examples/basic-agent/basic-agent.ias
./agentspec fmt --check examples/basic-agent/basic-agent.ias

# Validate
./agentspec validate examples/basic-agent/basic-agent.ias

# Plan and apply
./agentspec plan examples/basic-agent/basic-agent.ias
./agentspec apply examples/basic-agent/basic-agent.ias --auto-approve

# Environment overlays
./agentspec plan examples/multi-environment/multi-environment.ias --env dev
./agentspec plan examples/multi-environment/multi-environment.ias --env prod

# Export artifacts for a specific target
./agentspec export examples/multi-binding/multi-binding.ias --target compose --out-dir ./output

# Generate SDKs
./agentspec sdk examples/basic-agent/basic-agent.ias --language python --out-dir ./sdk
```

## IntentLang Syntax

IntentLang (`.ias` files) is a declarative language for defining agent specifications. Every file starts with a package header:

```
package "my-app" version "0.1.0" lang "2.0"
```

### Prompts

```
prompt "system" {
  content "You are a helpful assistant."
  version "1.0"
}
```

Prompts support template variables:

```
prompt "greeting" {
  content "Hello {{name}}, you are a {{role}}."
  variables {
    name string required
    role string required default "assistant"
  }
}
```

### Skills

Skills define tool capabilities with typed input/output schemas:

```
skill "web-search" {
  description "Search the web for information"
  input {
    query string required
  }
  output {
    results string
  }
  tool command {
    binary "search-tool"
  }
}
```

Tool types: `command`, `http`, `mcp`, `inline`.

### Agents

Agents compose prompts, skills, and runtime configuration:

```
agent "researcher" {
  uses prompt "system"
  uses skill "web-search"
  uses skill "summarize"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 10
  timeout "30s"
  temperature 0.7
  stream true
  on_error "fallback"
  fallback "backup-agent"
}
```

Agents can delegate to other agents:

```
agent "router" {
  uses prompt "router-prompt"
  model "claude-sonnet-4-20250514"
  delegate to agent "searcher" when "user asks for information"
  delegate to agent "calculator" when "user asks for math"
}
```

### Type Definitions

```
type "user" {
  name string required
  email string required
  age int
  active bool default "true"
}

type "status" enum ["pending", "active", "archived"]

type "tags" list string
```

### Pipelines

```
pipeline "data-report" {
  step "fetch" {
    agent "data-analyst"
    input "raw data source"
    output "processed data"
  }
  step "analyze" {
    agent "data-analyst"
    depends_on ["fetch"]
  }
  step "report" {
    agent "report-writer"
    depends_on ["analyze"]
    output "final report"
  }
}
```

### MCP Servers and Clients

```
server "file-server" {
  transport "stdio"
  command "file-mcp-server"
  exposes skill "file-read"
}

client "my-client" {
  connects to server "file-server"
}
```

### Deploy Targets

```
deploy "local" target "process" {
  default true
  port 8080
  health {
    path "/healthz"
  }
}

deploy "staging" target "docker-compose" {
}

deploy "production" target "kubernetes" {
  namespace "agents"
  replicas 3
}
```

### Secrets, Environments, and Policies

```
secret "api-key" {
  env(API_KEY)
}

environment "dev" {
  agent "assistant" {
    model "claude-haiku-latest"
  }
}

policy "production-safety" {
  deny model claude-haiku-latest
  require secret api-key
}
```

### Plugins

```
plugin "monitor" version "1.0.0"
```

Plugins are sandboxed WASM modules loaded from `~/.agentspec/plugins/`.

## Architecture

```
.ias source --> Lexer --> Parser --> AST --> Validator --> IR --> Plan --> Apply
                                                          |              |
                                                          v              v
                                                       Export        State File
```

The toolchain processes `.ias` files through a pipeline:

1. **Parser**: Hand-written recursive descent parser produces an AST
2. **Validator**: Two-phase validation (structural then semantic) with "did you mean?" suggestions
3. **IR**: Intermediate representation with SHA-256 content hashing for change detection
4. **Plan**: Desired-state diff engine computes create/update/delete/noop actions
5. **Apply**: Idempotent applier with partial failure handling and event emission
6. **Export**: Platform-specific artifact generation (JSON manifests, Docker Compose, etc.)

All outputs are deterministic — running the same command twice produces byte-identical results.

## Project Structure

```
cmd/agentspec/         CLI binary
internal/
  ast/                 Abstract syntax tree node types
  parser/              Recursive descent parser and lexer
  formatter/           Canonical deterministic formatter
  validate/            Structural and semantic validation
  ir/                  Intermediate representation
  plan/                Desired-state diff engine
  apply/               Idempotent applier
  state/               State backend (local JSON file)
  adapters/            Platform adapters (process, docker-compose)
  migrate/             IntentLang 1.0 to 2.0 migration
  runtime/             Agent runtime HTTP server
  loop/                ReAct agentic loop
  llm/                 LLM client abstraction
  mcp/                 MCP protocol integration
  tools/               Tool registry and executors
  session/             Session management
  memory/              Agent memory (sliding window)
  secrets/             Secret resolution
  policy/              Policy enforcement
  plugins/             WASM plugin system
  events/              Structured event emitter
  sdk/generator/       SDK code generation
examples/              IntentLang example files
integration_tests/     Integration test suite
```

## Examples

The `examples/` directory contains 10 self-contained examples:

| Example | Description |
|---------|-------------|
| [basic-agent](examples/basic-agent/) | Minimal agent with a prompt and deploy target |
| [multi-skill-agent](examples/multi-skill-agent/) | Agent with multiple skills |
| [mcp-server-client](examples/mcp-server-client/) | MCP server and client connectivity |
| [multi-environment](examples/multi-environment/) | Dev/prod environment overlays |
| [multi-binding](examples/multi-binding/) | Multiple deploy targets |
| [customer-support](examples/customer-support/) | Secrets, environments, escalation |
| [code-review-pipeline](examples/code-review-pipeline/) | Multi-agent code review |
| [data-pipeline](examples/data-pipeline/) | ETL with policies and secrets |
| [rag-chatbot](examples/rag-chatbot/) | RAG with vector search and MCP |
| [plugin-usage](examples/plugin-usage/) | WASM plugin integration |

## Development

### Prerequisites

- Go 1.25+
- [golangci-lint](https://golangci-lint.run/) v2.10+

### Make Targets

```bash
make all          # Lint, test, and build (default)
make build        # Build the agentspec binary
make test         # Run all tests
make test-v       # Run tests with verbose output
make test-race    # Run tests with the race detector
make lint         # Run golangci-lint
make fmt          # Format all Go source files
make fmt-check    # Check formatting without modifying files
make vet          # Run go vet
make validate     # Build and validate all example .ias files
make clean        # Remove the binary and state file
```

### Running Tests

```bash
# All tests
make test

# Verbose
make test-v

# Specific test
go test ./integration_tests/ -run TestV2Pipeline -v
```

The test suite includes 65 integration tests covering parsing, validation, formatting, IR lowering, plan/apply idempotency, export, runtime, sessions, tools, plugins, SDK generation, and secrets.

### CI

GitHub Actions runs on every push and pull request:

- Build
- Test (`go test ./... -count=1 -v`)
- Lint (golangci-lint v2.10.1)
- Validate all examples
- Format-check all examples
- Smoke tests (plan, apply, idempotency)

## License

See [LICENSE](LICENSE) for details.
