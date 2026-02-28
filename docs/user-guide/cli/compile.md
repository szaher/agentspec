# compile

Compile IntentLang spec files into a deployable agent artifact.

## Usage

```bash
agentspec compile [file.ias | directory]
```

## Description

The `compile` command transforms IntentLang (`.ias`) agent definitions into a standalone executable or framework-specific source code. The default `standalone` target produces a self-contained Go binary with the agent configuration, health checks, API endpoints, and an optional built-in frontend all embedded directly into the binary.

You can pass one or more `.ias` files, or a directory containing them. When given a directory, the compiler discovers all `.ias` files within it.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--target` | | `standalone` | Compilation target (see targets below) |
| `--output` | `-o` | `./build` | Output directory for compiled artifacts |
| `--platform` | | *(host platform)* | Target platform for cross-compilation (e.g. `linux/amd64`, `darwin/arm64`) |
| `--name` | | *(package name)* | Output binary or project name |
| `--embed-frontend` | | `true` | Embed the built-in frontend in the compiled binary |
| `--verbose` | `-v` | `false` | Enable verbose output |
| `--json` | | `false` | Output result as JSON |

## Targets

| Target | Description |
|--------|-------------|
| `standalone` | Self-contained Go binary with embedded config, API server, and frontend |
| `crewai` | Generate a CrewAI-compatible Python project |
| `langgraph` | Generate a LangGraph-compatible Python project |
| `llamaindex` | Generate a LlamaIndex-compatible Python project |
| `llamastack` | Generate a LlamaStack-compatible Python project |

## Cross-Compilation Platforms

Use the `--platform` flag to cross-compile for a different OS and architecture. The value follows the `GOOS/GOARCH` convention:

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`

When `--platform` is omitted, the binary is compiled for the host platform.

## Examples

```bash
# Compile to a standalone binary (default)
agentspec compile agent.ias

# Compile all .ias files in a directory
agentspec compile ./specs/

# Compile for a specific platform
agentspec compile --platform linux/amd64 agent.ias

# Compile to a CrewAI project
agentspec compile --target crewai agent.ias

# Compile to a LangGraph project
agentspec compile --target langgraph --output ./dist agent.ias

# Compile without the built-in frontend
agentspec compile --embed-frontend=false agent.ias

# Compile with a custom binary name and JSON output
agentspec compile --name my-agent --json agent.ias
```

## Output

On success, the command prints a summary of the compilation:

```
Compiling agent.ias...
  Parsed 1 file(s)
  Compiled to standalone binary

Output: ./build/agent (12.4 MB)
Platform: darwin/arm64
Agents: my-agent
Config: embedded
Time: 1823ms
```

Pass `--json` to get machine-readable output instead.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Compilation succeeded |
| `1` | An error occurred (parse failure, compilation error, etc.) |

## See Also

- [CLI: package](package.md) -- Package a compiled binary for deployment
- [CLI: run](run.md) -- Run an agent locally without compiling
- [CLI: eval](eval.md) -- Evaluate agent quality with test cases
