# Data Model: Distributed State and Reconciliation

## Entities

### StateEntry (existing — `state.Entry`)

| Field       | Type      | Description                                      |
|-------------|-----------|--------------------------------------------------|
| FQN         | string    | Fully qualified name — unique identity for entry  |
| Hash        | string    | Content hash of the resource (SHA-256)            |
| Status      | Status    | Lifecycle status: `applied`, `failed`, `orphaned` |
| LastApplied | time.Time | Timestamp of last successful apply                |
| Adapter     | string    | Adapter name used for deployment                  |
| Error       | string    | Error message if status is `failed`               |
| OrphanedAt  | time.Time | Timestamp when entry was first marked orphaned (zero if not orphaned) |

**Changes**: Added `orphaned` status value and `OrphanedAt` field for grace period tracking.

### BudgetState (existing — `state.BudgetState`)

No changes. Migrated as-is between backends.

### VersionEntry (existing — `state.VersionEntry`)

No changes. Migrated as-is between backends.

### StateConfig (new — AST node)

| Field      | Type              | Description                                         |
|------------|-------------------|-----------------------------------------------------|
| Name       | string            | Optional name for the state configuration            |
| Type       | string            | Backend type: `local`, `kubernetes`, `etcd`, `postgres`, `s3` |
| Properties | map[string]string | Backend-specific properties (DSN, endpoint, bucket, etc.) |
| StartPos   | Pos               | Source position start                                |
| EndPos     | Pos               | Source position end                                  |

### StateConfig (new — IR type)

| Field      | Type              | Description                                         |
|------------|-------------------|-----------------------------------------------------|
| Type       | string            | Resolved backend type                                |
| Properties | map[string]string | Resolved properties (env vars interpolated)          |

### BackendConfig (new — runtime)

| Field    | Type              | Description                             |
|----------|-------------------|-----------------------------------------|
| Type     | string            | Backend type identifier                 |
| Props    | map[string]string | Resolved connection properties          |

Backend-specific property keys:

| Backend    | Required Properties                         | Optional Properties        |
|------------|---------------------------------------------|----------------------------|
| local      | `path` (default: `.agentspec.state.json`)   | —                          |
| kubernetes | —                                           | `namespace`, `name`        |
| etcd       | `endpoints`                                 | `dial_timeout`, `prefix`   |
| postgres   | `dsn`                                       | `table`, `max_conns`       |
| s3         | `bucket`                                    | `region`, `endpoint`, `prefix`, `key` |

### StateStore (new — Kubernetes CRD)

| Field            | Type                | Description                              |
|------------------|---------------------|------------------------------------------|
| metadata         | ObjectMeta          | Standard K8s metadata                    |
| spec.scope       | string              | Scope identifier (e.g., package name)    |
| status.entries   | []StateEntryStatus  | State entries stored as status           |
| status.lastWrite | metav1.Time         | Last write timestamp                     |
| status.healthy   | bool                | Health indicator                         |
| status.conditions| []metav1.Condition  | Standard K8s conditions                  |

### StateEntryStatus (new — embedded in CRD status)

| Field       | Type        | Description                       |
|-------------|-------------|-----------------------------------|
| fqn         | string      | Fully qualified name              |
| hash        | string      | Content hash                      |
| status      | string      | applied / failed / orphaned       |
| lastApplied | metav1.Time | Last apply timestamp              |
| adapter     | string      | Adapter name                      |
| error       | string      | Error message (optional)          |
| orphanedAt  | metav1.Time | Orphan detection timestamp        |

### MigrationResult (new — returned by migration)

| Field      | Type   | Description                              |
|------------|--------|------------------------------------------|
| Source     | string | Source backend type                       |
| Dest       | string | Destination backend type                 |
| Migrated   | int    | Number of entries successfully migrated  |
| Failed     | int    | Number of entries that failed            |
| Skipped    | int    | Number of entries skipped (already exist)|
| Duration   | time.Duration | Total migration time               |

## State Transitions

```text
StateEntry lifecycle:
  (new resource) → applied
  applied → applied (re-apply, hash updated)
  applied → failed (apply error)
  failed → applied (successful retry)
  applied → orphaned (resource no longer in .ias files)
  orphaned → applied (resource re-added to .ias files)
  orphaned → (deleted) (grace period expired, auto-cleaned)
```

## Relationships

- A **Backend** contains many **StateEntry** instances
- A **StateConfig** (in .ias file) configures exactly one **Backend**
- A **MigrationResult** references source and destination **Backend** instances
- A **StateStore** CRD maps 1:1 to a namespace's state entries in the Kubernetes backend
- **BudgetState** and **VersionEntry** are ancillary data stored alongside entries by backends that support it (local, postgres); other backends store entries only
