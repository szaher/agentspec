# Incident Response

A single-agent incident response system that triages alerts, looks up and executes remediation runbooks, and escalates to on-call engineers when automated resolution fails. Demonstrates a multi-skill agent for operational automation.

## Architecture Overview

```
Incoming Alert
    |
    v
incident-responder (agent)
    |
    +---> triage-alert      -- classifies severity and category
    +---> lookup-runbook    -- retrieves the matching remediation runbook
    +---> execute-runbook   -- runs the remediation steps
    +---> escalate-oncall   -- pages on-call if unresolved
```

The agent follows a sequential workflow: triage the alert, find the right runbook, attempt automated remediation, and escalate only if the runbook cannot resolve the issue.

## Prerequisites

1. Build the AgentSpec CLI from the repository root:

   ```bash
   go build -o agentspec ./cmd/agentspec
   ```

2. Set the required environment variable:

   ```bash
   export PAGERDUTY_API_KEY="your-pagerduty-api-key"
   ```

3. Ensure the following tool binaries are available on `$PATH` (or stubbed for testing):
   - `alert-triage`
   - `runbook-lookup`
   - `runbook-executor`
   - `pagerduty-escalate`

## Step-by-Step Run Instructions

```bash
# 1. Validate the AgentSpec
./agentspec validate examples/incident-response/incident-response.ias

# 2. Preview planned changes
./agentspec plan examples/incident-response/incident-response.ias

# 3. Apply the changes
./agentspec apply examples/incident-response/incident-response.ias --auto-approve

# 4. Run the agent with a sample alert
./agentspec dev examples/incident-response/incident-response.ias --input "CPU usage exceeded 95% on prod-web-03"

# 5. Export artifacts
./agentspec export examples/incident-response/incident-response.ias --out-dir ./output
```

## Customization Tips

- **Add more skills**: Define skills for specific remediation actions (e.g., `restart-service`, `scale-up-instances`, `rollback-deploy`) for finer-grained automation.
- **Add severity-based routing**: Upgrade to lang "3.0" and use `on input` with `if/else` to route critical alerts directly to escalation, bypassing runbook execution.
- **Add environment overlays**: Use a cheaper model for dev/test alert triage and a more capable model for production incidents.
- **Integrate monitoring**: Add a `check-metrics` skill that queries Prometheus or Datadog to verify remediation success.
- **Add policies**: Attach rate-limiting policies to prevent the agent from executing too many runbooks in a short window.
