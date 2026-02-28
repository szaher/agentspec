# package

Package a compiled agent for deployment.

## Usage

```bash
agentspec package [file.ias | directory | compiled-binary]
```

## Description

The `package` command wraps a compiled agent binary into deployment-ready artifacts such as Docker images, Kubernetes manifests, or Helm charts.

If you pass `.ias` files or a directory instead of a pre-compiled binary, the command will automatically compile them first using the `standalone` target. When using a container format (`docker`, `kubernetes`, `helm`), the compiler automatically cross-compiles for `linux/amd64` unless you override the platform with `--platform`.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | | `docker` | Package format (see formats below) |
| `--output` | `-o` | `./build` | Output directory |
| `--tag` | | `<binary-name>:latest` | Image tag (for Docker format) |
| `--registry` | | | Container registry URL (prefixed to the tag) |
| `--platform` | | *(auto-detected)* | Target platform (e.g. `linux/amd64`, `linux/arm64`). Auto-set to `linux/amd64` for container formats |
| `--push` | | `false` | Push image to registry after build |
| `--json` | | `false` | Output result as JSON |

## Formats

| Format | Output | Description |
|--------|--------|-------------|
| `docker` | `Dockerfile`, `.dockerignore`, compiled binary | Files needed to build a Docker image |
| `kubernetes` | Kubernetes JSON manifests | Deployment, Service, and related resources |
| `helm` | Helm chart directory | A complete Helm chart with templates and values |
| `binary` | Compiled binary only | No additional packaging; just the standalone binary |

## Examples

### Docker (end-to-end)

```bash
# Package from .ias files (auto-compiles for linux/amd64)
agentspec package --format docker agent.ias

# Build the Docker image
docker build -t my-agent:latest ./build

# Run the container
docker run -p 8080:8080 my-agent:latest
```

### Docker with a custom registry

```bash
agentspec package --format docker --registry ghcr.io/myorg --tag my-agent:v1.0 agent.ias
```

### Kubernetes manifests

```bash
agentspec package --format kubernetes --tag my-agent:latest agent.ias
kubectl apply -f ./build/
```

### Helm chart

```bash
agentspec package --format helm --tag my-agent:latest agent.ias
helm install my-agent ./build/chart
```

### Package a pre-compiled binary

```bash
# Compile first, then package separately
agentspec compile --platform linux/amd64 agent.ias
agentspec package --format docker ./build/agent
```

### Binary-only output

```bash
agentspec package --format binary agent.ias
```

## Output

On success, the command prints a summary and next steps:

```
Package format: docker
Output: ./build
Tag: my-agent:latest
Files generated: 3
  ./build/agent
  ./build/Dockerfile
  ./build/.dockerignore

Next steps:
  1. Build the image:  docker build -t my-agent:latest ./build
  2. Run the container: docker run -p 8080:8080 my-agent:latest
  3. Open the UI:       http://localhost:8080
```

Pass `--json` to get machine-readable output instead.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Packaging succeeded |
| `1` | An error occurred (compilation failure, invalid format, etc.) |

## See Also

- [CLI: compile](compile.md) -- Compile .ias files into a binary
- [CLI: apply](apply.md) -- Deploy agents to remote targets
- [CLI: publish](publish.md) -- Publish an AgentPack to a Git remote
