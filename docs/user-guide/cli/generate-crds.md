# generate crds

Generate Kubernetes Custom Resource Definition (CRD) manifests from an IntentLang specification file.

## Usage

```bash
agentspec generate crds [file.ias] [flags]
```

## Description

The `generate crds` command parses an IntentLang `.ias` file, builds an AST, converts it to an intermediate representation (IR), and emits Kubernetes CRD YAML files into the specified output directory. Each top-level resource in the spec produces one or more CRD manifests that can be applied directly to a Kubernetes cluster with `kubectl apply`.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output-dir` | `-o` | `generated-crds` | Directory to write generated CRD YAML files |
| `--namespace` | `-n` | `default` | Kubernetes namespace to set in generated manifests |

## Resource Mapping

The following table shows how IntentLang resources map to Kubernetes CRDs:

| IntentLang Resource | Kubernetes CRD |
|---------------------|----------------|
| Agent | Agent |
| Skill | ToolBinding |
| Pipeline | Workflow |
| Prompt | ConfigMap |
| Secret | Secret |
| MCPServer | ToolBinding |

## Examples

**Generate CRDs from a spec into the default output directory:**

```bash
agentspec generate crds my-agent.ias
```

This creates YAML files under `generated-crds/`.

**Specify a custom output directory and namespace:**

```bash
agentspec generate crds my-agent.ias -o k8s-manifests -n production
```

**Generate and apply in one step:**

```bash
agentspec generate crds my-agent.ias -o /tmp/crds -n my-namespace
kubectl apply -f /tmp/crds/
```

**Generate CRDs for a multi-agent pipeline:**

```bash
agentspec generate crds pipeline.ias -o deploy/crds
kubectl apply -f deploy/crds/
```
