# Research: Distributed State and Reconciliation

## R1: etcd Go Client

**Decision**: Use `go.etcd.io/etcd/client/v3` (v3.6.9)
**Rationale**: Official etcd client, widely adopted (4,800+ importers), provides KV interface, lease-based locking via concurrency package, and maintenance/health API.
**Alternatives considered**: None — this is the canonical client.

**Key findings**:
- Module: `go.etcd.io/etcd/client/v3`
- CRUD: `cli.Put(ctx, key, val)`, `cli.Get(ctx, key, clientv3.WithPrefix())`, `cli.Delete(ctx, key)`
- Locking: `go.etcd.io/etcd/client/v3/concurrency` — session + mutex with lease TTL
- Health: `cli.Status(ctx, endpoint)` via Maintenance interface
- Gotchas: Must close client to avoid goroutine leaks; default send limit 2 MiB

## R2: AWS SDK for Go v2 (S3)

**Decision**: Use `github.com/aws/aws-sdk-go-v2/service/s3`
**Rationale**: Official AWS SDK v2 for Go, supports S3-compatible stores (MinIO, DigitalOcean Spaces) via custom endpoint resolver.
**Alternatives considered**: `github.com/minio/minio-go/v7` — MinIO-native, but limits interop with non-MinIO S3 providers.

**Key findings**:
- Module: `github.com/aws/aws-sdk-go-v2/service/s3` + `github.com/aws/aws-sdk-go-v2/config`
- Operations: `PutObject`, `GetObject`, `HeadBucket` (connectivity check)
- Consistency: S3 provides strong read-after-write consistency since Dec 2020
- Conditional writes: Use `IfMatch` (ETag) for optimistic concurrency on PutObject
- S3-compatible: Custom `BaseEndpoint` in client options
- State stored as single JSON object per scope to minimize API calls

## R3: pgx PostgreSQL Driver

**Decision**: Use `github.com/jackc/pgx/v5` (v5.8.0)
**Rationale**: Most performant pure-Go PostgreSQL driver, supports connection pooling (pgxpool), advisory locks, and prepared statements.
**Alternatives considered**: `lib/pq` — unmaintained, no pool, fewer features.

**Key findings**:
- Module: `github.com/jackc/pgx/v5`, pool via `github.com/jackc/pgx/v5/pgxpool`
- Connect: `pgxpool.New(ctx, os.Getenv("DATABASE_URL"))` — supports URL and key=value formats
- CRUD: `pool.QueryRow(ctx, sql, args...)`, `pool.Exec(ctx, sql, args...)`
- Advisory locks: `SELECT pg_advisory_lock(key)` / `SELECT pg_advisory_unlock(key)` — session-level locking
- Health: `pool.Ping(ctx)`
- Auto-schema: `CREATE TABLE IF NOT EXISTS` on first use
- Pool defaults: MaxConns = max(4, NumCPU), HealthCheckPeriod = 1m

## R4: Kubernetes State Storage via CRD Status

**Decision**: Create a dedicated `StateStore` CRD rather than embedding state in existing CRD status fields.
**Rationale**: Existing CRDs (Agent, Task, etc.) have tightly scoped status fields. A dedicated CRD avoids polluting them and respects the 1.5 MB etcd object size limit. One `StateStore` per namespace holds entries for that namespace.
**Alternatives considered**: Embedding state entries as ConfigMap data — less structured, no status subresource.

**Key findings**:
- Use controller-runtime `client.StatusClient` for status updates
- Optimistic concurrency via `resourceVersion` — retry on conflict errors (`errors.IsConflict`)
- etcd object size limit: 1.5 MB — cap entries per CRD instance, split if needed
- Connectivity check: simple `List` call with limit=1 on the StateStore CRD
- Reconciliation: existing operator controller pattern (feature 014) can be extended

## R5: IntentLang `state` Block Design

**Decision**: Add a new `state` top-level block to the IntentLang grammar with type and backend-specific properties.
**Rationale**: Follows the same pattern as `deploy`, `secret`, and `binding` blocks. Env var interpolation via `${}` syntax already exists for secrets.

**Key findings**:
- New AST node: `StateConfig` with Type, Properties map, and env var interpolation
- New IR field: `StateConfig` added to `ir.Document`
- Parser change: handle `state` keyword as top-level block
- Formatter change: handle `StateConfig` node formatting
- Lowering: resolve env vars during lowering, validate backend type
- CLI override: `--state-backend` flag takes precedence over `.ias` block
