# Docker Image Contract

## Image Reference

```
ghcr.io/szaher/agentspec:<tag>
```

## Tags

| Tag Pattern | Example | Description |
|-------------|---------|-------------|
| `X.Y.Z` | `0.5.0` | Exact version |
| `X.Y` | `0.5` | Latest patch for minor |
| `X` | `0` | Latest minor+patch for major |
| `latest` | — | Most recent stable release |

## Usage

### Validate a file

```bash
docker run --rm -v "$(pwd):/work" -w /work ghcr.io/szaher/agentspec:latest validate agent.ias
```

### Format a file

```bash
docker run --rm -v "$(pwd):/work" -w /work ghcr.io/szaher/agentspec:latest fmt agent.ias
```

### Run an agent (with API key)

```bash
docker run --rm -it \
  -v "$(pwd):/work" -w /work \
  -e ANTHROPIC_API_KEY \
  ghcr.io/szaher/agentspec:latest run agent.ias
```

### CI/CD (GitHub Actions)

```yaml
- name: Validate AgentSpec
  run: |
    docker run --rm \
      -v "${{ github.workspace }}:/work" -w /work \
      ghcr.io/szaher/agentspec:latest validate agent.ias
```

## Image Specification

| Property | Value |
|----------|-------|
| Base image | `scratch` |
| Binary path | `/agentspec` |
| Entrypoint | `["/agentspec"]` |
| Working directory | `/work` |
| Compressed size | < 50 MB |
| Architectures | `linux/amd64`, `linux/arm64` |
