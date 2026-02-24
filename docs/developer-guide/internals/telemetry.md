# Telemetry and Events

The telemetry subsystem provides structured lifecycle events, logging, and observability for the AgentSpec toolchain. It covers both the build-time pipeline (validate, plan, apply) and the agent runtime.

## Packages

| Package | Path | Purpose |
|---------|------|---------|
| `events` | `internal/events/` | Structured lifecycle event types and emitters |
| `telemetry` | `internal/telemetry/` | Telemetry collection and export |

## Event Types

The `events` package defines structured event types for lifecycle operations:

```go
type Type string

const (
    PlanStarted    Type = "plan.started"
    ApplyStarted   Type = "apply.started"
    ApplyResource  Type = "apply.resource"
    ApplyCompleted Type = "apply.completed"
    RunStarted     Type = "run.started"
    RunProgress    Type = "run.progress"
    RunCompleted   Type = "run.completed"
    RunFailed      Type = "run.failed"
)
```

### Build-Time Events

| Event | When Emitted | Data Fields |
|-------|-------------|-------------|
| `plan.started` | Plan computation begins | Resource count, target adapter |
| `apply.started` | Apply begins | Action count, adapter name |
| `apply.resource` | Each resource is applied | FQN, action type, result status |
| `apply.completed` | All actions finish | Success count, failure count, duration |

### Runtime Events

| Event | When Emitted | Data Fields |
|-------|-------------|-------------|
| `run.started` | Agent invocation begins | Agent name, session ID, input preview |
| `run.progress` | During agentic loop | Turn number, tool calls, token usage |
| `run.completed` | Invocation succeeds | Output preview, total turns, total tokens |
| `run.failed` | Invocation fails | Error message, turns completed |

## Event Structure

Every event carries a type, timestamp, correlation ID, and optional data:

```go
type Event struct {
    Type          Type                   `json:"type"`
    Timestamp     time.Time              `json:"timestamp"`
    CorrelationID string                 `json:"correlation_id"`
    Data          map[string]interface{} `json:"data,omitempty"`
}
```

### Creating Events

```go
event := events.New(events.ApplyStarted, correlationID).
    WithData("adapter", "docker").
    WithData("action_count", len(actions))
```

The `WithData()` method chains for fluent event construction.

### Serialization

Events serialize to JSON for transmission or logging:

```go
jsonBytes, err := event.JSON()
```

Example output:

```json
{
  "type": "apply.resource",
  "timestamp": "2026-02-24T10:30:15.123Z",
  "correlation_id": "abc-123",
  "data": {
    "fqn": "my-app/agent/assistant",
    "action": "create",
    "status": "success"
  }
}
```

## Emitter Interface

Events are dispatched through the `Emitter` interface:

```go
type Emitter interface {
    Emit(event *Event)
}
```

### Built-in Emitters

**NoopEmitter** -- Discards all events. Used when telemetry is disabled:

```go
type NoopEmitter struct{}

func (NoopEmitter) Emit(*Event) {}
```

**CollectorEmitter** -- Stores events in memory. Used in tests to assert event sequences:

```go
type CollectorEmitter struct {
    Events []*Event
}

func (c *CollectorEmitter) Emit(event *Event) {
    c.Events = append(c.Events, event)
}
```

## Logging

The runtime uses Go's `log/slog` package for structured logging:

```go
type Runtime struct {
    // ...
    logger *slog.Logger
}
```

Log levels:

| Level | Usage |
|-------|-------|
| `Info` | Normal operations: starting servers, registering tools |
| `Warn` | Non-fatal issues: MCP discovery failure, secret resolution failure |
| `Error` | Failures that affect functionality |
| `Debug` | Detailed internal state (token counts, message histories) |

### Structured Fields

All log entries use structured key-value pairs:

```go
rt.logger.Info("starting MCP server", "name", srv.Name, "command", srv.Command)
rt.logger.Warn("secret resolution failed", "key", k, "error", err)
rt.logger.Info("discovered MCP tool", "server", t.ServerName, "tool", t.Name)
```

## Metrics

The telemetry package tracks operational metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `agentspec.plan.duration_ms` | Histogram | Time to compute a plan |
| `agentspec.apply.duration_ms` | Histogram | Time to apply all actions |
| `agentspec.apply.actions` | Counter | Actions by type (create/update/delete) |
| `agentspec.runtime.invocations` | Counter | Agent invocations by agent name |
| `agentspec.runtime.turns` | Histogram | Loop turns per invocation |
| `agentspec.runtime.tokens` | Counter | Token usage (input/output) |
| `agentspec.runtime.tool_calls` | Counter | Tool calls by tool name |
| `agentspec.runtime.errors` | Counter | Runtime errors by type |

## Tracing

For distributed tracing, the telemetry package supports OpenTelemetry integration:

### Correlation IDs

Every operation receives a correlation ID that is propagated through all events and log entries. This allows tracing a single `apply` or `invoke` operation across all subsystems.

### Span Structure

```text
agentspec.apply
  |-- plan.compute
  |-- adapter.validate
  |-- adapter.apply
       |-- resource.create (fqn: my-app/agent/assistant)
       |-- resource.create (fqn: my-app/prompt/system)

agentspec.invoke
  |-- session.load
  |-- loop.react
       |-- llm.chat (turn 1)
       |-- tools.execute (search)
       |-- llm.chat (turn 2)
  |-- session.save
```

### Configuration

OpenTelemetry export can be configured via environment variables:

| Variable | Default | Purpose |
|----------|---------|---------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | (disabled) | OTLP collector endpoint |
| `OTEL_SERVICE_NAME` | `agentspec` | Service name in traces |
| `OTEL_RESOURCE_ATTRIBUTES` | (none) | Additional resource attributes |

## Implementing a Custom Emitter

To send events to a custom destination (e.g., a webhook or message queue):

```go
type WebhookEmitter struct {
    url    string
    client *http.Client
}

func (w *WebhookEmitter) Emit(event *events.Event) {
    data, err := event.JSON()
    if err != nil {
        return
    }
    resp, err := w.client.Post(w.url, "application/json", bytes.NewReader(data))
    if err != nil {
        return
    }
    resp.Body.Close()
}
```

Register the emitter with the runtime or apply engine to receive events.

## Testing with Events

The `CollectorEmitter` enables event-driven testing:

```go
func TestApplyEmitsEvents(t *testing.T) {
    collector := &events.CollectorEmitter{}

    // Run apply with the collector emitter
    // ...

    // Assert events
    if len(collector.Events) < 2 {
        t.Fatalf("expected at least 2 events, got %d", len(collector.Events))
    }

    if collector.Events[0].Type != events.ApplyStarted {
        t.Errorf("first event should be ApplyStarted, got %s", collector.Events[0].Type)
    }

    last := collector.Events[len(collector.Events)-1]
    if last.Type != events.ApplyCompleted {
        t.Errorf("last event should be ApplyCompleted, got %s", last.Type)
    }
}
```
