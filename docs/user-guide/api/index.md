# HTTP API Overview

The AgentSpec runtime exposes a RESTful HTTP API for invoking agents, managing sessions, running pipelines, and monitoring health. Every deployed agent automatically serves this API on the port configured in its `deploy` block.

---

## Base URL

All API endpoints are served under the `/v1/` prefix:

```
http://localhost:8080/v1/
```

The host and port are determined by the `port` attribute in your `deploy` block. The `/v1/` prefix ensures forward compatibility -- future breaking changes will use `/v2/`, while non-breaking additions will remain under `/v1/`.

---

## Authentication

The API supports two authentication methods. Include one of them with every request.

### API Key

Pass the key in the `X-API-Key` header:

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/v1/agents
```

### Bearer Token

Pass a token in the standard `Authorization` header:

```bash
curl -H "Authorization: Bearer your-token" http://localhost:8080/v1/agents
```

!!! tip "Development Mode"
    When running locally with `agentspec apply` and no authentication is configured, the API accepts unauthenticated requests. Always configure authentication before deploying to production.

---

## Error Response Format

All errors follow a consistent JSON structure:

```json
{
  "error": {
    "code": "not_found",
    "message": "Agent 'unknown-agent' does not exist."
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `error.code` | `string` | A machine-readable error code (e.g. `invalid_request`, `not_found`, `unauthorized`). |
| `error.message` | `string` | A human-readable description of what went wrong. |

---

## HTTP Status Codes

The API uses standard HTTP status codes:

| Status | Meaning | When It Occurs |
|--------|---------|----------------|
| `200 OK` | Request succeeded. | Successful GET, POST, or streaming request. |
| `400 Bad Request` | The request body is malformed or missing required fields. | Invalid JSON, missing `input` field. |
| `401 Unauthorized` | Authentication is missing or invalid. | Missing or expired API key or token. |
| `404 Not Found` | The requested resource does not exist. | Unknown agent name, session ID, or pipeline. |
| `500 Internal Server Error` | An unexpected error occurred on the server. | Runtime failures, plugin crashes, provider errors. |

---

## Content Type

All request and response bodies use JSON. Set the `Content-Type` header on requests that include a body:

```bash
curl -X POST http://localhost:8080/v1/agents/assistant/invoke \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"input": "Hello"}'
```

Streaming endpoints (`/stream`) return `text/event-stream` using the Server-Sent Events (SSE) protocol.

---

## Endpoint Index

| Method | Endpoint | Description | Reference |
|--------|----------|-------------|-----------|
| `GET` | `/v1/agents` | List all deployed agents. | [Agent Endpoints](agents.md) |
| `POST` | `/v1/agents/{name}/invoke` | Invoke an agent and receive the full response. | [Agent Endpoints](agents.md) |
| `POST` | `/v1/agents/{name}/stream` | Invoke an agent and stream the response via SSE. | [Agent Endpoints](agents.md) |
| `POST` | `/v1/agents/{name}/sessions` | Create a new session for multi-turn conversations. | [Session Endpoints](sessions.md) |
| `POST` | `/v1/agents/{name}/sessions/{id}/continue` | Continue an existing session with a new message. | [Session Endpoints](sessions.md) |
| `GET` | `/v1/agents/{name}/sessions` | List sessions for an agent. | [Session Endpoints](sessions.md) |
| `POST` | `/v1/pipelines/{name}/run` | Run a pipeline with the given input. | [Pipeline Endpoints](pipelines.md) |
| `GET` | `/v1/pipelines/{name}/status` | Check the status of a pipeline run. | [Pipeline Endpoints](pipelines.md) |
| `GET` | `/healthz` | Health check (not versioned). | [Health & Metrics](health-metrics.md) |
| `GET` | `/v1/metrics` | Prometheus-format metrics. | [Health & Metrics](health-metrics.md) |

---

## What's Next

- [Agent Endpoints](agents.md) -- Invoke agents and list deployed agents
- [Session Endpoints](sessions.md) -- Manage multi-turn conversation sessions
- [Pipeline Endpoints](pipelines.md) -- Run and monitor pipelines
- [Health & Metrics](health-metrics.md) -- Health checks and Prometheus metrics
- [Python SDK](../sdks/python.md) -- Python client library
- [TypeScript SDK](../sdks/typescript.md) -- TypeScript/JavaScript client library
- [Go SDK](../sdks/go.md) -- Go client library
