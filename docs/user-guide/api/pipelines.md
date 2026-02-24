# Pipeline Endpoints

Pipeline endpoints let you run multi-step agent pipelines and check their execution status. A pipeline chains multiple agents together, passing the output of each step as input to the next.

---

## Run Pipeline

Execute a pipeline with the given input. The response includes the output of each step and the final result.

**Request**

```
POST /v1/pipelines/{name}/run
```

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `name` | Path | Yes | The name of the pipeline to run. |
| `input` | Body | Yes | The input to the first step of the pipeline. |
| `options` | Body | No | Optional parameters for the pipeline run. |

**Request Body**

```json
{
  "input": "Summarize the latest quarterly earnings report for Acme Corp."
}
```

**Response** `200 OK`

```json
{
  "pipeline": "research-and-summarize",
  "status": "completed",
  "steps": [
    {
      "name": "researcher",
      "status": "completed",
      "output": "Acme Corp reported Q4 2025 revenue of $2.3B, up 15% year-over-year...",
      "usage": {
        "input_tokens": 25,
        "output_tokens": 340,
        "total_tokens": 365
      },
      "duration_ms": 4200
    },
    {
      "name": "summarizer",
      "status": "completed",
      "output": "Acme Corp had a strong Q4 2025 with $2.3B in revenue (+15% YoY)...",
      "usage": {
        "input_tokens": 350,
        "output_tokens": 120,
        "total_tokens": 470
      },
      "duration_ms": 1800
    }
  ],
  "output": "Acme Corp had a strong Q4 2025 with $2.3B in revenue (+15% YoY)...",
  "total_duration_ms": 6000,
  "usage": {
    "input_tokens": 375,
    "output_tokens": 460,
    "total_tokens": 835
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `pipeline` | `string` | The pipeline name. |
| `status` | `string` | Overall status: `completed`, `failed`, or `running`. |
| `steps` | `array` | Ordered list of step results. |
| `steps[].name` | `string` | The agent name for this step. |
| `steps[].status` | `string` | Step status: `completed`, `failed`, or `skipped`. |
| `steps[].output` | `string` | The output produced by this step. |
| `steps[].usage` | `object` | Token usage for this step. |
| `steps[].duration_ms` | `integer` | Execution time for this step in milliseconds. |
| `output` | `string` | The final output from the last step. |
| `total_duration_ms` | `integer` | Total pipeline execution time in milliseconds. |
| `usage` | `object` | Aggregate token usage across all steps. |

**Example**

```bash
curl -X POST http://localhost:8080/v1/pipelines/research-and-summarize/run \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"input": "Summarize the latest quarterly earnings report for Acme Corp."}'
```

**Error Responses**

| Status | Code | Description |
|--------|------|-------------|
| `400` | `invalid_request` | Missing or invalid `input` field. |
| `401` | `unauthorized` | Invalid or missing authentication. |
| `404` | `not_found` | Pipeline with the given name does not exist. |
| `500` | `pipeline_error` | A step in the pipeline failed. Check `steps` for details. |

---

## Check Pipeline Status

For long-running pipelines, check the current execution status without waiting for the full response.

**Request**

```
GET /v1/pipelines/{name}/status
```

| Parameter | Location | Required | Description |
|-----------|----------|----------|-------------|
| `name` | Path | Yes | The name of the pipeline. |
| `run_id` | Query | No | A specific run ID. If omitted, returns the status of the most recent run. |

**Response** `200 OK`

```json
{
  "pipeline": "research-and-summarize",
  "run_id": "run_abc123",
  "status": "running",
  "steps": [
    {
      "name": "researcher",
      "status": "completed",
      "duration_ms": 4200
    },
    {
      "name": "summarizer",
      "status": "running",
      "duration_ms": null
    }
  ],
  "started_at": "2026-02-24T10:30:00Z",
  "elapsed_ms": 5100
}
```

| Field | Type | Description |
|-------|------|-------------|
| `run_id` | `string` | Unique identifier for this pipeline run. |
| `status` | `string` | Current status: `running`, `completed`, or `failed`. |
| `steps` | `array` | Current status of each step. |
| `started_at` | `string` | ISO 8601 timestamp when the run started. |
| `elapsed_ms` | `integer` | Time elapsed since the run started. |

**Example**

```bash
curl -H "X-API-Key: your-api-key" \
  "http://localhost:8080/v1/pipelines/research-and-summarize/status"
```

**Example with a specific run ID**

```bash
curl -H "X-API-Key: your-api-key" \
  "http://localhost:8080/v1/pipelines/research-and-summarize/status?run_id=run_abc123"
```

---

## Pipeline Failure Handling

When a step in the pipeline fails, the response includes the error details in the failed step and the overall status is set to `failed`:

```json
{
  "pipeline": "research-and-summarize",
  "status": "failed",
  "steps": [
    {
      "name": "researcher",
      "status": "completed",
      "output": "...",
      "duration_ms": 4200
    },
    {
      "name": "summarizer",
      "status": "failed",
      "error": {
        "code": "provider_error",
        "message": "Model provider returned a rate limit error."
      },
      "duration_ms": 500
    }
  ],
  "output": null
}
```

Steps that were not reached due to an earlier failure have a status of `skipped`.

---

## What's Next

- [Agent Endpoints](agents.md) -- Invoke individual agents
- [Session Endpoints](sessions.md) -- Multi-turn conversations
- [Health & Metrics](health-metrics.md) -- Monitor pipeline performance
