# TypeScript SDK

The AgentSpec TypeScript SDK provides a fully typed client for the AgentSpec HTTP API. It supports Node.js, Deno, and modern browsers with built-in streaming, session management, and error handling.

---

## Installation

```bash
npm install @agentspec/sdk
```

Or with other package managers:

```bash
yarn add @agentspec/sdk
pnpm add @agentspec/sdk
```

**Requirements:** Node.js 18+ (or any runtime with `fetch` and `ReadableStream` support)

---

## Client Initialization

### Basic Setup

```typescript
import { AgentSpecClient } from "@agentspec/sdk";

const client = new AgentSpecClient({
  baseUrl: "http://localhost:8080",
  apiKey: "your-api-key",
});
```

### Bearer Token Authentication

```typescript
import { AgentSpecClient } from "@agentspec/sdk";

const client = new AgentSpecClient({
  baseUrl: "http://localhost:8080",
  token: "your-bearer-token",
});
```

### Environment Variables

When running in Node.js, the client reads configuration from environment variables if constructor options are not provided:

```bash
export AGENTSPEC_BASE_URL="http://localhost:8080"
export AGENTSPEC_API_KEY="your-api-key"
```

```typescript
import { AgentSpecClient } from "@agentspec/sdk";

// Picks up AGENTSPEC_BASE_URL and AGENTSPEC_API_KEY automatically
const client = new AgentSpecClient();
```

### Custom Options

```typescript
const client = new AgentSpecClient({
  baseUrl: "http://localhost:8080",
  apiKey: "your-api-key",
  timeout: 30_000,    // Request timeout in milliseconds
  maxRetries: 3,      // Retry on transient errors
  retryDelay: 1000,   // Initial retry delay in milliseconds
});
```

---

## Listing Agents

```typescript
const agents = await client.agents.list();

for (const agent of agents) {
  console.log(`${agent.name} - ${agent.model} - ${agent.status}`);
}
```

---

## Invoking an Agent

### Basic Invocation

```typescript
const response = await client.agents.invoke({
  name: "assistant",
  input: "What is the capital of France?",
});

console.log(response.output);
// "The capital of France is Paris."

console.log(`Tokens used: ${response.usage.totalTokens}`);
```

### With Options

```typescript
const response = await client.agents.invoke({
  name: "assistant",
  input: "Write a haiku about programming.",
  options: {
    temperature: 0.9,
    maxTokens: 100,
  },
});
```

---

## Streaming Responses

### Async Iterator

```typescript
const stream = client.agents.stream({
  name: "assistant",
  input: "Explain how compilers work.",
});

for await (const event of stream) {
  switch (event.type) {
    case "token":
      process.stdout.write(event.content);
      break;
    case "tool_call":
      console.log(`\n[Calling tool: ${event.name}]`);
      break;
    case "tool_result":
      console.log(`\n[Tool result: ${event.output}]`);
      break;
    case "done":
      console.log(`\n\nTokens used: ${event.usage.totalTokens}`);
      break;
    case "error":
      console.error(`\nError: ${event.message}`);
      break;
  }
}
```

### Callback-Based Streaming

```typescript
await client.agents.stream(
  {
    name: "assistant",
    input: "Tell me a story.",
  },
  {
    onToken: (content) => process.stdout.write(content),
    onToolCall: (name, args) => console.log(`\n[Tool: ${name}]`),
    onDone: (usage) => console.log(`\nTokens: ${usage.totalTokens}`),
    onError: (error) => console.error(`Error: ${error.message}`),
  }
);
```

### Collecting Stream into a String

```typescript
const stream = client.agents.stream({
  name: "assistant",
  input: "Hello",
});

let fullResponse = "";
for await (const event of stream) {
  if (event.type === "token") {
    fullResponse += event.content;
  }
}
console.log(fullResponse);
```

---

## Session Management

Sessions enable multi-turn conversations where the agent remembers previous messages.

### Creating and Using a Session

```typescript
// Create a session
const session = await client.sessions.create({
  agent: "assistant",
});
console.log(`Session ID: ${session.sessionId}`);

// First turn
const response1 = await client.sessions.continue({
  agent: "assistant",
  sessionId: session.sessionId,
  input: "My name is Alice and I work on robotics.",
});
console.log(response1.output);

// Second turn (agent remembers context)
const response2 = await client.sessions.continue({
  agent: "assistant",
  sessionId: session.sessionId,
  input: "What do I work on?",
});
console.log(response2.output);
// "You work on robotics."
```

### Session with Metadata

```typescript
const session = await client.sessions.create({
  agent: "assistant",
  metadata: {
    userId: "user-42",
    channel: "web",
  },
});
```

### Listing Sessions

```typescript
const sessions = await client.sessions.list({
  agent: "assistant",
  limit: 10,
});

for (const s of sessions) {
  console.log(`${s.sessionId} - ${s.turns} turns - last active: ${s.lastActiveAt}`);
}
```

---

## Running Pipelines

```typescript
const result = await client.pipelines.run({
  name: "research-and-summarize",
  input: "Summarize recent advances in battery technology.",
});

console.log(`Pipeline status: ${result.status}`);
console.log(`Final output: ${result.output}`);

for (const step of result.steps) {
  console.log(`  Step '${step.name}': ${step.status} (${step.durationMs}ms)`);
}
```

### Checking Pipeline Status

```typescript
const status = await client.pipelines.status({
  name: "research-and-summarize",
});
console.log(`Status: ${status.status}, Elapsed: ${status.elapsedMs}ms`);
```

---

## Error Handling

The SDK throws typed errors for different error conditions.

```typescript
import { AgentSpecClient } from "@agentspec/sdk";
import {
  AgentSpecError,
  AuthenticationError,
  NotFoundError,
  InvalidRequestError,
  ServerError,
} from "@agentspec/sdk/errors";

const client = new AgentSpecClient({ apiKey: "your-api-key" });

try {
  const response = await client.agents.invoke({
    name: "assistant",
    input: "Hello",
  });
} catch (error) {
  if (error instanceof AuthenticationError) {
    console.error(`Authentication failed: ${error.message}`);
    // error.code === "unauthorized"
  } else if (error instanceof NotFoundError) {
    console.error(`Agent not found: ${error.message}`);
    // error.code === "not_found"
  } else if (error instanceof InvalidRequestError) {
    console.error(`Bad request: ${error.message}`);
    // error.code === "invalid_request"
  } else if (error instanceof ServerError) {
    console.error(`Server error: ${error.message}`);
    // error.code === "internal_error"
  } else if (error instanceof AgentSpecError) {
    // Catch-all for any AgentSpec API error
    console.error(`API error [${error.code}]: ${error.message}`);
  } else {
    throw error;
  }
}
```

### Retry Logic

The client automatically retries on transient errors (HTTP 429, 502, 503, 504) with exponential backoff. Configure retry behavior during initialization:

```typescript
const client = new AgentSpecClient({
  apiKey: "your-api-key",
  maxRetries: 5,
  retryDelay: 500, // Initial delay in ms; doubles on each retry
});
```

---

## TypeScript Types

The SDK exports all request and response types for use in your application.

```typescript
import type {
  Agent,
  InvokeRequest,
  InvokeResponse,
  StreamEvent,
  TokenEvent,
  DoneEvent,
  Session,
  SessionContinueResponse,
  PipelineRunResponse,
  PipelineStatus,
  Usage,
} from "@agentspec/sdk/types";

function processResponse(response: InvokeResponse): string {
  console.log(`Tokens: ${response.usage.totalTokens}`);
  return response.output;
}

function handleEvents(events: StreamEvent[]): string {
  return events
    .filter((e): e is TokenEvent => e.type === "token")
    .map((e) => e.content)
    .join("");
}
```

---

## Complete Example

A full example that creates a session, has a multi-turn conversation with streaming, and handles errors.

```typescript
import { AgentSpecClient } from "@agentspec/sdk";
import { AgentSpecError } from "@agentspec/sdk/errors";

async function main() {
  const client = new AgentSpecClient({
    baseUrl: "http://localhost:8080",
    apiKey: "your-api-key",
  });

  // List available agents
  const agents = await client.agents.list();
  console.log("Available agents:");
  for (const agent of agents) {
    console.log(`  - ${agent.name} (${agent.model})`);
  }

  if (agents.length === 0) {
    console.log("No agents deployed.");
    process.exit(1);
  }

  const agentName = agents[0].name;

  // Create a session
  const session = await client.sessions.create({ agent: agentName });
  console.log(`\nSession created: ${session.sessionId}`);

  // Conversation loop
  const messages = [
    "Hi, I'm building a web scraper in TypeScript.",
    "What library should I use for parsing HTML?",
    "Can you show me a quick example?",
  ];

  for (const msg of messages) {
    console.log(`\nYou: ${msg}`);
    process.stdout.write("Agent: ");

    try {
      const stream = client.sessions.continueStream({
        agent: agentName,
        sessionId: session.sessionId,
        input: msg,
      });

      for await (const event of stream) {
        if (event.type === "token") {
          process.stdout.write(event.content);
        } else if (event.type === "done") {
          console.log(`\n  [Tokens: ${event.usage.totalTokens}]`);
        }
      }
    } catch (error) {
      if (error instanceof AgentSpecError) {
        console.error(`\nError: ${error.message}`);
        process.exit(1);
      }
      throw error;
    }
  }
}

main();
```

---

## What's Next

- [HTTP API Overview](../api/index.md) -- Full API reference
- [Python SDK](python.md) -- Python client library
- [Go SDK](go.md) -- Go client library
- [Agent Endpoints](../api/agents.md) -- Underlying API endpoints
