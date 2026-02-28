# Feature Specification: Agent Compilation & Deployment Framework

**Feature Branch**: `006-agent-compile-deploy`
**Created**: 2026-02-28
**Status**: Draft
**Input**: User description: "Package agents into deployable formats (Docker, K8s, standalone, local). Compile declarative .ias into working agents. Support imports, control flow, compilation to framework code (CrewAI, LangGraph, LlamaStack, LlamaIndex). Ship built-in agent frontend. Build own ecosystem distinct from Terraform."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Compile Agent to Standalone Service (Priority: P1)

A developer writes a declarative `.ias` file that defines an agent — its type, reasoning loop, tools, validation rules, and evaluation criteria. They run a single compile command and receive a self-contained executable service that implements the described agent. The compiled agent runs as a standalone process, accepting input and producing output without requiring the AgentSpec CLI at runtime.

**Why this priority**: This is the foundational capability. Without compilation, nothing else in this feature works. Converting declarative definitions into runnable agents is the core value proposition — it transforms AgentSpec from an infrastructure tool into an agent development platform.

**Independent Test**: Can be fully tested by compiling a simple agent `.ias` file, running the resulting executable, sending it a message, and verifying a valid agent response. Delivers an immediately usable agent from a declarative definition.

**Acceptance Scenarios**:

1. **Given** a valid `.ias` file defining an agent with a prompt, model, and skills, **When** the user runs the compile command, **Then** the system produces a runnable executable and reports compilation success with output path and size.
2. **Given** a compiled agent executable, **When** the user runs it, **Then** the agent starts, accepts input through its defined interface, processes it using the specified reasoning loop, and produces output.
3. **Given** an `.ias` file with syntax or semantic errors, **When** the user runs the compile command, **Then** the system reports clear, actionable error messages with file location and suggestions, and produces no output artifact.
4. **Given** an `.ias` file referencing undefined skills or invalid configurations, **When** the user compiles, **Then** the system catches all reference errors during compilation (not at runtime).

---

### User Story 2 - Enhanced IntentLang with Imports and Control Flow (Priority: P2)

A developer building a complex agent system wants to split their definitions across multiple `.ias` files for reuse and organization. They use `import` statements to reference other `.ias` files and published packages. They also use conditional logic and loops within their agent definitions to handle branching behavior (e.g., route different input types to different skills, iterate over a list of data sources).

**Why this priority**: Imports and control flow are language-level features that unlock composability and expressiveness. Without them, users are limited to flat, single-file agents with no reuse. This is the prerequisite for building a real ecosystem of shareable agent components.

**Independent Test**: Can be tested by creating a multi-file project where one `.ias` file imports another, uses conditionals to route between two different agent behaviors, and validates that both the import resolution and conditional execution work correctly.

**Acceptance Scenarios**:

1. **Given** a project with `main.ias` importing `skills/search.ias`, **When** the user compiles `main.ias`, **Then** the system resolves the import, merges the skill definitions, and produces a working agent with all imported capabilities.
2. **Given** an `.ias` file with a conditional block (e.g., route to different skills based on input category), **When** the agent processes input matching each condition, **Then** the correct branch executes for each case.
3. **Given** an `.ias` file with a loop construct iterating over a list of data sources, **When** the agent runs, **Then** it processes each data source in sequence and aggregates results.
4. **Given** an import referencing a non-existent file or circular dependency, **When** the user compiles, **Then** the system reports the specific import error with the dependency chain.

---

### User Story 3 - Framework Code Generation via Compilation Plugins (Priority: P3)

A developer has existing infrastructure built on a specific agentic framework (CrewAI, LangGraph, LlamaStack, or LlamaIndex). Instead of running AgentSpec's own runtime, they want to compile their `.ias` file into native code for their preferred framework. They select a compilation target, and AgentSpec generates well-structured, idiomatic code for that framework that implements the same agent behavior described in the `.ias` file.

**Why this priority**: Framework code generation makes AgentSpec a universal agent definition language rather than a closed ecosystem. It dramatically lowers adoption barriers — teams don't have to commit to AgentSpec's runtime to benefit from its declarative approach. However, it depends on the core compilation framework (P1) being in place.

**Independent Test**: Can be tested by compiling a standard agent `.ias` file to a specific framework target (e.g., CrewAI), inspecting the generated code for correctness and idiomatic patterns, and optionally running the generated code in that framework.

**Acceptance Scenarios**:

1. **Given** a valid `.ias` agent definition and a compilation target of "crewai", **When** the user compiles with the target flag, **Then** the system generates a complete, runnable CrewAI project with correct agent, task, and crew definitions.
2. **Given** an `.ias` file with features not supported by the target framework, **When** the user compiles, **Then** the system warns about unsupported features, generates code for supported features, and documents workarounds or manual steps needed.
3. **Given** a compilation plugin for a new framework target, **When** the user installs the plugin and compiles, **Then** the system uses the plugin to generate code for that framework target.
4. **Given** an `.ias` file previously compiled to CrewAI code, **When** the user modifies the `.ias` file and recompiles, **Then** only the changed portions of the generated code are updated (preserving any manual edits in designated safe zones).

---

### User Story 4 - Multi-Format Deployment Packaging (Priority: P4)

A developer has compiled their agent and wants to deploy it. They choose a deployment target — Docker container, Kubernetes cluster, or standalone binary for a specific OS — and the system packages the compiled agent with all necessary runtime dependencies, configuration, and deployment manifests into a ready-to-deploy artifact.

**Why this priority**: Deployment packaging bridges the gap between a compiled agent and a production deployment. It builds on the compilation output (P1) and makes agents operationally viable. It is lower priority than compilation and language features because existing adapter infrastructure partially covers this.

**Independent Test**: Can be tested by compiling an agent, packaging it as a Docker image, running the container, and verifying the agent responds correctly.

**Acceptance Scenarios**:

1. **Given** a compiled agent, **When** the user packages it for Docker, **Then** the system produces a container image with the agent, its dependencies, and a health check endpoint, and the image runs correctly.
2. **Given** a compiled agent, **When** the user packages it for Kubernetes, **Then** the system generates deployment manifests (deployment, service, config) and optionally a Helm chart, ready for `kubectl apply`.
3. **Given** a compiled agent, **When** the user packages it as a standalone binary for a specific OS/architecture, **Then** the system produces a single executable file that runs without external dependencies on the target platform.
4. **Given** a compiled multi-agent pipeline, **When** the user packages for deployment, **Then** all agents in the pipeline are packaged together with correct inter-agent communication configuration.

---

### User Story 5 - Built-in Agent Frontend (Priority: P5)

A user (developer, tester, or end user) wants to interact with a deployed agent without building a custom UI. AgentSpec ships a built-in web frontend that automatically connects to any running agent. The frontend supports chat-style interaction, displays agent reasoning activity and tool usage in real time, allows users to provide structured input, and presents agent output in a readable format.

**Why this priority**: The frontend is a developer experience and usability feature. It's essential for adoption and testing but not a prerequisite for the core compilation/deployment pipeline. Agents are fully functional without it.

**Independent Test**: Can be tested by starting a compiled agent with the frontend enabled, opening the frontend in a browser, sending a chat message, and verifying the response appears along with reasoning trace.

**Acceptance Scenarios**:

1. **Given** a running agent with the frontend enabled, **When** a user opens the frontend URL in a browser, **Then** they see a chat interface with the agent's name, description, and an input field.
2. **Given** an active chat session, **When** the user sends a message, **Then** the agent's response streams in real-time, and the reasoning steps (tool calls, intermediate thoughts) are visible in a collapsible activity panel.
3. **Given** an agent with structured input requirements, **When** the user interacts with the frontend, **Then** the frontend renders appropriate input controls (text fields, dropdowns, file uploads) based on the agent's input schema.
4. **Given** multiple agents running in the same deployment, **When** the user opens the frontend, **Then** they can switch between agents and see each agent's conversation history independently.

---

### User Story 6 - AgentSpec Ecosystem & Package Registry (Priority: P6)

A developer wants to share their reusable agent components (skills, prompts, tool definitions, agent templates) with the community or within their organization. They publish packages to a registry and other developers can import them by name and version in their `.ias` files. AgentSpec manages dependency resolution, version compatibility, and package caching.

**Why this priority**: The ecosystem and registry are the long-term moat for AgentSpec but depend on imports (P2) and compilation (P1) being solid first. This can start with a simple local/file-based registry and evolve.

**Independent Test**: Can be tested by publishing a skill package to a local registry, importing it in another `.ias` file by name and version, compiling successfully, and verifying the imported skill is available at runtime.

**Acceptance Scenarios**:

1. **Given** a developer with a reusable skill `.ias` file, **When** they run the publish command with a version tag, **Then** the package is published to the configured registry with metadata (name, version, description, dependencies).
2. **Given** an `.ias` file importing a published package by name and version, **When** the user compiles, **Then** the system resolves the package from the registry, downloads it if not cached, and makes its definitions available.
3. **Given** two packages with conflicting dependency versions, **When** the user compiles, **Then** the system reports the version conflict with both dependency chains and suggests resolution options.
4. **Given** a published package with a security vulnerability, **When** the registry maintainer marks it as deprecated, **Then** users who depend on it receive a warning during compilation.

---

### Edge Cases

- What happens when an `.ias` file imports a package that in turn imports an incompatible version of a shared dependency? The system must detect diamond dependency conflicts and report them clearly.
- How does the system handle compilation when a skill references an external tool (HTTP endpoint, MCP server) that is unavailable at compile time? The system should validate the tool definition structure but defer connectivity checks to runtime.
- What happens when a user compiles an agent targeting a framework (e.g., LangGraph) that doesn't support a feature used in the `.ias` file (e.g., a specific loop strategy)? The system must warn about unsupported features and either generate a best-effort equivalent or fail with a clear explanation.
- How does the system handle circular imports (A imports B, B imports A)? The compiler must detect cycles during import resolution and report the full cycle path.
- What happens when a compiled agent's input schema changes between versions? The frontend must gracefully handle schema mismatches and prompt users to refresh.
- How does the system behave when compiling for a target OS/architecture different from the host? Cross-compilation must either succeed or clearly state which targets are supported from the current host.
- What happens when a conditional block in an `.ias` file has no matching branch for a given input? The system should require an explicit default/fallback branch or report a warning during compilation.
- What happens when an agent response fails all validation retry attempts? The system must return the best available response along with the validation failure details rather than silently swallowing the output.
- How are evaluation test cases handled when the agent's LLM responses are non-deterministic? Evaluation scoring must support fuzzy matching, semantic similarity, or custom scoring functions rather than requiring exact output matches.
- What happens when a compiled agent starts with some required configuration missing? The agent must fail fast at startup, listing all missing parameters in a single error message rather than failing on the first request that needs them.
- What happens when an API key is not configured but auth is not explicitly disabled? The agent must refuse to start and report that either an API key must be set or auth must be explicitly disabled, preventing accidental open deployments.
- What happens when a runtime control flow expression references a property that doesn't exist on the current input (e.g., `input.category` when input has no `category` field)? The expression must evaluate to a well-defined null/missing value rather than crashing, and the default/else branch should handle it.

## Requirements *(mandatory)*

### Functional Requirements

#### Compilation Core

- **FR-001**: System MUST accept an `.ias` file (or directory of `.ias` files) as input and produce a compiled agent artifact as output.
- **FR-002**: System MUST validate all agent definitions, references, and type correctness at compile time, rejecting invalid inputs with clear error messages before producing any artifact.
- **FR-003**: System MUST support compiling a single agent, multiple agents, or a full pipeline from one or more `.ias` files into a unified artifact.
- **FR-004**: System MUST produce deterministic compilation output — the same input files must always produce byte-identical artifacts.
- **FR-005**: System MUST report compilation progress, warnings, and errors in a structured format suitable for both human and machine consumption.

#### IntentLang Extensions

- **FR-006**: IntentLang MUST support `import` statements to include definitions from other `.ias` files using relative paths (e.g., `import "./skills/search.ias"`).
- **FR-007**: IntentLang MUST support importing published packages by name and version using full host paths (e.g., `import "github.com/agentspec/web-tools" version "1.2.0"`).
- **FR-008**: IntentLang MUST support conditional logic (`if`/`else if`/`else`) within agent definitions to enable branching behavior based on runtime values: current request input, agent memory/session state, and tool outputs.
- **FR-009**: IntentLang MUST support loop constructs (`for each`) to iterate over runtime collections such as tool results, user-provided lists, and session data.
- **FR-047**: Control flow expressions MUST use a simple, sandboxed syntax limited to property access, comparisons, boolean logic, and type checks — not a full programming language.
- **FR-048**: The compiler MUST validate control flow expressions at compile time for syntactic correctness and type safety where determinable, deferring runtime-dependent checks to execution.
- **FR-010**: The compiler MUST detect and reject circular imports, reporting the full dependency cycle to the user.
- **FR-011**: IntentLang MUST require a default/fallback branch in conditional blocks, or emit a compilation warning if one is absent.

#### Compilation Targets & Plugins

- **FR-012**: System MUST support a default compilation target that produces a standalone, self-contained agent service.
- **FR-013**: System MUST support a plugin-based compilation architecture where third-party plugins can register new compilation targets.
- **FR-014**: System MUST ship with at least one framework compilation plugin as a reference implementation.
- **FR-015**: Compilation plugins MUST receive a well-defined intermediate representation (IR) of the agent and produce framework-specific source code.
- **FR-016**: System MUST warn users when an `.ias` feature is not supported by the selected compilation target and explain the gap.

#### Security & Authentication

- **FR-042**: Compiled agents MUST require API key authentication for all endpoints by default.
- **FR-043**: The API key MUST be provided via runtime configuration (environment variable or config file), not baked into the compiled artifact.
- **FR-044**: The built-in frontend MUST use the same API key mechanism, auto-injecting the key into requests via session management so users are not prompted for a key on every interaction.
- **FR-045**: System MUST support disabling authentication via an explicit flag or configuration option for local development use.
- **FR-046**: Unauthenticated requests to a secured agent MUST receive a clear error response indicating authentication is required, without leaking agent details.
- **FR-050**: Compiled agents MUST support configurable rate limiting on API endpoints to prevent abuse. Default limits MUST be documented and overridable via runtime configuration.

#### Runtime Configuration

- **FR-037**: IntentLang MUST support declaring required configuration parameters within agent definitions, including name, type, description, and optional default value.
- **FR-038**: Compiled agents MUST resolve configuration values from environment variables and/or configuration files at startup, not at compile time.
- **FR-039**: Compiled agents MUST fail fast at startup with a clear error listing all missing required configuration parameters rather than failing at first use.
- **FR-040**: Compiled artifacts MUST be portable — the same artifact MUST run in any environment by providing the appropriate configuration values, without recompilation.
- **FR-041**: System MUST generate a configuration reference (listing all required and optional parameters) as part of the compilation output.

#### Deployment Packaging

- **FR-017**: System MUST support packaging compiled agents as container images ready for container runtimes.
- **FR-018**: System MUST support packaging compiled agents as standalone executables for major operating systems and architectures.
- **FR-019**: System MUST support generating orchestration manifests for container orchestration platforms.
- **FR-020**: System MUST support running compiled agents as local processes for development and testing.
- **FR-021**: Packaged agents MUST include health check endpoints and graceful shutdown handling.

#### Built-in Frontend

- **FR-022**: System MUST ship a built-in web frontend that provides chat-style interaction with any running agent.
- **FR-023**: The frontend MUST display agent reasoning activity (tool calls, intermediate steps) in real-time during processing.
- **FR-024**: The frontend MUST render appropriate input controls based on the agent's declared input schema.
- **FR-025**: The frontend MUST support multiple concurrent agent sessions and conversation history.
- **FR-026**: The frontend MUST be embeddable — deployable alongside the agent with zero additional infrastructure.

#### Agent Validation & Evaluation

- **FR-031**: IntentLang MUST support declaring output validation rules within agent definitions, including format checks, schema compliance, and content guardrails.
- **FR-032**: Compiled agents MUST automatically execute declared validation rules on every agent response before returning it to the caller.
- **FR-033**: When a response fails validation, the system MUST either retry with the validation feedback (up to a configurable limit) or return the failure with a clear explanation of which rules were violated.
- **FR-034**: IntentLang MUST support declaring evaluation test cases as golden input/output pairs within agent definitions or in companion `.ias` files.
- **FR-035**: System MUST provide a batch evaluation command that runs all declared test cases against the agent and produces a quality score report.
- **FR-036**: Evaluation reports MUST include per-test-case pass/fail status, overall score, and comparison against previous evaluation runs when available.

#### Ecosystem & Registry

- **FR-027**: System MUST support publishing `.ias` packages to a registry with name, version, and metadata.
- **FR-028**: System MUST resolve package dependencies transitively during compilation, downloading missing packages from the configured registry.
- **FR-029**: System MUST cache downloaded packages locally to avoid redundant network requests.
- **FR-030**: System MUST detect and report version conflicts in transitive dependency graphs.
- **FR-049**: System MUST design package signing and provenance as first-class concepts. Published packages MUST include a signature field and verification endpoint. The MVP MAY stub the signing implementation with a placeholder that returns "unsigned" but the data structures and verification flow MUST exist.

### Key Entities

- **AgentSpec Package**: A versioned collection of `.ias` files forming a distributable unit. Has a name, version, description, author, and dependency list.
- **Compilation Target**: A named output format the compiler can produce. Includes the default (standalone service) and framework-specific targets (CrewAI, LangGraph, etc.). Each target is backed by a compilation plugin.
- **Compilation Plugin**: An extension module that transforms the agent IR into target-specific output. Has a name, supported features list, and code generation logic.
- **Compiled Artifact**: The output of compilation — either a standalone executable, framework source code, or an intermediate bundle. Has a target platform, size, and content hash.
- **Agent Frontend Session**: A user's interaction session with an agent through the built-in frontend. Contains conversation history, input/output records, and agent activity traces.
- **Package Registry**: A storage and discovery service for published AgentSpec packages. Supports name-based lookup, version resolution, and dependency metadata.

## Clarifications

### Session 2026-02-28

- Q: What do "validation" and "eval" mean for agents? → A: Validation = declarative output validation rules (format checks, schema compliance, guardrails) that run on every agent response. Eval = lightweight evaluation framework where agents define test cases with golden input/output pairs, runnable as a batch command to score agent quality.
- Q: How do compiled agents receive runtime configuration (API keys, model endpoints, secrets)? → A: Runtime injection. Compiled agents read configuration from environment variables and/or config files at startup. The `.ias` file declares what configuration the agent needs (names, types, defaults), but values are provided at runtime. Compiled artifacts are portable across environments.
- Q: What security/auth model for agent endpoints and frontend? → A: API key authentication, secure by default. Compiled agents require an API key for endpoint access. The key is provided via runtime configuration. Frontend uses the same key (auto-injected via session). Auth can be disabled via a flag for local development.
- Q: What can control flow constructs (if/else, for each) evaluate? → A: Runtime input and state. Conditions can inspect current request input, agent memory/session state, and tool outputs. Loops can iterate over runtime collections (e.g., search results, user-provided lists). Expressions use a simple, sandboxed syntax — not a full programming language.

## Assumptions

- The existing IR (Intermediate Representation) from the current AST → IR pipeline will serve as the input to the compilation framework, extended as needed for new language features.
- The existing WASM-based plugin system will be extended to support compilation plugins, maintaining the same sandboxing and isolation model.
- The existing deployment adapters (docker, kubernetes, compose, process) will be refactored to consume compiled artifacts instead of IR directly, maintaining backward compatibility during transition.
- Framework code generation (P3) will produce "ejectable" code — once generated, users own the output and can modify it freely. AgentSpec does not maintain a live link to the generated code.
- The built-in frontend is a single-page application bundled into the compiled agent binary, served on the same port as the agent's API.
- The package registry will initially support a local filesystem registry and a simple HTTP-based remote registry. Integration with existing package managers (npm, pip) is out of scope.
- Cross-compilation for standalone binaries will initially target the most common OS/architecture combinations: Linux (amd64, arm64), macOS (amd64, arm64), and Windows (amd64).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can go from a new `.ias` file to a running compiled agent in under 5 minutes, including compilation, packaging, and first interaction.
- **SC-002**: Compilation of a typical single-agent `.ias` file (under 500 lines) completes in under 10 seconds on standard developer hardware.
- **SC-003**: At least 4 framework compilation targets are available (CrewAI, LangGraph, LlamaStack, LlamaIndex), each generating runnable code that passes the target framework's own validation.
- **SC-004**: The compiled standalone agent binary starts and is ready to accept requests in under 3 seconds.
- **SC-005**: An agent defined across 5+ imported `.ias` files compiles successfully with correct dependency resolution and produces identical behavior to a single-file equivalent.
- **SC-006**: 90% of users can successfully chat with a deployed agent through the built-in frontend on their first attempt without documentation.
- **SC-007**: Published packages can be imported and resolved from the registry in under 5 seconds for packages under 10MB.
- **SC-008**: Compiled container images are under 100MB for a typical single-agent deployment, ensuring fast pull and startup times.
