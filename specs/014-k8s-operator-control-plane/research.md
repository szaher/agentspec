# Research: Kubernetes Operator and Control Plane

**Feature**: 014-k8s-operator-control-plane
**Date**: 2026-03-21

## 1. Operator Framework Selection

**Decision**: Use controller-runtime (sigs.k8s.io/controller-runtime) directly, not full Kubebuilder scaffolding.

**Rationale**: The project is an existing Go monorepo. Kubebuilder's scaffolding assumes a standalone project. Using controller-runtime directly lets us integrate into the existing `cmd/agentspec/` and `internal/` structure while still getting the manager, reconciler, envtest, and metrics infrastructure.

**Alternatives considered**:
- Full Kubebuilder scaffold: Too opinionated for an existing monorepo; would create parallel directory structures.
- Operator SDK: Adds another layer on top of controller-runtime; unnecessary complexity for our needs.
- Client-go only: Too low-level; would require reimplementing work queues, caching, leader election, and metrics.

## 2. Project Structure in Monorepo

**Decision**: Place API types in `internal/api/v1alpha1/`, controllers in `internal/operator/controller/`, operator entrypoint as a subcommand in `cmd/agentspec/operator.go` (not a separate binary).

**Rationale**: Single binary keeps deployment simple. The `agentspec operator start` command launches the controller manager. API types under `internal/` since external consumers aren't expected to import them directly (CRD YAML is the interface). Share the root `go.mod`.

**Alternatives considered**:
- Separate `cmd/operator/` binary: Simpler isolation but adds deployment complexity; single binary preferred for CLI+operator cohesion.
- API types in top-level `api/`: Convention for importable types, but our types are internal to this project.

## 3. API Versioning

**Decision**: Start with `v1alpha1` in API group `agentspec.io`.

**Rationale**: Universal convention for new operators. No stability guarantees needed during initial development. Allows free iteration on CRD schemas. Migration to v1beta1 once schemas stabilize, with conversion webhooks using hub-and-spoke model.

**Alternatives considered**:
- Starting at v1beta1: Premature stability commitment.
- Starting at v1: Would lock schema before we have production feedback.

## 4. CRD Scoping

**Decision**: All CRDs namespace-scoped except MemoryClass (cluster-scoped). Policy split into two CRDs: Policy (namespaced) + ClusterPolicy (cluster-scoped).

**Rationale**: Namespace-scoped is the Kubernetes default and enables namespace-per-tenant isolation. MemoryClass is infrastructure-level (like StorageClass) so cluster-scoped. Policy needs both levels — the Kyverno/Istio pattern of separate Policy/ClusterPolicy CRDs is the established convention.

**Alternatives considered**:
- Single Policy CRD with scope field: Mixing scopes in one CRD is not supported by Kubernetes.
- Everything namespace-scoped: Forces MemoryClass duplication per namespace.

## 5. Cross-Namespace References

**Decision**: Same-namespace references only for v1alpha1. Cluster-scoped resources (MemoryClass, ClusterPolicy) referenced by name only.

**Rationale**: Simplest starting point. Cross-namespace references add complexity (ReferenceGrant pattern from Gateway API). Can be added in v1alpha2+ if demand materializes.

**Alternatives considered**:
- ReferenceGrant from day one: Over-engineering for initial release.
- Allow arbitrary cross-namespace refs: Security risk without proper authorization model.

## 6. CRD Generation

**Decision**: Use `controller-gen` for CRD YAML generation from Go type definitions with kubebuilder markers.

**Rationale**: Universal standard. Go types with markers are the single source of truth. Generated OpenAPI v3 schemas, RBAC roles, and webhook configs. Commit generated YAML for review but never hand-edit.

**Alternatives considered**:
- Hand-written CRD YAML: Error-prone, especially for OpenAPI validation schemas. Falls out of sync with Go types.

## 7. Status and Conditions

**Decision**: Use `[]metav1.Condition` as source of truth. Derive a convenience `.status.phase` field deterministically from conditions.

**Rationale**: Standard Kubernetes API convention. `meta.SetStatusCondition()` handles deduplication and `LastTransitionTime`. Conditions are additive and composable. Phase is a derived convenience for `kubectl get` output.

**Standard condition types**: `Ready`, `Reconciling`, `Degraded`, `Available`, `Progressing`.

**Alternatives considered**:
- Phase-only status: Insufficient for expressing multiple concurrent states (e.g., running but degraded).
- Custom condition types: Reinvents what apimachinery already provides.

## 8. DAG Execution for Workflows

**Decision**: Hand-roll a lightweight DAG executor (~100 lines) using Kahn's algorithm for topological sort with goroutine-per-ready-node execution.

**Rationale**: Need result/output passing between steps, which most DAG libraries don't support. The core algorithm is simple. Avoids external dependency for a critical path. Full control over error propagation, context cancellation, and timeout handling.

**Key patterns adopted from Argo/Tekton**:
- Named-dependency model (steps declare `dependsOn` by name)
- Output passing via a shared `map[string]interface{}` results store keyed by step name
- Fail-fast default with optional fail-slow override
- `finally` steps that always run regardless of pipeline success/failure

**Alternatives considered**:
- `natessilva/dag`: Good for execution ordering but lacks result passing.
- `heimdalr/dag`: Supports data propagation but adds external dependency for a small amount of code.
- Argo Workflows as dependency: Far too heavy; we need an in-process executor, not a separate workflow engine.

## 9. Cron Scheduling

**Decision**: Use `robfig/cron/v3` inside the operator reconciler with requeue-based scheduling.

**Rationale**: Full programmatic control. The reconciler computes the next scheduled time, compares to now, and either creates a Task or requeues with `RequeueAfter: timeUntilNext`. This integrates naturally with the controller-runtime reconciliation model.

**Missed schedule handling**: On controller restart, check `status.lastScheduleTime` against the cron expression. Run missed schedules within a configurable deadline window; skip if too many missed.

**Concurrency policies**: `Allow` (concurrent runs OK), `Forbid` (skip if previous running), `Replace` (cancel previous, start new).

**Alternatives considered**:
- Kubernetes CronJob: Creates new Pods each run, limited control over coordination with operator state.
- Persistent cron goroutine: Doesn't survive operator restart; harder to make HA with leader election.

## 10. Testing Strategy

**Decision**: Three-tier testing — unit tests (pure Go logic), envtest integration tests (controller reconciliation), optional Kind e2e tests.

**Rationale**: envtest provides a real etcd + kube-apiserver without kubelet. Sufficient for testing reconciliation loops, status updates, cross-resource validation, and webhook behavior. Use standard Go testing (not Ginkgo) to match project conventions.

**envtest caveats**:
- Namespace deletion doesn't work (namespaces stay in Terminating). Create unique namespaces per test.
- No garbage collection. Test ownership assertions, not automatic deletion.
- Use `Eventually()` from Gomega for all async assertions.
- Re-fetch objects after status updates due to optimistic concurrency.

**Alternatives considered**:
- Fake client only: Controller-runtime explicitly recommends against this — leads to "poorly-written impressions of a real API server."
- Kind for all tests: Too slow for CI; reserve for full e2e validation.

## 11. Leader Election

**Decision**: Enable leader election via `manager.Options` with standard configuration.

**Configuration**: `LeaderElection: true`, `LeaderElectionID: "agentspec-operator.szaher.github.io"`, `LeaseDuration: 15s`, `RenewDeadline: 10s`, `RetryPeriod: 2s`, `LeaderElectionReleaseOnCancel: true`.

**Rationale**: Standard controller-runtime defaults. Lease-based election allows fast failover. Binary must exit immediately after manager stops to prevent split-brain.

## 12. Metrics

**Decision**: Use controller-runtime's built-in metrics (reconcile duration, queue depth, errors, API server latency) plus custom business metrics registered on the controller-runtime metrics registry.

**Built-in metrics**: `controller_runtime_reconcile_total`, `controller_runtime_reconcile_time_seconds`, `workqueue_depth`, `rest_client_requests_total`, plus Go runtime metrics.

**Custom metrics**: Register on `metrics.Registry` (not Prometheus global registry). Define in `internal/operator/metrics/`. Update inside `Reconcile()`.

**Rationale**: Controller-runtime exposes comprehensive operational metrics by default. Custom metrics add business-level observability (e.g., agents_ready, workflows_completed, policy_violations_total).

## 13. Owner References and Finalizers

**Decision**: Use owner references for in-cluster parent-child relationships (Agent → Session, Agent → Task, Workflow → Task steps). Use finalizers only for external resource cleanup or cross-scope dependencies.

**Rationale**: Owner references provide automatic garbage collection. Finalizers are only needed when the controller must coordinate cleanup that Kubernetes GC cannot handle (e.g., cleaning up external resources, deregistering from external services).

**Key rules**:
- Owner and dependent must be in the same namespace (or owner must be cluster-scoped).
- Finalizer names use domain-qualified format: `agentspec.io/<resource>-cleanup`.
- Add finalizers during first reconciliation, not via webhook.
- Finalizer cleanup must be idempotent.
