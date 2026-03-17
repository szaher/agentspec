# Data Model: Production Readiness & Advanced Features

**Feature Branch**: `012-production-readiness`
**Date**: 2026-03-17

## Entities

### User (new DSL block + IR entity)

Represents an authenticated user with per-agent access control.

| Field     | Type     | Required | Description                                    |
|-----------|----------|----------|------------------------------------------------|
| Name      | string   | yes      | Unique user identifier (DSL block name)        |
| KeyRef    | string   | yes      | Secret reference for API key (`secret("ENV")`) |
| Agents    | []string | yes      | List of agent names this user can access        |
| Role      | string   | no       | Permission level: `invoke` (default), `admin`  |

**Validation rules**:
- Name must be unique across all `user` blocks
- KeyRef must reference a declared `secret` block
- Each agent in `Agents` must reference a declared `agent` block
- Role must be one of: `invoke`, `admin`

**Relationships**: User → Secret (key reference), User → Agent (access list)

### AuditEntry (runtime entity, not persisted in state)

Written as one JSON line per invocation to the audit log file.

| Field       | Type   | Required | Description                                  |
|-------------|--------|----------|----------------------------------------------|
| Timestamp   | string | yes      | ISO 8601 timestamp                           |
| UserName    | string | yes      | Authenticated user name (or "anonymous")     |
| AgentName   | string | yes      | Invoked agent name                           |
| SessionID   | string | yes      | Session identifier                           |
| Action      | string | yes      | `invoke`, `stream`, `session_create`         |
| TokensIn    | int    | no       | Input tokens consumed                        |
| TokensOut   | int    | no       | Output tokens consumed                       |
| Duration    | string | no       | Request duration (e.g., "1.234s")            |
| Status      | string | yes      | `success`, `error`, `budget_exceeded`, `forbidden` |
| CorrelationID | string | yes    | Request correlation ID (ULID)                |

### AgentBudget (persisted in state file)

Tracks spending limits and current usage per agent.

| Field        | Type    | Required | Description                                |
|--------------|---------|----------|--------------------------------------------|
| AgentName    | string  | yes      | Agent this budget applies to               |
| Period       | string  | yes      | `daily` or `monthly`                       |
| LimitDollars | float64 | yes      | Maximum spend for the period               |
| UsedDollars  | float64 | yes      | Current accumulated spend                  |
| ResetAt      | string  | yes      | ISO 8601 timestamp for next reset          |
| Paused       | bool    | yes      | Whether agent is currently paused          |
| WarnedAt     | string  | no       | When 80% warning was last emitted          |

**State transitions**:
- `active` → `warned` (at 80% usage, sets WarnedAt)
- `warned` → `paused` (at 100% usage, sets Paused=true)
- `paused` → `active` (on period reset, clears UsedDollars and Paused)

### ModelChain (IR entity, configured in agent block)

Ordered list of models with fallback semantics.

| Field   | Type     | Required | Description                                |
|---------|----------|----------|--------------------------------------------|
| Models  | []string | yes      | Ordered model identifiers (primary first)  |

**Behavior**: Try models in order. On error/rate-limit from model N, try model N+1. Log a warning on each fallback. If all fail, return error with all failure reasons.

### Guardrail (new DSL block + IR entity)

Content filter applied to agent output.

| Field       | Type     | Required | Description                                |
|-------------|----------|----------|--------------------------------------------|
| Name        | string   | yes      | Unique guardrail identifier                |
| Mode        | string   | yes      | `warn` or `block`                          |
| Keywords    | []string | no       | Blocked keyword list (case-insensitive)    |
| Patterns    | []string | no       | Blocked regex patterns                     |
| FallbackMsg | string   | no       | Replacement message (block mode only)      |

**Validation rules**:
- At least one of Keywords or Patterns must be non-empty
- FallbackMsg required when Mode is `block`
- Mode must be one of: `warn`, `block`

**Relationships**: Agent → Guardrail (via `uses guardrail "name"`)

### AgentVersion (persisted in state file)

Snapshot of agent configuration for rollback.

| Field      | Type   | Required | Description                                 |
|------------|--------|----------|---------------------------------------------|
| Version    | int    | yes      | Sequential version number                   |
| AgentName  | string | yes      | Agent this version belongs to               |
| IRHash     | string | yes      | SHA-256 hash of the agent's IR at this version |
| IRSnapshot | object | yes      | Full IR for this agent (for restore)        |
| Timestamp  | string | yes      | ISO 8601 when this version was created      |
| Summary    | string | yes      | Human-readable change summary               |

**Retention**: Last 10 versions per agent. Oldest version is evicted when a new one is added.

### ModelPricing (configuration entity)

Static pricing table for cost estimation.

| Field          | Type    | Required | Description                          |
|----------------|---------|----------|--------------------------------------|
| Model          | string  | yes      | Model identifier (e.g., "claude-sonnet-4-20250514") |
| InputPerMTok   | float64 | yes      | Cost per million input tokens ($)    |
| OutputPerMTok  | float64 | yes      | Cost per million output tokens ($)   |

**Source**: Embedded default table, overridable via configuration.

## State File Schema Extensions

The existing `.agentspec.state.json` gains two new top-level keys:

```json
{
  "version": "1.0",
  "entries": [...],
  "budgets": [
    {
      "agent_name": "support-agent",
      "period": "daily",
      "limit_dollars": 10.0,
      "used_dollars": 3.50,
      "reset_at": "2026-03-18T00:00:00Z",
      "paused": false,
      "warned_at": ""
    }
  ],
  "agent_versions": {
    "support-agent": [
      {
        "version": 2,
        "ir_hash": "abc123...",
        "ir_snapshot": { ... },
        "timestamp": "2026-03-17T10:00:00Z",
        "summary": "Updated model to claude-sonnet-4-20250514"
      }
    ]
  }
}
```
