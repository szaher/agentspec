# Feature Specification: Kubernetes Operator and Control Plane

**Feature Branch**: `014-k8s-operator-control-plane`
**Created**: 2026-03-21
**Status**: Draft
**Input**: Introduce an AgentSpec operator with CRDs (Agent, Task, Session, Workflow, MemoryClass, ToolBinding, Policy, Schedule, EvalRun, Release) to replace simple manifest generation with reconciliation-driven lifecycle management.

## User Scenarios & Testing

### User Story 1 - Deploy Agents via Custom Resources (Priority: P1)

A platform engineer writes an Agent custom resource manifest and applies it to a Kubernetes cluster. The operator reconciles the desired state, provisions the agent runtime, and reports readiness via the resource status. The engineer can inspect, update, and delete agents using standard `kubectl` commands.

**Why this priority**: This is the foundational capability — without the operator reconciling Agent CRDs, no other resource types function. It proves the core reconciliation loop and replaces static manifest generation with a living control plane.

**Independent Test**: Can be fully tested by installing the operator, applying an Agent CR, and verifying the agent reaches a `Ready` status with correct configuration reflected in the status subresource.

**Acceptance Scenarios**:

1. **Given** a Kubernetes cluster with the operator installed, **When** a user applies a valid Agent CR, **Then** the operator provisions the agent runtime and the Agent resource status transitions to `Ready` within a defined time window.
2. **Given** a running Agent resource, **When** the user updates the Agent CR (e.g., changes the model or prompt reference), **Then** the operator reconciles the change and the agent reflects the updated configuration without downtime.
3. **Given** a running Agent resource, **When** the user deletes the Agent CR, **Then** the operator cleans up all associated resources (pods, services, secrets) and the Agent resource is removed.
4. **Given** an Agent CR with an invalid configuration (e.g., referencing a non-existent prompt), **When** the user applies it, **Then** the operator sets the status to `Failed` with a human-readable error message and does not provision any resources.

---

### User Story 2 - Compose Multi-Agent Workflows (Priority: P2)

A platform engineer defines a Workflow custom resource that orchestrates multiple agents with dependency ordering. The operator schedules agents according to the declared step dependencies, passes outputs between steps, and tracks overall workflow progress in the Workflow status.

**Why this priority**: Workflows unlock the multi-agent orchestration use case, which is the primary value proposition beyond single-agent deployment. This builds directly on the Agent CRD from US1.

**Independent Test**: Can be tested by creating Agent CRs for each step agent, applying a Workflow CR, and verifying that steps execute in dependency order with the Workflow reaching `Completed` status.

**Acceptance Scenarios**:

1. **Given** multiple Agent CRs exist, **When** a user applies a Workflow CR referencing those agents with step dependencies, **Then** the operator executes steps in topological order and the Workflow status reports progress per step.
2. **Given** a running Workflow, **When** an intermediate step fails, **Then** the operator halts dependent steps, marks the Workflow as `Failed`, and reports which step caused the failure.
3. **Given** a completed Workflow, **When** the user inspects the Workflow status, **Then** the status includes per-step results, execution durations, and the final output.

---

### User Story 3 - Manage Sessions and Conversation Memory (Priority: P3)

An application developer configures a MemoryClass custom resource that defines how conversation memory is stored and retained. When a Session resource is created for an agent, the operator provisions the backing memory store according to the MemoryClass and attaches it to the agent runtime.

**Why this priority**: Stateful conversations are essential for production agent deployments but require the Agent CRD (US1) to function. MemoryClass allows cluster-wide memory policies.

**Independent Test**: Can be tested by creating a MemoryClass CR, creating a Session CR bound to an Agent, and verifying that conversation history persists across agent interactions within that session.

**Acceptance Scenarios**:

1. **Given** a MemoryClass CR with a sliding-window strategy and a max message count, **When** a Session CR is created for an Agent, **Then** the operator provisions the memory backend and the session enforces the retention policy.
2. **Given** an active Session, **When** messages exceed the MemoryClass retention limit, **Then** older messages are evicted according to the configured strategy.
3. **Given** a Session CR is deleted, **When** the operator reconciles, **Then** the associated memory data is cleaned up according to the MemoryClass data-retention policy.

---

### User Story 4 - Bind Tools and External Services (Priority: P4)

A platform engineer creates ToolBinding custom resources that declare which tools (CLI commands, MCP servers, HTTP endpoints) are available to agents. Agents reference ToolBindings, and the operator validates that the referenced tools are accessible before marking the Agent as ready.

**Why this priority**: Tool access is what makes agents useful beyond conversation. ToolBindings decouple tool configuration from agent definitions, enabling cluster-wide tool governance.

**Independent Test**: Can be tested by creating a ToolBinding CR, referencing it from an Agent CR, and verifying the operator validates tool availability and the agent can invoke the bound tool.

**Acceptance Scenarios**:

1. **Given** a ToolBinding CR referencing an MCP server, **When** an Agent CR references that ToolBinding, **Then** the operator verifies the MCP server is reachable and the Agent status includes the bound tools.
2. **Given** an Agent referencing a ToolBinding, **When** the ToolBinding is deleted, **Then** the operator updates the Agent status to reflect the missing tool dependency and optionally degrades the Agent to a warning state.
3. **Given** a ToolBinding with access restrictions (e.g., namespace-scoped), **When** an Agent in a different namespace attempts to reference it, **Then** the operator rejects the binding and reports a clear authorization error.

---

### User Story 5 - Enforce Policies Across Agents (Priority: P5)

A cluster administrator creates Policy custom resources that define guardrails — cost budgets, allowed models, content filters, rate limits, and permitted tool access. The operator enforces these policies during agent provisioning and at runtime, preventing agents from exceeding their boundaries.

**Why this priority**: Governance is critical for production multi-tenant environments but only becomes meaningful once agents and workflows are operational (US1, US2).

**Independent Test**: Can be tested by creating a Policy CR with a cost budget, deploying an Agent that references it, and verifying the operator enforces the budget constraint.

**Acceptance Scenarios**:

1. **Given** a Policy CR with a cost budget, **When** an Agent CR is created in the policy's scope, **Then** the operator attaches the policy to the agent and the agent's runtime enforces the budget.
2. **Given** a Policy restricting allowed models, **When** a user attempts to deploy an Agent using a disallowed model, **Then** the operator rejects the Agent CR with a policy-violation error in the status.
3. **Given** multiple Policies with overlapping scopes, **When** they apply to the same Agent, **Then** the operator merges policies using a most-restrictive-wins strategy and reports the effective policy in the Agent status.

---

### User Story 6 - Schedule and Automate Agent Operations (Priority: P6)

A platform engineer creates Schedule custom resources that trigger agent or workflow executions at specified times or intervals. The operator creates the necessary Task resources at the scheduled times and tracks their outcomes.

**Why this priority**: Scheduled operations (batch processing, periodic evaluations, recurring workflows) extend agents beyond request-response patterns. Depends on US1 and US2.

**Independent Test**: Can be tested by creating a Schedule CR with a short interval, and verifying that Task resources are created at the expected times with correct agent references.

**Acceptance Scenarios**:

1. **Given** a Schedule CR with a cron expression, **When** the scheduled time arrives, **Then** the operator creates a Task resource that triggers the referenced agent or workflow.
2. **Given** a Schedule CR, **When** the user suspends it, **Then** no further Tasks are created until the schedule is resumed.
3. **Given** a Schedule with concurrency limits, **When** the previous Task is still running at the next trigger time, **Then** the operator skips the new invocation and records the skip in the Schedule status.

---

### User Story 7 - Track Agent Versions and Releases (Priority: P7)

A platform engineer creates Release custom resources that represent versioned snapshots of agent configurations. The operator supports promoting releases across environments (e.g., staging to production) and enables rollback to previous releases.

**Why this priority**: Release management is essential for production-grade operations but is a later-stage concern once the core agent lifecycle is stable (US1-US5).

**Independent Test**: Can be tested by creating a Release CR for an Agent, promoting it, then rolling back and verifying the agent reverts to the previous configuration.

**Acceptance Scenarios**:

1. **Given** a running Agent, **When** a Release CR is created referencing the current Agent configuration, **Then** the operator captures a versioned snapshot of the agent's spec.
2. **Given** a Release CR, **When** it is promoted to a target environment, **Then** the operator applies the release's configuration to the target namespace or cluster.
3. **Given** multiple Release versions exist, **When** a rollback is requested, **Then** the operator reverts the Agent configuration to the specified release version and reports the rollback in the Agent status.

---

### User Story 8 - Run Agent Evaluations (Priority: P8)

A quality engineer creates EvalRun custom resources that execute evaluation suites against agents. The operator runs the evaluation, collects results, and reports metrics (accuracy, latency, cost) in the EvalRun status.

**Why this priority**: Automated evaluation is critical for continuous improvement but requires a fully functional agent deployment pipeline (US1) and optionally workflows (US2).

**Independent Test**: Can be tested by creating an EvalRun CR with test cases, running it against a deployed Agent, and verifying the results are captured in the EvalRun status.

**Acceptance Scenarios**:

1. **Given** a deployed Agent and an EvalRun CR with evaluation criteria, **When** the EvalRun is created, **Then** the operator executes the evaluation and populates the EvalRun status with per-case results and aggregate metrics.
2. **Given** a completed EvalRun, **When** the user inspects the status, **Then** it includes pass/fail per test case, latency metrics, token usage, and an overall score.
3. **Given** a Schedule referencing an EvalRun template, **When** the schedule triggers, **Then** the operator creates periodic EvalRun resources and maintains a history of results for trend analysis.

---

### Edge Cases

- What happens when the operator is restarted mid-reconciliation? The operator MUST resume from the last known state without duplicating resources or losing progress.
- How does the system handle CRD version upgrades? The operator MUST support conversion webhooks for backward-compatible schema migrations.
- What happens when a referenced resource (e.g., a Prompt or ToolBinding) is deleted while an Agent depends on it? The operator MUST detect the broken reference and update the dependent resource's status with a clear error.
- How does the system handle namespace isolation? Resources MUST respect Kubernetes RBAC and namespace boundaries by default. Cross-namespace references MUST be explicitly opt-in.
- What happens when cluster resources are exhausted? The operator MUST report resource constraints in the status and queue pending operations rather than failing silently.
- How does the system handle concurrent modifications to the same resource? The operator MUST use Kubernetes optimistic concurrency (resource versions) to prevent lost updates.

## Requirements

### Functional Requirements

- **FR-001**: System MUST provide a Kubernetes operator that watches and reconciles custom resources for all 11 CRD types (Agent, Task, Session, Workflow, MemoryClass, ToolBinding, Policy, ClusterPolicy, Schedule, EvalRun, Release).
- **FR-002**: System MUST register CRDs with proper OpenAPI v3 validation schemas during operator installation.
- **FR-003**: System MUST implement a reconciliation loop for each CRD that converges toward the declared desired state.
- **FR-004**: System MUST report resource status via the status subresource with human-readable conditions and machine-parseable phase fields.
- **FR-005**: System MUST support finalizers to ensure clean resource deletion and dependent resource cleanup.
- **FR-006**: System MUST validate cross-resource references (e.g., Agent referencing a Prompt or ToolBinding) and report broken references as status conditions.
- **FR-007**: System MUST enforce Policy resources during agent provisioning, rejecting configurations that violate active policies.
- **FR-008**: System MUST support Workflow execution with topological ordering of steps based on declared dependencies.
- **FR-009**: System MUST manage Session lifecycle including memory provisioning according to the referenced MemoryClass.
- **FR-010**: System MUST execute Schedule-triggered operations (Task creation) with cron-expression-based timing and concurrency controls.
- **FR-011**: System MUST capture versioned snapshots of agent configurations in Release resources and support rollback.
- **FR-012**: System MUST run agent evaluations via EvalRun resources and report results in the status subresource.
- **FR-013**: System MUST support leader election for high availability in multi-replica operator deployments.
- **FR-014**: System MUST emit Kubernetes events for significant state transitions (provisioning, ready, failed, deleted) on all managed resources, produce structured logs with correlation IDs for all reconciliation operations, and expose a Prometheus-compatible metrics endpoint for operator health and reconciliation performance.
- **FR-015**: System MUST enforce a namespace-per-tenant isolation model where each tenant operates in a dedicated namespace with Kubernetes RBAC boundaries. Cross-namespace resource references MUST be explicitly opt-in.
- **FR-016**: System MUST handle operator restarts gracefully by resuming reconciliation from the last known resource state.
- **FR-017**: System MUST compile from IntentLang (.ias) source files to Kubernetes manifests. Agent, workflow, and deploy blocks map to their corresponding CRDs. Prompt blocks map to ConfigMaps (referenced by Agent CRs). Skill blocks map to ToolBinding CRs. Secret blocks map to Kubernetes Secret references. Pipeline blocks map to Workflow CRs with step definitions.

### Key Entities

- **Agent**: Represents a deployed AI agent with model, prompt, skill, and tool references. Core unit of deployment managed by the operator.
- **Task**: Represents a single invocation of an agent — an execution unit with inputs, outputs, and completion status.
- **Session**: Represents a conversation context bound to an agent, linking to a MemoryClass for state management.
- **Workflow**: Orchestrates multiple agents as a DAG of steps with dependency ordering and data passing between steps.
- **MemoryClass**: Cluster-scoped template defining memory storage strategy, retention policy, and backend configuration.
- **ToolBinding**: Declares available tools (command, MCP, HTTP) and their access policies, referenced by agents.
- **Policy**: Defines namespace-scoped governance guardrails — cost budgets, model allowlists, rate limits, content filters — applied to agents within a namespace.
- **ClusterPolicy**: Cluster-scoped variant of Policy that applies governance guardrails across all namespaces. Separate CRD following the Kyverno/Istio convention.
- **Schedule**: Defines time-based triggers (cron expressions) for creating Tasks or starting Workflows automatically.
- **EvalRun**: Represents an evaluation execution with test cases, metrics collection, and result reporting.
- **Release**: Captures a versioned snapshot of an agent's configuration for promotion across environments and rollback.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Platform engineers can deploy a fully functional agent on Kubernetes in under 5 minutes using a single Agent CR manifest and `kubectl apply`.
- **SC-002**: The operator reconciles resource changes within 30 seconds of a CR update under normal cluster load.
- **SC-003**: Multi-agent workflows with up to 10 steps complete with correct dependency ordering and the Workflow status accurately reflects per-step progress.
- **SC-004**: Agent rollback via Release resources restores the previous configuration within 60 seconds with zero manual intervention.
- **SC-005**: Policy violations are detected and reported at admission time — invalid Agent configurations are rejected before any resources are provisioned.
- **SC-006**: The operator handles up to 500 total custom resources across all types without reconciliation backlogs or degraded performance.
- **SC-007**: The operator runs stably for 7 days under continuous load without memory leaks, reconciliation backlogs, or unhandled errors.
- **SC-008**: All 11 CRD types pass validation, reconciliation, and cleanup integration tests covering both happy-path and error scenarios.
- **SC-009**: Scheduled operations trigger within 5 seconds of the configured cron time with less than 1% missed triggers over a 24-hour period.

## Clarifications

### Session 2026-03-21

- Q: What is the target scale for the operator in terms of total managed resources? → A: Medium scale — up to 500 total CRs across all types (department-level).
- Q: How should IntentLang source map to Kubernetes CRDs? → A: 1:1 mapping — each IntentLang block becomes one Kubernetes resource. Blocks with corresponding CRDs (agent, workflow, deploy) map directly. Blocks without dedicated CRDs map to native K8s types: prompt → ConfigMap, secret → Secret reference, skill → ToolBinding.
- Q: What is the multi-tenancy isolation model? → A: Namespace-per-tenant — each tenant operates in a dedicated namespace with RBAC boundaries enforcing isolation.
- Q: What observability signals should the operator expose? → A: Standard — Kubernetes events, structured logs with correlation IDs, and a Prometheus metrics endpoint.

## Assumptions

- The target Kubernetes version is 1.28+ with CRD v1 support.
- The operator will initially target a single-cluster deployment model; multi-cluster federation is a future enhancement.
- Agent runtimes are deployed as Kubernetes pods managed by the operator (not external processes).
- The operator reuses the existing AgentSpec IR as the canonical representation. Each IntentLang block maps 1:1 to a Kubernetes resource: blocks with dedicated CRDs (agent, workflow, deploy) map to their CRD types; blocks without dedicated CRDs map to native Kubernetes types (prompt → ConfigMap, skill → ToolBinding, secret → Secret reference).
- MemoryClass backends initially support in-memory and Redis; additional backends can be added via the existing plugin system.
- The `agentspec` CLI will gain a `generate crds` or equivalent command to bridge IntentLang source files and Kubernetes manifests.
