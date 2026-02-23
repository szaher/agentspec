# Runtime HTTP API Contract

**Branch**: `004-runtime-platform` | **Date**: 2026-02-23

## Overview

When an agent package is deployed, the runtime exposes an HTTP API for agent invocation, session management, pipeline execution, and operational endpoints.

**Base URL**: `http://{host}:{port}` (default port: 8080)

**Authentication**: API key via `Authorization: Bearer <key>` header or `X-API-Key: <key>` header.

**Content-Type**: All request and response bodies are `application/json` unless otherwise specified.

**Error Format**: All errors return a JSON body with `error` and `message` fields:

```json
{
  "error": "invalid_request",
  "message": "Agent 'unknown-bot' not found",
  "details": {}
}
```

**Error Codes**:

| HTTP Status | Error Code | Description |
| ----------- | ---------- | ----------- |
| 400 | `invalid_request` | Malformed request body or missing required fields |
| 401 | `unauthorized` | Missing or invalid API key |
| 404 | `not_found` | Agent, session, or pipeline not found |
| 408 | `timeout` | Invocation exceeded configured timeout |
| 429 | `rate_limited` | Per-agent rate limit exceeded |
| 500 | `internal_error` | Unexpected runtime error |
| 503 | `unavailable` | Runtime is starting up or shutting down |

---

## Health & Operations

### GET /healthz

Health check endpoint.

**Response** (200):
```json
{
  "status": "healthy",
  "uptime": "2h15m30s",
  "agents": 3,
  "version": "0.3.0"
}
```

### GET /v1/agents

List all deployed agents and their status.

**Response** (200):
```json
{
  "agents": [
    {
      "name": "support-bot",
      "fqn": "my-package/Agent/support-bot",
      "model": "claude-sonnet-4-20250514",
      "strategy": "react",
      "status": "running",
      "skills": ["lookup-order", "search-kb"],
      "active_sessions": 5,
      "total_invocations": 142
    }
  ]
}
```

### GET /v1/metrics

Prometheus-format metrics endpoint.

**Response** (200, `text/plain`):
```
# HELP agentspec_invocations_total Total agent invocations
# TYPE agentspec_invocations_total counter
agentspec_invocations_total{agent="support-bot",status="completed"} 140
agentspec_invocations_total{agent="support-bot",status="failed"} 2

# HELP agentspec_invocation_duration_seconds Invocation duration
# TYPE agentspec_invocation_duration_seconds histogram
agentspec_invocation_duration_seconds_bucket{agent="support-bot",le="1"} 50

# HELP agentspec_tokens_total Tokens consumed
# TYPE agentspec_tokens_total counter
agentspec_tokens_total{agent="support-bot",type="input"} 280000
agentspec_tokens_total{agent="support-bot",type="output"} 56000

# HELP agentspec_tool_calls_total Tool call count
# TYPE agentspec_tool_calls_total counter
agentspec_tool_calls_total{agent="support-bot",tool="lookup-order",status="success"} 89
```

---

## Agent Invocation

### POST /v1/agents/{name}/invoke

Invoke an agent and wait for the complete response.

**Request**:
```json
{
  "message": "I need help with my order #12345",
  "variables": {
    "customer_name": "Alice"
  },
  "session_id": "optional-session-id"
}
```

**Response** (200):
```json
{
  "output": "I found your order #12345. It was shipped on Feb 20 and...",
  "tool_calls": [
    {
      "id": "tc_001",
      "tool_name": "lookup-order",
      "input": {"order_id": "12345"},
      "output": {"status": "shipped", "tracking": "1Z999AA10"},
      "duration_ms": 150
    }
  ],
  "tokens": {
    "input": 1200,
    "output": 350,
    "cache_read": 800,
    "total": 2350
  },
  "turns": 2,
  "duration_ms": 3200,
  "session_id": "sess_abc123"
}
```

### POST /v1/agents/{name}/stream

Invoke an agent with streaming response (Server-Sent Events).

**Request**: Same as `/invoke`.

**Response** (200, `text/event-stream`):
```
event: text
data: {"text": "I found your order"}

event: text
data: {"text": " #12345. It was shipped"}

event: tool_call_start
data: {"id": "tc_001", "tool_name": "lookup-order", "input": {"order_id": "12345"}}

event: tool_call_end
data: {"id": "tc_001", "output": {"status": "shipped"}, "duration_ms": 150}

event: text
data: {"text": " on Feb 20 and is expected to arrive..."}

event: done
data: {"tokens": {"input": 1200, "output": 350, "total": 2350}, "turns": 2, "duration_ms": 3200}
```

**SSE Event Types**:

| Event | Data Fields | Description |
| ----- | ----------- | ----------- |
| `text` | text | Incremental text chunk from the agent |
| `tool_call_start` | id, tool_name, input | Tool call initiated |
| `tool_call_end` | id, output, duration_ms, error | Tool call completed |
| `error` | error, message | Error during invocation |
| `done` | tokens, turns, duration_ms, session_id | Invocation completed |

---

## Session Management

### POST /v1/agents/{name}/sessions

Create a new conversation session.

**Request**:
```json
{
  "metadata": {
    "user_id": "user_123"
  }
}
```

**Response** (201):
```json
{
  "session_id": "sess_abc123",
  "agent": "support-bot",
  "created_at": "2026-02-23T10:30:00Z"
}
```

### POST /v1/agents/{name}/sessions/{id}

Send a message within an existing session. The agent maintains conversation context.

**Request**:
```json
{
  "message": "The order number is #456"
}
```

**Response** (200): Same format as `/invoke`.

### DELETE /v1/agents/{name}/sessions/{id}

Close a session and release memory.

**Response** (204): No content.

---

## Pipeline Execution

### POST /v1/pipelines/{name}/run

Execute a multi-agent pipeline.

**Request**:
```json
{
  "trigger": {
    "pr_url": "https://github.com/org/repo/pull/42"
  }
}
```

**Response** (200):
```json
{
  "pipeline": "code-review",
  "status": "completed",
  "steps": {
    "analyze": {
      "agent": "code-analyzer",
      "output": {"findings": ["..."]},
      "duration_ms": 5200,
      "status": "completed"
    },
    "security": {
      "agent": "security-scanner",
      "output": {"vulnerabilities": []},
      "duration_ms": 3100,
      "status": "completed"
    },
    "summarize": {
      "agent": "review-summarizer",
      "output": {"review": "No critical issues found..."},
      "duration_ms": 2800,
      "status": "completed"
    }
  },
  "total_duration_ms": 8000,
  "tokens": {
    "total": 15000
  }
}
```
