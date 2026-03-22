# Contract: IntentLang `state` Block Grammar

## Syntax

```
state [<name>] {
  type <backend-type>
  <property> <value>
  ...
}
```

## Examples

### Local JSON (explicit)
```
state "dev" {
  type "local"
  path ".agentspec.state.json"
}
```

### Kubernetes
```
state "production" {
  type "kubernetes"
  namespace "agentspec"
}
```

### etcd
```
state "cluster" {
  type "etcd"
  endpoints "${ETCD_ENDPOINTS}"
  prefix "/agentspec/state/"
}
```

### PostgreSQL
```
state "db" {
  type "postgres"
  dsn "${DATABASE_URL}"
  table "agentspec_state"
}
```

### S3
```
state "cloud" {
  type "s3"
  bucket "my-agentspec-state"
  region "us-east-1"
  prefix "state/"
  endpoint "${S3_ENDPOINT}"
}
```

## AST Node

```go
type StateConfig struct {
    Name       string
    Type       string
    Properties map[string]string
    StartPos   Pos
    EndPos     Pos
}
```

## IR Representation

```json
{
  "state_config": {
    "type": "postgres",
    "properties": {
      "dsn": "postgres://user:pass@localhost/db",
      "table": "agentspec_state"
    }
  }
}
```

## Validation Rules

1. `type` is REQUIRED and MUST be one of: `local`, `kubernetes`, `etcd`, `postgres`, `s3`
2. Property values containing `${}` MUST be resolved to environment variables during lowering
3. Unresolved env vars MUST produce a validation error with source location
4. At most one `state` block is allowed per package
5. Backend-specific required properties:
   - `etcd`: `endpoints`
   - `postgres`: `dsn`
   - `s3`: `bucket`
   - `local`: none (all optional)
   - `kubernetes`: none (uses in-cluster config by default)

## CLI Override

```bash
# Override .ias state block for ad-hoc use
agentspec apply --state-backend=local myagent.ias
agentspec apply --state-backend=postgres --state-dsn="${DATABASE_URL}" myagent.ias
```

The `--state-backend` flag takes precedence over the `.ias` state block. Additional `--state-*` flags provide backend-specific properties when using CLI override.
