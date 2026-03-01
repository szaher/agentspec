# Feature Specification: Security Hardening & Compliance

**Feature Branch**: `007-security-hardening`
**Created**: 2026-03-01
**Status**: Draft
**Input**: Address all security findings from gap analysis (SEC-001–SEC-017, related BUG and GAP items). Server will be internet-facing with untrusted .ias files from registries/third parties.

**Gap Analysis References**: SEC-001 through SEC-017, BUG-001, BUG-002, BUG-003, BUG-009, BUG-018, BUG-019, BUG-023, BUG-024, BUG-025, GAP-001, GAP-003, GAP-004, GAP-005, GAP-006, GAP-007, GAP-008, GAP-009, GAP-010, GAP-012, GAP-019

## Clarifications

### Session 2026-03-01

- Q: Which inline tool languages must be sandboxed? → A: All 4 existing languages (Python, Node, Bash, Ruby) with a uniform sandbox mechanism.
- Q: What is the default behavior when no command tool allowlist is configured? → A: Block all command tool execution (secure default). Users must provide an explicit allowlist.
- Q: What policy requirement types must be implemented? → A: 4 types: `pinned imports`, `secret`, `deny command`, and `signed packages`.
- Q: Should the server rate-limit failed authentication attempts? → A: Yes, rate-limit per IP (max 10 failures per minute, then 429 for 5 minutes).
- Q: How should dev mode handle CORS for the built-in UI? → A: Dev mode auto-allows localhost origins for the built-in UI; production requires explicit CORS config.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Secure Session Management (Priority: P1)

An engineer deploys an agent server that is accessible over the internet. Users create sessions to interact with agents. Session identifiers must be cryptographically unpredictable so that an attacker cannot enumerate or hijack another user's session.

**Why this priority**: Session hijacking is the highest-impact vulnerability — it gives an attacker full access to another user's conversation history and agent invocations. The current timestamp-based IDs are trivially guessable.

**Independent Test**: Can be tested by creating 10,000 sessions concurrently and verifying zero collisions and that IDs contain sufficient entropy (>= 128 bits of randomness).

**Acceptance Scenarios**:

1. **Given** a running agent server, **When** two sessions are created within the same nanosecond, **Then** each session receives a unique, unpredictable identifier.
2. **Given** a valid session ID, **When** an attacker increments or decrements the ID, **Then** the system returns "session not found" (no information leakage).
3. **Given** session creation, **When** examining the generated ID, **Then** it contains at least 128 bits of cryptographic randomness (not derived from timestamps).

---

### User Story 2 - API Key Authentication Hardening (Priority: P1)

An engineer configures an API key for their agent server. The authentication mechanism must resist timing attacks, reject requests without valid keys, and warn operators when no key is configured.

**Why this priority**: The runtime server will be internet-facing. API key comparison must be constant-time to prevent character-by-character key discovery. Running without auth must be an explicit, conscious choice.

**Independent Test**: Can be tested by measuring response times for correct vs. incorrect keys of varying prefix lengths — timing variance must be statistically insignificant.

**Acceptance Scenarios**:

1. **Given** a server with API key configured, **When** a request arrives with an incorrect key, **Then** the comparison uses constant-time algorithms and returns 401.
2. **Given** a server started without an API key, **When** the server binds to an address, **Then** a prominent WARNING is logged stating the server is unauthenticated.
3. **Given** a server started without an API key, **When** a request arrives, **Then** access is allowed only if the `--no-auth` flag was explicitly provided at startup; otherwise, all requests are rejected.
4. **Given** a client IP that has failed authentication 10 times in one minute, **When** the next request arrives from that IP, **Then** the server returns 429 Too Many Requests regardless of whether the key is correct, and the IP is blocked for 5 minutes.

---

### User Story 3 - Inline Tool Sandboxing (Priority: P1)

An engineer installs an AgentSpec package from a public registry that includes an inline tool (Python/Node/Bash script). The inline tool must execute in a sandboxed environment that prevents access to the host filesystem, network, and system resources.

**Why this priority**: With untrusted .ias files from third-party registries, inline tool execution without sandboxing enables full host compromise. This is the highest-risk finding in the gap analysis.

**Independent Test**: Can be tested by creating an inline tool that attempts to read `/etc/passwd`, make a network request, or write to `/tmp` — all operations must fail with a sandbox violation error.

**Acceptance Scenarios**:

1. **Given** an inline tool in an installed package, **When** the tool attempts to read files outside its sandbox, **Then** the operation fails with a clear "sandbox violation" error.
2. **Given** an inline tool, **When** the tool attempts to make network connections, **Then** the connection is blocked unless explicitly allowed in the tool's policy.
3. **Given** an inline tool, **When** the tool exceeds its configured memory limit, **Then** execution is terminated with a resource limit error.
4. **Given** server startup, **When** no sandbox backend is available, **Then** inline tools are disabled by default with an error message explaining how to enable them.

---

### User Story 4 - Policy Engine Enforcement (Priority: P1)

An engineer defines security policies in their .ias files (e.g., `require pinned imports`, `deny command rm`). These policies must be enforced during `apply` — violations must block deployment.

**Why this priority**: The policy engine is a critical security feature that currently does nothing. Users defining policies believe they are enforced, creating a false sense of security.

**Independent Test**: Can be tested by defining a `require pinned imports` policy and attempting to apply a file with an unpinned import — the apply must fail.

**Acceptance Scenarios**:

1. **Given** a policy `require pinned imports`, **When** applying a file with an unpinned import, **Then** apply fails with an error listing the unpinned imports.
2. **Given** a policy `deny command rm`, **When** applying a file with a command tool using `rm`, **Then** apply fails with an error citing the denied command.
3. **Given** a policy `require secret api-key`, **When** applying without the referenced secret configured, **Then** apply fails with an error listing the missing secret.
4. **Given** a policy violation, **When** the `--policy=warn` flag is provided, **Then** violations are reported as warnings but apply proceeds.

---

### User Story 5 - HTTP Server Production Hardening (Priority: P2)

An engineer deploys the agent server to a cloud environment accessible from the internet. The server must resist denial-of-service attacks including slow-loris connections, oversized request bodies, and connection exhaustion.

**Why this priority**: An internet-facing server without timeouts, size limits, or proper CORS is vulnerable to trivial DoS attacks that can take it offline.

**Independent Test**: Can be tested by sending a slow-drip request (1 byte per second) and verifying the server times out the connection within the configured ReadHeaderTimeout.

**Acceptance Scenarios**:

1. **Given** a running server, **When** a client sends headers slower than ReadHeaderTimeout, **Then** the connection is closed.
2. **Given** a running server, **When** a request body exceeds the configured limit (default 10MB), **Then** the server returns 413 Request Entity Too Large.
3. **Given** a running server, **When** an idle connection exceeds IdleTimeout, **Then** the connection is closed.
4. **Given** CORS configuration, **When** a cross-origin request arrives from an unlisted origin, **Then** the request is rejected.

---

### User Story 6 - Tool Execution Security (Priority: P2)

An engineer configures command and HTTP tools in their agent. Command tools must only execute binaries from an approved list. HTTP tools must not reach internal networks or cloud metadata endpoints.

**Why this priority**: Without binary allowlists and SSRF protection, a compromised LLM or malicious .ias file can execute arbitrary commands or probe internal infrastructure.

**Independent Test**: Can be tested by configuring a command tool with an unlisted binary — execution must fail. HTTP tool to `169.254.169.254` must be blocked.

**Acceptance Scenarios**:

1. **Given** a command tool configured with binary `curl`, **When** `curl` is not in the allowlist, **Then** execution fails with "binary not in allowlist" error.
2. **Given** an HTTP tool configured with URL `http://169.254.169.254/latest/meta-data/`, **When** the tool executes, **Then** the request is blocked with "SSRF: private network access denied" error.
3. **Given** an HTTP tool, **When** the response body exceeds 10MB, **Then** reading stops at the limit with a truncation warning.
4. **Given** a command tool on the allowlist, **When** it executes, **Then** it inherits a minimal safe environment (PATH, HOME) plus configured secrets.

---

### User Story 7 - Concurrent Access Safety (Priority: P2)

Multiple agents and pipelines run concurrently on the same server. Shared data structures (MCP connection pool, secret redaction filter) must be safe for concurrent access without races or data corruption.

**Why this priority**: Race conditions cause intermittent failures that are difficult to diagnose and can lead to security-relevant issues like using the wrong MCP connection or incomplete secret redaction.

**Independent Test**: Can be tested by running with Go's race detector enabled — all tests must pass without data race warnings.

**Acceptance Scenarios**:

1. **Given** multiple goroutines accessing the MCP connection pool, **When** connections are created, used, and closed concurrently, **Then** no data race occurs and each goroutine gets its own connection.
2. **Given** secrets being added to the redact filter while another goroutine is filtering logs, **When** both operations execute concurrently, **Then** no data race occurs and redaction is complete.
3. **Given** all tests run with `-race` flag, **When** the test suite completes, **Then** zero race conditions are reported.

---

### User Story 8 - Error Transparency (Priority: P3)

When LLM responses contain malformed data or session saves fail, the system must report these errors rather than silently swallowing them. Engineers must be able to diagnose issues from logs.

**Why this priority**: Silent error swallowing causes hard-to-diagnose issues where tool calls are lost, sessions lose messages, or agents produce incorrect results without any visible error.

**Independent Test**: Can be tested by sending a malformed JSON tool input and verifying the error is logged and the agent reports the failure.

**Acceptance Scenarios**:

1. **Given** an LLM response with malformed JSON in a tool call input, **When** the system processes the response, **Then** the error is logged at WARNING level and the tool call is skipped with an error message to the LLM.
2. **Given** a session message save that fails, **When** the save error occurs, **Then** the error is logged at ERROR level and the API response includes a warning header.
3. **Given** a schema marshalling failure in the LLM client, **When** the marshal fails, **Then** the error propagates to the caller rather than producing an empty schema.

---

### Edge Cases

- What happens when the sandbox runtime is not available on the platform? Inline tools are disabled with a clear error message.
- What happens when a policy rule references an unknown requirement type? Apply fails with "unknown policy requirement" error listing supported types.
- What happens when an allowlisted binary does not exist on the system? Command tool fails with "binary not found" error (distinct from "not in allowlist").
- What happens when the CORS origin list is empty in production? Default to rejecting all cross-origin requests. In dev mode, localhost is auto-allowed for the built-in UI.
- What happens when multiple security violations occur in a single apply? All violations are reported together, not just the first one.
- What happens when no command tool allowlist is configured? All command tool execution is blocked with an error explaining how to configure an allowlist.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST generate session IDs using cryptographic randomness with at least 128 bits of entropy.
- **FR-002**: System MUST use constant-time comparison for all API key validation (both auth middleware and runtime server).
- **FR-003**: System MUST log a WARNING at startup when no API key is configured, and MUST require `--no-auth` flag to explicitly allow unauthenticated access.
- **FR-004**: System MUST sandbox inline tool execution for all 4 supported languages (Python, Node, Bash, Ruby) using a uniform mechanism, preventing access to host filesystem (except designated sandbox directory), network (unless explicitly allowed), and system resources.
- **FR-005**: System MUST enforce the configured memory limit for inline tool execution.
- **FR-006**: System MUST implement 4 policy requirement types with actual validation: `pinned imports` (version pins exist), `secret` (named secret is configured), `deny command` (binary blocked), and `signed packages` (imported packages have valid signatures).
- **FR-007**: System MUST support `--policy=warn` mode that reports violations without blocking.
- **FR-008**: System MUST set HTTP server connection timeouts: ReadHeaderTimeout (10s default), ReadTimeout (30s default), IdleTimeout (120s default).
- **FR-009**: System MUST limit request bodies to a configurable maximum (default 10MB) on all API endpoints.
- **FR-010**: System MUST limit HTTP tool response bodies to a configurable maximum (default 10MB).
- **FR-011**: System MUST validate HTTP tool URLs against private/internal IP ranges to prevent SSRF.
- **FR-012**: System MUST validate command tool binaries against a configurable allowlist. When no allowlist is configured, all command tool execution MUST be blocked (secure default).
- **FR-013**: System MUST provide a minimal safe environment (PATH, HOME) to command and inline tools.
- **FR-014**: System MUST make CORS origins configurable; wildcard (`*`) must not be the default. In dev mode (`agentspec dev`), localhost origins MUST be automatically allowed for the built-in UI. In production, explicit CORS configuration is required.
- **FR-015**: System MUST protect shared data structures (MCP pool, RedactFilter) with proper synchronization primitives.
- **FR-016**: System MUST propagate errors from JSON unmarshalling in LLM clients rather than discarding them.
- **FR-017**: System MUST log session save failures at ERROR level rather than discarding them.
- **FR-018**: System MUST capture plugin stdout/stderr into separate buffers rather than passing to host stdout.
- **FR-019**: System MUST use safe serialization for HTTP tool body values to prevent template injection.
- **FR-020**: System MUST rate-limit failed authentication attempts per client IP (max 10 failures per minute), returning 429 Too Many Requests and blocking the IP for 5 minutes after exceeding the threshold.

### Key Entities

- **SessionID**: Cryptographically random identifier for user sessions; prefixed format with sufficient entropy.
- **PolicyRule**: A deny/require rule defined in .ias files; supports 4 requirement types (`pinned imports`, `secret`, `deny command`, `signed packages`); evaluated during apply with actual validation logic.
- **ToolAllowlist**: Configurable list of permitted binary names for command tools; stored in agent or server configuration.
- **CORSConfig**: Origin allowlist for cross-origin requests; defaults to empty (deny all cross-origin).
- **SandboxConfig**: Resource limits and permissions for inline tool execution (memory, filesystem, network); applies uniformly to all 4 supported languages (Python, Node, Bash, Ruby).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Zero session ID collisions across 100,000 concurrent session creations.
- **SC-002**: API key comparison timing variance is statistically insignificant (< 1% variance) across 10,000 measurements with correct vs. incorrect keys of varying prefix lengths.
- **SC-003**: All policy `require` rules produce correct accept/reject results for their documented requirement types.
- **SC-004**: Inline tool attempts to access host filesystem, network, or exceed memory limits are blocked 100% of the time.
- **SC-005**: Server drops slow-loris connections within the configured timeout.
- **SC-006**: SSRF attempts to private IP ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 169.254.0.0/16, 127.0.0.0/8) are blocked 100% of the time.
- **SC-007**: All tests pass with the race detector enabled.
- **SC-008**: Zero silent error discards in LLM client and session save paths — all errors are either returned or logged at WARNING/ERROR level.

## Assumptions

- Inline tool sandboxing uses OS-level process isolation (ulimit/timeout/tmpdir), not the WASM runtime (wazero). Wazero sandboxes compiled WASM plugins; inline tools are native scripts requiring a different isolation approach (see research.md R3).
- The existing constant-time comparison function in the auth package is correct and can be reused in the runtime server.
- Private IP ranges for SSRF protection follow RFC 1918 and RFC 3927 (link-local) standards.
- The command tool allowlist will be configured per-agent or per-server, not globally.
- `--policy=warn` mode is needed for gradual adoption; operators can switch to enforce mode (default) when ready.
