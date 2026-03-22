# Implementation Plan: Distributed State and Reconciliation

**Branch**: `015-distributed-state-reconciliation` | **Date**: 2026-03-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/015-distributed-state-reconciliation/spec.md`

## Summary

Replace the hardcoded local JSON state backend with a pluggable backend system supporting five storage providers (local JSON, Kubernetes CRDs, etcd, PostgreSQL, S3). Add a `state` block to the IntentLang grammar for backend configuration, a state migration CLI command, operator-driven reconciliation with drift detection, and state health observability.

## Technical Context

**Language/Version**: Go 1.25+ (existing)
**Primary Dependencies**:
- `go.etcd.io/etcd/client/v3` v3.6+ (etcd backend)
- `github.com/jackc/pgx/v5` v5.8+ (PostgreSQL backend)
- `github.com/aws/aws-sdk-go-v2/service/s3` (S3 backend)
- `sigs.k8s.io/controller-runtime` (existing, Kubernetes backend)
- `github.com/spf13/cobra` v1.10.2 (existing CLI)
**Storage**: Local JSON file (existing), Kubernetes CRDs, etcd, PostgreSQL, S3-compatible object storage
**Testing**: `go test`, `envtest` (Kubernetes), integration tests with testcontainers or mocks
**Target Platform**: Linux/macOS CLI + Kubernetes operator
**Project Type**: CLI + operator (existing)
**Performance Goals**: Migration of 1,000 entries < 30s; health status < 2s
**Constraints**: Backward compatible — local JSON remains default; no breaking changes to existing CLI commands
**Scale/Scope**: State stores up to 10,000 entries per backend instance

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Determinism | PASS | Same state entries produce same serialized output regardless of backend |
| II. Idempotency | PASS | Apply twice with no changes produces no mutations — verified per-backend |
| III. Portability | PASS | State backend is abstracted behind `Backend` interface; platform-specific code isolated per backend |
| IV. Separation of Concerns | PASS | `state` block in AST/IR is configuration; semantic state operations remain in `internal/state` |
| V. Reproducibility | PASS | State entries include content hashes for drift detection |
| VI. Safe Defaults | PASS | Backend credentials use env var interpolation, never plaintext. Local JSON is the safe default |
| VII. Minimal Surface Area | PASS | One new keyword (`state`) justified by pluggable backend use case |
| VIII. English-Friendly Syntax | PASS | `state "production" { type "postgres" dsn "${PG_DSN}" }` is readable |
| IX. Canonical Formatting | PASS | Formatter extended to handle `state` block |
| X. Strict Validation | PASS | Backend type validated; connection parameters validated at startup |
| XI. Explicit References | N/A | No external imports involved |
| XII. No Hidden Behavior | PASS | Backend selection is explicit in config or CLI flag |
| Drift Detection | NOTE | Constitution says "MUST report drift and MUST NOT silently reconcile". CLI `plan`/`apply` will report drift explicitly. Operator reconciliation is an opt-in K8s-native mode, not silent — it runs as a separate controller the user explicitly deploys. |
| Pre-Commit Validation | PASS | All lint, format, build, test gates apply |

## Project Structure

### Documentation (this feature)

```text
specs/015-distributed-state-reconciliation/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── backend-interface.md
│   ├── state-block-grammar.md
│   └── cli-commands.md
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── state/
│   ├── state.go              # Backend interface (existing, extended)
│   ├── local.go              # LocalBackend (existing)
│   ├── local_test.go         # (existing)
│   ├── etcd.go               # EtcdBackend (new)
│   ├── etcd_test.go          # (new)
│   ├── postgres.go           # PostgresBackend (new)
│   ├── postgres_test.go      # (new)
│   ├── s3.go                 # S3Backend (new)
│   ├── s3_test.go            # (new)
│   ├── kubernetes.go         # KubernetesBackend (new)
│   ├── kubernetes_test.go    # (new)
│   ├── registry.go           # Backend registry and factory (new)
│   ├── registry_test.go      # (new)
│   ├── migrate.go            # Migration logic (new)
│   └── migrate_test.go       # (new)
├── ast/
│   └── ast.go                # StateConfig node (extended)
├── ir/
│   └── ir.go                 # StateConfig in Document (extended)
├── parser/
│   └── parser.go             # Parse `state` block (extended)
├── formatter/
│   └── formatter.go          # Format `state` block (extended)
├── lowering/
│   └── lower.go              # Lower StateConfig with env var resolution (extended)
├── api/v1alpha1/
│   └── statestore_types.go   # StateStore CRD types (new)
├── operator/controller/
│   └── statestore_controller.go  # StateStore reconciler (new)
│
cmd/agentspec/
├── main.go                   # Add --state-backend flag (extended)
├── apply.go                  # Use backend registry (extended)
├── state_cmd.go              # `agentspec state status/migrate` commands (new)

config/crd/bases/
└── agentspec.io_statestores.yaml  # StateStore CRD manifest (new)

integration_tests/
└── state_backend_test.go     # Backend parity integration tests (new)
```

**Structure Decision**: All new backend implementations go in `internal/state/` alongside the existing `local.go`. This keeps the backend abstraction self-contained. The new CLI commands are in `cmd/agentspec/state_cmd.go` following the existing pattern.
