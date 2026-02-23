# SDK API Contract

## Overview

SDKs provide programmatic access to AgentSpec resources from Python,
TypeScript, and Go. In MVP (no API server), SDKs operate against
the local state file and compiled IR artifacts.

## Common API Surface

All three SDKs MUST expose the following operations:

### Initialization

```
client = AgentSpecClient(state_file="path/to/.agentspec.state.json")
```

### List Resources

```
agents = client.list_agents()
prompts = client.list_prompts()
skills = client.list_skills()
servers = client.list_servers()
clients = client.list_clients()
```

Returns a list of resource summaries:
- `name`: Resource name
- `kind`: Resource type
- `fqn`: Fully-qualified name
- `status`: `applied` | `failed` | `pending`
- `hash`: Content hash
- `last_applied`: Timestamp

### Get Resource

```
agent = client.get_agent("research-assistant")
```

Returns full resource details including all resolved attributes.

### Resolve Endpoint

```
endpoint = client.resolve_endpoint("research-assistant")
```

Returns the endpoint/address for a resource as determined by its
binding and adapter (e.g., local socket path, HTTP URL, Docker
service name).

### Invoke Run

```
run = client.invoke("research-assistant", input={"query": "..."})
```

Returns:
- `run_id`: Unique run identifier
- `correlation_id`: Correlation ID for tracing
- `status`: `started` | `running` | `completed` | `failed`

### Stream Events

```
for event in client.stream_events(run_id):
    print(event.type, event.data)
```

Event types:
- `run.started` — Run began
- `run.progress` — Intermediate output
- `run.completed` — Run finished successfully
- `run.failed` — Run failed with error
- `run.log` — Structured log entry

### Error Handling

All SDK methods raise/return typed errors:
- `ResourceNotFoundError` — Resource does not exist in state
- `StateFileError` — State file missing or corrupted
- `InvocationError` — Run invocation failed
- `ConnectionError` — Cannot connect to local runtime

## Language-Specific Notes

### Python SDK

- Package name: `agentspec`
- Minimum Python: 3.10+
- Async support: `AsyncAgentSpecClient` with `async for` streaming
- Type hints via dataclasses

### TypeScript SDK

- Package name: `@agentspec/sdk`
- Minimum Node: 18+
- Async/await with `AsyncIterable` for streaming
- Full TypeScript types generated from IR schema

### Go SDK

- Module: `github.com/agentz/sdk-go`
- Minimum Go: 1.25+
- Context-based cancellation
- Channel-based streaming: `func StreamEvents(ctx, runID) <-chan Event`
