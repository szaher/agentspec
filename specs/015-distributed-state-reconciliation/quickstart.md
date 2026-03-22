# Quickstart: Distributed State and Reconciliation

## Scenario 1: Local Development (default)

No changes needed — existing behavior is preserved.

```bash
# Apply with local JSON state (default)
agentspec apply myagent.ias
# State saved to .agentspec.state.json
```

## Scenario 2: Configure PostgreSQL Backend

Add a `state` block to your `.ias` file:

```
package "my-agents" version "1.0.0" lang "3.0"

state "production" {
  type "postgres"
  dsn "${DATABASE_URL}"
}

agent "assistant" {
  model "claude-sonnet-4-5-20250929"
  ...
}
```

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/agentspec"
agentspec apply myagent.ias
# State saved to PostgreSQL
```

## Scenario 3: Migrate Local State to PostgreSQL

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/agentspec"

# Preview migration
agentspec state migrate --from local --to postgres --to-dsn="${DATABASE_URL}" --dry-run

# Execute migration
agentspec state migrate --from local --to postgres --to-dsn="${DATABASE_URL}"
```

## Scenario 4: Check State Backend Health

```bash
agentspec state status myagent.ias
# State Backend: postgres
# Status:        healthy
# Entries:       42
# Last Write:    2026-03-22T10:30:00Z
```

## Scenario 5: Override Backend via CLI Flag

```bash
# Use local backend even though .ias configures postgres
agentspec apply --state-backend=local myagent.ias

# Use S3 backend ad-hoc
agentspec apply --state-backend=s3 --state-bucket=my-bucket myagent.ias
```

## Scenario 6: Kubernetes Operator Reconciliation

Deploy the operator with state reconciliation enabled:

```yaml
apiVersion: agentspec.io/v1alpha1
kind: Agent
metadata:
  name: my-assistant
  namespace: production
spec:
  model: claude-sonnet-4-5-20250929
  promptRef: assistant-prompt
```

The operator detects drift between declared CRDs and actual state, re-applying when needed. Orphaned entries are auto-deleted after 24 hours.

## Integration Test Scenarios

1. **Backend parity**: Save 100 entries to each backend, load them back, verify identical data
2. **Concurrent writes**: Two goroutines write to the same backend simultaneously — no corruption
3. **Migration roundtrip**: local → postgres → local preserves all entries
4. **Health check**: Ping healthy backend returns nil; ping unreachable backend returns error
5. **CLI override**: `--state-backend` flag overrides `.ias` state block
6. **Env var interpolation**: `${DB_URL}` resolves correctly; unset var produces error
7. **Orphan detection**: Remove resource from `.ias`, reconcile, verify orphan marking and eventual deletion
