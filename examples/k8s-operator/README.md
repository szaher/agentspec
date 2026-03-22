# Kubernetes Operator Example

A DevOps assistant agent deployed to Kubernetes via the AgentSpec operator. This example demonstrates CRD generation from IntentLang, operator-driven reconciliation, and dual deployment modes (direct apply vs. operator-managed).

## What's included

| Component | Purpose |
|-----------|---------|
| `devops-assistant` agent | Kubernetes cluster operations assistant with kubectl access |
| `kubectl-get` skill | Command tool that queries cluster resources via `kubectl` |
| `check-pods` skill | Inline bash tool that inspects pod status and recent events |
| `kubernetes` deploy block | Targets a Kubernetes cluster with 2 replicas, health checks, and the `agents` namespace |

The agent uses `claude-sonnet-4-20250514` with a ReAct strategy and a 10-turn limit. It includes a validation rule to prevent leaking secrets or kubeconfig data in responses.

## Prerequisites

1. `kubectl` installed and configured for your cluster
2. A running Kubernetes cluster (kind, minikube, or cloud-hosted)
3. The `agentspec` CLI built from the repo root:

```bash
go build -o agentspec ./cmd/agentspec
```

4. An `ANTHROPIC_API_KEY` environment variable set (for the agent's LLM calls)

## Deployment modes

AgentSpec supports two ways to deploy agents to Kubernetes. Choose the one that fits your workflow.

### Mode 1: Operator-managed (recommended for production)

Generate CRD manifests from the IntentLang file, install the operator, and let it reconcile the desired state.

**Generate CRDs:**

```bash
./agentspec generate crds examples/k8s-operator/k8s-operator.ias -o /tmp/k8s-crds
```

This produces YAML manifests for the `Agent` and `ToolBinding` custom resources in `/tmp/k8s-crds/`.

**Build and deploy the operator to a kind cluster:**

```bash
# Create a kind cluster (if you don't already have one)
kind create cluster --name agentspec

# Build the operator image
docker build -t agentspec-operator:latest -f Dockerfile.operator .

# Load the image into kind
kind load docker-image agentspec-operator:latest --name agentspec

# Deploy the operator
kubectl apply -f config/manager/
```

**Apply the generated CRDs:**

```bash
kubectl apply --server-side -f /tmp/k8s-crds
```

**Create the API key secret:**

```bash
kubectl create secret generic anthropic-api-key \
  --namespace agents \
  --from-literal=ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY"
```

**Verify the resources:**

```bash
kubectl get agents,toolbindings -n agents
# NAME                            MODEL                      PHASE   AGE
# agent/devops-assistant          claude-sonnet-4-20250514   Ready   45s
#
# NAME                            TYPE      AGE
# toolbinding/kubectl-get         command   45s
# toolbinding/check-pods          inline    45s
```

### Mode 2: Direct deploy (quick iteration)

Apply the IntentLang file directly to the cluster without installing the operator. AgentSpec generates and applies the manifests in one step.

```bash
./agentspec apply examples/k8s-operator/k8s-operator.ias \
  --target kubernetes \
  --auto-approve
```

This creates the namespace, deploys the agent pods, and configures the service. It is useful for development and testing but does not provide operator reconciliation (no self-healing, no drift detection).

### Mode selection summary

| Mode | Command | Reconciliation | Best for |
|------|---------|----------------|----------|
| Operator | `generate crds` + `kubectl apply` | Yes (self-healing, drift detection) | Production, multi-team |
| Direct | `apply --target kubernetes` | No (one-shot) | Development, quick tests |
| Auto | `apply --target auto` | Operator if installed, direct otherwise | CI/CD pipelines |

## Querying the agent

Once the agent is running, forward the service port and send requests:

```bash
# Port-forward the agent service
kubectl port-forward -n agents svc/devops-assistant 8080:8080

# Send a query
curl -s http://localhost:8080/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What pods are running in the kube-system namespace?"}' | jq .
```

## Validate and plan

Before deploying, you can validate the spec and preview planned changes:

```bash
# Validate the IntentLang file
./agentspec validate examples/k8s-operator/k8s-operator.ias

# Show what would be created
./agentspec plan examples/k8s-operator/k8s-operator.ias
```

## Customization

To change the target namespace, replica count, or health check settings, edit the `deploy` block in `k8s-operator.ias`:

```
deploy "kubernetes" target "kubernetes" {
  namespace "my-namespace"
  replicas 3
  port 9090
  health {
    path "/healthz"
    interval 15
    timeout 3
  }
}
```
