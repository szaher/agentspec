# Runtime Architecture

The AgentSpec runtime is the execution engine that runs deployed agents. It manages the agent lifecycle, handles HTTP API requests, executes the agentic loop, dispatches tool calls, and maintains conversation sessions.

## Packages

| Package | Path | Purpose |
|---------|------|---------|
| `runtime` | `internal/runtime/` | Runtime server, lifecycle management, tool registration |
| `loop` | `internal/loop/` | Agentic loop strategies (ReAct, etc.) |
| `llm` | `internal/llm/` | LLM client abstraction (Anthropic, etc.) |
| `tools` | `internal/tools/` | Tool registry and executors (HTTP, command, inline, MCP) |
| `mcp` | `internal/mcp/` | Model Context Protocol client pool and tool discovery |
| `session` | `internal/session/` | Session lifecycle (in-memory, Redis) |
| `memory` | `internal/memory/` | Conversation memory strategies (sliding window, summary) |
| `secrets` | `internal/secrets/` | Secret resolution for tool configuration |

## Runtime Lifecycle

```text
  RuntimeConfig (from IR)
        |
        v
  +------------------+
  |  Runtime.New()   |  Create LLM client, MCP pool, tool registry,
  +--------+---------+  secret resolver, session manager, strategy
           |
           v
  +------------------+
  | Runtime.Start()  |  1. Connect to MCP servers
  +--------+---------+  2. Discover and register MCP tools
           |             3. Start HTTP server
           v
  +------------------+
  |  HTTP Server     |  /healthz, /v1/agents/*, /v1/pipelines/*
  +--------+---------+
           |
   (on shutdown signal)
           |
           v
  +------------------+
  | Runtime.Shutdown()|  Gracefully stop HTTP server, close MCP pool
  +------------------+
```

### Initialization

The `Runtime` struct orchestrates all runtime components:

```go
type Runtime struct {
    config   *RuntimeConfig
    server   *Server
    mcpPool  *agentmcp.Pool
    registry *tools.Registry
    logger   *slog.Logger
    apiKey   string
    port     int
}
```

`Runtime.New()` creates all subsystems:

1. **LLM Client** -- Default is `llm.NewAnthropicClient()`, can be overridden in options.
2. **MCP Pool** -- Connection pool for MCP server processes.
3. **Tool Registry** -- Central registry for all tool definitions and executors.
4. **Secret Resolver** -- Resolves secret references (env vars by default).
5. **Session Manager** -- In-memory session store with 30-minute TTL and sliding-window memory (50 messages).
6. **Strategy** -- Default is `loop.ReActStrategy{}`.

### Tool Registration

During initialization, the runtime registers tools from the `RuntimeConfig`:

| Tool Type | Executor | Registration |
|-----------|----------|-------------|
| `mcp` | `mcpToolExecutor` | Registered during `Start()` via MCP discovery |
| `http` | `tools.NewHTTPExecutor()` | Registered during `New()` |
| `command` | `tools.NewCommandExecutor()` | Registered during `New()` with resolved secrets |
| `inline` | `tools.NewInlineExecutor()` | Registered during `New()` with resolved secrets |

### MCP Integration

During `Start()`, the runtime:

1. Connects to each configured MCP server via the pool.
2. Runs tool discovery to find available tools.
3. Registers each discovered tool with the composite name `server-name/tool-name`.

## Agentic Loop

The agentic loop (`internal/loop/`) defines how agents reason and act. The primary interface:

```go
type Strategy interface {
    Execute(ctx context.Context, inv Invocation, llmClient llm.Client,
            tools ToolExecutor, onEvent StreamCallback) (*Response, error)
    Name() string
}
```

### ReAct Strategy

The default `ReActStrategy` implements the Reason-Act-Observe pattern:

```text
  User Input
      |
      v
  +-------+     +----------+     +----------+
  | Think  | --> | Act      | --> | Observe  |
  | (LLM)  |    | (Tools)  |    | (Results) |
  +-------+     +----------+     +----+-----+
      ^                               |
      |                               |
      +---------- loop ---------------+
      |
  (no more tool calls OR max turns reached)
      |
      v
  Final Output
```

The loop in detail:

1. Build the message history (existing context + new user input).
2. Send to LLM with available tool definitions.
3. If the LLM response contains tool calls:
   a. Execute all tool calls concurrently via `ToolExecutor.ExecuteConcurrent()`.
   b. Append tool results to the message history.
   c. Go to step 2 (next turn).
4. If no tool calls or stop reason is not `tool_use`, return the final output.

Constraints:

- **Max Turns** -- Configurable per agent, default 10. Prevents infinite loops.
- **Token Budget** -- Optional cap on total token consumption across all turns.
- **Streaming** -- When enabled, the loop pipes LLM stream events through the `StreamCallback`.

### Invocation

An `Invocation` captures all parameters for a single agent execution:

```go
type Invocation struct {
    AgentName   string
    Model       string
    System      string            // System prompt content
    Input       string            // User message
    Messages    []llm.Message     // Existing conversation context
    Variables   map[string]string // Template variables
    MaxTurns    int
    MaxTokens   int
    TokenBudget int
    Temperature *float64
    Stream      bool
}
```

### Response

The loop returns a `Response` with the output, tool call audit trail, token usage, and timing:

```go
type Response struct {
    Output    string
    ToolCalls []ToolCallRecord
    Tokens    llm.TokenUsage
    Turns     int
    Duration  time.Duration
    Error     string
}
```

## Tool Dispatch

The `ToolExecutor` interface supports both sequential and concurrent execution:

```go
type ToolExecutor interface {
    Execute(ctx context.Context, call llm.ToolCall) (string, error)
    ExecuteConcurrent(ctx context.Context, calls []llm.ToolCall) []llm.ToolResult
}
```

When the LLM requests multiple tool calls in a single turn, they are executed concurrently for better latency.

## Session Management

Sessions enable multi-turn conversations with persistent context.

```text
  Create Session -> Session ID
       |
       v
  Send Message (with Session ID)
       |
       v
  Load previous messages from memory store
       |
       v
  Run agentic loop with full context
       |
       v
  Save new messages to memory store
       |
       v
  Return response
```

The `session.Manager` coordinates two subsystems:

- **Store** (`session.Store`) -- Tracks session metadata (ID, agent name, timestamps). Implementations: `MemoryStore` (default), `RedisStore` (opt-in).
- **Memory** (`memory.Store`) -- Stores conversation message history. Implementations: `SlidingWindow` (keeps last N messages), `Summary` (summarizes older messages).

```go
type Manager struct {
    store  Store
    memory memory.Store
}
```

Key operations:

```go
func (m *Manager) Create(ctx, agentName, metadata) (*Session, error)
func (m *Manager) LoadMessages(ctx, sessionID) ([]llm.Message, error)
func (m *Manager) SaveMessages(ctx, sessionID, messages) error
func (m *Manager) Close(ctx, sessionID) error
```

## Streaming

When streaming is enabled, the runtime uses Server-Sent Events (SSE) to push incremental responses:

1. The HTTP handler sets `Content-Type: text/event-stream`.
2. The agentic loop calls the `StreamCallback` with each `llm.StreamEvent`.
3. Events include: `content_delta` (text chunks), `tool_call_start`, `tool_call_end`, `done`.
4. The client reads the SSE stream and processes events incrementally.
