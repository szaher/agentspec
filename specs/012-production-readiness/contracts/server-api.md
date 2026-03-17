# Contract: Server API Extensions

## TLS Configuration

### CLI Flags (added to `run` command)

| Flag         | Type   | Default | Description                          |
|--------------|--------|---------|--------------------------------------|
| `--tls-cert` | string | ""      | Path to TLS certificate file (PEM)   |
| `--tls-key`  | string | ""      | Path to TLS private key file (PEM)   |

**Behavior**:
- Both flags must be provided together; providing one without the other is an error
- When both are set, server listens on HTTPS only; plain HTTP requests are rejected
- When neither is set, server listens on HTTP with a warning log: `"TLS disabled, serving over HTTP"`
- Invalid/expired certificates fail startup with exit code 1 and a clear error message
- Certificate file changes are detected via fsnotify and reloaded without restart

### New Endpoint: Metrics (enhanced)

`GET /v1/metrics` — existing endpoint, enhanced with new metrics:

```
# New cost metrics
agentspec_cost_dollars_total{agent="...", model="..."} 0.0
agentspec_budget_usage_ratio{agent="...", period="daily"} 0.35

# New fallback metrics
agentspec_fallback_total{agent="...", from_model="...", to_model="..."} 0

# New guardrail metrics
agentspec_guardrail_violations_total{agent="...", guardrail="...", mode="warn|block"} 0
```

## Authentication Extensions

### Per-User API Key Resolution

When multi-user auth is configured (via `user` blocks in `.ias` file):

1. Server extracts API key from `X-API-Key` header or `Authorization: Bearer <key>`
2. Key is looked up against the user table (resolved from secret references at startup)
3. If matched: request proceeds with user identity in context
4. If not matched: 401 Unauthorized
5. If user lacks access to the requested agent: 403 Forbidden

### Audit Log Format

Each line in `agentspec-audit.log` is a JSON object:

```json
{"timestamp":"2026-03-17T10:00:00Z","user":"alice","agent":"support-agent","session":"ses_01J...","action":"invoke","tokens_in":150,"tokens_out":300,"duration":"1.234s","status":"success","correlation_id":"01JXYZ..."}
```

## Budget Enforcement API

### Budget-Exceeded Response

When an agent's budget is exceeded, invocation endpoints return:

```
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 3600

{"error":"budget_exceeded","message":"Agent 'support-agent' daily budget of $10.00 exceeded (used: $10.50). Resets at 2026-03-18T00:00:00Z.","reset_at":"2026-03-18T00:00:00Z"}
```
