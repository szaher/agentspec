# Health & Metrics

The AgentSpec runtime provides health check and metrics endpoints for monitoring deployed agents in production. These endpoints are available outside the `/v1/` prefix (health check) and within it (metrics).

---

## Health Check

The health check endpoint reports whether the runtime is operational and ready to accept requests.

**Request**

```
GET /healthz
```

!!! note "No Version Prefix"
    The health check endpoint is served at `/healthz` (not `/v1/healthz`). This follows Kubernetes conventions and makes it easy to configure liveness and readiness probes without versioned paths.

**Response** `200 OK`

```json
{
  "status": "ok",
  "version": "0.5.0",
  "uptime_seconds": 3621
}
```

| Field | Type | Description |
|-------|------|-------------|
| `status` | `string` | `"ok"` when the runtime is healthy. |
| `version` | `string` | The AgentSpec runtime version. |
| `uptime_seconds` | `integer` | Seconds since the runtime started. |

**Unhealthy Response** `503 Service Unavailable`

```json
{
  "status": "degraded",
  "version": "0.5.0",
  "uptime_seconds": 3621,
  "checks": {
    "session_store": "unavailable",
    "plugin_loader": "ok"
  }
}
```

When the runtime is degraded, the `checks` object indicates which subsystems are unhealthy.

**Example**

```bash
curl http://localhost:8080/healthz
```

**Example -- scripted health check**

```bash
STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/healthz)
if [ "$STATUS" -ne 200 ]; then
  echo "Health check failed with status $STATUS"
  exit 1
fi
echo "Runtime is healthy"
```

---

## Prometheus Metrics

The metrics endpoint exposes runtime telemetry in Prometheus exposition format. Use it to monitor agent performance, request rates, token usage, and error rates.

**Request**

```
GET /v1/metrics
```

**Response** `200 OK` (`text/plain; charset=utf-8`)

```
# HELP agentspec_requests_total Total number of API requests.
# TYPE agentspec_requests_total counter
agentspec_requests_total{agent="assistant",endpoint="invoke",status="200"} 1542
agentspec_requests_total{agent="assistant",endpoint="invoke",status="500"} 3
agentspec_requests_total{agent="assistant",endpoint="stream",status="200"} 876

# HELP agentspec_request_duration_seconds Request duration in seconds.
# TYPE agentspec_request_duration_seconds histogram
agentspec_request_duration_seconds_bucket{agent="assistant",endpoint="invoke",le="0.5"} 120
agentspec_request_duration_seconds_bucket{agent="assistant",endpoint="invoke",le="1.0"} 890
agentspec_request_duration_seconds_bucket{agent="assistant",endpoint="invoke",le="5.0"} 1530
agentspec_request_duration_seconds_bucket{agent="assistant",endpoint="invoke",le="+Inf"} 1542

# HELP agentspec_tokens_total Total tokens consumed.
# TYPE agentspec_tokens_total counter
agentspec_tokens_total{agent="assistant",type="input"} 245000
agentspec_tokens_total{agent="assistant",type="output"} 189000

# HELP agentspec_sessions_active Number of active sessions.
# TYPE agentspec_sessions_active gauge
agentspec_sessions_active{agent="assistant"} 12

# HELP agentspec_pipeline_steps_total Total pipeline steps executed.
# TYPE agentspec_pipeline_steps_total counter
agentspec_pipeline_steps_total{pipeline="research-and-summarize",step="researcher",status="completed"} 340
agentspec_pipeline_steps_total{pipeline="research-and-summarize",step="summarizer",status="completed"} 338
agentspec_pipeline_steps_total{pipeline="research-and-summarize",step="summarizer",status="failed"} 2
```

**Available Metrics**

| Metric | Type | Description |
|--------|------|-------------|
| `agentspec_requests_total` | Counter | Total API requests by agent, endpoint, and HTTP status. |
| `agentspec_request_duration_seconds` | Histogram | Request latency distribution in seconds. |
| `agentspec_tokens_total` | Counter | Total tokens consumed by agent and direction (input/output). |
| `agentspec_sessions_active` | Gauge | Number of currently active sessions per agent. |
| `agentspec_pipeline_steps_total` | Counter | Pipeline step executions by pipeline, step, and status. |
| `agentspec_errors_total` | Counter | Total errors by agent and error code. |

**Example**

```bash
curl http://localhost:8080/v1/metrics
```

---

## Structured Log Metrics

The memory and performance subsystems emit structured log entries that provide operational visibility without requiring a metrics pipeline. These entries are written as JSON-structured log lines via the runtime's `slog` logger and can be ingested by any log aggregation tool (e.g., Loki, Elasticsearch, CloudWatch Logs).

| Component | Log Message | Fields | Description |
|-----------|------------|--------|-------------|
| Rate limiter | `rate limiter eviction` | `evicted`, `remaining` | Emitted each eviction cycle. Shows how many stale client buckets were removed and how many remain. |
| Session store | `session cleanup` | `evicted`, `active` | Emitted each background cleanup cycle. Shows expired sessions removed and active sessions remaining. |
| Conversation memory | `memory session eviction` | `evicted`, `remaining`, `max` | Emitted when LRU eviction occurs in sliding window or summary memory stores. |
| State cache | `state cache` | `hits`, `misses` | Emitted periodically (every 100th `Get()` call) with cumulative cache hit/miss counts. |
| Redis session list | `redis session list` | `count` | Emitted after listing sessions via Redis SCAN. |

**Example log output**

```json
{"time":"2026-03-04T12:00:01Z","level":"INFO","msg":"rate limiter eviction","evicted":3,"remaining":42}
{"time":"2026-03-04T12:00:01Z","level":"INFO","msg":"session cleanup","evicted":7,"active":128}
{"time":"2026-03-04T12:00:02Z","level":"INFO","msg":"memory session eviction","evicted":2,"remaining":98,"max":100}
{"time":"2026-03-04T12:00:05Z","level":"INFO","msg":"state cache","hits":4837,"misses":163}
{"time":"2026-03-04T12:00:06Z","level":"INFO","msg":"redis session list","count":54}
```

### Correlation IDs

Every HTTP request processed by the AgentSpec runtime receives a ULID correlation ID -- a time-sortable, 128-bit unique identifier. This enables end-to-end request tracing across logs, metrics, and downstream services.

- If the client sends an `X-Correlation-ID` request header, that value is used as the correlation ID for the request.
- If no `X-Correlation-ID` header is present, the runtime generates a new ULID automatically.
- The correlation ID is always returned in the `X-Correlation-ID` response header, regardless of whether it was client-supplied or server-generated.
- All log entries emitted during the lifetime of that request include a `correlation_id` field, making it straightforward to filter and trace a single request across the entire log stream.

**Example log entry with correlation ID**

```json
{
  "time": "2026-03-04T12:00:01Z",
  "level": "INFO",
  "msg": "rate limiter eviction",
  "correlation_id": "01JQXG5K8R3MZPN0VWTYBC4DEF",
  "evicted": 3,
  "remaining": 42
}
```

!!! tip "Filtering by Correlation ID"
    To trace a single request through your logs, filter on the `correlation_id` field:

    ```bash
    # Using jq
    cat logs.json | jq 'select(.correlation_id == "01JQXG5K8R3MZPN0VWTYBC4DEF")'

    # Using grep (quick filter)
    grep '01JQXG5K8R3MZPN0VWTYBC4DEF' logs.json
    ```

---

## Prometheus Integration

Add the AgentSpec runtime as a scrape target in your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: "agentspec"
    scrape_interval: 15s
    metrics_path: "/v1/metrics"
    static_configs:
      - targets: ["localhost:8080"]
        labels:
          environment: "production"
```

For Kubernetes deployments with Prometheus Operator, add annotations to the service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: agentspec-assistant
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/v1/metrics"
spec:
  ports:
    - port: 8080
      targetPort: 8080
  selector:
    app: agentspec-assistant
```

---

## Grafana Dashboard

Use the Prometheus metrics to build Grafana dashboards. Here are useful PromQL queries to get started:

**Request rate (requests per second)**

```
rate(agentspec_requests_total[5m])
```

**Average response time**

```
rate(agentspec_request_duration_seconds_sum[5m]) / rate(agentspec_request_duration_seconds_count[5m])
```

**Error rate (percentage)**

```
sum(rate(agentspec_requests_total{status=~"5.."}[5m])) / sum(rate(agentspec_requests_total[5m])) * 100
```

**Token consumption rate**

```
rate(agentspec_tokens_total[5m])
```

**p95 latency**

```
histogram_quantile(0.95, rate(agentspec_request_duration_seconds_bucket[5m]))
```

---

## Docker Compose Health Check

When deploying with Docker Compose, configure the health check to use the `/healthz` endpoint:

```yaml
services:
  agentspec:
    image: agentspec/assistant:0.5.0
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
```

---

## Kubernetes Probes

Configure liveness and readiness probes in your Kubernetes deployment:

```yaml
containers:
  - name: agentspec
    image: agentspec/assistant:0.5.0
    ports:
      - containerPort: 8080
    livenessProbe:
      httpGet:
        path: /healthz
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 30
      timeoutSeconds: 5
    readinessProbe:
      httpGet:
        path: /healthz
        port: 8080
      initialDelaySeconds: 3
      periodSeconds: 10
      timeoutSeconds: 3
```

---

## What's Next

- [Agent Endpoints](agents.md) -- Invoke and list agents
- [Session Endpoints](sessions.md) -- Multi-turn conversation management
- [Pipeline Endpoints](pipelines.md) -- Run multi-step pipelines
- [Deployment Best Practices](../deployment/best-practices.md) -- Production monitoring guidance
