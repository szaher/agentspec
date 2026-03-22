# CRD API Contract: AgentSpec Operator

**API Group**: `agentspec.io`
**API Version**: `v1alpha1`
**Date**: 2026-03-21

## CRD Registration

The operator registers 11 CRDs (10 logical entities, Policy split into Policy + ClusterPolicy):

| Kind | Group | Plural | Scope | Short Names |
|------|-------|--------|-------|-------------|
| Agent | agentspec.io | agents | Namespaced | ag |
| Task | agentspec.io | tasks | Namespaced | at |
| Session | agentspec.io | sessions | Namespaced | as |
| Workflow | agentspec.io | workflows | Namespaced | wf |
| MemoryClass | agentspec.io | memoryclasses | Cluster | mc |
| ToolBinding | agentspec.io | toolbindings | Namespaced | tb |
| Policy | agentspec.io | policies | Namespaced | pol |
| ClusterPolicy | agentspec.io | clusterpolicies | Cluster | cpol |
| Schedule | agentspec.io | schedules | Namespaced | sched |
| EvalRun | agentspec.io | evalruns | Namespaced | eval |
| Release | agentspec.io | releases | Namespaced | rel |

## Status Subresource Contract

All CRDs use the `/status` subresource. Status updates do not trigger spec watches.

### Standard Conditions

All resources report conditions using `[]metav1.Condition` with these standard types:

| Condition Type | Meaning |
|---------------|---------|
| `Ready` | Resource has reached its desired state and is fully operational |
| `Reconciling` | Controller is actively working toward the desired state |
| `Degraded` | Resource is operational but not at full capability |

Each condition includes:
- `type`: Condition name
- `status`: `True`, `False`, or `Unknown`
- `reason`: Machine-readable reason code (PascalCase)
- `message`: Human-readable description
- `lastTransitionTime`: When the condition last changed
- `observedGeneration`: The `.metadata.generation` the condition reflects

### Phase Derivation

Phase is derived deterministically from conditions:
- `Pending`: No conditions set, or `Reconciling=True` and `Ready=False`
- `Running`: Resource-specific (Task, Workflow, EvalRun only)
- `Ready`/`Available`: `Ready=True`
- `Failed`: Any condition with `status=False` and a failure reason
- `Completed`: Resource-specific (Task, Workflow, EvalRun only)

## Kubernetes Events

The operator emits events for significant state transitions:

| Event Type | Reason | Resources | Description |
|-----------|--------|-----------|-------------|
| Normal | `Provisioning` | Agent | Agent runtime provisioning started |
| Normal | `Ready` | Agent | Agent reached ready state |
| Normal | `Reconciled` | All | Resource successfully reconciled |
| Warning | `ReconcileError` | All | Reconciliation failed |
| Warning | `BrokenReference` | Agent | Referenced resource not found |
| Warning | `PolicyViolation` | Agent | Policy check failed |
| Normal | `StepStarted` | Workflow | Workflow step execution started |
| Normal | `StepCompleted` | Workflow | Workflow step completed |
| Warning | `StepFailed` | Workflow | Workflow step failed |
| Normal | `TaskCreated` | Schedule | Scheduled task created |
| Warning | `ScheduleMissed` | Schedule | Scheduled execution was missed |
| Normal | `ReleaseCreated` | Release | Version snapshot captured |
| Normal | `Promoted` | Release | Release promoted to target |
| Normal | `RolledBack` | Release | Agent rolled back to this release |

## Finalizer Contract

| Finalizer Name | Resource | Cleanup Action |
|---------------|----------|----------------|
| `agentspec.io/agent-cleanup` | Agent | Delete owned Sessions, Tasks; deregister from ToolBindings |
| `agentspec.io/workflow-cleanup` | Workflow | Cancel running Tasks, clean up step state |
| `agentspec.io/schedule-cleanup` | Schedule | Cancel pending scheduled Tasks |

## Metrics Contract

Custom metrics exposed on the operator's metrics endpoint (`:8443/metrics`):

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `agentspec_agents_total` | Gauge | `namespace`, `phase` | Total agents by phase |
| `agentspec_tasks_total` | Counter | `namespace`, `result` | Total tasks by result |
| `agentspec_workflow_duration_seconds` | Histogram | `namespace`, `workflow` | Workflow execution duration |
| `agentspec_policy_violations_total` | Counter | `namespace`, `policy` | Policy violation count |
| `agentspec_schedule_triggers_total` | Counter | `namespace`, `schedule` | Scheduled trigger count |
| `agentspec_schedule_misses_total` | Counter | `namespace`, `schedule` | Missed schedule count |
| `agentspec_evalrun_score` | Gauge | `namespace`, `agent` | Latest eval run score |

## CLI Contract

### `agentspec operator start`

Starts the operator controller manager.

**Flags**:
- `--metrics-bind-address` (default: `:8443`) — metrics endpoint
- `--health-probe-bind-address` (default: `:8081`) — health/readiness probes
- `--leader-elect` (default: `true`) — enable leader election
- `--leader-election-id` (default: `agentspec-operator.szaher.github.io`)
- `--namespace` (default: `""` = all namespaces) — restrict to specific namespace

**Exit codes**: 0 = clean shutdown, 1 = startup error, 2 = leader election lost.

### `agentspec generate crds <file.ias>`

Generates Kubernetes CRD manifests from IntentLang source.

**Flags**:
- `--output-dir` (default: `.`) — output directory for generated YAML
- `--namespace` (default: `default`) — target namespace for namespaced resources

**Output**: One YAML file per IntentLang block, following the 1:1 mapping convention.

**Exit codes**: 0 = success, 1 = parse/validation error.

## RBAC Contract

The operator requires a ClusterRole with the following permissions:

```
agentspec.io/agents: get, list, watch, create, update, patch, delete
agentspec.io/agents/status: get, update, patch
agentspec.io/tasks: get, list, watch, create, update, patch, delete
agentspec.io/tasks/status: get, update, patch
agentspec.io/sessions: get, list, watch, create, update, patch, delete
agentspec.io/sessions/status: get, update, patch
agentspec.io/workflows: get, list, watch, create, update, patch, delete
agentspec.io/workflows/status: get, update, patch
agentspec.io/memoryclasses: get, list, watch, create, update, patch, delete
agentspec.io/memoryclasses/status: get, update, patch
agentspec.io/toolbindings: get, list, watch, create, update, patch, delete
agentspec.io/toolbindings/status: get, update, patch
agentspec.io/policies: get, list, watch
agentspec.io/policies/status: get, update, patch
agentspec.io/clusterpolicies: get, list, watch
agentspec.io/clusterpolicies/status: get, update, patch
agentspec.io/schedules: get, list, watch, create, update, patch, delete
agentspec.io/schedules/status: get, update, patch
agentspec.io/evalruns: get, list, watch, create, update, patch, delete
agentspec.io/evalruns/status: get, update, patch
agentspec.io/releases: get, list, watch, create, update, patch, delete
agentspec.io/releases/status: get, update, patch
core/events: create, patch
core/pods: get, list, watch, create, delete
core/services: get, list, watch, create, update, delete
core/secrets: get, list, watch
coordination.k8s.io/leases: get, list, watch, create, update, patch, delete
```
