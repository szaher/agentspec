# Research Swarm

A multi-agent research system with a coordinator and two specialized research agents (web and academic) connected via a pipeline. The research agents run in parallel, and a coordinator agent synthesizes their findings into a unified report.

## Prerequisites

- AgentSpec CLI installed
- `ANTHROPIC_API_KEY` environment variable set
- MCP-compatible search services for web and academic search

## Configuration

Set required environment variables:

```bash
export ANTHROPIC_API_KEY="your-key-here"
```

## Run

```bash
agentspec validate research-swarm.ias
agentspec run research-swarm.ias
```

## Customization

- Add more research agents (e.g., news, patents, internal docs) by defining new agent and skill blocks and adding them as pipeline steps.
- Adjust `depends_on` in the pipeline to change execution order or add sequential dependencies.
- Modify the coordinator prompt to change the output format (e.g., bullet points vs. narrative report).
- Tune `max_turns` per agent based on the complexity of each research domain.
