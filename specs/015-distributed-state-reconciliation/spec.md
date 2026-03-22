# Feature Specification: Distributed State and Reconciliation

**Feature Branch**: `015-distributed-state-reconciliation`
**Created**: 2026-03-22
**Status**: Draft
**Input**: User description: "Move beyond .agentspec.state.json. Support backends: local JSON (dev), k8s, etcd, postgres, s3, ...etc. The operator reconciles desired state from this control-plane."

## Clarifications

### Session 2026-03-22

- Q: Should all listed backends (k8s, etcd, postgres, s3) be in scope for this feature, or only a subset? → A: All five backends are in scope: local JSON (existing), Kubernetes, etcd, PostgreSQL, and S3-compatible object storage.
- Q: When reconciliation finds orphaned state entries (resources no longer in IntentLang files), should they be auto-deleted or only flagged? → A: Auto-delete after a configurable grace period (default 24 hours).
- Q: How should backend-specific connection parameters be supplied? → A: Defined as a dedicated block inside the IntentLang `.ias` file, with environment variable interpolation support for sensitive values (credentials, connection strings).
- Q: Should multi-backend replication (writing to two backends simultaneously) be in scope? → A: Out of scope — single active backend at a time; replication is a future feature.
- Q: When an `.ias` file has no `state` block, should a CLI flag override be supported? → A: Yes — `.ias` block is primary, CLI `--state-backend` flag can override it for ad-hoc use. Local JSON is the default when neither is specified.

## User Scenarios & Testing

### User Story 1 - Pluggable State Backend Selection (Priority: P1)

As a platform operator, I want to choose where AgentSpec stores its state (local JSON, Kubernetes CRDs, etcd, PostgreSQL, or S3-compatible object storage) so that I can match state storage to my deployment environment — single-machine for development, database-backed or cloud-native for production.

**Why this priority**: Without pluggable backends, all deployments are locked to a single-machine JSON file, blocking any multi-node or production use.

**Independent Test**: Can be fully tested by configuring different backend types via CLI flags or configuration, then verifying that state operations (save, load, get, list) work identically across each backend.

**Acceptance Scenarios**:

1. **Given** a user has a local development environment, **When** they run `agentspec apply` without specifying a backend, **Then** state is persisted to the local JSON file (`.agentspec.state.json`) as it does today.
2. **Given** a user has a Kubernetes cluster, **When** they declare a `state` block with type `kubernetes` in their `.ias` file, **Then** state entries are stored as Kubernetes custom resource status fields accessible via `kubectl`.
3. **Given** a user has an etcd cluster, **When** they declare a `state` block with type `etcd` and an endpoint using env var interpolation, **Then** state entries are stored as key-value pairs in etcd.
4. **Given** a user has a PostgreSQL database, **When** they declare a `state` block with type `postgres` and a DSN referencing an environment variable, **Then** state entries are stored in a dedicated table.
5. **Given** a user has S3-compatible storage, **When** they declare a `state` block with type `s3` and bucket/region configuration, **Then** state entries are stored as a JSON object in the configured bucket.
6. **Given** a user specifies an unsupported backend type, **When** they run any state-dependent command, **Then** they receive a clear error message listing available backend types.
7. **Given** a user switches backend type in configuration, **When** they run a state-dependent command, **Then** the system uses the newly configured backend without requiring data migration.

---

### User Story 2 - State Migration Between Backends (Priority: P2)

As a platform operator transitioning from development to production, I want to migrate my existing state from one backend to another so that I preserve my resource history and avoid re-applying all definitions from scratch.

**Why this priority**: Migration enables smooth transitions between environments. Without it, users must re-apply all resources when changing backends, losing history and version data.

**Independent Test**: Can be tested by populating state in one backend, running a migration command, and verifying all entries exist identically in the destination backend.

**Acceptance Scenarios**:

1. **Given** state exists in the local JSON backend with 50 entries, **When** the user runs a state migration command targeting any other backend, **Then** all 50 entries appear in the destination backend with identical data.
2. **Given** a migration is in progress, **When** the source backend becomes temporarily unavailable, **Then** the migration halts with a clear error and no partial writes corrupt the destination.
3. **Given** a migration completes successfully, **When** the user inspects both source and destination, **Then** the source remains unmodified (non-destructive copy).

---

### User Story 3 - Operator-Driven State Reconciliation (Priority: P3)

As a platform operator running AgentSpec in a Kubernetes cluster, I want the AgentSpec operator to continuously reconcile desired state from the control plane so that any drift between declared resources and actual state is automatically detected and corrected.

**Why this priority**: Reconciliation is the core value of a Kubernetes-native workflow. It builds on the pluggable backend (P1) and ensures declared agent configurations always converge to the desired state.

**Independent Test**: Can be tested by deploying an AgentSpec CRD, manually modifying the underlying state, and verifying the operator detects drift and re-applies the correct state.

**Acceptance Scenarios**:

1. **Given** an Agent CRD is created in Kubernetes, **When** the operator runs its reconciliation loop, **Then** the agent's desired state is applied and tracked in the configured state backend.
2. **Given** state drift occurs (an entry's hash no longer matches the declared resource), **When** the next reconciliation cycle runs, **Then** the operator detects the drift and re-applies the correct state.
3. **Given** a CRD is deleted, **When** the operator reconciles, **Then** the corresponding state entry is marked as removed or cleaned up.
4. **Given** the operator restarts, **When** it begins reconciling, **Then** it resumes from the current state without duplicating or losing entries.

---

### User Story 4 - State Observability and Health (Priority: P4)

As a platform operator, I want to inspect the health and status of my state backend so that I can quickly diagnose issues like connectivity failures, stale locks, or synchronization lag.

**Why this priority**: Observability is essential for operating distributed state in production but is not blocking for core functionality.

**Independent Test**: Can be tested by running a status command and verifying it reports backend type, connectivity, entry count, and last synchronization time.

**Acceptance Scenarios**:

1. **Given** the state backend is healthy, **When** the user runs a state status command, **Then** they see backend type, entry count, last write time, and a "healthy" indicator.
2. **Given** the state backend is unreachable, **When** the user runs a state status command, **Then** they see a "degraded" or "unreachable" status with the error details.
3. **Given** reconciliation is active, **When** the user queries status, **Then** they see the last reconciliation time and any pending drift items.

---

### Edge Cases

- What happens when two processes attempt to write to the same backend simultaneously? The backend must provide locking or optimistic concurrency to prevent data loss.
- What happens when the Kubernetes API server is temporarily unavailable? The Kubernetes backend must retry with exponential backoff and not lose pending writes.
- What happens when state entries reference resources that no longer exist in the source IntentLang files? The reconciliation loop must mark these as orphaned, surface them to the operator, and auto-delete them after a configurable grace period (default 24 hours).
- What happens when the state file format version in the source backend differs from what the destination backend expects? Migration must validate version compatibility before proceeding.
- What happens when a backend is configured but its dependencies are not available (e.g., Kubernetes backend outside a cluster)? The system must fail fast with a clear diagnostic message at startup.
- What happens when S3 storage has eventual consistency and a read-after-write returns stale data? The S3 backend must handle read-after-write consistency by verifying writes before confirming success.
- What happens when the PostgreSQL connection pool is exhausted? The system must queue requests or return a clear "backend busy" error rather than silently dropping state writes.

## Out of Scope

- **Multi-backend replication**: Writing state to two or more backends simultaneously for redundancy. Only one active backend is supported at a time. Replication is deferred to a future feature.
- **Backend-specific query languages**: Backends expose only the standard `Backend` interface; no backend-specific query or filtering capabilities beyond `List` with status filter.
- **Real-time state streaming**: No live event stream or webhook notifications for state changes. Polling or reconciliation-based approaches only.

## Requirements

### Functional Requirements

- **FR-001**: System MUST support five state backends: local JSON file (existing), Kubernetes CRD-based storage, etcd, PostgreSQL, and S3-compatible object storage.
- **FR-002**: System MUST allow backend configuration via a dedicated `state` block in the IntentLang `.ias` file, specifying backend type and connection parameters. Sensitive values (credentials, connection strings) MUST support environment variable interpolation. A CLI `--state-backend` flag MUST be available to override the `.ias` block for ad-hoc use. When neither is specified, local JSON is the default.
- **FR-003**: All backends MUST implement the existing `Backend` interface (`Load`, `Save`, `Get`, `List`) to ensure behavioral parity.
- **FR-004**: System MUST provide a state migration capability that copies all entries from one backend to another without modifying the source.
- **FR-005**: The Kubernetes operator MUST reconcile desired state by comparing declared CRD resources against actual state entries and correcting drift.
- **FR-006**: System MUST detect state drift by comparing resource content hashes between declared and stored state.
- **FR-007**: System MUST provide a status/health check command that reports backend type, connectivity, entry count, and last operation time.
- **FR-008**: All backends MUST support concurrent access safely, either through locking (local), optimistic concurrency (Kubernetes), transactions (PostgreSQL/etcd), or object versioning (S3).
- **FR-009**: System MUST preserve backward compatibility — the local JSON backend remains the default when no backend is explicitly configured.
- **FR-010**: Migration MUST be atomic at the entry level — either an entry is fully migrated or not at all; partial entry writes are not permitted.
- **FR-011**: Each backend MUST provide a connection validation method that verifies connectivity and access permissions at startup.
- **FR-012**: The reconciliation loop MUST auto-delete orphaned state entries (those with no corresponding IntentLang resource) after a configurable grace period (default 24 hours), emitting a warning when entries are first marked orphaned.

### Key Entities

- **StateEntry**: A record of a single resource's lifecycle state (FQN, hash, status, last applied time, adapter, error). Already exists as `state.Entry`.
- **StateBackend**: A pluggable storage provider implementing Load/Save/Get/List. Currently only `LocalBackend`; extended with `KubernetesBackend`, `EtcdBackend`, `PostgresBackend`, and `S3Backend`.
- **MigrationResult**: Represents the outcome of a state migration operation — source backend, destination backend, entries migrated, failed, skipped, and duration.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Users can switch between any of the five state backends with a single configuration change, with zero code modifications to their agent definitions.
- **SC-002**: State migration between any two backends completes for 1,000 entries in under 30 seconds.
- **SC-003**: The operator detects and corrects state drift within one reconciliation cycle (default interval).
- **SC-004**: All existing CLI commands (`apply`, `plan`, `validate`, `run`) work identically regardless of which backend is configured.
- **SC-005**: State health status is retrievable in under 2 seconds for any configured backend.
- **SC-006**: Zero data loss during backend migration — 100% of source entries are verifiable in the destination after migration.

## Assumptions

- The existing `Backend` interface (`Load`, `Save`, `Get`, `List`) is sufficient for all new backends. If a backend requires additional operations (e.g., `Watch`), they will be added as optional interface extensions rather than modifying the core interface.
- The Kubernetes backend uses a dedicated `StateStore` CRD (one per namespace) rather than embedding state in existing CRD status fields, to avoid polluting tightly scoped CRDs and to respect the 1.5 MB etcd object size limit.
- Budget and version state (currently stored in the local JSON file) will also be migrated when using the migration capability.
- The reconciliation loop reuses the existing operator controller pattern established in feature 014.
- Lock semantics differ by backend: local uses file locks (existing), Kubernetes uses resource version optimistic concurrency, etcd uses lease-based locking, PostgreSQL uses advisory locks, S3 uses conditional writes.
- The S3 backend stores the full state as a single JSON object per state scope, not one object per entry, to minimize API calls.
- The PostgreSQL backend assumes the user provides a connection string; schema creation is handled automatically on first use.
