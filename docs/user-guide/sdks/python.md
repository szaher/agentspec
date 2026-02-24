# Python SDK

The AgentSpec Python SDK provides a typed client for the AgentSpec HTTP API with support for synchronous and asynchronous invocation, streaming responses, and session management.

---

## Installation

```bash
pip install agentspec
```

For async support (optional):

```bash
pip install agentspec[async]
```

**Requirements:** Python 3.9+

---

## Client Initialization

### Basic Setup

```python
from agentspec import AgentSpecClient

client = AgentSpecClient(
    base_url="http://localhost:8080",
    api_key="your-api-key",
)
```

### Bearer Token Authentication

```python
from agentspec import AgentSpecClient

client = AgentSpecClient(
    base_url="http://localhost:8080",
    token="your-bearer-token",
)
```

### Environment Variables

The client reads configuration from environment variables when arguments are not provided:

```bash
export AGENTSPEC_BASE_URL="http://localhost:8080"
export AGENTSPEC_API_KEY="your-api-key"
```

```python
from agentspec import AgentSpecClient

# Picks up AGENTSPEC_BASE_URL and AGENTSPEC_API_KEY automatically
client = AgentSpecClient()
```

### Custom Options

```python
from agentspec import AgentSpecClient

client = AgentSpecClient(
    base_url="http://localhost:8080",
    api_key="your-api-key",
    timeout=30.0,           # Request timeout in seconds
    max_retries=3,          # Retry on transient errors
    retry_delay=1.0,        # Initial retry delay in seconds
)
```

---

## Listing Agents

```python
agents = client.agents.list()

for agent in agents:
    print(f"{agent.name} - {agent.model} - {agent.status}")
```

---

## Invoking an Agent

### Synchronous Invocation

```python
response = client.agents.invoke(
    name="assistant",
    input="What is the capital of France?",
)

print(response.output)
# "The capital of France is Paris."

print(f"Tokens used: {response.usage.total_tokens}")
```

### With Options

```python
response = client.agents.invoke(
    name="assistant",
    input="Write a haiku about programming.",
    options={
        "temperature": 0.9,
        "max_tokens": 100,
    },
)
```

---

## Async Invocation

Use the async client for non-blocking calls in async applications.

```python
import asyncio
from agentspec import AsyncAgentSpecClient

async def main():
    client = AsyncAgentSpecClient(
        base_url="http://localhost:8080",
        api_key="your-api-key",
    )

    response = await client.agents.invoke(
        name="assistant",
        input="Explain quantum computing in simple terms.",
    )
    print(response.output)

    await client.close()

asyncio.run(main())
```

### Async Context Manager

```python
import asyncio
from agentspec import AsyncAgentSpecClient

async def main():
    async with AsyncAgentSpecClient(api_key="your-api-key") as client:
        response = await client.agents.invoke(
            name="assistant",
            input="Hello!",
        )
        print(response.output)

asyncio.run(main())
```

---

## Streaming Responses

### Synchronous Streaming

```python
for event in client.agents.stream(name="assistant", input="Tell me a story."):
    if event.type == "token":
        print(event.content, end="", flush=True)
    elif event.type == "tool_call":
        print(f"\n[Calling tool: {event.name}]")
    elif event.type == "tool_result":
        print(f"\n[Tool result: {event.output}]")
    elif event.type == "done":
        print(f"\n\nTokens used: {event.usage.total_tokens}")
    elif event.type == "error":
        print(f"\nError: {event.message}")
```

### Async Streaming

```python
async for event in client.agents.stream(name="assistant", input="Tell me a story."):
    if event.type == "token":
        print(event.content, end="", flush=True)
    elif event.type == "done":
        print(f"\n\nTokens used: {event.usage.total_tokens}")
```

### Collecting Stream into a String

```python
full_response = ""
async for event in client.agents.stream(name="assistant", input="Hello"):
    if event.type == "token":
        full_response += event.content

print(full_response)
```

---

## Session Management

Sessions enable multi-turn conversations where the agent remembers previous messages.

### Creating and Using a Session

```python
# Create a session
session = client.sessions.create(agent="assistant")
print(f"Session ID: {session.session_id}")

# First turn
response = client.sessions.continue_(
    agent="assistant",
    session_id=session.session_id,
    input="My name is Alice and I work on robotics.",
)
print(response.output)

# Second turn (agent remembers context)
response = client.sessions.continue_(
    agent="assistant",
    session_id=session.session_id,
    input="What do I work on?",
)
print(response.output)
# "You work on robotics."
```

### Session with Metadata

```python
session = client.sessions.create(
    agent="assistant",
    metadata={
        "user_id": "user-42",
        "channel": "web",
    },
)
```

### Listing Sessions

```python
sessions = client.sessions.list(agent="assistant", limit=10)

for s in sessions:
    print(f"{s.session_id} - {s.turns} turns - last active: {s.last_active_at}")
```

---

## Running Pipelines

```python
result = client.pipelines.run(
    name="research-and-summarize",
    input="Summarize recent advances in battery technology.",
)

print(f"Pipeline status: {result.status}")
print(f"Final output: {result.output}")

for step in result.steps:
    print(f"  Step '{step.name}': {step.status} ({step.duration_ms}ms)")
```

### Checking Pipeline Status

```python
status = client.pipelines.status(name="research-and-summarize")
print(f"Status: {status.status}, Elapsed: {status.elapsed_ms}ms")
```

---

## Error Handling

The SDK raises typed exceptions for different error conditions.

```python
from agentspec import AgentSpecClient
from agentspec.exceptions import (
    AgentSpecError,
    AuthenticationError,
    NotFoundError,
    InvalidRequestError,
    ServerError,
)

client = AgentSpecClient(api_key="your-api-key")

try:
    response = client.agents.invoke(
        name="assistant",
        input="Hello",
    )
except AuthenticationError as e:
    print(f"Authentication failed: {e.message}")
    # e.code == "unauthorized"
except NotFoundError as e:
    print(f"Agent not found: {e.message}")
    # e.code == "not_found"
except InvalidRequestError as e:
    print(f"Bad request: {e.message}")
    # e.code == "invalid_request"
except ServerError as e:
    print(f"Server error: {e.message}")
    # e.code == "internal_error"
except AgentSpecError as e:
    # Catch-all for any AgentSpec API error
    print(f"API error [{e.code}]: {e.message}")
```

### Retry Logic

The client automatically retries on transient errors (HTTP 429, 502, 503, 504) with exponential backoff. Configure retry behavior during initialization:

```python
client = AgentSpecClient(
    api_key="your-api-key",
    max_retries=5,
    retry_delay=0.5,  # Initial delay; doubles on each retry
)
```

---

## Type Hints

The SDK is fully typed and works with mypy and pyright. All response objects are dataclasses with documented fields.

```python
from agentspec.types import (
    Agent,
    InvokeResponse,
    StreamEvent,
    Session,
    SessionContinueResponse,
    PipelineRunResponse,
    PipelineStatus,
    Usage,
)

def process_response(response: InvokeResponse) -> str:
    """Extract and log the agent's output."""
    print(f"Tokens: {response.usage.total_tokens}")
    return response.output

def handle_stream(events: list[StreamEvent]) -> str:
    """Collect stream tokens into a complete response."""
    return "".join(
        event.content for event in events if event.type == "token"
    )
```

---

## Complete Example

A full example that creates a session, has a multi-turn conversation with streaming, and handles errors.

```python
import sys
from agentspec import AgentSpecClient
from agentspec.exceptions import AgentSpecError

def main():
    client = AgentSpecClient(
        base_url="http://localhost:8080",
        api_key="your-api-key",
    )

    # List available agents
    agents = client.agents.list()
    print("Available agents:")
    for agent in agents:
        print(f"  - {agent.name} ({agent.model})")

    if not agents:
        print("No agents deployed.")
        sys.exit(1)

    agent_name = agents[0].name

    # Create a session
    session = client.sessions.create(agent=agent_name)
    print(f"\nSession created: {session.session_id}")

    # Conversation loop
    messages = [
        "Hi, I'm building a web scraper in Python.",
        "What library should I use for parsing HTML?",
        "Can you show me a quick example?",
    ]

    for msg in messages:
        print(f"\nYou: {msg}")
        print(f"Agent: ", end="")

        try:
            for event in client.sessions.continue_stream(
                agent=agent_name,
                session_id=session.session_id,
                input=msg,
            ):
                if event.type == "token":
                    print(event.content, end="", flush=True)
                elif event.type == "done":
                    print(f"\n  [Tokens: {event.usage.total_tokens}]")
        except AgentSpecError as e:
            print(f"\nError: {e.message}")
            sys.exit(1)

if __name__ == "__main__":
    main()
```

---

## What's Next

- [HTTP API Overview](../api/index.md) -- Full API reference
- [TypeScript SDK](typescript.md) -- TypeScript/JavaScript client
- [Go SDK](go.md) -- Go client library
- [Agent Endpoints](../api/agents.md) -- Underlying API endpoints
