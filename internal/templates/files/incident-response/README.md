# Incident Response

An incident triage agent with escalation skills for handling production incidents. Connects to monitoring, logging, ticketing, and paging systems to assess severity, gather diagnostics, attempt automated remediation, and escalate when needed.

## Prerequisites

- AgentSpec CLI installed
- `ANTHROPIC_API_KEY` environment variable set
- MCP-compatible monitoring service (e.g., Prometheus, Datadog)
- MCP-compatible logging service (e.g., Loki, ELK)
- MCP-compatible ticketing service (e.g., Jira, ServiceNow)
- MCP-compatible paging service (e.g., PagerDuty)

## Configuration

Set required environment variables:

```bash
export ANTHROPIC_API_KEY="your-key-here"
```

## Run

```bash
agentspec validate incident-response.ias
agentspec run incident-response.ias
```

## Customization

- Update the severity definitions in the system prompt to match your organization's incident classification.
- Change the MCP `server` values in each skill to point to your actual monitoring, logging, and ticketing endpoints.
- Add skills for additional remediation actions (e.g., restart services, scale infrastructure).
- Adjust `max_turns` based on the complexity of your incident workflows.
- Change the `port` in the deploy block if 9090 conflicts with existing services.
