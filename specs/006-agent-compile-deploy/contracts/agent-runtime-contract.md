# Agent Runtime Contract

**Feature**: 006-agent-compile-deploy

## Overview

A compiled agent binary exposes an HTTP API for agent invocation, streaming, session management, health checks, and the built-in frontend. This contract defines the endpoints that every compiled agent must serve.

## Base URL

```
http://{host}:{port}
```

Default: `http://0.0.0.0:8080`

## Authentication

All API endpoints (except `/healthz` and static frontend assets) require authentication by default.

**Header**: `Authorization: Bearer <api-key>`

**Response when unauthenticated** (401):
```json
{
  "error": "authentication_required",
  "message": "API key required. Set AGENTSPEC_API_KEY environment variable."
}
```

Authentication can be disabled via `--no-auth` flag or `AGENTSPEC_NO_AUTH=true`.

## API Endpoints

### Health Check

```
GET /healthz
```

No authentication required.

**Response** (200):
```json
{
  "status": "healthy",
  "agents": ["support-agent", "escalation-agent"],
  "uptime_seconds": 3600,
  "version": "0.3.0"
}
```

**Response** (503 — agent not ready):
```json
{
  "status": "unhealthy",
  "error": "missing configuration: AGENTSPEC_SUPPORT_AGENT_ANTHROPIC_API_KEY"
}
```

---

### Agent Invocation

```
POST /v1/agents/{name}/invoke
```

**Request**:
```json
{
  "input": "How do I reset my password?",
  "session_id": "optional-session-id",
  "metadata": {}
}
```

**Response** (200):
```json
{
  "output": "To reset your password, go to Settings > Security > Reset Password...",
  "session_id": "sess_a1b2c3d4",
  "usage": {
    "input_tokens": 45,
    "output_tokens": 120,
    "total_tokens": 165
  },
  "validation": {
    "passed": true,
    "rules_checked": 3,
    "warnings": []
  },
  "activity": [
    {
      "type": "thought",
      "content": "User is asking about password reset. I should check the FAQ skill.",
      "timestamp": "2026-02-28T12:00:00Z"
    },
    {
      "type": "tool_call",
      "content": "faq_search(query='password reset')",
      "timestamp": "2026-02-28T12:00:01Z",
      "duration_ms": 250
    }
  ]
}
```

---

### Streaming Invocation (SSE)

```
POST /v1/agents/{name}/stream
```

**Request**: Same as `/invoke`

**Response**: `Content-Type: text/event-stream`

```
event: thought
data: {"content":"Looking up password reset procedure...","timestamp":"2026-02-28T12:00:00Z"}

event: tool_call
data: {"tool":"faq_search","input":{"query":"password reset"},"timestamp":"2026-02-28T12:00:01Z"}

event: tool_result
data: {"tool":"faq_search","output":"Go to Settings > Security...","duration_ms":250}

event: token
data: {"content":"To ","timestamp":"2026-02-28T12:00:02Z"}

event: token
data: {"content":"reset ","timestamp":"2026-02-28T12:00:02Z"}

event: token
data: {"content":"your password...","timestamp":"2026-02-28T12:00:02Z"}

event: validation
data: {"passed":true,"rules_checked":3}

event: done
data: {"session_id":"sess_a1b2c3d4","usage":{"input_tokens":45,"output_tokens":120}}
```

---

### Session Management

```
GET /v1/agents/{name}/sessions
```

**Response** (200):
```json
{
  "sessions": [
    {
      "session_id": "sess_a1b2c3d4",
      "created_at": "2026-02-28T12:00:00Z",
      "last_active": "2026-02-28T12:05:00Z",
      "message_count": 5
    }
  ]
}
```

```
DELETE /v1/agents/{name}/sessions/{session_id}
```

**Response** (204): No content

---

### Agent Metadata

```
GET /v1/agents/{name}
```

**Response** (200):
```json
{
  "name": "support-agent",
  "description": "Customer support agent with FAQ search and ticket creation",
  "model": "claude-sonnet-4-20250514",
  "skills": ["faq_search", "create_ticket", "escalate"],
  "loop_strategy": "react",
  "input_schema": {
    "type": "object",
    "properties": {
      "input": {"type": "string"},
      "category": {"type": "string", "enum": ["billing", "technical", "general"]}
    },
    "required": ["input"]
  },
  "validation_rules": ["output_format", "no_pii", "tone_check"],
  "config_params": [
    {"name": "anthropic_api_key", "type": "string", "secret": true, "required": true},
    {"name": "escalation_email", "type": "string", "secret": false, "required": false, "default": "support@example.com"}
  ]
}
```

---

### List All Agents

```
GET /v1/agents
```

**Response** (200):
```json
{
  "agents": [
    {"name": "support-agent", "description": "Customer support agent"},
    {"name": "escalation-agent", "description": "Handles escalated tickets"}
  ]
}
```

---

## Frontend Endpoints

### Serve Frontend

```
GET /
GET /ui/*
```

Serves the embedded SPA (single-page application). No authentication required for static assets. The frontend auto-injects the API key into API requests via session storage.

### Frontend → API Flow

1. User opens `http://agent:8080/` → serves `index.html`
2. Frontend prompts for API key (stored in session storage)
3. Frontend calls `GET /v1/agents` to list available agents
4. User selects agent → frontend calls `GET /v1/agents/{name}` for metadata
5. User sends message → frontend calls `POST /v1/agents/{name}/stream`
6. Frontend renders streaming response with activity panel

---

## Error Responses

All errors follow this format:

```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": {}
}
```

| Status | Error Code | When |
|--------|-----------|------|
| 400 | `invalid_input` | Malformed request body |
| 401 | `authentication_required` | Missing or invalid API key |
| 404 | `agent_not_found` | Unknown agent name |
| 404 | `session_not_found` | Unknown session ID |
| 422 | `validation_failed` | Agent output failed all validation retries |
| 429 | `rate_limited` | Too many requests (FR-050) |
| 500 | `internal_error` | Unexpected server error |
| 503 | `not_ready` | Agent not fully initialized |
