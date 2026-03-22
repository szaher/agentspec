# Tasks: Kubernetes Operator and Control Plane

**Input**: Design documents from `/specs/014-k8s-operator-control-plane/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Integration tests included per constitution requirement (integration tests are primary quality gate).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, dependency setup, and shared type definitions

- [x] T001 Add controller-runtime, client-go, apimachinery, and controller-tools dependencies to go.mod
- [x] T002 Create API group and version registration in internal/api/v1alpha1/groupversion_info.go with group `agentspec.io` and version `v1alpha1`
- [x] T003 [P] Create shared status condition helpers (SetReady, SetReconciling, SetDegraded, DerivePhase) in internal/operator/status/conditions.go
- [x] T004 [P] Create custom Prometheus metrics registration (agents_total, tasks_total, workflow_duration_seconds, policy_violations_total, schedule_triggers_total, schedule_misses_total, evalrun_score) in internal/operator/metrics/metrics.go
- [x] T005 [P] Create Makefile targets for `make manifests` (controller-gen CRD/RBAC generation) and `make generate` (deepcopy) in Makefile

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: CRD type definitions for ALL 11 types and operator manager setup. MUST complete before any controller can be implemented.

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T006 [P] Define Agent CRD types (AgentSpec, AgentStatus with conditions, phase, boundTools, effectivePolicy) in internal/api/v1alpha1/agent_types.go with kubebuilder markers for validation, status subresource, and printcolumns
- [x] T007 [P] Define Task CRD types (TaskSpec with agentRef/input/parameters/timeout, TaskStatus with phase/output/tokenUsage/startTime/completionTime) in internal/api/v1alpha1/task_types.go
- [x] T008 [P] Define Session CRD types (SessionSpec with agentRef/memoryClassRef, SessionStatus with phase/messageCount/lastActivityTime) in internal/api/v1alpha1/session_types.go
- [x] T009 [P] Define Workflow CRD types (WorkflowSpec with steps/failFast/finally, WorkflowStep with name/agentRef/dependsOn, WorkflowStatus with stepStatuses) in internal/api/v1alpha1/workflow_types.go
- [x] T010 [P] Define MemoryClass CRD types (cluster-scoped, MemoryClassSpec with strategy/maxMessages/ttl/backend/backendConfig, MemoryClassStatus) in internal/api/v1alpha1/memoryclass_types.go with `+kubebuilder:resource:scope=Cluster`
- [x] T011 [P] Define ToolBinding CRD types (ToolBindingSpec with toolType/name/command/mcp/http/accessPolicy, ToolBindingStatus with phase/lastProbeTime/boundAgentCount) in internal/api/v1alpha1/toolbinding_types.go
- [x] T012 [P] Define Policy CRD types (PolicySpec with costBudget/allowedModels/deniedModels/rateLimits/contentFilters/toolRestrictions/targetSelector, PolicyStatus) in internal/api/v1alpha1/policy_types.go
- [x] T013 [P] Define ClusterPolicy CRD types (cluster-scoped, same spec as Policy) in internal/api/v1alpha1/clusterpolicy_types.go with `+kubebuilder:resource:scope=Cluster`
- [x] T014 [P] Define Schedule CRD types (ScheduleSpec with schedule/timezone/targetRef/taskTemplate/concurrencyPolicy/suspend, ScheduleStatus with lastScheduleTime/nextScheduleTime/activeTaskRefs) in internal/api/v1alpha1/schedule_types.go
- [x] T015 [P] Define EvalRun CRD types (EvalRunSpec with agentRef/testCases/parallelism, EvalRunStatus with results/summary/phase) in internal/api/v1alpha1/evalrun_types.go
- [x] T016 [P] Define Release CRD types (ReleaseSpec with agentRef/version/snapshot/notes/promoteTo, ReleaseStatus with phase/promotedAt/supersededBy) in internal/api/v1alpha1/release_types.go
- [x] T017 Run `make generate` to produce zz_generated.deepcopy.go for all types in internal/api/v1alpha1/
- [x] T018 Run `make manifests` to generate CRD YAML in config/crd/bases/ and RBAC manifests in config/rbac/
- [x] T019 Create operator manager setup with leader election, metrics endpoint, health probes, and controller registration in cmd/agentspec/operator.go as `agentspec operator start` subcommand
- [x] T020 Create envtest test suite setup (TestMain with envtest.Environment, manager start/stop, CRD installation) in integration_tests/operator/suite_test.go

**Checkpoint**: All CRD types defined, CRD YAML generated, operator binary builds, envtest suite runs empty

---

## Phase 3: User Story 1 - Deploy Agents via Custom Resources (Priority: P1) MVP

**Goal**: Platform engineers can deploy, update, and delete agents on Kubernetes using Agent CRs with full lifecycle management.

**Independent Test**: Install operator, apply Agent CR, verify `Ready` status, update spec, verify reconciliation, delete and verify cleanup.

### Implementation for User Story 1

- [x] T021 [US1] Implement AgentReconciler with Reconcile loop handling create/update/delete in internal/operator/controller/agent_controller.go: provision agent runtime (create Pod+Service), set status conditions (Ready/Reconciling/Degraded), set finalizer `agentspec.io/agent-cleanup`
- [x] T022 [US1] Implement cross-resource reference validation in AgentReconciler: verify promptRef, skillRefs, toolBindingRefs exist in same namespace; verify memoryClassRef exists as cluster-scoped resource; set BrokenReference condition on failure
- [x] T023 [US1] Implement finalizer cleanup in AgentReconciler: delete owned Pods, Services, Sessions, Tasks on Agent deletion; remove finalizer after cleanup
- [x] T024 [US1] Implement Kubernetes event emission in AgentReconciler: emit Provisioning, Ready, ReconcileError, BrokenReference events using record.EventRecorder
- [x] T025 [US1] Update custom metrics in AgentReconciler: increment/decrement agentspec_agents_total gauge by phase on each reconciliation
- [x] T026 [US1] Register AgentReconciler with manager via SetupWithManager in internal/operator/controller/agent_controller.go: watch Agents, own Pods and Services
- [x] T027 [US1] Write integration test for Agent create → Ready lifecycle in integration_tests/operator/agent_test.go: apply Agent CR, assert status transitions to Ready, verify Pod created
- [x] T028 [US1] Write integration test for Agent update → re-reconcile in integration_tests/operator/agent_test.go: update Agent model field, assert status re-converges to Ready
- [x] T029 [US1] Write integration test for Agent delete → cleanup in integration_tests/operator/agent_test.go: delete Agent CR, assert finalizer runs, verify owned resources cleaned up
- [x] T030 [US1] Write integration test for invalid Agent (broken reference) in integration_tests/operator/agent_test.go: apply Agent with non-existent promptRef, assert status shows Failed with BrokenReference condition
- [x] T031 [US1] Create sample Agent CR manifest in config/samples/agent_v1alpha1_sample.yaml

**Checkpoint**: Agents can be deployed, updated, and deleted via kubectl. Status conditions report state accurately. Integration tests pass.

---

## Phase 4: User Story 2 - Compose Multi-Agent Workflows (Priority: P2)

**Goal**: Platform engineers can define Workflow CRs that orchestrate multiple agents with DAG-based dependency ordering.

**Independent Test**: Create Agent CRs, apply Workflow CR with step dependencies, verify steps execute in order and Workflow reaches Completed.

**Depends on**: US1 (Agent CRD must be functional)

### Implementation for User Story 2

- [x] T032 [US2] Implement DAG executor with topological sort (Kahn's algorithm), goroutine-per-ready-node execution, result passing via shared map, and fail-fast/fail-slow modes in internal/operator/controller/dag.go
- [x] T033 [US2] Implement WorkflowReconciler with Reconcile loop in internal/operator/controller/workflow_controller.go: validate DAG (cycle detection), create Task CRs for each step, track per-step status, set Workflow phase (Pending/Running/Completed/Failed)
- [x] T034 [US2] Implement TaskReconciler with Reconcile loop in internal/operator/controller/task_controller.go: invoke agent, capture output/tokenUsage, set phase (Pending/Running/Completed/Failed/TimedOut), handle timeout
- [x] T035 [US2] Implement `finally` steps execution in WorkflowReconciler: always run regardless of DAG success/failure
- [x] T036 [US2] Register WorkflowReconciler and TaskReconciler with manager in their respective controller files
- [x] T037 [US2] Write integration test for Workflow DAG execution in integration_tests/operator/workflow_test.go: create 3-step workflow with dependencies, assert steps run in correct order, verify Completed status
- [x] T038 [US2] Write integration test for Workflow failure handling in integration_tests/operator/workflow_test.go: create workflow where middle step fails, assert dependent steps not started, verify Failed status with failing step identified
- [x] T039 [US2] Create sample Workflow CR manifest in config/samples/workflow_v1alpha1_sample.yaml

**Checkpoint**: Workflows execute step DAGs correctly. Failure handling works. Integration tests pass.

---

## Phase 5: User Story 3 - Manage Sessions and Conversation Memory (Priority: P3)

**Goal**: Application developers can create MemoryClass and Session CRs to manage stateful conversation memory for agents.

**Independent Test**: Create MemoryClass CR, create Session CR for an Agent, verify memory provisioning and retention enforcement.

**Depends on**: US1 (Agent CRD must be functional)

### Implementation for User Story 3

- [x] T040 [P] [US3] Implement MemoryClassReconciler in internal/operator/controller/memoryclass_controller.go: validate strategy/backend, track sessionCount in status, set conditions
- [x] T041 [US3] Implement SessionReconciler in internal/operator/controller/session_controller.go: validate agentRef and memoryClassRef, provision memory backend (in-memory or Redis per MemoryClass), set owner reference to Agent, manage lifecycle (Active/Expired/Terminated), enforce retention (sliding_window eviction)
- [x] T042 [US3] Register MemoryClassReconciler and SessionReconciler with manager
- [x] T043 [US3] Write integration test for Session lifecycle in integration_tests/operator/session_test.go: create MemoryClass, create Agent, create Session, verify Active status and memory provisioning
- [x] T044 [US3] Create sample MemoryClass and Session CR manifests in config/samples/memoryclass_v1alpha1_sample.yaml and config/samples/session_v1alpha1_sample.yaml

**Checkpoint**: Sessions are created with correct memory backends. MemoryClass retention enforced. Integration tests pass.

---

## Phase 6: User Story 4 - Bind Tools and External Services (Priority: P4)

**Goal**: Platform engineers can create ToolBinding CRs to declare tools and bind them to agents with availability validation.

**Independent Test**: Create ToolBinding CR, reference from Agent CR, verify operator validates tool availability and Agent status includes bound tools.

**Depends on**: US1 (Agent CRD must be functional)

### Implementation for User Story 4

- [x] T045 [US4] Implement ToolBindingReconciler in internal/operator/controller/toolbinding_controller.go: validate tool type (command/mcp/http), probe tool availability, set phase (Available/Unavailable/Degraded), track boundAgentCount, emit events
- [x] T046 [US4] Update AgentReconciler to resolve toolBindingRefs: fetch ToolBindings, validate accessPolicy (namespace restrictions), populate status.boundTools, set Degraded condition if ToolBinding deleted
- [x] T047 [US4] Register ToolBindingReconciler with manager; add watch on ToolBinding changes from AgentReconciler
- [x] T048 [US4] Write integration test for ToolBinding lifecycle in integration_tests/operator/toolbinding_test.go: create ToolBinding, reference from Agent, verify boundTools in Agent status; delete ToolBinding, verify Agent status degrades
- [x] T049 [US4] Create sample ToolBinding CR manifest in config/samples/toolbinding_v1alpha1_sample.yaml

**Checkpoint**: ToolBindings validate availability. Agents reflect bound tools in status. Cross-namespace access blocked. Integration tests pass.

---

## Phase 7: User Story 5 - Enforce Policies Across Agents (Priority: P5)

**Goal**: Cluster administrators can create Policy/ClusterPolicy CRs to enforce governance guardrails on agents.

**Independent Test**: Create Policy CR with cost budget and model allowlist, deploy Agent referencing it, verify policy enforcement.

**Depends on**: US1 (Agent CRD must be functional)

### Implementation for User Story 5

- [x] T050 [P] [US5] Implement PolicyReconciler in internal/operator/controller/policy_controller.go: validate policy spec, match agents via targetSelector, track affectedAgentCount and violationCount, set conditions
- [x] T051 [P] [US5] Implement ClusterPolicyReconciler in internal/operator/controller/clusterpolicy_controller.go: same as PolicyReconciler but cluster-scoped, applies to all namespaces
- [x] T052 [US5] Implement policy merge logic (most-restrictive-wins) in internal/operator/controller/policy_merge.go: merge cost budgets (lowest wins), intersect allowed models, union denied models, take lowest rate limits
- [x] T053 [US5] Update AgentReconciler to enforce policies: resolve policyRef + matching ClusterPolicies, merge, validate agent spec against merged policy, set PolicyViolation condition and reject if non-compliant, populate effectivePolicy in status
- [x] T054 [US5] Implement validating webhook for Agent admission in internal/operator/webhook/agent_webhook.go: reject Agent create/update if it violates active policies at admission time
- [x] T055 [US5] Register PolicyReconciler, ClusterPolicyReconciler, and Agent webhook with manager
- [x] T056 [US5] Write integration test for policy enforcement in integration_tests/operator/policy_test.go: create Policy with model allowlist, deploy Agent with disallowed model, assert PolicyViolation status; deploy Agent with allowed model, assert Ready
- [x] T057 [US5] Write integration test for policy merge in integration_tests/operator/policy_test.go: create two overlapping policies, verify most-restrictive-wins merge, verify effectivePolicy in Agent status
- [x] T058 [US5] Create sample Policy and ClusterPolicy CR manifests in config/samples/policy_v1alpha1_sample.yaml and config/samples/clusterpolicy_v1alpha1_sample.yaml

**Checkpoint**: Policies enforce guardrails on Agents. Policy merging works. Webhook rejects violations at admission. Integration tests pass.

---

## Phase 8: User Story 6 - Schedule and Automate Agent Operations (Priority: P6)

**Goal**: Platform engineers can create Schedule CRs to trigger agent/workflow executions at specified times or intervals.

**Independent Test**: Create Schedule CR with short interval, verify Task resources created at expected times.

**Depends on**: US1 (Agent), US2 (Task/Workflow CRDs must be functional)

### Implementation for User Story 6

- [x] T059 [US6] Add robfig/cron/v3 dependency to go.mod
- [x] T060 [US6] Implement ScheduleReconciler in internal/operator/controller/schedule_controller.go: parse cron expression (robfig/cron), compute next run time, create Task/Workflow/EvalRun at trigger time, handle concurrencyPolicy (Allow/Forbid/Replace), manage suspend flag, track lastScheduleTime/nextScheduleTime/activeTaskRefs, requeue with RequeueAfter, handle missed schedules within startingDeadlineSeconds
- [x] T061 [US6] Update schedule metrics: increment schedule_triggers_total and schedule_misses_total counters
- [x] T062 [US6] Register ScheduleReconciler with manager
- [x] T063 [US6] Write integration test for Schedule triggering in integration_tests/operator/schedule_test.go: create Schedule with short interval, verify Task created, verify Forbid concurrency skips when previous running
- [x] T064 [US6] Create sample Schedule CR manifest in config/samples/schedule_v1alpha1_sample.yaml

**Checkpoint**: Schedules trigger Task creation on time. Concurrency policies enforced. Missed schedules handled. Integration tests pass.

---

## Phase 9: User Story 7 - Track Agent Versions and Releases (Priority: P7)

**Goal**: Platform engineers can create Release CRs to capture versioned agent snapshots and support rollback.

**Independent Test**: Create Release CR for an Agent, promote it, roll back, verify agent reverts.

**Depends on**: US1 (Agent CRD must be functional)

### Implementation for User Story 7

- [x] T065 [US7] Implement ReleaseReconciler in internal/operator/controller/release_controller.go: capture AgentSpec snapshot at creation, validate semver version, set owner reference to Agent, handle promotion (apply snapshot to target namespace), handle rollback (revert Agent spec), manage phase (Created/Promoted/RolledBack/Superseded), mark older releases as Superseded
- [x] T066 [US7] Register ReleaseReconciler with manager
- [x] T067 [US7] Write integration test for Release lifecycle in integration_tests/operator/release_test.go: create Agent, create Release, update Agent, rollback to Release, verify Agent spec reverted
- [x] T068 [US7] Create sample Release CR manifest in config/samples/release_v1alpha1_sample.yaml

**Checkpoint**: Releases capture agent snapshots. Rollback restores previous config. Integration tests pass.

---

## Phase 10: User Story 8 - Run Agent Evaluations (Priority: P8)

**Goal**: Quality engineers can create EvalRun CRs to execute evaluation suites and report results.

**Independent Test**: Create EvalRun CR with test cases, run against Agent, verify results in status.

**Depends on**: US1 (Agent CRD must be functional)

### Implementation for User Story 8

- [x] T069 [US8] Implement EvalRunReconciler in internal/operator/controller/evalrun_controller.go: iterate test cases (with parallelism control), invoke Agent per test case, match output against expected (exact/contains/regex), capture per-case results (passed/actualOutput/latencyMs/tokenUsage), compute summary (total/passed/failed/score), set phase (Pending/Running/Completed/Failed)
- [x] T070 [US8] Update evalrun_score metric gauge on EvalRun completion
- [x] T071 [US8] Register EvalRunReconciler with manager
- [x] T072 [US8] Write integration test for EvalRun execution in integration_tests/operator/evalrun_test.go: create Agent, create EvalRun with test cases, verify results populated in status with score
- [x] T073 [US8] Create sample EvalRun CR manifest in config/samples/evalrun_v1alpha1_sample.yaml

**Checkpoint**: EvalRuns execute test cases and report metrics. Integration tests pass.

---

## Phase 11: CLI Integration - Generate CRDs from IntentLang

**Purpose**: Bridge IntentLang source files to Kubernetes CRD manifests via CLI

- [x] T074 Implement `agentspec generate crds` command in cmd/agentspec/generate.go: parse .ias file, convert each IR block to corresponding CRD manifest (1:1 mapping), write YAML files to --output-dir with --namespace flag
- [x] T075 Implement IR-to-Kubernetes-resource converter in internal/k8s/converter/converter.go: map agent IR → Agent CR, prompt IR → ConfigMap (referenced by Agent via promptRef), skill IR → ToolBinding CR, deploy IR → annotation on Agent CR, secret IR → K8s Secret reference, pipeline IR → Workflow CR with step dependsOn mappings
- [x] T076 Write integration test for `agentspec generate crds` in integration_tests/operator/generate_test.go: parse example .ias file, generate CRDs, validate generated YAML against CRD schemas
- [x] T077 Create operator deployment manifests (Deployment, ServiceAccount, ClusterRoleBinding) in config/manager/manager.yaml

---

## Phase 12: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T078 [P] Add `+kubebuilder:printcolumn` markers to all CRD types for useful kubectl get output (phase, age, key fields) in internal/api/v1alpha1/*_types.go
- [x] T079 [P] Create RBAC ClusterRole manifest with all required permissions per CRD API contract in config/rbac/role.yaml
- [x] T080 [P] Add structured logging with correlation IDs (using log/slog) to all reconcilers — log reconcile start/end, status transitions, errors with resource name/namespace/generation
- [x] T081 Run `gofmt -l .` to verify all new files are formatted
- [x] T082 Run `go build ./...` to verify the project builds cleanly
- [x] T083 Run `go test ./... -count=1` to verify all tests pass
- [x] T084 Run `golangci-lint run ./...` to verify zero lint errors
- [x] T085 Validate all sample CR manifests in config/samples/ can be parsed by kubectl dry-run

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Foundational — MVP
- **US2 (Phase 4)**: Depends on US1 (needs Agent + Task CRDs)
- **US3 (Phase 5)**: Depends on US1 (needs Agent CRD); parallel with US2
- **US4 (Phase 6)**: Depends on US1 (needs Agent CRD); parallel with US2, US3
- **US5 (Phase 7)**: Depends on US1 (needs Agent CRD); parallel with US2-US4
- **US6 (Phase 8)**: Depends on US1 + US2 (needs Task/Workflow CRDs)
- **US7 (Phase 9)**: Depends on US1 (needs Agent CRD); parallel with US2-US5
- **US8 (Phase 10)**: Depends on US1 (needs Agent CRD); parallel with US2-US7
- **CLI Integration (Phase 11)**: Depends on Foundational (CRD types defined)
- **Polish (Phase 12)**: Depends on all story phases being complete

### User Story Dependencies

```
Phase 1: Setup
    ↓
Phase 2: Foundational (ALL CRD types)
    ↓
Phase 3: US1 (Agent) ← MVP
    ↓         ↓              ↓
Phase 4:   Phase 5:       Phase 6:       Phase 7:       Phase 9:    Phase 10:
US2        US3             US4            US5            US7         US8
(Workflow) (Session/Mem)   (ToolBinding)  (Policy)       (Release)   (EvalRun)
    ↓
Phase 8: US6 (Schedule) ← needs US2 for Workflow triggers
    ↓
Phase 11: CLI Integration (parallel with US3-US8)
    ↓
Phase 12: Polish
```

### Parallel Opportunities

- **Phase 2**: All T006-T016 CRD type definitions run in parallel (different files)
- **Phase 3**: T027-T030 integration tests can run in parallel after T021-T026
- **After US1**: US3, US4, US5, US7, US8 can all run in parallel (independent controllers)
- **Phase 11**: CLI integration can run in parallel with US3-US8

---

## Parallel Example: Foundational Phase

```bash
# Launch all CRD type definitions in parallel (T006-T016):
Task: "Define Agent CRD types in internal/api/v1alpha1/agent_types.go"
Task: "Define Task CRD types in internal/api/v1alpha1/task_types.go"
Task: "Define Session CRD types in internal/api/v1alpha1/session_types.go"
Task: "Define Workflow CRD types in internal/api/v1alpha1/workflow_types.go"
Task: "Define MemoryClass CRD types in internal/api/v1alpha1/memoryclass_types.go"
Task: "Define ToolBinding CRD types in internal/api/v1alpha1/toolbinding_types.go"
Task: "Define Policy CRD types in internal/api/v1alpha1/policy_types.go"
Task: "Define ClusterPolicy CRD types in internal/api/v1alpha1/clusterpolicy_types.go"
Task: "Define Schedule CRD types in internal/api/v1alpha1/schedule_types.go"
Task: "Define EvalRun CRD types in internal/api/v1alpha1/evalrun_types.go"
Task: "Define Release CRD types in internal/api/v1alpha1/release_types.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRD types + operator manager)
3. Complete Phase 3: User Story 1 (Agent lifecycle)
4. **STOP and VALIDATE**: Test Agent create/update/delete via kubectl
5. Deploy to a test cluster if ready

### Incremental Delivery

1. Setup + Foundational → CRD types defined, operator builds
2. US1 (Agent) → Deploy/test agents via kubectl (MVP!)
3. US2 (Workflow) + US3 (Session) + US4 (ToolBinding) → Multi-agent + stateful + tools
4. US5 (Policy) → Governance layer
5. US6 (Schedule) → Automation
6. US7 (Release) + US8 (EvalRun) → Production operations
7. CLI Integration → Bridge IntentLang → CRDs
8. Polish → Production-ready

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Constitution requires integration tests as primary quality gate — all story phases include integration tests
