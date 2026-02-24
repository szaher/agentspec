# Multi-Binding

Deploy the same AgentSpec to multiple platforms simultaneously using different deploy targets.

## What This Demonstrates

- **Multiple deploy targets** targeting different targets from one AgentSpec
- **Default deploy target** selection for `plan` and `apply` without `--target`
- **Target-specific artifacts** produced by `export`
- **Write once, deploy anywhere** workflow

## AgentSpec Structure

### Default Deploy Target

```
deploy "local" target "process" {
  default true
}
```

The `process` target is marked as default. Commands like `agentspec plan` and `agentspec apply` use this deploy target when no `--target` flag is specified.

### Additional Deploy Target

```
deploy "compose" target "docker-compose" {
  output_dir "./compose-deploy"
}
```

The `docker-compose` target produces container deployment artifacts. The `output_dir` configures where exported files are written.

## How to Run

```bash
# Validate
./agentspec validate examples/multi-binding.ias

# Plan for default deploy target (process)
./agentspec plan examples/multi-binding.ias

# Plan for docker-compose deploy target
./agentspec plan examples/multi-binding.ias --target compose

# Apply to default
./agentspec apply examples/multi-binding.ias --auto-approve

# Export to process target
./agentspec export examples/multi-binding.ias --out-dir ./local-output

# Export to docker-compose target
./agentspec export examples/multi-binding.ias --target compose --out-dir ./compose-output
```

## Resources Created

| Kind | Name | Description |
|------|------|-------------|
| Prompt | system | System prompt |
| Skill | query | Data query capability |
| Agent | data-bot | Agent using the query skill |

## Exported Artifacts by Target

### process

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

## How Deploy Target Resolution Works

1. `agentspec plan` checks for a `--target` flag
2. If no target is specified, it looks for a deploy target with `default true`
3. If exactly one deploy target exists and none is marked default, it uses that one implicitly
4. If multiple deploy targets exist with no default and no `--target`, the tool reports an error

## Next Steps

- Add environment overlays to multi-deploy-target: see [customer-support](../customer-support/)
- Deploy a multi-agent pipeline to CI: see [code-review-pipeline](../code-review-pipeline/)
