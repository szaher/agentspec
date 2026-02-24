# Session Endpoints

Sessions enable multi-turn conversations with an agent. When you create a session, the runtime stores conversation history so the agent maintains context across multiple interactions. Sessions use either the default in-memory store or an optional Redis store for persistence.

---

## Create Session

Create a new session for an agent. The returned `session_id` is used in subsequent requests to continue the conversation.

**Request**

```
POST /v1/agents/{name}/sessions
```

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `name` | Path | Yes | The name of the agent. |
| `metadata` | Body | No | Arbitrary key-value pairs to attach to the session. |

**Request Body**

```json
{
  "metadata": {
    "user_id": "user-42",
    "channel": "web"
  }
}
```

An empty body `{}` is also valid if no metadata is needed.

**Response** `200 OK`

```json
{
  "session_id": "sess_a1b2c3d4e5f6",
  "agent": "assistant",
  "created_at": "2026-02-24T10:30:00Z",
  "metadata": {
    "user_id": "user-42",
    "channel": "web"
  }
}
```

**Example**

```bash
curl -X POST http://localhost:8080/v1/agents/assistant/sessions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"metadata": {"user_id": "user-42"}}'
```

---

## Continue Session

Send a new message within an existing session. The agent receives the full conversation history and responds in context.

**Request**

```
POST /v1/agents/{name}/sessions/{id}/continue
```

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `name` | Path | Yes | The name of the agent. |
| `id` | Path | Yes | The session ID returned from the create endpoint. |
| `input` | Body | Yes | The next user message. |

**Request Body**

```json
{
  "input": "What did I ask you earlier?"
}
```

**Response** `200 OK`

```json
{
  "output": "You asked me about the capital of France, which is Paris.",
  "usage": {
    "input_tokens": 45,
    "output_tokens": 14,
    "total_tokens": 59
  },
  "turn": 3
}
```

| Field | Type | Description |
|-------|------|-------------|
| `output` | `string` | The agent's response for this turn. |
| `usage` | `object` | Token usage for this turn. |
| `turn` | `integer` | The turn number within the session (1-indexed). |

**Example -- Multi-turn conversation**

```bash
# Turn 1: Create a session
SESSION=$(curl -s -X POST http://localhost:8080/v1/agents/assistant/sessions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{}' | jq -r '.session_id')

# Turn 2: First message
curl -X POST "http://localhost:8080/v1/agents/assistant/sessions/${SESSION}/continue" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"input": "My name is Alice."}'

# Turn 3: Follow-up (agent remembers context)
curl -X POST "http://localhost:8080/v1/agents/assistant/sessions/${SESSION}/continue" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"input": "What is my name?"}'
```

**Error Responses**

| Status | Code | Description |
|--------|------|-------------|
| `400` | `invalid_request` | Missing `input` field in the request body. |
| `404` | `not_found` | Agent or session ID does not exist. |
| `401` | `unauthorized` | Invalid or missing authentication. |

---

## List Sessions

Retrieve all sessions for a given agent. Results are ordered by creation time (most recent first).

**Request**

```
GET /v1/agents/{name}/sessions
```

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `name` | Path | Yes | The name of the agent. |
| `limit` | Query | No | Maximum number of sessions to return (default: 20, max: 100). |
| `offset` | Query | No | Number of sessions to skip for pagination (default: 0). |

**Response** `200 OK`

```json
{
  "sessions": [
    {
      "session_id": "sess_a1b2c3d4e5f6",
      "agent": "assistant",
      "turns": 5,
      "created_at": "2026-02-24T10:30:00Z",
      "last_active_at": "2026-02-24T10:45:00Z",
      "metadata": {
        "user_id": "user-42"
      }
    },
    {
      "session_id": "sess_x7y8z9w0v1u2",
      "agent": "assistant",
      "turns": 2,
      "created_at": "2026-02-24T09:15:00Z",
      "last_active_at": "2026-02-24T09:20:00Z",
      "metadata": {}
    }
  ],
  "total": 2
}
```

**Example**

```bash
curl -H "X-API-Key: your-api-key" \
  "http://localhost:8080/v1/agents/assistant/sessions?limit=10"
```

**Example with pagination**

```bash
curl -H "X-API-Key: your-api-key" \
  "http://localhost:8080/v1/agents/assistant/sessions?limit=10&offset=10"
```

---

## Session Storage

By default, sessions are stored in memory and are lost when the runtime restarts. For persistent sessions, configure a Redis session store in your deploy block.

!!! warning "In-Memory Sessions"
    In-memory sessions are suitable for development and testing. For production deployments, use Redis or another persistent session store to ensure sessions survive restarts and can be shared across replicas.

---

## What's Next

- [Agent Endpoints](agents.md) -- Stateless agent invocation
- [Pipeline Endpoints](pipelines.md) -- Run multi-step pipelines
- [Python SDK](../sdks/python.md) -- Session management in Python
- [Go SDK](../sdks/go.md) -- Session management in Go
