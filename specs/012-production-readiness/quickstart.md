# Quickstart: Production Readiness & Advanced Features

## Scenario 1: TLS Encryption

```bash
# Generate self-signed cert for testing
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes -subj '/CN=localhost'

# Start server with TLS
./agentspec run examples/basic-agent/basic-agent.ias --tls-cert cert.pem --tls-key key.pem --port 8443

# Verify HTTPS works
curl -k https://localhost:8443/v1/agents

# Verify HTTP is rejected (should fail)
curl http://localhost:8443/v1/agents  # connection refused or protocol error
```

## Scenario 2: Multi-User Access Control

```ias
package "team-agents" version "1.0.0" lang "2.0"

secret "alice-key" { env(ALICE_API_KEY) }
secret "bob-key" { env(BOB_API_KEY) }

user "alice" {
  key secret("alice-key")
  agents ["support-agent"]
  role "invoke"
}

user "bob" {
  key secret("bob-key")
  agents ["support-agent", "admin-agent"]
  role "admin"
}

prompt "system" { content "You are helpful." }
agent "support-agent" { uses prompt "system" model "claude-sonnet-4-20250514" }
agent "admin-agent" { uses prompt "system" model "claude-sonnet-4-20250514" }
```

```bash
# Alice can invoke support-agent
curl -H "X-API-Key: $ALICE_API_KEY" https://localhost:8443/v1/agents/support-agent/invoke -d '{"input":"hello"}'
# → 200 OK

# Alice cannot invoke admin-agent
curl -H "X-API-Key: $ALICE_API_KEY" https://localhost:8443/v1/agents/admin-agent/invoke -d '{"input":"hello"}'
# → 403 Forbidden

# Check audit log
tail -1 agentspec-audit.log | jq .
# → {"user":"alice","agent":"support-agent","action":"invoke","status":"success",...}
```

## Scenario 3: Cost Tracking and Budgets

```ias
agent "support-agent" {
  uses prompt "system"
  model "claude-sonnet-4-20250514"
  budget daily 10.0
  budget monthly 200.0
}
```

```bash
# Run the server
./agentspec run budget-agent.ias

# Invoke until budget exceeded
# After ~$10 of estimated usage:
# → 429 Too Many Requests: "Agent 'support-agent' daily budget of $10.00 exceeded"

# Check metrics
curl https://localhost:8443/v1/metrics | grep budget
# agentspec_budget_usage_ratio{agent="support-agent",period="daily"} 1.05
```

## Scenario 4: Multi-Model Fallback

```ias
agent "resilient-agent" {
  uses prompt "system"
  models ["claude-sonnet-4-20250514", "gpt-4o-mini"]
}
```

```bash
# If Claude API is down, agent automatically falls back to GPT-4o-mini
# Logs show: "primary model failed, falling back to gpt-4o-mini"
```

## Scenario 5: Content Guardrails

```ias
guardrail "pii-filter" {
  mode "block"
  keywords ["SSN", "social security"]
  patterns ["\\d{3}-\\d{2}-\\d{4}"]
  fallback "I cannot share personal identification information."
}

agent "support-agent" {
  uses prompt "system"
  uses guardrail "pii-filter"
  model "claude-sonnet-4-20250514"
}
```

## Scenario 6: Versioning and Rollback

```bash
# Deploy version 1
./agentspec apply agent-v1.ias

# Deploy version 2 (introduces regression)
./agentspec apply agent-v2.ias

# Check history
./agentspec history --agent support-agent
# Version  Timestamp                 Summary
# 2        2026-03-17T10:00:00Z      Updated model
# 1        2026-03-16T15:00:00Z      Initial deployment

# Roll back
./agentspec rollback --agent support-agent
# Rolled back "support-agent" from version 2 to version 1
```

## Scenario 7: Observability Dashboard

```bash
# Start server (metrics already exposed at /v1/metrics)
./agentspec run agents.ias

# Import the Grafana dashboard
# File: dashboards/agentspec-overview.json
# → Import in Grafana UI → Data source: Prometheus → Dashboard appears

# Dashboard panels:
# - Agent invocation rate (req/s)
# - Latency percentiles (p50, p95, p99)
# - Token consumption by agent and model
# - Cost tracking ($)
# - Tool call patterns
# - Guardrail violation rate
```

## Scenario 8: Release Automation

```bash
# Tag a release
git tag v1.2.0
git push origin v1.2.0

# GitHub Actions automatically:
# 1. Builds binaries for 6 platforms
# 2. Creates GitHub Release with changelog
# 3. Uploads binaries and checksums
# 4. Binary reports: ./agentspec version → "agentspec v1.2.0 (commit abc123)"
```
