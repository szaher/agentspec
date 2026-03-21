# Multi-Agent Router

A router agent that uses lang "3.0" control flow to classify incoming customer messages and delegate them to specialized sub-agents: billing, technical support, or general inquiries. Demonstrates if/else routing, multi-agent composition, and intent-based dispatch.

## Architecture Overview

```
Customer Message
    |
    v
router (agent, lang 3.0)
    |
    +---> on input: if/else keyword matching
    |         |
    |         +--- "billing" / "invoice" / "refund" ---> billing-agent
    |         +--- "bug" / "error" / "api"          ---> technical-agent
    |         +--- (default)                        ---> general-agent
    |
    v
Specialist Agent
    |
    +---> billing-agent:    lookup-account, process-refund, create-ticket
    +---> technical-agent:  search-docs, create-ticket
    +---> general-agent:    create-ticket
```

The router uses lang "3.0" `on input` blocks with `if/else` conditions to inspect the incoming message and route to the appropriate specialist. Each specialist agent has its own prompt and skill set tailored to its domain.

## Prerequisites

1. Build the AgentSpec CLI from the repository root:

   ```bash
   go build -o agentspec ./cmd/agentspec
   ```

2. Set the required environment variable:

   ```bash
   export PLATFORM_API_KEY="your-platform-api-key"
   ```

3. Ensure the following tool binaries are available on `$PATH` (or stubbed for testing):
   - `intent-classifier`
   - `account-lookup`
   - `refund-processor`
   - `docs-search`
   - `ticket-creator`

## Step-by-Step Run Instructions

```bash
# 1. Validate the AgentSpec
./agentspec validate examples/multi-agent-router/multi-agent-router.ias

# 2. Preview planned changes
./agentspec plan examples/multi-agent-router/multi-agent-router.ias

# 3. Apply the changes
./agentspec apply examples/multi-agent-router/multi-agent-router.ias --auto-approve

# 4. Test routing with a billing message
./agentspec dev examples/multi-agent-router/multi-agent-router.ias --input "I need a refund for order #12345"

# 5. Test routing with a technical message
./agentspec dev examples/multi-agent-router/multi-agent-router.ias --input "I'm getting an API error 500 on the /users endpoint"

# 6. Test routing with a general message
./agentspec dev examples/multi-agent-router/multi-agent-router.ias --input "What products do you offer?"

# 7. Export artifacts
./agentspec export examples/multi-agent-router/multi-agent-router.ias --out-dir ./output
```

## Customization Tips

- **Add more specialists**: Define new agents (e.g., `sales-agent`, `returns-agent`) and add corresponding `else if` branches in the router's `on input` block.
- **Use LLM-based classification**: Replace keyword matching with the `classify-intent` skill for more nuanced routing based on semantic understanding.
- **Add fallback behavior**: Add an escalation skill in the default `else` branch for messages that no specialist can handle.
- **Add environment overlays**: Use a cheaper model for the router (which only classifies) and a more capable model for specialists that need to reason.
- **Chain with a pipeline**: Wrap the routing pattern inside a `pipeline` for post-processing steps like satisfaction surveys or ticket summaries.
