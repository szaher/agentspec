# Implementation Plan: Kubernetes Operator and Control Plane

**Branch**: `014-k8s-operator-control-plane` | **Date**: 2026-03-21 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/014-k8s-operator-control-plane/spec.md`

## Summary

Introduce a Kubernetes operator for AgentSpec that manages 10 CRD types (Agent, Task, Session, Workflow, MemoryClass, ToolBinding, Policy, Schedule, EvalRun, Release) through reconciliation-driven lifecycle management. The operator replaces static manifest generation with a living control plane that watches, validates, and converges custom resources toward their declared desired state. Each IntentLang block maps 1:1 to a CRD instance, maintaining the existing IR as the canonical representation. The operator targets medium scale (up to 500 CRs) with namespace-per-tenant isolation.

## Technical Context

**Language/Version**: Go 1.25+ (existing project language)
**Primary Dependencies**: controller-runtime (kubebuilder framework), client-go, apimachinery, cobra v1.10.2 (existing CLI), sigs.k8s.io/controller-tools (CRD generation)
**Storage**: Kubernetes etcd (via CRDs), existing AgentSpec state file (`.agentspec.state.json`) for CLI bridge
**Testing**: Go testing + envtest (controller-runtime test harness for integration tests), go test (unit tests)
**Target Platform**: Kubernetes 1.28+ clusters, Linux containers (operator), cross-platform CLI (existing)
**Project Type**: Kubernetes operator + CLI extension
**Performance Goals**: Reconciliation within 30 seconds of CR update, 500 total CRs without backlogs, scheduled triggers within 5 seconds of cron time
**Constraints**: Single-cluster deployment (multi-cluster deferred), namespace-per-tenant isolation, leader election for HA
**Scale/Scope**: Up to 500 total custom resources across all 10 CRD types (department-level)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Determinism | PASS | CRDs map 1:1 to IR; reconciliation produces deterministic state from spec |
| II. Idempotency | PASS | Reconciliation loops are inherently idempotent — re-reconciling a converged resource produces no mutations |
| III. Portability | PASS | CRDs are a Kubernetes-specific adapter; the DSL and IR remain platform-neutral per adapter contract |
| IV. Separation of Concerns | PASS | CRDs are surface syntax mapping to IR; operator is an adapter, no business logic in CRD definitions |
| V. Reproducibility | PASS | CRD manifests generated from pinned IntentLang source; Release CRD captures versioned snapshots |
| VI. Safe Defaults | PASS | Secrets referenced via K8s Secrets (not plaintext); namespace isolation by default; Policy CRDs enforce least-privilege |
| VII. Minimal Surface Area | PASS | Each CRD maps to an existing IntentLang concept; no new constructs without existing use cases |
| VIII. English-Friendly Syntax | PASS | Users author IntentLang source; CRD YAML is the generated adapter output |
| IX. Canonical Formatting | PASS | IntentLang formatter applies to source; CRD manifests follow K8s conventions |
| X. Strict Validation | PASS | OpenAPI v3 schemas on CRDs + operator-level semantic validation with status conditions |
| XI. Explicit References | PASS | Cross-resource references validated; broken references reported as status conditions |
| XII. No Hidden Behavior | PASS | All reconciliation actions emit K8s events; structured logs with correlation IDs |
| Adapter Contract | PASS | Operator accepts IR, not raw DSL; thin mapping and deployment layer |
| Plugin Contract | PASS | Existing plugin system reused for extensibility (e.g., MemoryClass backends) |
| Pre-Commit Validation | PASS | Existing CI pipeline applies; operator code follows same lint/format/build/test gates |
| Testing Strategy | PASS | Integration tests via envtest; contract tests for CRD schemas; golden fixtures for determinism |

**Gate Result**: PASS — no violations.

## Project Structure

### Documentation (this feature)

```text
specs/014-k8s-operator-control-plane/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (CRD schemas)
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── operator/
│   ├── controller/          # Reconcilers for each CRD type
│   │   ├── agent_controller.go
│   │   ├── task_controller.go
│   │   ├── session_controller.go
│   │   ├── workflow_controller.go
│   │   ├── memoryclass_controller.go
│   │   ├── toolbinding_controller.go
│   │   ├── policy_controller.go
│   │   ├── schedule_controller.go
│   │   ├── evalrun_controller.go
│   │   └── release_controller.go
│   ├── webhook/             # Validation and conversion webhooks
│   │   ├── agent_webhook.go
│   │   └── policy_webhook.go
│   ├── metrics/             # Prometheus metrics registration
│   │   └── metrics.go
│   └── status/              # Shared status/condition helpers
│       └── conditions.go
├── api/
│   └── v1alpha1/            # CRD type definitions
│       ├── agent_types.go
│       ├── task_types.go
│       ├── session_types.go
│       ├── workflow_types.go
│       ├── memoryclass_types.go
│       ├── toolbinding_types.go
│       ├── policy_types.go
│       ├── schedule_types.go
│       ├── evalrun_types.go
│       ├── release_types.go
│       ├── groupversion_info.go
│       └── zz_generated.deepcopy.go
├── k8s/
│   ├── converter/           # IntentLang IR → CRD manifest converter
│   │   └── converter.go
│   └── installer/           # CRD installation helpers
│       └── installer.go

cmd/agentspec/
├── operator.go              # `agentspec operator start` command
└── generate.go              # `agentspec generate crds` command

config/
├── crd/                     # Generated CRD YAML manifests
│   └── bases/
├── rbac/                    # RBAC role/binding manifests
├── manager/                 # Operator deployment manifests
└── samples/                 # Example CR manifests for each type

integration_tests/
└── operator/
    ├── agent_test.go
    ├── workflow_test.go
    ├── policy_test.go
    └── suite_test.go
```

**Structure Decision**: Follows existing project conventions — CRD types under `internal/api/`, controllers under `internal/operator/controller/`, CLI extensions in `cmd/agentspec/`. Uses kubebuilder-style layout adapted to the existing monorepo structure. Integration tests use envtest for in-process API server testing.

## Phases

### Phase 1: Core Operator Foundation (US1)

- CRD type definitions for all 10 types in `internal/api/v1alpha1/`
- Agent reconciler with full lifecycle (create, update, delete, status)
- Finalizers and owner references for dependent resource cleanup
- Leader election and health probes
- `agentspec operator start` CLI command
- `agentspec generate crds` CLI command (IntentLang → CRD manifests)
- CRD installation and RBAC manifests
- Integration tests for Agent lifecycle via envtest
- Prometheus metrics endpoint (reconciliation duration, queue depth, errors)
- Structured logging with correlation IDs

### Phase 2: Workflows and Task Execution (US2)

- Workflow reconciler with DAG execution engine
- Task reconciler for individual agent invocations
- Step dependency resolution and topological ordering
- Output passing between workflow steps
- Workflow status tracking (per-step progress, durations)
- Integration tests for workflow execution

### Phase 3: Sessions, Memory, and Tools (US3, US4)

- MemoryClass reconciler (cluster-scoped)
- Session reconciler with memory provisioning
- ToolBinding reconciler with reachability validation
- Cross-resource reference validation (Agent → ToolBinding, Agent → Session)
- Integration tests for session lifecycle and tool binding

### Phase 4: Governance and Automation (US5, US6)

- Policy reconciler with admission-time enforcement
- Policy merging (most-restrictive-wins)
- Validation webhook for policy enforcement
- Schedule reconciler with cron-based Task creation
- Concurrency controls for scheduled operations
- Integration tests for policy enforcement and scheduling

### Phase 5: Releases and Evaluations (US7, US8)

- Release reconciler with versioned snapshots
- Promotion and rollback operations
- EvalRun reconciler with test execution and result collection
- Integration with Schedule for periodic evaluations
- Integration tests for release management and evaluations

## Complexity Tracking

No constitution violations to justify.

## Post-Design Constitution Re-Check

All gates continue to PASS after Phase 1 design:
- **Policy split** (Policy + ClusterPolicy): Two CRDs for the same logical concept at different scopes — follows Kyverno/Istio convention. Does not introduce new DSL constructs (Principle VII satisfied).
- **controller-gen code generation**: Generated CRD YAML is deterministic from Go types with markers (Principle I satisfied). Generated files are committed for review (Principle XII satisfied).
- **DAG executor**: Hand-rolled in Go, no hidden transforms — all step executions are declared in the Workflow spec and visible as Tasks (Principle XII satisfied).
- **robfig/cron**: Schedule execution is explicit — Tasks are created as visible K8s resources, not hidden background operations (Principle XII satisfied).
