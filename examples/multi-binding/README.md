# Multi-Binding

Deploy the same agent definition to multiple platforms simultaneously using different adapter bindings.

## What This Demonstrates

- **Multiple bindings** targeting different adapters from one definition
- **Default binding** selection for `plan` and `apply` without `--target`
- **Adapter-specific artifacts** produced by `export`
- **Write once, deploy anywhere** workflow

## Definition Structure

### Default Binding

```
binding "local" adapter "local-mcp" {
  default true
}
```

The `local-mcp` adapter is marked as default. Commands like `agentz plan` and `agentz apply` use this binding when no `--target` flag is specified.

### Additional Binding

```
binding "compose" adapter "docker-compose" {
  output_dir "./compose-deploy"
}
```

The `docker-compose` adapter produces container deployment artifacts. The `output_dir` configures where exported files are written.

## How to Run

```bash
# Validate
./agentz validate examples/multi-binding.az

# Plan for default binding (local-mcp)
./agentz plan examples/multi-binding.az

# Plan for docker-compose binding
./agentz plan examples/multi-binding.az --target compose

# Apply to default
./agentz apply examples/multi-binding.az --auto-approve

# Export to local-mcp
./agentz export examples/multi-binding.az --out-dir ./local-output

# Export to docker-compose
./agentz export examples/multi-binding.az --target compose --out-dir ./compose-output
```

## Resources Created

| Kind | Name | Description |
|------|------|-------------|
| Prompt | system | System prompt |
| Skill | query | Data query capability |
| Agent | data-bot | Agent using the query skill |

## Exported Artifacts by Adapter

### local-mcp

```
local-output/
  agents.json          # Agent definitions with resolved references
  mcp-servers.json     # MCP server configurations (if any)
  mcp-clients.json     # MCP client configurations (if any)
```

### docker-compose

```
compose-output/
  docker-compose.yml   # Container service definitions
  config/              # Configuration files for each service
  .env                 # Environment variables template
```

## How Binding Resolution Works

1. `agentz plan` checks for a `--target` flag
2. If no target is specified, it looks for a binding with `default true`
3. If exactly one binding exists and none is marked default, it uses that one implicitly
4. If multiple bindings exist with no default and no `--target`, the tool reports an error

## Next Steps

- Add environment overlays to multi-binding: see [customer-support](../customer-support/)
- Deploy a multi-agent pipeline to CI: see [code-review-pipeline](../code-review-pipeline/)
