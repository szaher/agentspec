# Kubernetes Deployment

The `kubernetes` target deploys your agent to a Kubernetes cluster with full support for namespaces, replicas, resource limits, health probes, autoscaling, and secret management. AgentSpec generates Kubernetes manifests (Deployment, Service, HorizontalPodAutoscaler) and applies them to your cluster.

---

## Prerequisites

- `kubectl` installed and configured with access to a Kubernetes cluster (`kubectl cluster-info` to verify).
- The `agentspec` CLI binary installed and available on your `PATH`.
- A container registry accessible to the cluster for pulling agent images.

---

## Deploy Block

A Kubernetes deployment specifies the namespace, replicas, resources, health probes, and autoscaling:

```ias novalidate
deploy "production" target "kubernetes" {
  namespace "agents"
  image "agentspec/my-agent:0.1.0"
  replicas 3
  port 8080
  resources {
    cpu "1"
    memory "1Gi"
  }
  health {
    path "/healthz"
    interval "30s"
    timeout "5s"
  }
  autoscale {
    min 3
    max 10
    metric "cpu"
    target 80
  }
  env {
    LOG_LEVEL "warn"
    ENVIRONMENT "production"
  }
  secrets {
    API_KEY "api-key"
  }
}
```

### Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `default` | bool | Mark this as the default deploy target. |
| `namespace` | string | Kubernetes namespace to deploy into. Created if it does not exist. |
| `image` | string | Container image name and tag. Must be pullable by the cluster. |
| `replicas` | int | Number of pod replicas. |
| `port` | int | Port the container listens on. Used in the Service definition. |
| `resources` | block | CPU and memory requests/limits for each pod. |
| `health` | block | Liveness and readiness probe configuration. |
| `autoscale` | block | Horizontal Pod Autoscaler (HPA) configuration. |
| `env` | block | Environment variables set on each pod. |
| `secrets` | block | Secret mappings. Values are stored as Kubernetes Secrets and mounted as environment variables. |

---

## How It Works

When you run `agentspec apply` with a `kubernetes` target, AgentSpec:

1. Generates Kubernetes manifests from the deploy block configuration:
    - **Deployment** -- Defines the pod template, replica count, resource limits, health probes, and environment variables.
    - **Service** -- Exposes the deployment within the cluster on the configured port.
    - **HorizontalPodAutoscaler** -- (If `autoscale` is configured) Creates an HPA resource targeting the deployment.
    - **Secret** -- (If `secrets` is configured) Creates Kubernetes Secret objects from resolved secret values.
2. Creates the namespace if it does not exist.
3. Applies the manifests to the cluster using `kubectl apply`.
4. Waits for the deployment to reach the desired replica count and pass health checks.
5. Records the deployment state in `.agentspec.state.json`.

---

## Namespace

The `namespace` attribute specifies which Kubernetes namespace the resources are deployed into:

```ias novalidate
deploy "production" target "kubernetes" {
  namespace "agents"
}
```

If the namespace does not exist, AgentSpec creates it before applying the manifests. Use namespaces to isolate agent deployments by environment or team:

| Namespace | Purpose |
|-----------|---------|
| `agents-dev` | Development agents |
| `agents-staging` | Staging and integration testing |
| `agents` or `agents-prod` | Production workloads |

---

## Replicas

The `replicas` attribute sets the initial number of pod replicas in the Deployment:

```ias novalidate
deploy "production" target "kubernetes" {
  replicas 3
}
```

If `autoscale` is also configured, the `replicas` value serves as the initial count. The HPA adjusts the actual count between `autoscale.min` and `autoscale.max` based on the configured metric.

---

## Resource Limits

The `resources` block sets CPU and memory requests and limits for each pod:

```ias novalidate
resources {
  cpu "1"
  memory "1Gi"
}
```

AgentSpec sets both the Kubernetes `requests` and `limits` to the same value, ensuring guaranteed Quality of Service (QoS) for your agent pods.

| Attribute | Format | Examples |
|-----------|--------|----------|
| `cpu` | Millicores or whole cores | `"250m"` (0.25 cores), `"500m"` (0.5 cores), `"1"` (1 core), `"2"` (2 cores) |
| `memory` | Mebibytes or gibibytes | `"128Mi"`, `"256Mi"`, `"512Mi"`, `"1Gi"`, `"2Gi"` |

### Sizing Guidelines

| Strategy | Recommended CPU | Recommended Memory |
|----------|-----------------|--------------------|
| `react` | `"500m"` - `"1"` | `"256Mi"` - `"512Mi"` |
| `plan-and-execute` | `"1"` - `"2"` | `"512Mi"` - `"1Gi"` |
| `reflexion` | `"1"` - `"2"` | `"512Mi"` - `"1Gi"` |
| `router` | `"250m"` - `"500m"` | `"128Mi"` - `"256Mi"` |
| `map-reduce` | `"1"` - `"4"` | `"1Gi"` - `"4Gi"` |

---

## Health Probes

The `health` block configures both liveness and readiness probes for the pods:

```ias novalidate
health {
  path "/healthz"
  interval "30s"
  timeout "5s"
}
```

AgentSpec generates Kubernetes probe definitions from the health block:

- **Liveness probe** -- Restarts the pod if the health check fails. Prevents stuck processes from consuming resources.
- **Readiness probe** -- Removes the pod from the Service's endpoint list if the check fails. Prevents traffic from reaching pods that are not ready to serve requests.

Both probes use the same path, interval, and timeout. The liveness probe has a higher failure threshold to avoid premature restarts.

| Attribute | Kubernetes Mapping |
|-----------|-------------------|
| `path` | `httpGet.path` |
| `interval` | `periodSeconds` |
| `timeout` | `timeoutSeconds` |

!!! warning "Always Configure Health Checks"
    Without health probes, Kubernetes cannot detect when an agent pod is unhealthy. Always configure a `health` block for production deployments.

---

## Autoscaling

The `autoscale` block creates a HorizontalPodAutoscaler (HPA) resource that automatically adjusts the number of replicas based on observed metrics:

```ias novalidate
autoscale {
  min 3
  max 10
  metric "cpu"
  target 80
}
```

| Attribute | Description |
|-----------|-------------|
| `min` | Minimum number of replicas. The HPA will never scale below this count. |
| `max` | Maximum number of replicas. The HPA will never scale above this count. |
| `metric` | The metric used to trigger scaling decisions. |
| `target` | Target utilization percentage. Scaling occurs when the average exceeds this value. |

### Supported Metrics

| Metric | Description | When to Use |
|--------|-------------|-------------|
| `"cpu"` | Average CPU utilization across all pods | General-purpose workloads |
| `"memory"` | Average memory utilization across all pods | Memory-intensive agents |
| `"requests"` | Request rate per second | Traffic-driven scaling |

### How Autoscaling Works

1. The HPA monitors the configured metric across all pods in the deployment.
2. When the average utilization exceeds the `target` percentage, the HPA increases the replica count.
3. When utilization drops below the target, the HPA decreases the replica count (respecting `min`).
4. Scaling decisions are made at regular intervals (controlled by the Kubernetes HPA controller).

!!! tip "Setting Target Values"
    A target of `80` means the HPA adds replicas when average CPU or memory usage exceeds 80%. Lower targets (e.g., `60`) provide more headroom for traffic spikes but use more resources. Higher targets (e.g., `90`) are more cost-efficient but leave less room for bursts.

---

## Secret Management in Kubernetes

The `secrets` block maps environment variable names to declared `secret` resources. AgentSpec creates Kubernetes Secret objects and mounts them as environment variables in the pods:

```ias novalidate
secret "api-key" {
  env(API_KEY)
}

secret "db-password" {
  store(production/database/password)
}

deploy "production" target "kubernetes" {
  namespace "agents"
  secrets {
    API_KEY "api-key"
    DB_PASSWORD "db-password"
  }
}
```

At deploy time, AgentSpec:

1. Resolves each secret from its source (`env()` from environment variables, `store()` from a secret manager).
2. Creates a Kubernetes Secret object in the target namespace.
3. Configures the Deployment to mount the secret values as environment variables.

!!! warning "Secret Rotation"
    When a secret value changes, re-run `agentspec apply` to update the Kubernetes Secret and trigger a rolling restart of the pods.

---

## Ingress Configuration

AgentSpec generates a Service of type `ClusterIP` by default. To expose the agent externally, configure an Ingress resource separately or use a LoadBalancer service type.

For external access, you can set up an Ingress controller and create an Ingress resource that points to the generated Service:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: agent-ingress
  namespace: agents
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
  - host: agent.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: assistant
            port:
              number: 8080
```

!!! note "Ingress Is External"
    AgentSpec does not generate Ingress resources. Ingress configuration is managed separately because it depends on your cluster's ingress controller, TLS certificates, and DNS setup.

---

## Complete Example

A production-ready agent with Kubernetes deployment, autoscaling, and secret management:

```ias
package "production-agent" version "1.0.0" lang "2.0"

prompt "system" {
  content "You are a production-grade assistant. Provide accurate,\nhelpful responses. Follow security best practices and\nnever expose sensitive information."
}

skill "search" {
  description "Search the knowledge base for information"
  input {
    query string required
  }
  output {
    results string
  }
  tool command {
    binary "search-tool"
  }
}

skill "create-report" {
  description "Generate a structured report from data"
  input {
    data string required
    format string required
  }
  output {
    report string
  }
  tool command {
    binary "report-tool"
  }
}

agent "assistant" {
  uses prompt "system"
  uses skill "search"
  uses skill "create-report"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 15
  timeout "60s"
  token_budget 200000
  on_error "retry"
  max_retries 3
}

secret "api-key" {
  env(ANTHROPIC_API_KEY)
}

secret "db-url" {
  env(DATABASE_URL)
}

# Local development
deploy "dev" target "process" {
  default true
  port 8080
  health {
    path "/healthz"
  }
}

# Production Kubernetes
deploy "production" target "kubernetes" {
  namespace "agents"
  image "agentspec/assistant:1.0.0"
  replicas 3
  port 8080
  resources {
    cpu "1"
    memory "1Gi"
  }
  health {
    path "/healthz"
    interval "30s"
    timeout "5s"
  }
  autoscale {
    min 3
    max 10
    metric "cpu"
    target 80
  }
  env {
    LOG_LEVEL "warn"
    ENVIRONMENT "production"
  }
  secrets {
    ANTHROPIC_API_KEY "api-key"
    DATABASE_URL "db-url"
  }
}
```

---

## Deploying

Validate, plan, and apply the Kubernetes deployment:

```bash
# Validate the .ias file
agentspec validate production-agent.ias

# Preview the Kubernetes deployment
agentspec plan production-agent.ias --target production

# Apply to the cluster
agentspec apply production-agent.ias --target production
```

---

## Verification

After applying, verify the deployment using `kubectl`.

### Check Deployment Status

```bash
kubectl get deployment -n agents
```

Expected output:

```
NAME        READY   UP-TO-DATE   AVAILABLE   AGE
assistant   3/3     3            3           2m
```

### Check Pod Status

```bash
kubectl get pods -n agents
```

Expected output:

```
NAME                         READY   STATUS    RESTARTS   AGE
assistant-7d4f8b6c9a-abc12   1/1     Running   0          2m
assistant-7d4f8b6c9a-def34   1/1     Running   0          2m
assistant-7d4f8b6c9a-ghi56   1/1     Running   0          2m
```

### Check Service

```bash
kubectl get service -n agents
```

### Check HPA

```bash
kubectl get hpa -n agents
```

Expected output:

```
NAME        REFERENCE              TARGETS   MINPODS   MAXPODS   REPLICAS   AGE
assistant   Deployment/assistant   45%/80%   3         10        3          2m
```

### Health Check (via port-forward)

```bash
kubectl port-forward -n agents service/assistant 8080:8080
curl http://localhost:8080/healthz
```

### View Logs

```bash
kubectl logs -n agents -l app=assistant --follow
```

---

## Updating the Deployment

To update a running Kubernetes deployment, modify your `.ias` file and re-apply:

```bash
agentspec plan production-agent.ias --target production
agentspec apply production-agent.ias --target production
```

AgentSpec performs a rolling update. Kubernetes gradually replaces old pods with new ones, ensuring zero downtime as long as the health probes pass.

---

## Tearing Down

To remove all Kubernetes resources created by the deployment:

```bash
agentspec destroy production-agent.ias --target production
```

This deletes the Deployment, Service, HPA, and Secrets from the target namespace and updates `.agentspec.state.json`.

---

## See Also

- [Deployment Overview](index.md) -- Compare all deployment targets
- [Docker Compose Deployment](compose.md) -- Multi-agent stacks for staging
- [Deploy Block Reference](../language/deploy.md) -- Full attribute reference
- [Best Practices](best-practices.md) -- Production security, monitoring, and scaling guidance
