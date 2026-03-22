# Tasks: Distributed State and Reconciliation

**Input**: Design documents from `/specs/015-distributed-state-reconciliation/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Constitution mandates >=80% line coverage for the `state` package (security-critical). Unit tests are included for all new backend files.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add new dependencies and prepare project structure

- [x] T001 Add `go.etcd.io/etcd/client/v3` v3.6+, `github.com/jackc/pgx/v5` v5.8+, `github.com/aws/aws-sdk-go-v2/service/s3`, and `github.com/aws/aws-sdk-go-v2/config` to `go.mod`
- [x] T002 Run `go mod tidy` to resolve transitive dependencies and verify `go build ./...` succeeds

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core abstractions that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T003 Add `StateConfig` AST node (Name, Type, Properties, StartPos, EndPos) to `internal/ast/ast.go`
- [x] T004 Add `StateConfig` IR type (Type, Properties map) and add `StateConfig *StateConfig` field to `ir.Document` in `internal/ir/ir.go`
- [x] T005 Extend parser to handle `state` keyword as a top-level block, parsing name, type, and property key-value pairs into `ast.StateConfig` in `internal/parser/parser.go`
- [x] T006 Extend formatter to emit canonical formatting for `state` blocks in `internal/formatter/formatter.go`
- [x] T007 Extend lowering to resolve `${}` env var interpolation in `StateConfig` properties and validate backend type (local, kubernetes, etcd, postgres, s3) in `internal/lowering/lower.go`
- [x] T008 Add `HealthChecker`, `Closer`, `Locker`, `BudgetStore`, and `VersionStore` optional interfaces to `internal/state/state.go`
- [x] T009 Add `orphaned` status value to the `Status` type and `OrphanedAt` field to `Entry` struct in `internal/state/state.go`
- [x] T010 Create backend registry with `Register`, `New`, and `Available` functions in `internal/state/registry.go`
- [x] T011 Register the existing `LocalBackend` as the `local` backend in the registry in `internal/state/registry.go`

**Checkpoint**: Foundation ready — backend interface extended, grammar supports `state` block, registry is operational with local backend registered

---

## Phase 3: User Story 1 — Pluggable State Backend Selection (Priority: P1) MVP

**Goal**: Enable users to choose where AgentSpec stores state via configuration or CLI flag, with five backend implementations

**Independent Test**: Configure different backend types via CLI flags or `.ias` state block, then verify save/load/get/list work identically across each backend

### Implementation for User Story 1

- [x] T012 [P] [US1] Implement `EtcdBackend` (Backend, HealthChecker, Locker, Closer interfaces) with KV CRUD, lease-based locking, Status() health check in `internal/state/etcd.go`
- [x] T013 [P] [US1] Implement `PostgresBackend` (Backend, HealthChecker, Locker, Closer, BudgetStore, VersionStore interfaces) with pgxpool, advisory locks, auto-schema creation, and pool exhaustion error handling (return clear "backend busy" error) in `internal/state/postgres.go`
- [x] T014 [P] [US1] Implement `S3Backend` (Backend, HealthChecker, Closer interfaces) with single JSON object storage, ETag optimistic concurrency, HeadBucket health check in `internal/state/s3.go`
- [x] T015 [P] [US1] Implement `KubernetesBackend` (Backend, HealthChecker, Closer interfaces) with StateStore CRD status updates, resourceVersion optimistic concurrency in `internal/state/kubernetes.go`
- [x] T016 [P] [US1] Write unit tests for `EtcdBackend` (CRUD, locking, health check, close) targeting >=80% coverage in `internal/state/etcd_test.go`
- [x] T017 [P] [US1] Write unit tests for `PostgresBackend` (CRUD, advisory locks, auto-schema, health check, pool exhaustion) targeting >=80% coverage in `internal/state/postgres_test.go`
- [x] T018 [P] [US1] Write unit tests for `S3Backend` (CRUD, ETag concurrency, health check) targeting >=80% coverage in `internal/state/s3_test.go`
- [x] T019 [P] [US1] Write unit tests for `KubernetesBackend` (CRUD, resourceVersion concurrency, health check) targeting >=80% coverage in `internal/state/kubernetes_test.go`
- [x] T020 [US1] Write unit tests for backend registry (`Register`, `New`, `Available`, unknown type error) in `internal/state/registry_test.go`
- [x] T021 [US1] Register all new backends (etcd, postgres, s3, kubernetes) in `internal/state/registry.go`
- [x] T022 [US1] Add `--state-backend`, `--state-dsn`, `--state-bucket`, `--state-endpoint` flags to root command in `cmd/agentspec/main.go`
- [x] T023 [US1] Refactor `apply` command to resolve backend from `.ias` state block or CLI flags via registry instead of hardcoded `NewLocalBackend`, validate backend connectivity at startup via `HealthChecker.Ping()` before proceeding, in `cmd/agentspec/apply.go`
- [x] T024 [US1] Refactor `plan` command to use the same backend resolution and startup validation logic in `cmd/agentspec/plan.go`
- [x] T025 [US1] Ensure `Close()` is called on backends that implement `Closer` at command exit in `cmd/agentspec/apply.go`

**Checkpoint**: All five backends are functional. Users can configure any backend via `.ias` state block or CLI flags. Existing local JSON behavior is preserved as default.

---

## Phase 4: User Story 2 — State Migration Between Backends (Priority: P2)

**Goal**: Enable non-destructive copy of all state entries from one backend to another

**Independent Test**: Populate state in local backend, run migration to postgres, verify all entries exist identically in destination

### Implementation for User Story 2

- [x] T026 [US2] Implement `Migrate` function (load from source, save to destination, entry-level atomicity, dry-run support, source/destination version compatibility validation) returning `MigrationResult` in `internal/state/migrate.go`
- [x] T027 [US2] Write unit tests for `Migrate` function (roundtrip, dry-run, partial failure, version validation) targeting >=80% coverage in `internal/state/migrate_test.go`
- [x] T028 [US2] Create `agentspec state migrate` CLI command with `--from`, `--to`, `--from-*`, `--to-*`, `--dry-run` flags, progress output, and exit codes (0/1/3) in `cmd/agentspec/state_cmd.go`

**Checkpoint**: State can be migrated between any two backends. Source remains unmodified. Dry-run shows what would be migrated.

---

## Phase 5: User Story 3 — Operator-Driven State Reconciliation (Priority: P3)

**Goal**: Kubernetes operator detects drift between declared CRDs and actual state, re-applying when needed, with orphan auto-deletion

**Independent Test**: Deploy an Agent CRD, manually modify underlying state, verify operator detects drift and re-applies

### Implementation for User Story 3

- [x] T029 [P] [US3] Define `StateStore` CRD types (StateStoreSpec, StateStoreStatus, StateEntryStatus) in `internal/api/v1alpha1/statestore_types.go`
- [x] T030 [P] [US3] Generate StateStore CRD manifest in `config/crd/bases/agentspec.io_statestores.yaml`
- [x] T031 [US3] Implement `StateStoreReconciler` with drift detection (hash comparison), re-apply on drift, orphan marking, and grace-period auto-deletion (default 24h) in `internal/operator/controller/statestore_controller.go`
- [x] T032 [US3] Register `StateStoreReconciler` with the operator manager in `cmd/agentspec/operator.go`

**Checkpoint**: Operator reconciles declared vs actual state. Orphaned entries are marked and auto-deleted after grace period. Drift is detected and corrected.

---

## Phase 6: User Story 4 — State Observability and Health (Priority: P4)

**Goal**: Provide CLI command to inspect state backend health, entry count, and last write time

**Independent Test**: Run `agentspec state status` against a configured backend and verify output shows type, connectivity, entries, and last write

### Implementation for User Story 4

- [x] T033 [US4] Create `agentspec state status` CLI command that calls `HealthChecker.Ping()`, counts entries via `List()`, and reports backend type, status, entry count, last write time, and masked connection info in `cmd/agentspec/state_cmd.go`
- [x] T034 [US4] Add exit code 2 for backend unreachable errors in `cmd/agentspec/state_cmd.go`

**Checkpoint**: Users can check backend health with a single command. Healthy and unhealthy states produce distinct, actionable output.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Integration validation and cross-cutting improvements

- [x] T035 Create backend parity integration test (save/load/get/list 100 entries across all backends, verify identical data) in `integration_tests/state_backend_test.go`
- [x] T036 Create concurrent write integration test (two goroutines write to same backend simultaneously, verify no corruption) in `integration_tests/state_backend_test.go`
- [x] T037 Verify all existing CLI commands (`apply`, `plan`, `validate`, `run`, `dev`, `eval`, `diff`, `export`) work with non-local backends — fix any hardcoded `LocalBackend` references
- [x] T038 Run `gofmt -l .`, `go build ./...`, `go test ./... -count=1` pre-commit checks and fix any issues
- [x] T039 Run quickstart.md scenarios manually and verify expected output

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational (Phase 2)
- **User Story 2 (Phase 4)**: Depends on User Story 1 (needs registry + at least 2 backends)
- **User Story 3 (Phase 5)**: Depends on Foundational (Phase 2), can proceed in parallel with US1 for CRD types (T023-T024)
- **User Story 4 (Phase 6)**: Depends on User Story 1 (needs HealthChecker implementations)
- **Polish (Phase 7)**: Depends on all user stories being complete

### Within Each User Story

- Backend implementations (T012-T015) can run in parallel
- Unit tests (T016-T019) can run in parallel (after their respective backend impl)
- Registry tests and registration (T020-T021) depend on backend implementations
- CLI integration (T022-T025) depends on registry
- CRD types (T029-T030) can run in parallel with each other
- Reconciler (T031) depends on CRD types

### Parallel Opportunities

```text
# Phase 2: These can run in parallel (different files):
T003 (ast.go) || T004 (ir.go) || T008 (state.go) || T010 (registry.go)

# Phase 3: All backend implementations in parallel:
T012 (etcd.go) || T013 (postgres.go) || T014 (s3.go) || T015 (kubernetes.go)

# Phase 3: All backend unit tests in parallel (after impl):
T016 (etcd_test.go) || T017 (postgres_test.go) || T018 (s3_test.go) || T019 (kubernetes_test.go)

# Phase 5: CRD types in parallel:
T029 (statestore_types.go) || T030 (CRD manifest)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (add dependencies)
2. Complete Phase 2: Foundational (grammar, interfaces, registry)
3. Complete Phase 3: User Story 1 (all 5 backends + CLI flags)
4. **STOP and VALIDATE**: Test each backend independently with `agentspec apply`
5. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. User Story 1 → Pluggable backends working → MVP!
3. User Story 2 → Migration between backends
4. User Story 3 → Operator reconciliation with drift detection
5. User Story 4 → Health/status observability
6. Polish → Integration tests, cross-command validation

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Unit tests included per constitution mandate (state package >=80% line coverage)
- US2 depends on US1 because migration needs at least two backend implementations
- US4 depends on US1 because health checks require HealthChecker implementations
- US3 CRD types can start early but reconciler needs backend registry from US1
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
