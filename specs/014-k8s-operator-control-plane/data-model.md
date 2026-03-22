# Data Model: Kubernetes Operator and Control Plane

**Feature**: 014-k8s-operator-control-plane
**Date**: 2026-03-21
**API Group**: `agentspec.io`
**API Version**: `v1alpha1`

## Entity Overview

| Entity | Scope | Owner | Description |
|--------|-------|-------|-------------|
| Agent | Namespace | — | Deployed AI agent with model, prompt, skill, and tool references |
| Task | Namespace | Agent or Workflow | Single agent invocation with inputs, outputs, completion status |
| Session | Namespace | Agent | Conversation context bound to an agent with memory provisioning |
| Workflow | Namespace | — | DAG of agent steps with dependency ordering |
| MemoryClass | Cluster | — | Template for memory storage strategy and retention |
| ToolBinding | Namespace | — | Declares available tools and access policies |
| Policy | Namespace | — | Governance guardrails scoped to a namespace |
| ClusterPolicy | Cluster | — | Governance guardrails applied cluster-wide |
| Schedule | Namespace | — | Time-based triggers for Tasks or Workflows |
| EvalRun | Namespace | Agent or Schedule | Evaluation execution with test cases and results |
| Release | Namespace | Agent | Versioned snapshot of agent configuration |

## IntentLang Block → Kubernetes Resource Mapping

Not all IntentLang blocks have dedicated CRDs. The mapping is:

| IntentLang Block | Kubernetes Resource | Notes |
|-----------------|---------------------|-------|
| `agent` | Agent CRD | Direct 1:1 mapping |
| `prompt` | ConfigMap | Referenced by Agent via `promptRef` field |
| `skill` | ToolBinding CRD | Skills map to ToolBindings with appropriate tool type |
| `secret` | K8s Secret reference | Agent references existing Secrets via `secretRefs` |
| `deploy` | Annotation on Agent CR | Deployment target metadata stored as Agent annotations |
| `pipeline` | Workflow CRD | Pipeline steps become Workflow steps with `dependsOn` |

## Entity Definitions

### Agent

```
Agent (namespace-scoped)
├── spec
│   ├── model: string (required) — LLM model identifier
│   ├── promptRef: string (optional) — reference to a Prompt resource name
│   ├── strategy: string (optional, default: "react") — agent strategy
│   ├── maxTurns: int (optional, default: 10) — max conversation turns
│   ├── stream: bool (optional, default: false) — enable streaming
│   ├── skillRefs: []string (optional) — references to Skill resource names
│   ├── toolBindingRefs: []string (optional) — references to ToolBinding names
│   ├── memoryClassRef: string (optional) — reference to cluster-scoped MemoryClass
│   ├── policyRef: string (optional) — reference to Policy name (same namespace)
│   └── secretRefs: []SecretRef (optional) — references to K8s Secrets
│       ├── name: string
│       └── key: string
├── status
│   ├── phase: string (Pending | Provisioning | Ready | Failed | Terminating)
│   ├── conditions: []metav1.Condition
│   ├── observedGeneration: int64
│   ├── boundTools: []string — resolved tool names
│   ├── effectivePolicy: string — merged policy summary
│   └── lastReconcileTime: metav1.Time
```

**Validation rules**:
- `model` must be non-empty
- `promptRef`, `skillRefs`, `toolBindingRefs` must reference existing resources in same namespace
- `memoryClassRef` must reference an existing cluster-scoped MemoryClass
- `policyRef` must reference an existing Policy in same namespace

**State transitions**: Pending → Provisioning → Ready | Failed. Updates trigger Provisioning → Ready. Deletion triggers Terminating.

---

### Task

```
Task (namespace-scoped)
├── spec
│   ├── agentRef: string (required) — reference to Agent name
│   ├── input: string (optional) — input message or payload
│   ├── parameters: map[string]string (optional) — key-value parameters
│   └── timeout: duration (optional, default: "5m") — execution timeout
├── status
│   ├── phase: string (Pending | Running | Completed | Failed | TimedOut)
│   ├── conditions: []metav1.Condition
│   ├── output: string — task result
│   ├── startTime: metav1.Time
│   ├── completionTime: metav1.Time
│   ├── tokenUsage: TokenUsage
│   │   ├── inputTokens: int64
│   │   └── outputTokens: int64
│   └── error: string — error message if failed
```

**Owner reference**: Set to owning Agent or Workflow.

**State transitions**: Pending → Running → Completed | Failed | TimedOut.

---

### Session

```
Session (namespace-scoped)
├── spec
│   ├── agentRef: string (required) — reference to Agent name
│   ├── memoryClassRef: string (optional) — override agent's MemoryClass
│   └── metadata: map[string]string (optional) — session metadata
├── status
│   ├── phase: string (Active | Expired | Terminated)
│   ├── conditions: []metav1.Condition
│   ├── messageCount: int32
│   ├── createdAt: metav1.Time
│   └── lastActivityTime: metav1.Time
```

**Owner reference**: Set to owning Agent.

**State transitions**: Active → Expired (TTL) | Terminated (manual deletion).

---

### Workflow

```
Workflow (namespace-scoped)
├── spec
│   ├── steps: []WorkflowStep (required, min: 1)
│   │   ├── name: string (required, unique within workflow)
│   │   ├── agentRef: string (required) — reference to Agent name
│   │   ├── input: string (optional) — static input or template referencing prior step outputs
│   │   ├── dependsOn: []string (optional) — step names this depends on
│   │   └── timeout: duration (optional, default: "5m")
│   ├── failFast: bool (optional, default: true) — halt on first step failure
│   └── finally: []WorkflowStep (optional) — steps that always run
├── status
│   ├── phase: string (Pending | Running | Completed | Failed)
│   ├── conditions: []metav1.Condition
│   ├── stepStatuses: []StepStatus
│   │   ├── name: string
│   │   ├── phase: string
│   │   ├── taskRef: string — reference to created Task
│   │   ├── startTime: metav1.Time
│   │   ├── completionTime: metav1.Time
│   │   └── output: string
│   ├── startTime: metav1.Time
│   └── completionTime: metav1.Time
```

**Validation rules**:
- Steps must form a valid DAG (no cycles)
- All `dependsOn` references must name existing steps within the same workflow
- All `agentRef` values must reference Agents in the same namespace

**State transitions**: Pending → Running → Completed | Failed.

---

### MemoryClass

```
MemoryClass (cluster-scoped)
├── spec
│   ├── strategy: string (required) — "sliding_window" | "summary" | "full"
│   ├── maxMessages: int32 (optional, default: 100) — max messages retained
│   ├── ttl: duration (optional) — session data TTL
│   ├── backend: string (optional, default: "in-memory") — "in-memory" | "redis"
│   └── backendConfig: map[string]string (optional) — backend-specific config
├── status
│   ├── conditions: []metav1.Condition
│   └── sessionCount: int32 — number of sessions using this class
```

**Validation rules**:
- `strategy` must be one of the allowed values
- `maxMessages` must be > 0
- `backend` must be one of the supported backends

---

### ToolBinding

```
ToolBinding (namespace-scoped)
├── spec
│   ├── toolType: string (required) — "command" | "mcp" | "http"
│   ├── name: string (required) — tool name exposed to agents
│   ├── description: string (optional) — tool description
│   ├── command: CommandToolSpec (if toolType=command)
│   │   ├── binary: string
│   │   └── args: []string
│   ├── mcp: MCPToolSpec (if toolType=mcp)
│   │   └── serverRef: string — MCP server endpoint or service name
│   ├── http: HTTPToolSpec (if toolType=http)
│   │   ├── url: string
│   │   └── method: string
│   └── accessPolicy: AccessPolicy (optional)
│       └── allowedNamespaces: []string (optional) — empty = same namespace only
├── status
│   ├── phase: string (Available | Unavailable | Degraded)
│   ├── conditions: []metav1.Condition
│   ├── lastProbeTime: metav1.Time
│   └── boundAgentCount: int32
```

**Validation rules**:
- Exactly one of `command`, `mcp`, `http` must be set (matching `toolType`)
- `name` must be a valid DNS subdomain

---

### Policy / ClusterPolicy

```
Policy (namespace-scoped) / ClusterPolicy (cluster-scoped)
├── spec
│   ├── costBudget: CostBudget (optional)
│   │   ├── maxDailyCost: string (decimal) — max daily spend
│   │   └── currency: string (default: "USD")
│   ├── allowedModels: []string (optional) — model allowlist (empty = all allowed)
│   ├── deniedModels: []string (optional) — model denylist
│   ├── rateLimits: RateLimits (optional)
│   │   ├── requestsPerMinute: int32
│   │   └── tokensPerMinute: int64
│   ├── contentFilters: []ContentFilter (optional)
│   │   ├── type: string — "input" | "output" | "both"
│   │   └── pattern: string — regex pattern to block
│   ├── toolRestrictions: ToolRestrictions (optional)
│   │   ├── allowedTools: []string
│   │   └── deniedTools: []string
│   └── targetSelector: LabelSelector (optional) — which Agents this policy applies to
├── status
│   ├── conditions: []metav1.Condition
│   ├── affectedAgentCount: int32
│   └── violationCount: int64
```

**Merge strategy**: When multiple policies apply to an Agent, use most-restrictive-wins: lowest cost budget, intersection of allowed models, union of denied models, lowest rate limits.

---

### Schedule

```
Schedule (namespace-scoped)
├── spec
│   ├── schedule: string (required) — cron expression
│   ├── timezone: string (optional, default: "UTC")
│   ├── targetRef: TargetRef (required)
│   │   ├── kind: string — "Agent" | "Workflow" | "EvalRun"
│   │   └── name: string
│   ├── taskTemplate: TaskTemplate (optional) — template for created Tasks
│   │   ├── input: string
│   │   └── parameters: map[string]string
│   ├── concurrencyPolicy: string (optional, default: "Forbid") — "Allow" | "Forbid" | "Replace"
│   ├── startingDeadlineSeconds: int64 (optional, default: 100)
│   ├── suspend: bool (optional, default: false)
│   └── successfulTasksHistoryLimit: int32 (optional, default: 3)
├── status
│   ├── conditions: []metav1.Condition
│   ├── lastScheduleTime: metav1.Time
│   ├── nextScheduleTime: metav1.Time
│   ├── activeTaskRefs: []string
│   └── missedScheduleCount: int64
```

**Validation rules**:
- `schedule` must be a valid cron expression
- `targetRef.kind` must be one of the allowed kinds
- `targetRef.name` must reference an existing resource

---

### EvalRun

```
EvalRun (namespace-scoped)
├── spec
│   ├── agentRef: string (required) — reference to Agent name
│   ├── testCases: []EvalTestCase (required, min: 1)
│   │   ├── name: string (required)
│   │   ├── input: string (required)
│   │   ├── expectedOutput: string (optional) — exact match or pattern
│   │   ├── matchType: string (optional, default: "contains") — "exact" | "contains" | "regex"
│   │   └── timeout: duration (optional, default: "30s")
│   └── parallelism: int32 (optional, default: 1) — max concurrent test cases
├── status
│   ├── phase: string (Pending | Running | Completed | Failed)
│   ├── conditions: []metav1.Condition
│   ├── results: []EvalResult
│   │   ├── name: string
│   │   ├── passed: bool
│   │   ├── actualOutput: string
│   │   ├── latencyMs: int64
│   │   └── tokenUsage: TokenUsage
│   ├── summary: EvalSummary
│   │   ├── total: int32
│   │   ├── passed: int32
│   │   ├── failed: int32
│   │   ├── score: string (decimal, 0.0-1.0)
│   │   └── totalTokens: int64
│   ├── startTime: metav1.Time
│   └── completionTime: metav1.Time
```

---

### Release

```
Release (namespace-scoped)
├── spec
│   ├── agentRef: string (required) — reference to Agent name
│   ├── version: string (required) — semantic version tag
│   ├── snapshot: AgentSpec (required) — captured agent spec at release time
│   ├── notes: string (optional) — release notes
│   └── promoteTo: string (optional) — target namespace for promotion
├── status
│   ├── phase: string (Created | Promoted | RolledBack | Superseded)
│   ├── conditions: []metav1.Condition
│   ├── promotedAt: metav1.Time
│   ├── promotedTo: string — namespace where promoted
│   └── supersededBy: string — newer Release name
```

**Owner reference**: Set to owning Agent.

**Validation rules**:
- `version` must be valid semver
- `agentRef` must reference an existing Agent

## Relationship Diagram

```
                    ┌──────────────┐
                    │ ClusterPolicy│ (cluster-scoped)
                    └──────┬───────┘
                           │ applies to
                           ▼
┌─────────────┐    ┌───────────────┐    ┌──────────────┐
│ MemoryClass │◄───│    Agent      │───►│  ToolBinding │
│  (cluster)  │    │               │    │              │
└─────────────┘    └───┬───┬───┬───┘    └──────────────┘
                       │   │   │
              owns ┌───┘   │   └───┐ owns
                   ▼       │       ▼
            ┌──────────┐   │  ┌─────────┐
            │ Session  │   │  │ Release │
            └──────────┘   │  └─────────┘
                           │
                    refs   │
                   ┌───────┘
                   ▼
            ┌──────────┐        ┌──────────┐
            │   Task   │◄───────│ Schedule │
            └──────────┘creates └──────────┘
                   ▲                  │
                   │ creates          │ triggers
            ┌──────┴───┐      ┌──────┴───┐
            │ Workflow  │      │ EvalRun  │
            └──────────┘      └──────────┘

            ┌──────────┐
            │  Policy  │ (namespace-scoped)
            └──────────┘ applies to Agents via selector
```
