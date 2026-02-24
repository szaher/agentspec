# SDK Generation

The SDK generator produces typed client libraries for the AgentSpec runtime HTTP API. It generates clients in Python, TypeScript, and Go from the runtime configuration, enabling consumers to invoke agents, manage sessions, stream responses, and run pipelines with a native developer experience.

## Package

| Package | Path | Purpose |
|---------|------|---------|
| `sdk/generator` | `internal/sdk/generator/` | Template engine and language-specific generators |

Related sub-packages for generated output:

| Path | Purpose |
|------|---------|
| `internal/sdk/python/` | Python SDK templates and generated code |
| `internal/sdk/typescript/` | TypeScript SDK templates and generated code |
| `internal/sdk/go/` | Go SDK templates and generated code |

## Architecture

```text
  RuntimeConfig (from IR)
        |
        v
  +---------------------+
  | buildTemplateData() |  Extract agents, pipelines, package name
  +----------+----------+
             |
             v
  +---------------------+
  |  Language Generator  |  generatePython() / generateTypeScript() / generateGo()
  +----------+----------+
             |
             v
  +---------------------+
  |  Go text/template   |  Render templates with agent/pipeline data
  +----------+----------+
             |
             v
  Generated SDK files
```

## Entry Points

### Single Language

```go
func Generate(cfg Config) error
```

```go
cfg := generator.Config{
    Language:      generator.LangPython,
    OutDir:        "./sdk/python",
    RuntimeConfig: runtimeConfig,
}
err := generator.Generate(cfg)
```

### All Languages

```go
func GenerateAll(baseDir string, rc *runtime.RuntimeConfig) error
```

Generates Python, TypeScript, and Go SDKs under `baseDir/python/`, `baseDir/typescript/`, and `baseDir/go/` respectively.

## Configuration

```go
type Language string

const (
    LangPython     Language = "python"
    LangTypeScript Language = "typescript"
    LangGo         Language = "go"
)

type Config struct {
    Language      Language
    OutDir        string
    RuntimeConfig *runtime.RuntimeConfig
}
```

If `RuntimeConfig` is nil, a generic client is generated without agent-specific type constants. When provided, the generator extracts agent names, models, strategies, skills, and pipeline definitions to produce typed constants.

## Template Data

The generator builds a `templateData` struct from the runtime configuration:

```go
type templateData struct {
    PackageName string
    Agents      []agentData
    Pipelines   []pipelineData
}

type agentData struct {
    Name       string   // original name (e.g., "code-reviewer")
    NameTitle  string   // PascalCase (e.g., "CodeReviewer")
    NameConst  string   // UPPER_SNAKE_CASE (e.g., "CODE_REVIEWER")
    Model      string
    Strategy   string
    Skills     []string
    HasSession bool
}

type pipelineData struct {
    Name      string
    NameTitle string
    NameConst string
    Steps     []string
}
```

Name transformations:

| Input | PascalCase (`NameTitle`) | UPPER_SNAKE_CASE (`NameConst`) |
|-------|--------------------------|-------------------------------|
| `code-reviewer` | `CodeReviewer` | `CODE_REVIEWER` |
| `data_processor` | `DataProcessor` | `DATA_PROCESSOR` |
| `assistant` | `Assistant` | `ASSISTANT` |

## Generated Code Structure

### Python

```text
sdk/python/
  __init__.py         # Package exports
  client.py           # AgentSpecClient class + all types
  pyproject.toml      # Package metadata
```

The generated `AgentSpecClient` class provides:

```text
health()                    -> dict
list_agents()               -> list[AgentInfo]
invoke(agent, message, ...) -> InvokeResponse
stream(agent, message, ...) -> Generator[StreamEvent]
create_session(agent, ...)  -> SessionInfo
send_message(agent, sid, msg) -> InvokeResponse
delete_session(agent, sid)  -> None
run_pipeline(name, trigger) -> PipelineResult
```

Agent and pipeline name constants are generated:

```text
AGENT_CODE_REVIEWER = "code-reviewer"
PIPELINE_CI_CHECK = "ci-check"
```

### TypeScript

```text
sdk/typescript/
  index.ts            # Client class, types, and constants
  package.json        # npm package metadata
  tsconfig.json       # TypeScript configuration
```

The generated `AgentSpecClient` class mirrors the Python client with TypeScript types:

```text
health(): Promise<Record<string, unknown>>
listAgents(): Promise<AgentInfo[]>
invoke(agent, message, opts?): Promise<InvokeResponse>
createSession(agent, metadata?): Promise<SessionInfo>
sendMessage(agent, sessionId, message): Promise<InvokeResponse>
deleteSession(agent, sessionId): Promise<void>
runPipeline(name, trigger): Promise<PipelineResult>
```

Constants:

```text
export const AGENT_CODE_REVIEWER = "code-reviewer";
export const PIPELINE_CI_CHECK = "ci-check";
```

### Go

```text
sdk/go/
  go.mod              # Go module file
  agentspec/
    client.go         # Client struct, methods, and types
```

The generated Go client follows standard Go conventions:

```text
NewClient(baseURL, opts...) -> *Client
client.Invoke(ctx, agent, message, vars) -> (*InvokeResponse, error)
client.Stream(ctx, agent, message, callback) -> error
client.ListAgents(ctx) -> ([]AgentInfo, error)
client.CreateSession(ctx, agent) -> (*SessionInfo, error)
client.SendMessage(ctx, agent, sessionID, message) -> (*InvokeResponse, error)
client.DeleteSession(ctx, agent, sessionID) -> error
client.RunPipeline(ctx, name, trigger) -> (*PipelineResult, error)
```

Constants:

```text
const AgentCodeReviewer = "code-reviewer"
const PipelineCiCheck = "ci-check"
```

## Template System

The generator uses Go's `text/template` package. Each language has its client template defined as a Go string constant:

```go
const pythonClientTemplate = `"""Generated AgentSpec SDK client..."""
...
{{ range .Agents }}
AGENT_{{ .NameConst }} = "{{ .Name }}"
{{ end }}`
```

The `writeTemplate()` helper parses and executes the template:

```go
func writeTemplate(path, name, tmplStr string, data interface{}) error {
    t, err := template.New(name).Parse(tmplStr)
    if err != nil {
        return fmt.Errorf("parse template %s: %w", name, err)
    }
    f, err := os.Create(path)
    if err != nil {
        return fmt.Errorf("create %s: %w", path, err)
    }
    if err := t.Execute(f, data); err != nil {
        f.Close()
        return err
    }
    return f.Close()
}
```

## Generated API Coverage

All generated clients cover the same API surface:

| Endpoint | Method | Client Method |
|----------|--------|---------------|
| `/healthz` | GET | `health()` |
| `/v1/agents` | GET | `listAgents()` |
| `/v1/agents/{name}/invoke` | POST | `invoke()` |
| `/v1/agents/{name}/stream` | POST | `stream()` |
| `/v1/agents/{name}/sessions` | POST | `createSession()` |
| `/v1/agents/{name}/sessions/{id}` | POST | `sendMessage()` |
| `/v1/agents/{name}/sessions/{id}` | DELETE | `deleteSession()` |
| `/v1/pipelines/{name}/run` | POST | `runPipeline()` |

## Error Handling

All generated clients define typed error types:

| Language | Base Error | API Error |
|----------|-----------|-----------|
| Python | `AgentSpecError` | `APIError(status_code, error_code, message)` |
| TypeScript | `AgentSpecError` | `APIError` with `statusCode` and `errorCode` |
| Go | `*APIError` | `APIError{StatusCode, ErrorCode, Message}` |

API errors are parsed from the server's JSON error response:

```json
{
  "error": "agent_not_found",
  "message": "agent 'unknown' is not deployed"
}
```

## Adding a New Language Target

To add support for a new language (e.g., Java):

1. Define a `LangJava` constant in the `Language` type.
2. Write the client template as a Go string constant (following the existing patterns).
3. Implement `generateJava(cfg Config) error`:
   - Create the output directory.
   - Build template data with `buildTemplateData(cfg)`.
   - Write the client source via `writeTemplate()`.
   - Write any package metadata files (e.g., `pom.xml`).
4. Add the case to the `Generate()` switch.
5. Add the language to the `GenerateAll()` loop.
6. Add tests verifying the generated code compiles and matches expected output.
