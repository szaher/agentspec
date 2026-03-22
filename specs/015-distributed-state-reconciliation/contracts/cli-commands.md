# Contract: CLI Commands

## New Commands

### `agentspec state status`

Show state backend health and statistics.

```
Usage:
  agentspec state status [file.ias] [flags]

Flags:
  --state-backend string   Override state backend type

Output (healthy):
  State Backend: postgres
  Status:        healthy
  Entries:       142
  Last Write:    2026-03-22T10:30:00Z
  Connection:    postgres://***@db.example.com/agentspec

Output (unhealthy):
  State Backend: postgres
  Status:        unreachable
  Error:         dial tcp 10.0.0.5:5432: connection refused
```

### `agentspec state migrate`

Migrate state entries between backends.

```
Usage:
  agentspec state migrate --from <type> --to <type> [flags]

Flags:
  --from string          Source backend type (required)
  --to string            Destination backend type (required)
  --from-* string        Source backend-specific properties
  --to-* string          Destination backend-specific properties
  --dry-run              Show what would be migrated without executing

Output:
  Migrating state: local → postgres
  [========================================] 142/142 entries

  Migration complete:
    Migrated: 142
    Failed:   0
    Skipped:  0
    Duration: 2.3s

Output (dry-run):
  Dry run: local → postgres
  Would migrate 142 entries
  Source: .agentspec.state.json (142 entries)
  Destination: postgres://***@db.example.com/agentspec (0 entries)
```

## Modified Commands

### `agentspec apply` (extended)

New flags:
- `--state-backend string` — Override state backend type from .ias file
- `--state-dsn string` — Backend DSN (postgres, etcd endpoints)
- `--state-bucket string` — S3 bucket name
- `--state-endpoint string` — Custom endpoint (S3-compatible, etcd)

### `agentspec plan` (extended)

Same new flags as `apply`. Plans use the configured backend to load current state for diff.

### Global Flag

- `--state-file string` — (existing, renamed conceptually) Path to local state file. Still works for backward compatibility but is equivalent to `--state-backend=local --state-path=<value>`.

## Exit Codes

| Code | Meaning                              |
|------|--------------------------------------|
| 0    | Success                              |
| 1    | General error                        |
| 2    | Backend unreachable / connection error |
| 3    | Migration failed (partial)           |
