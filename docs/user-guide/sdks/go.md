# Go SDK

The AgentSpec Go SDK provides a typed client for the AgentSpec HTTP API. It follows Go conventions with context-based cancellation, streaming via channels, and structured error handling.

---

## Installation

```bash
go get github.com/szaher/designs/agentz/sdk
```

**Requirements:** Go 1.25+

---

## Client Initialization

### Basic Setup

```go
package main

import (
    "context"
    "fmt"
    "log"

    agentspec "github.com/szaher/designs/agentz/sdk"
)

func main() {
    client, err := agentspec.NewClient(
        agentspec.WithBaseURL("http://localhost:8080"),
        agentspec.WithAPIKey("your-api-key"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    fmt.Println("Client initialized")
}
```

### Bearer Token Authentication

```go
client, err := agentspec.NewClient(
    agentspec.WithBaseURL("http://localhost:8080"),
    agentspec.WithToken("your-bearer-token"),
)
```

### Environment Variables

The client reads configuration from environment variables when options are not provided:

```bash
export AGENTSPEC_BASE_URL="http://localhost:8080"
export AGENTSPEC_API_KEY="your-api-key"
```

```go
// Picks up AGENTSPEC_BASE_URL and AGENTSPEC_API_KEY automatically
client, err := agentspec.NewClient()
```

### Custom Options

```go
client, err := agentspec.NewClient(
    agentspec.WithBaseURL("http://localhost:8080"),
    agentspec.WithAPIKey("your-api-key"),
    agentspec.WithTimeout(30 * time.Second),
    agentspec.WithMaxRetries(3),
    agentspec.WithRetryDelay(time.Second),
)
```

---

## Listing Agents

```go
ctx := context.Background()

agents, err := client.Agents.List(ctx)
if err != nil {
    log.Fatal(err)
}

for _, agent := range agents {
    fmt.Printf("%s - %s - %s\n", agent.Name, agent.Model, agent.Status)
}
```

---

## Invoking an Agent

### Basic Invocation

```go
ctx := context.Background()

response, err := client.Agents.Invoke(ctx, &agentspec.InvokeRequest{
    Name:  "assistant",
    Input: "What is the capital of France?",
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Output)
// "The capital of France is Paris."

fmt.Printf("Tokens used: %d\n", response.Usage.TotalTokens)
```

### With Options

```go
response, err := client.Agents.Invoke(ctx, &agentspec.InvokeRequest{
    Name:  "assistant",
    Input: "Write a haiku about programming.",
    Options: &agentspec.InvokeOptions{
        Temperature: agentspec.Float64(0.9),
        MaxTokens:   agentspec.Int(100),
    },
})
```

### With Context Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

response, err := client.Agents.Invoke(ctx, &agentspec.InvokeRequest{
    Name:  "assistant",
    Input: "Hello",
})
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("Request timed out")
        return
    }
    log.Fatal(err)
}
```

---

## Streaming Responses

### Channel-Based Streaming

```go
ctx := context.Background()

stream, err := client.Agents.Stream(ctx, &agentspec.StreamRequest{
    Name:  "assistant",
    Input: "Explain how compilers work.",
})
if err != nil {
    log.Fatal(err)
}

for event := range stream.Events() {
    switch event.Type {
    case agentspec.EventToken:
        fmt.Print(event.Content)
    case agentspec.EventToolCall:
        fmt.Printf("\n[Calling tool: %s]\n", event.Name)
    case agentspec.EventToolResult:
        fmt.Printf("\n[Tool result: %s]\n", event.Output)
    case agentspec.EventDone:
        fmt.Printf("\n\nTokens used: %d\n", event.Usage.TotalTokens)
    case agentspec.EventError:
        fmt.Printf("\nError: %s\n", event.Message)
    }
}

if err := stream.Err(); err != nil {
    log.Fatal(err)
}
```

### Callback-Based Streaming

```go
err := client.Agents.StreamWith(ctx, &agentspec.StreamRequest{
    Name:  "assistant",
    Input: "Tell me a story.",
}, agentspec.StreamHandlers{
    OnToken: func(content string) {
        fmt.Print(content)
    },
    OnToolCall: func(name string, args map[string]any) {
        fmt.Printf("\n[Tool: %s]\n", name)
    },
    OnDone: func(usage agentspec.Usage) {
        fmt.Printf("\nTokens: %d\n", usage.TotalTokens)
    },
    OnError: func(code, message string) {
        fmt.Printf("Error: %s\n", message)
    },
})
```

### Collecting Stream into a String

```go
stream, err := client.Agents.Stream(ctx, &agentspec.StreamRequest{
    Name:  "assistant",
    Input: "Hello",
})
if err != nil {
    log.Fatal(err)
}

var buf strings.Builder
for event := range stream.Events() {
    if event.Type == agentspec.EventToken {
        buf.WriteString(event.Content)
    }
}

fmt.Println(buf.String())
```

---

## Session Management

Sessions enable multi-turn conversations where the agent remembers previous messages.

### Creating and Using a Session

```go
ctx := context.Background()

// Create a session
session, err := client.Sessions.Create(ctx, &agentspec.CreateSessionRequest{
    Agent: "assistant",
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Session ID: %s\n", session.SessionID)

// First turn
resp1, err := client.Sessions.Continue(ctx, &agentspec.ContinueSessionRequest{
    Agent:     "assistant",
    SessionID: session.SessionID,
    Input:     "My name is Alice and I work on robotics.",
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp1.Output)

// Second turn (agent remembers context)
resp2, err := client.Sessions.Continue(ctx, &agentspec.ContinueSessionRequest{
    Agent:     "assistant",
    SessionID: session.SessionID,
    Input:     "What do I work on?",
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp2.Output)
// "You work on robotics."
```

### Session with Metadata

```go
session, err := client.Sessions.Create(ctx, &agentspec.CreateSessionRequest{
    Agent: "assistant",
    Metadata: map[string]string{
        "user_id": "user-42",
        "channel": "web",
    },
})
```

### Listing Sessions

```go
sessions, err := client.Sessions.List(ctx, &agentspec.ListSessionsRequest{
    Agent: "assistant",
    Limit: 10,
})
if err != nil {
    log.Fatal(err)
}

for _, s := range sessions {
    fmt.Printf("%s - %d turns - last active: %s\n",
        s.SessionID, s.Turns, s.LastActiveAt)
}
```

---

## Running Pipelines

```go
result, err := client.Pipelines.Run(ctx, &agentspec.PipelineRunRequest{
    Name:  "research-and-summarize",
    Input: "Summarize recent advances in battery technology.",
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Pipeline status: %s\n", result.Status)
fmt.Printf("Final output: %s\n", result.Output)

for _, step := range result.Steps {
    fmt.Printf("  Step '%s': %s (%dms)\n", step.Name, step.Status, step.DurationMs)
}
```

### Checking Pipeline Status

```go
status, err := client.Pipelines.Status(ctx, &agentspec.PipelineStatusRequest{
    Name: "research-and-summarize",
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Status: %s, Elapsed: %dms\n", status.Status, status.ElapsedMs)
```

---

## Error Handling

The SDK returns typed errors that can be inspected using `errors.As`.

```go
import (
    "errors"

    agentspec "github.com/szaher/designs/agentz/sdk"
)

response, err := client.Agents.Invoke(ctx, &agentspec.InvokeRequest{
    Name:  "assistant",
    Input: "Hello",
})
if err != nil {
    var apiErr *agentspec.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.StatusCode {
        case 401:
            fmt.Printf("Authentication failed: %s\n", apiErr.Message)
        case 404:
            fmt.Printf("Agent not found: %s\n", apiErr.Message)
        case 400:
            fmt.Printf("Bad request: %s\n", apiErr.Message)
        case 500:
            fmt.Printf("Server error: %s\n", apiErr.Message)
        default:
            fmt.Printf("API error [%s]: %s\n", apiErr.Code, apiErr.Message)
        }
        return
    }
    // Non-API error (network, timeout, etc.)
    log.Fatal(err)
}
```

### Error Type

```go
type APIError struct {
    StatusCode int    // HTTP status code (400, 401, 404, 500)
    Code       string // Machine-readable error code ("not_found", "unauthorized")
    Message    string // Human-readable error description
}

func (e *APIError) Error() string {
    return fmt.Sprintf("agentspec: %s (%d): %s", e.Code, e.StatusCode, e.Message)
}
```

### Retry Logic

The client automatically retries on transient errors (HTTP 429, 502, 503, 504) with exponential backoff. Configure retry behavior during initialization:

```go
client, err := agentspec.NewClient(
    agentspec.WithAPIKey("your-api-key"),
    agentspec.WithMaxRetries(5),
    agentspec.WithRetryDelay(500 * time.Millisecond), // Doubles on each retry
)
```

---

## Go Types

The SDK exports all request and response types as Go structs.

```go
package agentspec

// Agent represents a deployed agent.
type Agent struct {
    Name   string   `json:"name"`
    Model  string   `json:"model"`
    Skills []string `json:"skills"`
    Status string   `json:"status"`
}

// InvokeRequest is the request body for agent invocation.
type InvokeRequest struct {
    Name    string         `json:"-"`
    Input   string         `json:"input"`
    Options *InvokeOptions `json:"options,omitempty"`
}

// InvokeResponse is the response from agent invocation.
type InvokeResponse struct {
    Output string `json:"output"`
    Usage  Usage  `json:"usage"`
}

// Usage contains token usage statistics.
type Usage struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
    TotalTokens  int `json:"total_tokens"`
}

// StreamEvent represents a single event in a streaming response.
type StreamEvent struct {
    Type    EventType `json:"type"`
    Content string    `json:"content,omitempty"`
    Name    string    `json:"name,omitempty"`
    Output  string    `json:"output,omitempty"`
    Usage   *Usage    `json:"usage,omitempty"`
    Code    string    `json:"code,omitempty"`
    Message string    `json:"message,omitempty"`
}

// EventType identifies the kind of stream event.
type EventType string

const (
    EventToken      EventType = "token"
    EventToolCall   EventType = "tool_call"
    EventToolResult EventType = "tool_result"
    EventDone       EventType = "done"
    EventError      EventType = "error"
)

// Session represents a multi-turn conversation session.
type Session struct {
    SessionID    string            `json:"session_id"`
    Agent        string            `json:"agent"`
    Turns        int               `json:"turns"`
    CreatedAt    time.Time         `json:"created_at"`
    LastActiveAt time.Time         `json:"last_active_at"`
    Metadata     map[string]string `json:"metadata"`
}

// PipelineRunResponse is the response from running a pipeline.
type PipelineRunResponse struct {
    Pipeline       string         `json:"pipeline"`
    Status         string         `json:"status"`
    Steps          []PipelineStep `json:"steps"`
    Output         string         `json:"output"`
    TotalDurationMs int           `json:"total_duration_ms"`
    Usage          Usage          `json:"usage"`
}

// PipelineStep contains the result of a single pipeline step.
type PipelineStep struct {
    Name       string `json:"name"`
    Status     string `json:"status"`
    Output     string `json:"output,omitempty"`
    DurationMs int    `json:"duration_ms"`
    Usage      *Usage `json:"usage,omitempty"`
}
```

---

## Complete Example

A full example that creates a session, has a multi-turn conversation with streaming, and handles errors.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "os"
    "strings"

    agentspec "github.com/szaher/designs/agentz/sdk"
)

func main() {
    ctx := context.Background()

    client, err := agentspec.NewClient(
        agentspec.WithBaseURL("http://localhost:8080"),
        agentspec.WithAPIKey("your-api-key"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // List available agents
    agents, err := client.Agents.List(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Available agents:")
    for _, agent := range agents {
        fmt.Printf("  - %s (%s)\n", agent.Name, agent.Model)
    }

    if len(agents) == 0 {
        fmt.Println("No agents deployed.")
        os.Exit(1)
    }

    agentName := agents[0].Name

    // Create a session
    session, err := client.Sessions.Create(ctx, &agentspec.CreateSessionRequest{
        Agent: agentName,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("\nSession created: %s\n", session.SessionID)

    // Conversation loop
    messages := []string{
        "Hi, I'm building a web scraper in Go.",
        "What library should I use for parsing HTML?",
        "Can you show me a quick example?",
    }

    for _, msg := range messages {
        fmt.Printf("\nYou: %s\n", msg)
        fmt.Print("Agent: ")

        stream, err := client.Sessions.ContinueStream(ctx, &agentspec.ContinueSessionStreamRequest{
            Agent:     agentName,
            SessionID: session.SessionID,
            Input:     msg,
        })
        if err != nil {
            var apiErr *agentspec.APIError
            if errors.As(err, &apiErr) {
                fmt.Printf("\nError: %s\n", apiErr.Message)
                os.Exit(1)
            }
            log.Fatal(err)
        }

        var buf strings.Builder
        for event := range stream.Events() {
            switch event.Type {
            case agentspec.EventToken:
                fmt.Print(event.Content)
                buf.WriteString(event.Content)
            case agentspec.EventDone:
                fmt.Printf("\n  [Tokens: %d]\n", event.Usage.TotalTokens)
            }
        }

        if err := stream.Err(); err != nil {
            log.Fatal(err)
        }
    }
}
```

---

## What's Next

- [HTTP API Overview](../api/index.md) -- Full API reference
- [Python SDK](python.md) -- Python client library
- [TypeScript SDK](typescript.md) -- TypeScript/JavaScript client library
- [Agent Endpoints](../api/agents.md) -- Underlying API endpoints
