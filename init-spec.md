You are Claude Code acting as a staff-level language designer + platform engineer.

Mission
Design and build a high-level, English-friendly, declarative language (DSL) for packaging, managing, deploying, and operating:
- Agents
- Prompts
- Skills/tools
- MCP servers
- MCP clients
- Platform bindings/adapters (multiple agentic frameworks/providers)

Users express desired state once; it applies idempotently across runs and across platforms.

Core goals
1) Minimal declarative definitions (convention-over-configuration).
2) Human-readable and “English-friendly” syntax (not just low-level YAML/JSON), while remaining machine-parseable and deterministic.
3) Platform-independent intermediate representation (IR) that compiles to multiple targets via adapters.
4) Extension/plugin mechanism for custom logic at:
   - compile-time (transforms, validators, codegen hooks)
   - apply-time (pre/post hooks, policy checks)
   - runtime (custom skill execution, routing, observability)
5) Built-in SDK generation (Python, TypeScript/JS, Go minimum) to:
   - query resources (agents/prompts/skills/servers/clients)
   - resolve endpoints/URIs
   - trigger runs/invocations
   - stream events/logs
6) Strong validation, versioning, and backwards-compatible evolution.

Operating constraints
- Prioritize end-to-end and integration testing over unit tests.
- Prefer a reference implementation that is straightforward to run locally (single binary or simple dev stack).
- Keep security sane by default (authn/authz, secrets handling, least privilege).
- Deterministic behavior: same input => same rendered plans/manifests.

Your first action
Ask me clarifying/qualifying questions that materially change design decisions. Do not start implementation until the questions are answered. Questions must be grouped by theme and limited to the minimum needed.

Then proceed with this workflow
A) Requirements & scope
- Convert goals into explicit functional requirements + non-goals.
- Define target users and 3–5 primary use cases (with “happy path” flows).
- Define success criteria and acceptance tests for MVP.

B) Language design (spec-first)
Deliver an initial spec in /spec/spec.md that includes:
- Design principles and terminology
- Resource model (Agent, Prompt, Skill, MCPServer, MCPClient, Package, Environment, Secret, Policy, Binding, Runtime, Observability, Registry)
- Desired-state semantics (apply/plan/diff/drift)
- Versioning strategy (language versions, package versions, schema evolution)
- Validation model (schema + semantic checks)
- Referencing/imports (packages, registries, local paths, git refs)
- Variables, templates, and safe interpolation rules
- Extension model (plugins): packaging, discovery, capabilities, sandboxing (e.g., WASM), lifecycle
- Compilation model:
  - Source DSL -> canonical AST -> IR -> target manifests/adapters
  - Determinism rules
- Execution model:
  - control plane (optional for MVP) vs local apply
  - state backend options (local file for MVP; pluggable later)
- Security model:
  - secrets, signing, provenance, supply-chain
  - authn/authz for API
- Observability model:
  - events, traces, logs, metrics, run history
- Error model and diagnostics (line/column, suggestions)

C) Syntax proposal (English-friendly)
Propose 2 viable syntaxes, pick 1 for MVP:
Option 1: “Sentence-like” DSL (custom grammar)
Option 2: YAML/JSON compatible surface with English-y keys + optional “sugar” layer
For each option:
- Example configs (at least 6) covering:
  1) Agent referencing a prompt + skills
  2) MCP server definition + transport + auth
  3) Client that consumes an MCP server and exposes skills to an agent
  4) Multi-environment overrides (dev/stage/prod)
  5) Plugin-defined custom resource
  6) Platform binding compiling to at least two targets
- Pros/cons and why MVP chooses one
- Canonical formatting rules (“fmt”) to avoid bikeshedding

D) Architecture & repo plan
Create /ARCHITECTURE.md with:
- Components:
  - Parser + formatter
  - Validator
  - Planner (diff/plan)
  - Applier (idempotent apply)
  - Adapter interface (targets)
  - Plugin host
  - Registry client (optional MVP: local registry)
  - API server (if included) + SDK codegen pipeline
- Data formats:
  - Canonical IR schema (JSON Schema or protobuf)
  - OpenAPI schema for control plane endpoints (if included)
- Threat model summary
- Deployment modes (local CLI; optional daemon)

E) MVP definition
Define MVP to ship in a week of focused engineering:
- Must-have features (small)
- Explicitly deferred features
- A single “golden path” demo scenario

F) Implementation
Implement in a new repo with this layout (adjust if needed):
/cmd/dslc                  # CLI: fmt, validate, plan, apply, export, run
/internal/parser
/internal/ast
/internal/ir
/internal/validate
/internal/plan
/internal/apply
/internal/plugins
/internal/adapters
/adapters/<target1>
/adapters/<target2>
/api (optional)            # control plane server
/sdk/python
/sdk/typescript
/sdk/go
/examples
/integration_tests
/spec

Decide primary language for implementation (Go or Rust preferred for single-binary + SDK tooling friendliness). Justify choice.

G) SDK generation
If API server exists:
- Define endpoints:
  - GET /v1/agents, /v1/agents/{name}
  - GET /v1/prompts, /v1/prompts/{name}
  - GET /v1/skills, /v1/skills/{name}
  - GET /v1/mcp/servers, /v1/mcp/clients
  - POST /v1/runs (invoke agent) + streaming events
- Generate SDKs from OpenAPI (or protobuf/gRPC gateway).
If no API server in MVP:
- SDKs operate on local state + compiled artifacts with a stable library API.

H) Extension/plugin system
Implement a minimal plugin mechanism in MVP:
- Plugin manifest (name, version, capabilities, hooks)
- One example plugin that adds:
  - a custom resource type
  - a compile-time transform
  - a validation rule
Prefer sandboxing (WASM) if feasible; otherwise strict process isolation.

I) Integration tests
Write integration tests that:
- Parse + validate examples
- Plan/diff with deterministic output
- Apply twice with no changes (idempotency)
- Compile to two target adapters and validate produced artifacts
- Exercise plugin loading and hook execution
Avoid unit tests unless they unblock integration tests.

Targets/adapters (pick at least two)
Choose two concrete targets for MVP to prove portability, for example:
- “Generic MCP runtime” (local) and one external agentic platform adapter
- Or two different agent frameworks/providers
Adapters must be thin; most logic stays in IR.

Deliverables checklist (must produce files)
- /spec/spec.md
- /spec/ir.schema.json (or protobuf)
- /ARCHITECTURE.md
- /examples/* (at least 6)
- /integration_tests/* (runnable)
- Working CLI: dslc fmt|validate|plan|apply|export
- At least one plugin + docs
- SDKs (python, ts, go) with minimal examples

Quality bar
- Deterministic formatting and plans
- Clear, stable naming conventions
- Error messages include location + actionable hints
- Documentation is sufficient for a new user to run the demo in <15 minutes

Start now:
1) Ask your clarifying questions.
2) After answers, write the spec and MVP plan.
3) Then implement the repo and ship the demo.
