# Kubernetes Operator

The AgentSpec Kubernetes operator provides a native control plane for managing AI agents as first-class Kubernetes resources. Instead of generating raw Deployments and Services, the operator introduces Custom Resource Definitions (CRDs) that model the full agent lifecycle -- from creation through runtime orchestration to teardown.

---

## Architecture Overview

The operator follows the standard Kubernetes controller pattern:

1. **CRDs** define the desired state of agents, workflows, policies, and supporting resources.
2. **Controllers** watch for changes to CRDs and reconcile the cluster toward the desired state.
3. **Owned resources** (ConfigMaps, Deployments, Services) are created and managed automatically, linked to their parent CR via owner references.

The operator runs as a single manager Deployment in the `agentspec-system` namespace. It reconciles all 11 CRD types and handles cross-reference validation, workload provisioning, and cascade deletion.

---

## Custom Resource Definitions

The operator defines 11 CRDs across two scopes:

| CRD | API Group | Scope | Description |
|-----|-----------|-------|-------------|
| **Agent** | `agentspec.io/v1alpha1` | Namespaced | Core agent definition -- model, prompt, tools, and runtime configuration |
| **Task** | `agentspec.io/v1alpha1` | Namespaced | A unit of work assigned to an agent with input/output schema |
| **Session** | `agentspec.io/v1alpha1` | Namespaced | Conversation state and message history for an agent interaction |
| **Workflow** | `agentspec.io/v1alpha1` | Namespaced | DAG-based multi-step pipeline with dependency resolution |
| **ToolBinding** | `agentspec.io/v1alpha1` | Namespaced | Declares an external tool (MCP, HTTP, command) available to agents |
| **Policy** | `agentspec.io/v1alpha1` | Namespaced | Namespace-scoped guardrails -- rate limits, content filters, cost budgets |
| **ClusterPolicy** | `agentspec.io/v1alpha1` | Cluster | Cluster-wide policy enforced across all namespaces |
| **MemoryClass** | `agentspec.io/v1alpha1` | Namespaced | Storage backend configuration for agent conversation memory |
| **Schedule** | `agentspec.io/v1alpha1` | Namespaced | Cron-based trigger for recurring agent tasks |
| **EvalRun** | `agentspec.io/v1alpha1` | Namespaced | Evaluation execution record with metrics and pass/fail results |
| **Release** | `agentspec.io/v1alpha1` | Namespaced | Versioned agent release with rollout strategy and rollback support |

---

## Installation

### Build the operator

```bash
go build -o agentspec-operator ./cmd/operator
```

### Build the container image

```bash
docker build -t agentspec-operator:latest -f Dockerfile.operator .
```

### Load into a local cluster (kind)

```bash
kind load docker-image agentspec-operator:latest
```

### Install CRDs, RBAC, and the manager

```bash
kubectl apply -f config/crd/bases/
kubectl apply -f config/rbac/
kubectl apply -f config/manager/
```

Verify the operator is running:

```bash
kubectl -n agentspec-system get pods
```

---

## Generating CRD Manifests

The `agentspec generate crds` command converts an IntentLang file into Kubernetes CRD manifests:

```bash
agentspec generate crds examples/support-agent.ias
```

This produces YAML manifests for every resource defined in the `.ias` file -- Agent, ToolBinding, Policy, and any other applicable CRDs. The output is written to stdout by default; redirect to a file or pipe directly into `kubectl apply`:

```bash
agentspec generate crds examples/support-agent.ias | kubectl apply -f -
```

### Inline tool mapping

When your IntentLang file contains `tool inline {}` blocks, the generator maps them to `command` type ToolBindings with `sh -c` as the entrypoint. This allows inline scripts to run as sidecar commands in the agent pod.

---

## Using `agentspec apply` with Auto-Detection

The `agentspec apply` command automatically detects whether the AgentSpec operator is installed in the target cluster:

- **Operator present** -- generates and applies CRD manifests (Agent, ToolBinding, Policy, etc.).
- **Operator absent** -- falls back to direct mode, generating raw Deployment, Service, and ConfigMap resources.

```bash
# Auto-detects operator presence and chooses the right mode
agentspec apply examples/support-agent.ias
```

You can force a specific mode with the `--mode` flag:

```bash
agentspec apply examples/support-agent.ias --mode=operator
agentspec apply examples/support-agent.ias --mode=direct
```

---

## Agent CR Lifecycle

When you create an Agent CR, the operator provisions the full workload stack:

1. **ConfigMap** -- contains the agent's prompt, model configuration, and tool references.
2. **Deployment** -- runs the agent runtime container with the ConfigMap mounted.
3. **Service** -- exposes the agent's HTTP API within the cluster.

```
Agent CR created
    |
    v
Operator reconciles
    |
    +---> Creates ConfigMap (agent config)
    +---> Creates Deployment (agent runtime)
    +---> Creates Service (cluster networking)
    |
    v
Agent status transitions: Pending -> Provisioning -> Ready
```

All owned resources carry an `ownerReference` pointing back to the Agent CR. Deleting the Agent CR triggers Kubernetes garbage collection, which removes the ConfigMap, Deployment, and Service automatically.

### Status conditions

The Agent CR reports its status through standard Kubernetes conditions:

```bash
kubectl get agents
```

```
NAME            MODEL          STATUS   AGE
support-agent   claude-sonnet  Ready    5m
```

---

## Workflow Orchestration

The Workflow CRD models multi-step agent pipelines as a directed acyclic graph (DAG):

- **Steps** declare which agent or task to run, with optional `dependsOn` references.
- **DAG validation** runs at admission time -- the operator rejects workflows with circular dependencies.
- **Parallel execution** -- steps with no unmet dependencies run concurrently.
- **Finally steps** -- steps marked with `finally: true` run after all other steps complete, regardless of success or failure. Use these for cleanup, notification, or summary tasks.

```yaml
apiVersion: agentspec.io/v1alpha1
kind: Workflow
metadata:
  name: support-pipeline
spec:
  steps:
    - name: classify
      agentRef: classifier-agent
    - name: respond
      agentRef: support-agent
      dependsOn: [classify]
    - name: log-result
      agentRef: logger-agent
      finally: true
```

---

## Cross-Reference Validation

The operator validates all resource references at reconciliation time:

- **ToolBindings** -- every tool referenced in an Agent spec must have a matching ToolBinding CR in the same namespace.
- **Policies** -- referenced Policy CRs must exist; ClusterPolicy CRs are resolved cluster-wide.
- **MemoryClasses** -- if an Agent references a memory class, the corresponding MemoryClass CR must be present.
- **Secrets** -- Kubernetes Secrets referenced in tool configurations or agent env blocks must exist in the namespace.

If any reference is missing, the Agent CR enters a `Degraded` status with a descriptive message indicating which reference could not be resolved.

---

## Monitoring

### List agents

```bash
kubectl get agents -n default
```

### Inspect an agent

```bash
kubectl describe agent support-agent -n default
```

This shows the full spec, status conditions, and recent events (provisioning, scaling, errors).

### View operator logs

```bash
kubectl logs -n agentspec-system deploy/agentspec-operator-manager -f
```

### Check owned resources

```bash
kubectl get configmap,deployment,service -l agentspec.io/agent=support-agent
```

---

## See Also

- [Kubernetes Deployment](kubernetes.md) -- Direct deployment mode without the operator
- [Best Practices](best-practices.md) -- Production deployment recommendations
