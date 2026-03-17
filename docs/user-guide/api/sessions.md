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

### Redis Store Performance

The Redis session store uses optimized data structures and access patterns for production workloads:

- **Cursor-based listing.** Session listing uses the Redis `SCAN` command with a cursor instead of the blocking `KEYS` command. This avoids locking the Redis server when the session count is large and allows the runtime to page through results incrementally.
- **O(1) message append.** New messages are appended to a session's history using the Redis `RPUSH` (List push) operation. This is an O(1) operation that avoids reloading and re-serializing the entire message array on every turn.
- **Transparent migration.** Sessions that were stored under the legacy format (a single JSON string containing the full message array) are automatically migrated to the new Redis List format on first access. No manual migration step is required; the runtime detects the legacy format, converts it, and deletes the old key atomically.

!!! note "Backwards Compatibility"
    Existing Redis sessions created before the List-based storage change will continue to work. The runtime transparently migrates them on first read, so deployments can upgrade without downtime or data loss.

---

## Session Expiry and Eviction

In-memory sessions have a configurable idle timeout. If a session is not accessed (created, continued, or listed individually) within the timeout window, it becomes eligible for eviction.

| Setting | Default | Description |
|---------|---------|-------------|
| Idle timeout | 30 minutes | Duration of inactivity after which a session is considered expired. |
| Sweep interval | 5 minutes | How often the background goroutine scans for and removes expired sessions. |

A background goroutine runs on the configured sweep interval, iterating over all tracked sessions and removing those whose last-access time exceeds the idle timeout. This keeps steady-state memory usage proportional to the number of *active* sessions rather than the total number of sessions ever created.

When listing sessions, expired sessions are also **lazily evicted**: any session whose idle timeout has elapsed is removed from the store and excluded from the response before results are returned to the caller.

!!! tip "Rate Limiting"
    The `AGENTSPEC_RATE_LIMIT` environment variable configures request rate limiting for session endpoints. The format is `"rate:burst"` -- for example, `"10:20"` allows 10 requests per second with a burst capacity of 20. When the rate limit is exceeded, the runtime responds with `429 Too Many Requests`.

---

## Conversation Memory Limits

Each conversation memory store -- the sliding-window store and the summary store -- enforces a maximum number of concurrent sessions. This prevents unbounded memory growth when many users interact with the same agent.

| Setting | Default | Description |
|---------|---------|-------------|
| Max concurrent sessions | 10,000 | Maximum number of sessions tracked per memory store instance. |
| Sliding-window size | 50 messages | Maximum number of messages retained per session. |

When the concurrent session limit is exceeded, the **least-recently-used (LRU)** session is evicted to make room for the new one. The evicted session's entire message history is discarded.

Within a single session, the sliding-window memory store retains the most recent messages up to the configured window size. When a new message is appended and the window is full, the oldest message is dropped. This keeps per-session memory usage bounded while preserving the most relevant recent context for the agent.

!!! warning "LRU Eviction"
    LRU eviction is silent -- no error is returned to the caller whose session was evicted. If a subsequent request references an evicted session, the runtime returns a `404 not_found` error. Long-lived integrations should handle this case by creating a new session.

---

## What's Next

- [Agent Endpoints](agents.md) -- Stateless agent invocation
- [Pipeline Endpoints](pipelines.md) -- Run multi-step pipelines
- [Python SDK](../sdks/python.md) -- Session management in Python
- [Go SDK](../sdks/go.md) -- Session management in Go
