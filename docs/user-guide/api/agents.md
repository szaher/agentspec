# Agent Endpoints

Agent endpoints let you list deployed agents, invoke them for a complete response, or stream responses in real time using Server-Sent Events.

---

## List Agents

Retrieve a list of all agents deployed in the current runtime.

**Request**

```
GET /v1/agents
```

**Response** `200 OK`

```json
{
  "agents": [
    {
      "name": "assistant",
      "model": "claude-sonnet-4-20250514",
      "skills": ["search", "calculator"],
      "status": "ready"
    },
    {
      "name": "reviewer",
      "model": "claude-sonnet-4-20250514",
      "skills": ["code-review"],
      "status": "ready"
    }
  ]
}
```

**Example**

```bash
curl -H "X-API-Key: your-api-key" \
  http://localhost:8080/v1/agents
```

---

## Invoke Agent

Send an input to an agent and receive the complete response once processing finishes.

**Request**

```
POST /v1/agents/{name}/invoke
```

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `name` | Path | Yes | The name of the agent to invoke. |
| `input` | Body | Yes | The user message or query string. |
| `options` | Body | No | Optional invocation parameters. |

**Request Body**

```json
{
  "input": "What is the capital of France?",
  "options": {
    "temperature": 0.7,
    "max_tokens": 1024
  }
}
```

**Response** `200 OK`

```json
{
  "output": "The capital of France is Paris.",
  "usage": {
    "input_tokens": 12,
    "output_tokens": 8,
    "total_tokens": 20
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `output` | `string` | The agent's complete response. |
| `usage.input_tokens` | `integer` | Number of tokens in the input. |
| `usage.output_tokens` | `integer` | Number of tokens in the output. |
| `usage.total_tokens` | `integer` | Total tokens consumed. |

**Example**

```bash
curl -X POST http://localhost:8080/v1/agents/assistant/invoke \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"input": "What is the capital of France?"}'
```

**Error Responses**

| Status | Code | Description |
|--------|------|-------------|
| `400` | `invalid_request` | Missing or invalid `input` field. |
| `401` | `unauthorized` | Invalid or missing authentication. |
| `404` | `not_found` | Agent with the given name does not exist. |
| `500` | `internal_error` | Runtime or provider error during invocation. |

---

## Stream Agent Response

Send an input to an agent and receive the response as a stream of Server-Sent Events (SSE). This is useful for displaying partial results in real time.

**Request**

```
POST /v1/agents/{name}/stream
```

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `name` | Path | Yes | The name of the agent to invoke. |
| `input` | Body | Yes | The user message or query string. |
| `options` | Body | No | Optional invocation parameters. |

**Request Body**

```json
{
  "input": "Explain how neural networks work."
}
```

**Response** `200 OK` (`text/event-stream`)

The response uses the SSE protocol. Each event contains a JSON `data` payload:

```
data: {"type": "token", "content": "Neural"}

data: {"type": "token", "content": " networks"}

data: {"type": "token", "content": " are"}

data: {"type": "done", "usage": {"input_tokens": 8, "output_tokens": 150, "total_tokens": 158}}
```

**Event Types**

| Type | Description |
|------|-------------|
| `token` | A partial response token. The `content` field contains the text fragment. |
| `tool_call` | The agent is invoking a skill/tool. Includes `name` and `arguments` fields. |
| `tool_result` | The result of a tool invocation. Includes `name` and `output` fields. |
| `error` | An error occurred during streaming. Includes `code` and `message` fields. |
| `done` | The stream is complete. Includes final `usage` statistics. |

**Example**

```bash
curl -N -X POST http://localhost:8080/v1/agents/assistant/stream \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"input": "Explain how neural networks work."}'
```

!!! tip "The `-N` flag"
    Use `curl -N` (or `--no-buffer`) to disable output buffering so SSE events appear in real time.

---

## What's Next

- [Session Endpoints](sessions.md) -- Multi-turn conversations with state
- [Pipeline Endpoints](pipelines.md) -- Run multi-step pipelines
- [Python SDK](../sdks/python.md) -- Python client with streaming support
- [TypeScript SDK](../sdks/typescript.md) -- TypeScript client with streaming support
