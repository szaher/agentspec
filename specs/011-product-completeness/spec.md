# Feature Specification: Product Completeness & UX

**Feature Branch**: `011-product-completeness`
**Created**: 2026-03-01
**Status**: Draft
**Input**: Address product/UX gaps from gap analysis: incomplete documentation, non-functional compiler output, missing frontend states, CLI naming confusion, broken eval command, and developer experience improvements.

**Gap Analysis References**: UX-001 through UX-008, GAP-013, GAP-014, GAP-015, GAP-016, GAP-020, GAP-021, GAP-022, BUG-020, BUG-028, FEAT-007, FEAT-008

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Complete CLI Documentation (Priority: P1)

A new user reads the README to understand what AgentSpec can do. All available commands must be documented with descriptions and usage examples so the user can discover and learn the full capability set.

**Why this priority**: 8 of 19 CLI commands are currently undocumented in the README (init, compile, publish, install, eval, status, logs, destroy). Users cannot discover these capabilities, leading to underutilization and confusion.

**Independent Test**: Can be tested by comparing the list of commands in the README against the list of commands registered in the CLI — they must match exactly.

**Acceptance Scenarios**:

1. **Given** the README, **When** a user reads the CLI commands section, **Then** all 19+ commands are listed with descriptions and usage examples.
2. **Given** a command, **When** a user runs `agentspec <command> --help`, **Then** the output matches the README documentation and provides actionable guidance.
3. **Given** a new command added in the future, **When** it is not documented in the README, **Then** CI validation detects the mismatch.

---

### User Story 2 - Functional Compiler Output (Priority: P1)

An engineer compiles their agent to a target framework (e.g., CrewAI, LangGraph). The generated code must be a functional starting point that handles tool invocations, not a file full of "not implemented" stubs.

**Why this priority**: Users compiling to framework targets get code that looks complete but fails at runtime with "not implemented" errors for every tool call. This undermines trust in the compilation feature and wastes time.

**Independent Test**: Can be tested by compiling a sample agent with tools and running the generated code — tool stubs must call the appropriate backend (HTTP, command) rather than returning "not implemented."

**Acceptance Scenarios**:

1. **Given** an agent with an HTTP tool, **When** compiled to a target framework, **Then** the generated code includes a working HTTP client call to the configured URL.
2. **Given** an agent with a command tool, **When** compiled to a target framework, **Then** the generated code includes subprocess execution calling the configured binary.
3. **Given** an agent with an inline tool, **When** compiled to a target framework, **Then** the generated code includes the inline script execution.
4. **Given** a compiled agent, **When** generated code is reviewed, **Then** any remaining stubs are clearly marked with TODO comments explaining what needs customization.

---

### User Story 3 - Polished Frontend Experience (Priority: P2)

An engineer opens the built-in web UI (`agentspec dev --ui`) to interact with their agent. The interface must provide clear feedback during loading, graceful error handling, and helpful guidance when no conversations exist.

**Why this priority**: First impressions matter. The current frontend shows a blank screen during load, no error messages on failure, and no guidance for new users. This creates confusion about whether the system is working.

**Independent Test**: Can be tested by opening the frontend in three states: initial load (loading indicator visible), connection failure (error banner with retry), and empty session (welcome message with instructions).

**Acceptance Scenarios**:

1. **Given** the frontend loading, **When** agents are being fetched, **Then** a loading indicator is displayed.
2. **Given** a server connection failure, **When** the frontend cannot reach the API, **Then** an error banner appears with a retry button.
3. **Given** a new session with no messages, **When** the chat area loads, **Then** a welcome message with usage instructions is displayed.
4. **Given** an agent response containing markdown, **When** rendered in the chat, **Then** formatting is preserved (headings, code blocks, lists).

---

### User Story 4 - Live Agent Evaluation (Priority: P2)

An engineer wants to test their agent's behavior using the `eval` command with actual LLM invocations, not just expression evaluation. They need to verify that the agent produces correct responses for defined test cases.

**Why this priority**: The eval command currently uses a stub invoker that always fails, making it impossible to test agent behavior end-to-end. Engineers must manually deploy and test, which is slow and error-prone.

**Independent Test**: Can be tested by defining eval cases in an .ias file and running `agentspec eval --live` — the agent must produce responses that match expected patterns.

**Acceptance Scenarios**:

1. **Given** an eval case with expected output patterns, **When** `agentspec eval --live` is run, **Then** the agent is invoked with the configured LLM and responses are compared against expectations.
2. **Given** the `--live` flag is not provided, **When** `agentspec eval` is run, **Then** only expression evaluations are executed (existing behavior preserved).
3. **Given** eval results, **When** the command completes, **Then** a summary report shows pass/fail for each test case with expected vs. actual comparison.

---

### User Story 5 - Clear Command Naming (Priority: P3)

An engineer expects `agentspec run` to start a server (similar to `docker run`). The command names must match common conventions and their behavior must be clearly differentiated.

**Why this priority**: `run` does one-shot invocation while `dev` starts a server — the opposite of what most users expect. This causes confusion and wasted time.

**Independent Test**: Can be tested by verifying that command help text clearly explains the behavior, and that the README accurately describes each command's purpose.

**Acceptance Scenarios**:

1. **Given** the `run` command, **When** a user reads its help text, **Then** the description clearly states it performs a one-shot agent invocation (not a server).
2. **Given** the `dev` command, **When** a user reads its help text, **Then** the description clearly states it starts a development server with file watching.
3. **Given** command documentation, **When** comparing `run` vs `dev`, **Then** the differences are explicit and easy to understand.

---

### User Story 6 - Fast Dev Mode File Watching (Priority: P3)

An engineer edits an .ias file while the dev server is running. Changes should be detected and applied within 500ms, not after a 2-second polling interval.

**Why this priority**: The current 2-second polling with directory walking introduces noticeable latency and wastes CPU cycles. Event-based file watching is more responsive and efficient.

**Independent Test**: Can be tested by saving an .ias file change and measuring the time until the dev server reloads — must be under 500ms.

**Acceptance Scenarios**:

1. **Given** a running dev server, **When** an .ias file is saved, **Then** the change is detected within 500ms.
2. **Given** a large project with 100+ .ias files, **When** one file is modified, **Then** only the changed file triggers a reload (not a full directory walk).
3. **Given** a dev server on a platform without event-based watching, **When** the server starts, **Then** it falls back to polling with a message explaining the limitation.

---

### User Story 7 - Honest Feature Flags (Priority: P3)

An engineer uses `agentspec publish --sign` expecting their package to be cryptographically signed. Features that are advertised (via CLI flags) but not implemented must either work or not be visible.

**Why this priority**: Flags that accept input but do nothing create a false sense of security and erode trust. The `--sign` flag prints "not yet implemented" but continues silently.

**Independent Test**: Can be tested by verifying that `--sign` either performs signing or returns an error explaining the feature is not yet available.

**Acceptance Scenarios**:

1. **Given** the `--sign` flag on publish, **When** the feature is not implemented, **Then** the command returns an error (not a warning) and does not publish.
2. **Given** a flag marked as "coming soon" in help text, **When** the user invokes it, **Then** the behavior is clear and the command does not proceed as if the flag worked.

---

### Edge Cases

- What happens when the README documentation validation discovers a new undocumented command? CI fails with a report listing the undocumented commands.
- What happens when a compiler target framework does not support a specific tool type? Generated code includes a clear comment explaining the limitation and suggests alternatives.
- What happens when the frontend loses connection mid-conversation? Existing messages are preserved and a reconnection attempt is made automatically.
- What happens when fsnotify is not available on the platform? Dev mode falls back to polling with a logged message.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: README MUST document all CLI commands with descriptions and usage examples.
- **FR-002**: CI MUST validate that all registered CLI commands are documented in the README.
- **FR-003**: Compiler targets MUST generate functional tool implementations that call the configured backend (HTTP URL, command binary, or inline script).
- **FR-004**: Compiler-generated code MUST clearly mark remaining customization points with TODO comments.
- **FR-005**: Frontend MUST display a loading indicator during initial agent fetch.
- **FR-006**: Frontend MUST display an error banner with retry button on connection failure.
- **FR-007**: Frontend MUST display a welcome message with instructions when no messages exist.
- **FR-008**: Frontend MUST render markdown formatting in agent responses.
- **FR-009**: The `eval` command MUST support a `--live` flag that invokes agents with a real LLM client.
- **FR-010**: The `eval` command MUST produce a summary report with pass/fail per test case.
- **FR-011**: Dev mode MUST detect file changes within 500ms using event-based file watching (with polling fallback).
- **FR-012**: CLI flags for unimplemented features MUST return an error, not silently continue.
- **FR-013**: All command help text MUST clearly describe the command's behavior and differentiate it from similar commands.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of CLI commands documented in README with descriptions and examples.
- **SC-002**: Compiled agent code for all 4 targets produces functional tool calls (not "not implemented" stubs).
- **SC-003**: Frontend displays appropriate state (loading, error, empty) within 200ms of state change.
- **SC-004**: `eval --live` successfully invokes agents and compares against expected outputs.
- **SC-005**: Dev mode file change detection latency is under 500ms on supported platforms.
- **SC-006**: Zero CLI flags exist that accept input but produce no effect.
- **SC-007**: CI documentation validation catches 100% of undocumented commands.

## Assumptions

- Compiler-generated tool implementations will use the same HTTP/command/inline patterns as the AgentSpec runtime, adapted to each target framework's conventions.
- Event-based file watching is available on Linux, macOS, and Windows; polling fallback covers edge cases.
- The `eval --live` command requires the same LLM provider configuration as the `run` command (API keys in environment).
- Markdown rendering in the frontend will use a lightweight library, not a full framework.
- The `--sign` flag will be removed (not implemented) as package signing is deferred to a future feature.
