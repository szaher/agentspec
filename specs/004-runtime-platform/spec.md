# Feature Specification: AgentSpec Runtime Platform

**Feature Branch**: `004-runtime-platform`
**Created**: 2026-02-23
**Status**: Draft
**Input**: User description: "Transform AgentSpec from a declarative state-tracking shell into a working agent deployment platform with runtime, agentic loop, deployment adapters, and SDKs"

## Clarifications

### Session 2026-02-23

- Q: Does `agentspec apply` start one runtime process serving all agents in a package, or one process per agent? → A: Single process serves all agents in a package (path-routed, shared resources).
- Q: When `agentspec apply` detects changes to a running agent, what update strategy is used? → A: Graceful restart — stop the running process, start a new one with updated config (brief downtime accepted).
- Q: What happens when two `agentspec apply` commands run simultaneously against the same package? → A: Fail the second invocation with a lock error (state file locking). User retries after the first completes.
- Q: How is `tool inline` (user-provided code) executed securely? → A: Subprocess with resource limits (timeout, memory cap, restricted filesystem access). Users can pass env variables and secrets to the inline tool execution runtime.
- Q: When a pipeline step fails, what happens to independent parallel steps already running? → A: Fail-fast — cancel all running steps immediately when any step fails (saves cost, simpler error reporting).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run an Agent Locally (Priority: P1)

As an agent builder, I want to write an `.ias` file defining an agent with a prompt, model, and skills, then run a single CLI command to get a running agent that accepts messages and responds with AI-generated text using the declared skills.

Today, `agentspec apply` produces a plan but executes nothing. After this feature, `apply` must start a real process that accepts requests, calls an LLM, dispatches tools, and returns responses. This is the core missing capability that makes everything else possible.

**Why this priority**: Without a working runtime, AgentSpec is a configuration validator, not an agent platform. Every other feature depends on agents actually running.

**Independent Test**: Can be fully tested by writing a minimal `.ias` file with one agent, one prompt, and one skill, running `agentspec apply`, sending an HTTP request to the agent, and verifying a meaningful AI-generated response is returned.

**Acceptance Scenarios**:

1. **Given** a valid `.ias` file with an agent definition, **When** the user runs `agentspec apply`, **Then** the system starts a local process and reports the endpoint URL where the agent is accessible.
2. **Given** a running agent, **When** the user sends a message via HTTP, **Then** the agent responds with AI-generated text informed by the declared system prompt.
3. **Given** a running agent with skills, **When** the LLM decides to use a skill during conversation, **Then** the skill's backing tool is invoked and the result is incorporated into the agent's response.
4. **Given** a running agent, **When** the user sends a request that requires multiple tool calls, **Then** the agent iterates through a reason-act-observe loop until it produces a final answer or reaches the configured turn limit.
5. **Given** a running agent, **When** the agent exceeds the configured maximum turns or timeout, **Then** the system returns a clear error indicating the limit was reached.
6. **Given** a running agent, **When** the user runs `agentspec apply` again with no changes, **Then** the system detects no drift and does not restart the agent.
7. **Given** a running agent, **When** the user runs `agentspec apply` with changes (e.g., prompt updated, skill added), **Then** the system gracefully stops the running process and starts a new one with the updated configuration.
8. **Given** a running agent, **When** the user runs a one-shot command like `agentspec run <agent> --input "message"`, **Then** the system invokes the agent, prints the response to stdout, and exits.

---

### User Story 2 - Executable Skills with Real Tool Backends (Priority: P1)

As an agent builder, I want to define skills that connect to real tool backends (MCP servers, HTTP APIs, or local commands) so that when my agent decides to use a skill, the tool actually executes and returns real results.

Currently, skills are metadata labels with no runtime behavior. After this feature, each skill must be backed by a concrete tool execution mechanism.

**Why this priority**: Agents without tools are just chatbots. Skills are what make agents useful for real tasks. This is co-equal with P1 because the agentic loop is meaningless without executable tools.

**Independent Test**: Can be tested by defining a skill backed by an MCP server, starting the agent, sending a message that triggers tool use, and verifying the MCP server received the call and the agent incorporated the result.

**Acceptance Scenarios**:

1. **Given** a skill with `tool mcp "server/tool"` and a running MCP server, **When** the agent decides to use this skill, **Then** the system connects to the MCP server and calls the specified tool with the correct input.
2. **Given** a skill with `tool http` configuration, **When** the agent invokes this skill, **Then** the system makes the configured HTTP request and returns the response body as the tool result.
3. **Given** a skill with `tool command` configuration, **When** the agent invokes this skill, **Then** the system spawns the configured subprocess, passes input, and captures the output.
4. **Given** a skill with typed input/output schemas, **When** the tool returns a response, **Then** the system validates the response against the declared output schema.
5. **Given** an MCP server that is not reachable, **When** the agent tries to use a skill backed by that server, **Then** the system returns a descriptive error to the LLM and the agent handles it gracefully (retry or inform the user).

---

### User Story 3 - IntentLang 2.0 Language Constructs (Priority: P2)

As an agent builder, I want the IntentLang language to support richer constructs for tool bindings, deployment targets, agent runtime configuration, prompt variables, and multi-agent pipelines, so that I can express complex agent definitions declaratively.

The current language (1.0) has `execution command "..."` for skills and `binding` for deployment, both of which are placeholders. IntentLang 2.0 replaces these with semantically rich constructs that the runtime can interpret.

**Why this priority**: The new runtime requires new language constructs to express its capabilities. Without `tool`, `deploy`, and agent runtime config in the language, users cannot configure the features from P1.

**Independent Test**: Can be tested by writing `.ias` files using the new 2.0 syntax, running `agentspec validate`, and verifying the parser accepts them with correct AST output.

**Acceptance Scenarios**:

1. **Given** an `.ias` file with `lang "2.0"`, **When** the user runs `agentspec validate`, **Then** the parser accepts the file and recognizes all 2.0 constructs (`tool`, `deploy`, `pipeline`, agent runtime config, prompt variables, `type` definitions).
2. **Given** an `.ias` file with `lang "1.0"`, **When** the user runs `agentspec validate`, **Then** the parser rejects the file with an actionable error message directing the user to run `agentspec migrate --to-v2`.
3. **Given** a valid 1.0 `.ias` file, **When** the user runs `agentspec migrate --to-v2`, **Then** the file is rewritten to 2.0 syntax with `execution command` replaced by `tool command`, `binding` replaced by `deploy`, and `lang "2.0"` set.
4. **Given** an agent block with `strategy`, `max_turns`, `timeout`, `token_budget`, and `temperature`, **When** validated, **Then** each attribute is recognized and its value constrained to valid ranges.
5. **Given** a prompt with `{{variable}}` placeholders and a `variables` block, **When** validated, **Then** all referenced variables are checked against the declarations.
6. **Given** a `type` definition with nested fields, enums, and lists, **When** used in a skill's input/output, **Then** the validator ensures type references resolve correctly.

---

### User Story 4 - Multi-Target Deployment (Priority: P3)

As an agent builder, I want to deploy my agents to Docker containers or Kubernetes clusters using the same `.ias` file with a different `deploy` target, so that I can promote agents from development to production without rewriting definitions.

**Why this priority**: Local process deployment (P1) proves the runtime works. Docker and Kubernetes deployment makes it production-ready. This is the third priority because it builds on a working runtime.

**Independent Test**: Can be tested by writing an `.ias` file with a `deploy "staging" target "docker"` block, running `agentspec apply --target staging`, and verifying a container is running and the agent responds to requests.

**Acceptance Scenarios**:

1. **Given** an `.ias` file with `deploy "staging" target "docker"`, **When** the user runs `agentspec apply --target staging`, **Then** the system builds a container image and starts a container with the agent accessible on the configured port.
2. **Given** an `.ias` file with `deploy "production" target "kubernetes"`, **When** the user runs `agentspec apply --target production`, **Then** the system creates the necessary cluster resources (deployment, service, configuration) and waits for the agent to become healthy.
3. **Given** a deployed agent on any target, **When** the user runs `agentspec status`, **Then** the system shows the health status, endpoint URL, and resource utilization of each deployed agent.
4. **Given** a deployed agent, **When** the user runs `agentspec logs <agent>`, **Then** the system streams recent logs from the deployed agent regardless of target.
5. **Given** a deployed agent, **When** the user runs `agentspec destroy`, **Then** all resources for the deployment are torn down and the state file is updated.
6. **Given** a Kubernetes deploy block with `autoscale` configuration, **When** deployed, **Then** the system configures horizontal scaling based on the declared metric and thresholds.

---

### User Story 5 - Multi-Agent Coordination (Priority: P4)

As an agent builder, I want to define pipelines where multiple agents coordinate on a complex task, with steps running in parallel where possible and data flowing between agents, so that I can build sophisticated workflows declaratively.

**Why this priority**: Single-agent operation (P1) covers most use cases. Multi-agent coordination enables advanced workflows like code review pipelines and research assistants but is not required for the platform to be useful.

**Independent Test**: Can be tested by defining a pipeline with 3 agents (two parallel, one dependent), invoking the pipeline, and verifying all agents execute in the correct order with data passing between steps.

**Acceptance Scenarios**:

1. **Given** a pipeline with steps that have `parallel true`, **When** the pipeline is invoked, **Then** parallel steps execute concurrently and the pipeline completes faster than sequential execution.
2. **Given** a pipeline with `depends_on` declarations, **When** the pipeline is invoked, **Then** dependent steps wait for their dependencies to complete before starting.
3. **Given** a pipeline step that fails, **When** the failure occurs, **Then** all running steps (including independent parallel steps) are cancelled immediately and the pipeline returns an error with details about which step failed and why.
4. **Given** an agent with `delegate to agent "other" when "condition"`, **When** the agent receives a message matching the condition, **Then** the conversation is handed off to the delegate agent.
5. **Given** a pipeline invocation, **When** it completes, **Then** the system returns results from all steps, including intermediate outputs, for auditability.

---

### User Story 6 - Developer SDKs and Programmatic Access (Priority: P5)

As an application developer, I want to invoke deployed agents programmatically from Python, TypeScript, or Go, with support for streaming responses and session continuity, so that I can integrate agents into my applications.

**Why this priority**: Direct HTTP access (from P1) is sufficient for early adopters. Typed SDK clients improve developer experience and adoption but are not required for agents to be useful.

**Independent Test**: Can be tested by installing the Python SDK, pointing it at a running agent, and verifying that `client.invoke()` returns a response, `client.stream()` yields chunks, and session-based conversations maintain context.

**Acceptance Scenarios**:

1. **Given** a deployed agent, **When** a developer uses the Python SDK to invoke it, **Then** the SDK returns a typed response object with the agent's output, token usage, and tool call audit trail.
2. **Given** a deployed agent with streaming enabled, **When** a developer uses the SDK's streaming method, **Then** response chunks arrive incrementally as the agent generates them.
3. **Given** a deployed agent, **When** a developer creates a session and sends multiple messages, **Then** the agent maintains conversation context across messages within that session.
4. **Given** a deployed agent, **When** a developer uses the TypeScript or Go SDK, **Then** the same capabilities (invoke, stream, session) are available with idiomatic language patterns.
5. **Given** an `.ias` file with typed skill schemas, **When** the SDK is generated, **Then** the generated types match the declared input/output schemas.

---

### User Story 7 - IDE Support for IntentLang (Priority: P5)

As an agent builder, I want syntax highlighting, autocomplete, inline diagnostics, and format-on-save when editing `.ias` files in my IDE, so that I can write agent definitions efficiently with immediate feedback.

**Why this priority**: The CLI provides all validation feedback. IDE integration improves productivity but is not required for the platform to function.

**Independent Test**: Can be tested by installing the VSCode extension, opening an `.ias` file, and verifying that keywords are highlighted, autocomplete suggests valid constructs, and validation errors appear inline.

**Acceptance Scenarios**:

1. **Given** an `.ias` file open in VSCode, **When** the file is loaded, **Then** IntentLang keywords, strings, numbers, and block structure are syntax-highlighted.
2. **Given** an `.ias` file with a validation error, **When** the file is saved, **Then** the error appears as an inline diagnostic with file, line, and column information.
3. **Given** the cursor inside an agent block, **When** the user triggers autocomplete, **Then** valid attributes for agent blocks are suggested (e.g., `model`, `strategy`, `max_turns`).
4. **Given** a reference like `uses prompt "name"`, **When** the user invokes "Go to Definition", **Then** the editor navigates to the prompt definition.
5. **Given** an `.ias` file, **When** the user saves the file with format-on-save enabled, **Then** the file is formatted according to the canonical IntentLang style.

---

### User Story 8 - Production Observability and Operations (Priority: P6)

As an operations engineer, I want deployed agents to emit structured metrics, traces, and logs, and I want secrets to be resolved from secure stores, so that I can run agents reliably in production.

**Why this priority**: Development use is possible without production observability. This is the final priority because it polishes the platform for enterprise use but does not enable new capabilities.

**Independent Test**: Can be tested by deploying an agent, sending requests, and verifying that metrics (request count, latency, token usage) are available at the metrics endpoint and that secrets declared as `env(VAR)` are resolved from actual environment variables.

**Acceptance Scenarios**:

1. **Given** a deployed agent, **When** requests are processed, **Then** the system emits metrics including request count, latency distribution, token consumption, and tool call duration.
2. **Given** a deployed agent with distributed tracing configured, **When** a request is processed, **Then** a trace is generated spanning the full request lifecycle (receive → LLM call → tool calls → response).
3. **Given** a secret declared as `secret "key" { env(VAR) }`, **When** the agent is deployed, **Then** the system reads the environment variable and injects the value into the runtime without exposing it in logs or state files.
4. **Given** a secret declared with a secure store reference, **When** the agent is deployed, **Then** the system resolves the secret from the configured store at startup.
5. **Given** an agent with `token_budget` configured, **When** an invocation approaches the budget, **Then** the system enforces the limit and returns a clear error rather than silently exceeding it.

---

### Edge Cases

- What happens when the LLM API is unreachable during an agent invocation? The system should return an error to the caller with retry guidance, not hang indefinitely.
- What happens when an MCP server crashes mid-tool-call? The system should return a tool error to the LLM so it can decide to retry or inform the user.
- What happens when multiple agents in a pipeline share the same MCP server? The system should reuse connections and avoid resource exhaustion.
- What happens when a deploy target (Docker daemon, Kubernetes cluster) is unavailable? The system should fail fast with a diagnostic message, not leave partial resources.
- What happens when an `.ias` file references a prompt or skill that doesn't exist? The validator should catch this before apply, with "did you mean?" suggestions (existing behavior to preserve).
- What happens when `agentspec apply` is interrupted mid-deployment? The system should leave state consistent so the next `apply` can recover.
- What happens when two `agentspec apply` commands run concurrently against the same package? The system acquires a lock on the state file; the second invocation fails immediately with a lock error and the user retries.
- What happens when a streaming response connection is dropped by the client? The system should cancel the in-progress LLM call and tool executions to avoid wasted cost.
- What happens when concurrent tool calls have different timeouts? Each tool call should respect its own timeout independently, and the overall invocation timeout applies to the full turn.
- What happens when a session reaches its maximum message count? The system should apply the configured memory strategy (sliding window or summarization) transparently.
- What happens when a pipeline step fails while independent parallel steps are running? The system cancels all running steps immediately (fail-fast) to avoid wasting LLM tokens and tool calls on results that will be discarded.

## Requirements *(mandatory)*

### Functional Requirements

#### Runtime & Agentic Loop

- **FR-001**: The system MUST start a single runtime process per package when `agentspec apply` is executed against a valid `.ias` file, serving all agents in that package via path-routed HTTP endpoints (e.g., `/v1/agents/{name}/invoke`).
- **FR-002**: The system MUST implement an agentic loop that iterates through LLM calls and tool executions until the agent produces a final response or a configured limit is reached.
- **FR-003**: The system MUST support a ReAct (reason-act-observe) execution strategy as the default agentic loop pattern.
- **FR-004**: The system MUST support additional execution strategies: plan-and-execute, reflexion, router, and map-reduce.
- **FR-005**: The system MUST call real LLM APIs (starting with Anthropic Claude) to generate agent responses, not mock or simulated outputs.
- **FR-006**: The system MUST support streaming responses, delivering LLM output tokens incrementally to the caller.
- **FR-007**: The system MUST enforce configured limits: maximum turns, timeout per invocation, and token budget.
- **FR-008**: The system MUST support error handling strategies: retry with configurable max retries, fail immediately, or fallback to a designated agent.

#### Tool Execution

- **FR-009**: The system MUST execute skills backed by MCP servers using the MCP protocol (stdio transport at minimum, SSE and streamable-HTTP as additional transports).
- **FR-010**: The system MUST execute skills backed by HTTP APIs, making configured requests and returning response bodies.
- **FR-011**: The system MUST execute skills backed by local commands, spawning subprocesses with input/output capture.
- **FR-012**: When the LLM returns multiple tool calls in a single response, the system MUST execute them concurrently.
- **FR-013**: The system MUST maintain persistent connections to MCP servers (connection pooling) to avoid setup overhead per tool call.
- **FR-014**: The system MUST automatically start MCP servers declared in the `.ias` file when using stdio transport.
- **FR-014a**: The system MUST execute `tool inline` code in a sandboxed subprocess with configurable resource limits (timeout, memory cap, restricted filesystem access).
- **FR-014b**: The system MUST allow users to declare env variables and secrets that are passed to the `tool inline` execution runtime, using the same secret resolution mechanism as deploy blocks.

#### Language (IntentLang 2.0)

- **FR-015**: The parser MUST accept `lang "2.0"` files and recognize all new constructs: `tool` blocks, `deploy` blocks, `pipeline` blocks, agent runtime attributes, prompt `variables`, and `type` definitions.
- **FR-016**: The parser MUST reject `lang "1.0"` files with a message directing users to the `migrate --to-v2` command.
- **FR-017**: The system MUST provide an `agentspec migrate --to-v2` command that rewrites 1.0 syntax to 2.0 syntax automatically.
- **FR-018**: The `execution command "..."` syntax MUST be replaced by `tool mcp`, `tool http`, `tool command`, and `tool inline` blocks.
- **FR-019**: The `binding` block MUST be replaced by `deploy "name" target "type" { ... }` with support for process, docker, docker-compose, and kubernetes targets.
- **FR-020**: Prompts MUST support `{{variable}}` template syntax with a `variables` block for declarations, including types, required flags, and defaults.
- **FR-021**: The `type` keyword MUST allow users to define custom types with fields, enums, lists, and nesting for use in skill input/output schemas.
- **FR-022**: The `pipeline` keyword MUST allow users to define multi-step agent workflows with step ordering, parallel execution, and data flow between steps.
- **FR-023**: The `delegate` keyword MUST allow agents to hand off conversations to other agents based on conditions.

#### Deployment

- **FR-024**: The local process adapter MUST start the runtime as a local process, perform health checks, and record the PID and endpoint in state.
- **FR-025**: The Docker adapter MUST build a container image with the runtime and agent config, start a container, and verify health.
- **FR-026**: The Kubernetes adapter MUST generate deployment manifests, apply them to a cluster, and wait for rollout completion.
- **FR-027**: The Docker Compose adapter MUST generate a multi-service compose file with real images, health checks, and networking, and manage the stack lifecycle.
- **FR-028**: The system MUST provide `agentspec status` to check health of deployed agents across all target types.
- **FR-029**: The system MUST provide `agentspec logs` to stream logs from deployed agents.
- **FR-030**: The system MUST provide `agentspec destroy` to tear down deployed resources with confirmation.

#### Sessions & Memory

- **FR-031**: The system MUST support conversation sessions where context is maintained across multiple messages from the same user.
- **FR-032**: The system MUST support configurable memory strategies: sliding window (fixed message count) and summarization-based compression.
- **FR-033**: Sessions MUST support both in-memory storage (development) and persistent storage (production).

#### Developer Experience

- **FR-034**: The system MUST provide `agentspec run <agent> --input "message"` for one-shot agent invocation from the CLI.
- **FR-035**: The system MUST provide `agentspec dev` for development mode with hot reload on `.ias` file changes.
- **FR-036**: The system MUST provide `agentspec init --template <name>` to scaffold new projects from pre-built templates.
- **FR-037**: The system MUST provide at least 5 pre-built templates: customer-support, rag-chatbot, code-review-pipeline, data-extraction, and research-assistant.
- **FR-038**: The system MUST generate typed SDK clients for Python, TypeScript, and Go that support invoke, stream, and session operations.

#### IDE & Editor Support

- **FR-039**: A VSCode extension MUST provide syntax highlighting for `.ias` files covering keywords, strings, numbers, comments, and block structure.
- **FR-040**: The VSCode extension MUST provide inline diagnostics by running validation on save.
- **FR-041**: The VSCode extension MUST provide autocomplete for keywords, resource types, and cross-references (prompt/skill/agent names).
- **FR-042**: The VSCode extension MUST provide go-to-definition for resource references (e.g., `uses prompt "name"` navigates to the prompt block).

#### Observability & Security

- **FR-043**: The runtime MUST expose a metrics endpoint with request count, latency, token usage, and tool call metrics.
- **FR-044**: The runtime MUST support distributed tracing across the full request lifecycle.
- **FR-045**: The system MUST resolve secrets declared as `env(VAR)` from environment variables at deployment time.
- **FR-046**: The system MUST support resolving secrets from external secure stores (e.g., vault-style key-value stores).
- **FR-047**: Secrets MUST never appear in logs, state files, or exported artifacts.
- **FR-048**: The runtime MUST enforce per-agent rate limits and token budgets.

#### Deprecations

- **FR-049**: The system MUST remove `.az` file support entirely. Files with `.az` extension must be rejected with a migration message.
- **FR-050**: The system MUST remove stub `Apply()` and `Plan()` methods from existing adapters, replacing them with working implementations.
- **FR-051**: The system MUST implement WASM plugin hooks, validators, and transforms that currently return hardcoded success.

### Key Entities

- **Agent**: A defined AI agent with a model, system prompt, set of skills, execution strategy, and runtime limits. The primary deployable unit.
- **Skill**: A capability available to an agent, backed by a concrete tool execution mechanism (MCP, HTTP, command, or inline code). Has typed input/output schemas.
- **Prompt**: A system prompt template with optional variable placeholders. Injected into the agent's LLM context at invocation time.
- **Deploy Target**: A deployment configuration specifying where and how an agent runs (local process, Docker, Kubernetes). Includes resource limits, health checks, and scaling rules.
- **Pipeline**: A multi-step workflow coordinating multiple agents. Steps can run in parallel and pass data to downstream steps.
- **Session**: A stateful conversation context between a caller and an agent. Maintains message history according to a configured memory strategy.
- **Invocation**: A single request-response cycle with an agent. Includes the full agentic loop execution, all tool calls, and token accounting.
- **Type**: A user-defined data type with fields, enums, and lists. Used to declare skill input/output schemas for validation.

## Assumptions

- **Default LLM provider**: The initial implementation targets Anthropic Claude models. OpenAI-compatible models are a follow-on addition.
- **MCP transport priority**: Stdio transport is implemented first as it is the simplest and most common for local development. SSE and streamable-HTTP follow.
- **Default execution strategy**: ReAct is the default strategy when no `strategy` attribute is specified.
- **Authentication**: The runtime HTTP API uses API key authentication by default. More sophisticated auth (OAuth2, mTLS) is deferred.
- **Session storage**: In-memory session storage is the default. Persistent storage (e.g., Redis-backed) is an explicit opt-in.
- **Template registry**: Templates are bundled with the CLI binary initially. A remote registry for community templates is a future consideration.
- **WASM plugins**: The existing wazero-based plugin host is the foundation for custom strategies and hooks. No additional plugin runtimes are planned.
- **Backward compatibility**: IntentLang 1.0 files are not supported in validate/plan/apply. The `migrate --to-v2` command is the only path. Users on 1.0 can use the previous CLI version.
- **Kubernetes access**: The Kubernetes adapter assumes `kubectl` or equivalent client access is configured. The system does not manage cluster provisioning.
- **Runtime process model**: One runtime process per package, serving all agents in that package. Agents within a package share MCP connections, session storage, and the HTTP server. Per-agent isolation is achieved by defining agents in separate packages.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can go from a new `.ias` file to a running, message-responsive agent in a single CLI command (`agentspec apply`).
- **SC-002**: Agents complete invocations with real LLM responses and real tool executions, producing verifiable side effects (e.g., MCP tool calls appear in server logs).
- **SC-003**: The same `.ias` file can be deployed to at least 3 different target environments (local process, Docker, Kubernetes) without modification to the agent definition.
- **SC-004**: 100% of example `.ias` files in the repository are runnable end-to-end with `agentspec dev` or `agentspec apply`.
- **SC-005**: Agent invocations with streaming deliver the first response token to the caller within 2 seconds of the LLM generating it (excluding LLM API latency).
- **SC-006**: Concurrent tool calls complete in `max(individual_latencies)` time, not `sum(individual_latencies)`, verifiable via timing assertions.
- **SC-007**: SDK clients in all 3 supported languages (Python, TypeScript, Go) can invoke an agent, stream a response, and maintain a session.
- **SC-008**: The IntentLang 2.0 migration command successfully converts 100% of valid 1.0 files without manual intervention.
- **SC-009**: The VSCode extension provides syntax highlighting, inline diagnostics, and autocomplete for all IntentLang 2.0 constructs.
- **SC-010**: Deployed agents report accurate metrics: request count, latency percentiles, token usage per invocation, and tool call success/failure rates.
