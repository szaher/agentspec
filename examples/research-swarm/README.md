# Research Swarm

A multi-agent research system where a coordinator delegates sub-tasks to specialist agents (web researcher, academic researcher) and a synthesizer combines their findings into a final report. Demonstrates multi-agent collaboration using the pipeline DSL.

## Architecture Overview

```
User Query
    |
    v
pipeline "research"
    |
    +---> [gather-web]       web-agent        -- searches the web
    +---> [gather-academic]  academic-agent   -- searches scholarly databases
              |                    |
              +--------+-----------+
                       |
                       v
              [synthesize]  synthesis-agent   -- merges findings
                       |
                       v
              [finalize]    coordinator       -- produces final report
```

The `gather-web` and `gather-academic` steps run in parallel (no dependency between them). The `synthesize` step waits for both, and `finalize` produces the polished output.

## Prerequisites

1. Build the AgentSpec CLI from the repository root:

   ```bash
   go build -o agentspec ./cmd/agentspec
   ```

2. Set the required environment variable:

   ```bash
   export SEARCH_API_KEY="your-search-api-key"
   ```

3. Ensure the following tool binaries are available on `$PATH` (or stubbed for testing):
   - `web-search-tool`
   - `scholar-search-tool`
   - `summarize-tool`

## Step-by-Step Run Instructions

```bash
# 1. Validate the AgentSpec
./agentspec validate examples/research-swarm/research-swarm.ias

# 2. Preview planned changes
./agentspec plan examples/research-swarm/research-swarm.ias

# 3. Apply the changes
./agentspec apply examples/research-swarm/research-swarm.ias --auto-approve

# 4. Export artifacts
./agentspec export examples/research-swarm/research-swarm.ias --out-dir ./output
```

## Customization Tips

- **Add more specialists**: Define additional agents (e.g., `patent-agent`, `news-agent`) and add corresponding pipeline steps with `depends_on` wiring.
- **Change parallelism**: Adjust `depends_on` arrays to control which steps run concurrently vs. sequentially.
- **Swap models**: Use a cheaper model for research agents and a more capable model for the coordinator to optimize cost.
- **Add environment overlays**: Use `environment` blocks to switch models or tools between dev and production.
- **Add policies**: Attach rate-limiting or content-filtering policies to control agent behavior.
