# Feature Specification: AgentSpec Documentation Site

**Feature Branch**: `005-docs-site`
**Created**: 2026-02-23
**Status**: Draft
**Input**: User description: "Create a comprehensive and thorough docs site split between developer docs and user docs (AI engineers, data scientists, ML engineers). Complete reference and guide for the IntentLang programming language covering all directives. Add use-cases covering all agentic architectures, structures, and use-cases. Cover deployment on all different targets and best practices. Publishable on GitHub Pages."

## Clarifications

### Session 2026-02-23

- Q: Should the User Guide include a dedicated HTTP API reference section documenting all runtime endpoints independently from the SDK documentation? -> A: Yes, add a standalone HTTP API reference with endpoint documentation, request/response schemas, authentication details, and curl examples.

## User Scenarios & Testing

### User Story 1 - IntentLang Language Reference (Priority: P1)

An AI engineer new to AgentSpec visits the documentation site and navigates to the IntentLang language reference. They find a complete listing of every keyword, directive, and construct in IntentLang 2.0 with syntax definitions, attribute tables, and annotated examples. They copy an example, adapt it for their use case, and successfully define their first agent.

**Why this priority**: Without a complete language reference, users cannot learn or use IntentLang effectively. This is the foundational documentation that all other content depends on.

**Independent Test**: Can be fully tested by verifying that every IntentLang 2.0 keyword and construct documented in `spec/spec.md` has a corresponding reference page with syntax definition, attribute table, and at least one working example.

**Acceptance Scenarios**:

1. **Given** a user visits the language reference section, **When** they look up any IntentLang 2.0 keyword (e.g., `agent`, `skill`, `tool`, `deploy`, `pipeline`, `type`, `prompt`, `server`, `client`, `policy`, `secret`, `environment`, `plugin`), **Then** they find a dedicated page with syntax definition, attribute descriptions, valid values, and at least one annotated code example. Note: `delegate to agent` is documented within the `agent` reference page.
2. **Given** a user reads a code example in the reference, **When** they copy and paste the example into a `.ias` file, **Then** the file passes `agentspec validate` without errors.
3. **Given** a user searches for a specific construct, **When** they use the site search, **Then** relevant reference pages appear in results within the first 5 entries.

---

### User Story 2 - Getting Started and Tutorials (Priority: P1)

A data scientist who has never used AgentSpec visits the documentation site. They follow a getting-started guide that walks them through installation, creating their first agent definition, validating it, and running it locally. The tutorial takes them from zero to a working agent in a single guided flow.

**Why this priority**: First-time user experience determines adoption. A clear onboarding path is equally critical as the reference for driving usage.

**Independent Test**: Can be fully tested by having a new user follow the getting-started guide end-to-end and successfully run a working agent locally.

**Acceptance Scenarios**:

1. **Given** a new user with no prior AgentSpec experience, **When** they follow the getting-started guide step by step, **Then** they have a validated `.ias` file and understand how to run `agentspec validate`, `agentspec plan`, and `agentspec apply`.
2. **Given** a user completes the getting-started guide, **When** they look for next steps, **Then** they find links to at least 3 tutorials covering different use cases (customer support agent, RAG chatbot, code review pipeline).
3. **Given** a user encounters an error while following a tutorial, **When** they look at the troubleshooting section, **Then** they find the error message documented with an explanation and fix.

---

### User Story 3 - Agentic Architecture Use Cases (Priority: P2)

An ML engineer evaluating AgentSpec for their team needs to understand which agentic patterns are supported and how to implement them. They browse a use-case catalog that covers all major agentic architectures (ReAct, Plan-and-Execute, Reflexion, Router, Map-Reduce, multi-agent pipelines, agent delegation). Each use case includes a problem description, architecture diagram, complete `.ias` example, and deployment instructions.

**Why this priority**: Use cases demonstrate the practical value of the platform. They help users understand whether AgentSpec fits their needs and how to structure their agents.

**Independent Test**: Can be fully tested by verifying that each documented agentic architecture has a complete `.ias` example that passes validation and includes deployment instructions for at least one target.

**Acceptance Scenarios**:

1. **Given** a user visits the use-case catalog, **When** they browse available architectures, **Then** they find documented examples for at least: single-agent ReAct, plan-and-execute, reflexion, router/triage, map-reduce, multi-agent pipeline, and agent delegation.
2. **Given** a user reads a use-case page, **When** they look for implementation guidance, **Then** they find a complete `.ias` file, an architecture diagram, a description of when to use this pattern, and at least one deployment target configuration.
3. **Given** a user wants to compare architectures, **When** they visit the architecture overview page, **Then** they find a comparison table showing trade-offs between strategies (latency, cost, complexity, use case fit).

---

### User Story 4 - Deployment Guide (Priority: P2)

An AI engineer has a working agent definition and needs to deploy it. They find a deployment guide that covers all supported targets (local process, Docker, Docker Compose, Kubernetes) with step-by-step instructions, configuration options, health check setup, scaling configuration, and production best practices.

**Why this priority**: Deployment is the bridge between defining agents and running them. Users need clear guidance for each supported target to get agents into production.

**Independent Test**: Can be fully tested by verifying that each deployment target has a complete guide with prerequisites, configuration reference, step-by-step instructions, and a working example `deploy` block.

**Acceptance Scenarios**:

1. **Given** a user has a working `.ias` file, **When** they navigate to the deployment section, **Then** they find guides for local process, Docker, Docker Compose, and Kubernetes targets.
2. **Given** a user reads a deployment guide for a specific target, **When** they follow the instructions, **Then** the guide includes prerequisites, a complete `deploy` block example, health check configuration, and a verification step.
3. **Given** a user wants to deploy to production, **When** they read the Kubernetes guide, **Then** they find guidance on resource limits, autoscaling, ingress configuration, secret management, and monitoring setup.

---

### User Story 5 - CLI Command Reference (Priority: P2)

A user working with AgentSpec needs to look up the exact syntax and options for a CLI command. They find a CLI reference section with every command (`validate`, `fmt`, `plan`, `apply`, `run`, `dev`, `status`, `logs`, `destroy`, `init`, `migrate`, `export`, `diff`, `sdk`, `version`) documented with usage syntax, all flags/options, examples, and expected output.

**Why this priority**: CLI commands are the primary interface for interacting with AgentSpec. A complete reference prevents users from needing to rely on `--help` output alone.

**Independent Test**: Can be fully tested by verifying that every CLI subcommand has a corresponding reference page with usage syntax, flags, and at least one example.

**Acceptance Scenarios**:

1. **Given** a user wants to use a CLI command, **When** they visit the CLI reference, **Then** they find a page for every `agentspec` subcommand with usage syntax, all available flags, and at least one usage example.
2. **Given** a user reads a CLI command page, **When** they look for output format details, **Then** they find documented output examples for both success and error cases.

---

### User Story 6 - SDK Documentation (Priority: P3)

A developer building an application that interacts with deployed agents needs SDK documentation. They find guides for the Python, TypeScript, and Go SDKs with installation instructions, client initialization, invocation patterns (sync, streaming, sessions), type definitions, and code examples.

**Why this priority**: SDKs enable programmatic integration with deployed agents. This is essential for building production applications but is a secondary concern after users can define and deploy agents.

**Independent Test**: Can be fully tested by verifying that each SDK (Python, TypeScript, Go) has a quickstart, API reference, and at least 3 code examples covering invocation, streaming, and session management.

**Acceptance Scenarios**:

1. **Given** a Python developer, **When** they visit the Python SDK page, **Then** they find installation instructions, client initialization, and code examples for synchronous invocation, async streaming, and session management.
2. **Given** a TypeScript developer, **When** they visit the TypeScript SDK page, **Then** they find equivalent documentation and examples.
3. **Given** a developer reading SDK docs, **When** they look for error handling patterns, **Then** they find documented error types and recommended handling strategies.

---

### User Story 7 - Developer/Contributor Documentation (Priority: P3)

A Go developer wants to contribute to AgentSpec or build custom adapters/plugins. They find developer documentation covering the internal architecture (parser pipeline, IR, adapters, runtime), how to build from source, how to write a custom adapter, how to write a WASM plugin, coding conventions, and how to run tests.

**Why this priority**: Contributor documentation grows the community and enables extensibility. It is important for long-term project health but secondary to user-facing documentation.

**Independent Test**: Can be fully tested by verifying a developer can clone the repo, follow the build guide, run tests, and find architecture documentation for each major internal package.

**Acceptance Scenarios**:

1. **Given** a new contributor, **When** they visit the developer docs, **Then** they find build-from-source instructions, testing guide, and code contribution guidelines.
2. **Given** a developer wanting to write a custom adapter, **When** they read the adapter development guide, **Then** they find the adapter interface definition, step-by-step implementation guide, and a working example adapter.
3. **Given** a developer wanting to write a WASM plugin, **When** they read the plugin development guide, **Then** they find the plugin contract, hook/validator/transform interfaces, and a working example plugin.

---

### User Story 8 - Site Navigation, Search, and Publishing (Priority: P3)

A user of any type needs to find specific information quickly across the documentation site. The site has a clear navigation structure with separate sections for user docs and developer docs, full-text search, and is published on GitHub Pages with automated builds.

**Why this priority**: Discoverability and publishing are infrastructure concerns that enable all other user stories but can use standard documentation tooling defaults initially.

**Independent Test**: Can be fully tested by verifying the site builds successfully, deploys to GitHub Pages, has working navigation between all sections, and search returns relevant results.

**Acceptance Scenarios**:

1. **Given** the documentation source files are in the repository, **When** a CI pipeline runs, **Then** the site builds successfully and deploys to GitHub Pages without manual intervention.
2. **Given** a user visits the site, **When** they navigate between user docs and developer docs, **Then** the navigation clearly separates these audiences with distinct sections and breadcrumbs.
3. **Given** a user searches for a term, **When** search results appear, **Then** results include page titles, section matches, and direct links to the matching content.

---

### Edge Cases

- What happens when a user searches for a deprecated IntentLang 1.0 construct (e.g., `execution command`, `binding`)? The site shows a result explaining the deprecated construct and links to the 2.0 replacement.
- How does the site handle linking to code examples that reference external MCP servers or API keys? Examples clearly mark prerequisites and use placeholder values with explanatory comments.
- What happens when the language spec is updated but documentation lags behind? The automated build pipeline validates all `.ias` examples embedded in documentation pages against the current parser.
- How does the site handle users who arrive from search engines on deep-linked pages? Every page includes breadcrumb navigation and contextual links to prerequisite content.

## Requirements

### Functional Requirements

- **FR-001**: The documentation site MUST have two distinct audience sections: "User Guide" (for AI engineers, data scientists, ML engineers using AgentSpec) and "Developer Guide" (for contributors extending AgentSpec).
- **FR-002**: The User Guide MUST include a complete IntentLang 2.0 language reference with a dedicated page for each resource type (`agent`, `prompt`, `skill`, `tool`, `server`, `client`, `deploy`, `pipeline`, `type`, `secret`, `environment`, `policy`, `plugin`) documenting syntax, attributes, valid values, and annotated examples. See also FR-008 for tool variant coverage.
- **FR-003**: The User Guide MUST include a getting-started guide that walks a new user from installation through defining, validating, and running their first agent.
- **FR-004**: The User Guide MUST include a use-case catalog with at least 7 agentic architecture patterns: single-agent ReAct, plan-and-execute, reflexion, router/triage, map-reduce, multi-agent pipeline, and agent delegation. Each use case MUST include a problem description, architecture diagram, complete `.ias` example, and deployment instructions.
- **FR-005**: The User Guide MUST include deployment guides for all supported targets: local process, Docker, Docker Compose, and Kubernetes. Each guide MUST include prerequisites, configuration reference, step-by-step instructions, and a working `deploy` block example.
- **FR-006**: The User Guide MUST include a CLI command reference with a page for every `agentspec` subcommand, documenting usage syntax, all flags/options, usage examples, and expected output.
- **FR-007**: The User Guide MUST include SDK documentation for Python, TypeScript, and Go, each with installation instructions, client initialization, and code examples covering invocation, streaming, and session management.
- **FR-008**: The User Guide MUST include a tool configuration guide covering all four tool types: `tool mcp`, `tool http`, `tool command`, and `tool inline`, with examples for each.
- **FR-009**: The Developer Guide MUST include architecture documentation covering the parser pipeline (lexer, tokens, parser, AST, IR), validator, formatter, plan engine, adapter system, runtime, agentic loop, and plugin host.
- **FR-010**: The Developer Guide MUST include guides for writing custom adapters and WASM plugins, with interface definitions and working examples.
- **FR-011**: The Developer Guide MUST include build-from-source instructions, test running instructions, and code contribution guidelines.
- **FR-012**: The documentation site MUST be buildable as a static site and deployable to GitHub Pages via a CI pipeline (GitHub Actions workflow).
- **FR-013**: The documentation site MUST include full-text search functionality.
- **FR-014**: The documentation site MUST include navigation that clearly separates user docs from developer docs with breadcrumbs and cross-linking.
- **FR-015**: Every `.ias` code example embedded in documentation MUST be validated against the current parser as part of the build process to prevent documentation drift.
- **FR-016**: The User Guide MUST include an architecture comparison table showing trade-offs between agentic strategies (latency, cost, complexity, recommended use case).
- **FR-017**: The User Guide MUST document prompt template variables, including `{{variable}}` syntax, the `variables` block, `required` and `default` modifiers, with examples.
- **FR-018**: The User Guide MUST document agent runtime configuration attributes: `model`, `strategy`, `max_turns`, `timeout`, `token_budget`, `temperature`, `stream`, `on_error`, `max_retries`, `fallback`.
- **FR-019**: The User Guide MUST document MCP server and client configuration, including `transport`, `command`, `url`, `auth`, and `connects`/`exposes` references.
- **FR-020**: The User Guide MUST document security policies (`policy` blocks), secret management (`secret` blocks), and environment overlays (`environment` blocks).
- **FR-021**: The User Guide MUST include a migration guide for users upgrading from IntentLang 1.0 to 2.0, documenting deprecated constructs and the `agentspec migrate --to-v2` command.
- **FR-022**: The documentation site MUST include a changelog or release notes section that reflects the project's versioned history.
- **FR-023**: The User Guide MUST document pipeline constructs including `step`, `depends_on`, `parallel`, and input/output data flow between steps.
- **FR-024**: The documentation site MUST provide a sitemap and structured metadata for search engine discoverability.
- **FR-025**: The User Guide MUST include a dedicated HTTP API reference section documenting all runtime endpoints (`/healthz`, `/v1/agents`, `/v1/agents/{name}/invoke`, `/v1/agents/{name}/stream`, `/v1/agents/{name}/sessions`, `/v1/pipelines/{name}/run`, `/v1/metrics`), including request/response schemas, authentication requirements (API key via `X-API-Key` header or Bearer token), and curl examples for each endpoint.

### Key Entities

- **Page**: A single documentation page with title, content, audience section (user/developer), navigation breadcrumb, and optional code examples.
- **Code Example**: An IntentLang `.ias` snippet embedded in a page, with syntax highlighting and validation status. Each example is a complete, valid file or a clearly-marked fragment.
- **Use Case**: A documented agentic architecture pattern with problem statement, architecture diagram, `.ias` example, and deployment instructions.
- **Navigation Section**: A grouping of pages under a titled section (e.g., "Language Reference", "Deployment", "SDKs") within either the user or developer audience area.
- **Build Pipeline**: An automated process that compiles documentation source files into a static site, validates embedded examples, and deploys to GitHub Pages.

## Success Criteria

### Measurable Outcomes

- **SC-001**: The documentation site contains at least 40 distinct pages covering all functional requirements listed above.
- **SC-002**: Every IntentLang 2.0 keyword documented in the language specification has a corresponding reference page in the site.
- **SC-003**: 100% of `.ias` code examples embedded in documentation pass `agentspec validate` during the build process.
- **SC-004**: The use-case catalog includes at least 7 agentic architecture patterns, each with a complete, validated `.ias` example.
- **SC-005**: A new user with no prior AgentSpec experience can follow the getting-started guide and have a validated agent definition within 15 minutes.
- **SC-006**: The site builds and deploys to GitHub Pages in under 5 minutes via an automated CI workflow.
- **SC-007**: Every CLI subcommand and every SDK (Python, TypeScript, Go) has a dedicated documentation page.
- **SC-008**: Site search returns relevant results for any IntentLang keyword or CLI command within the first 5 results.
- **SC-009**: The navigation structure provides a maximum of 3 clicks to reach any documentation page from the site homepage.
- **SC-010**: The deployment section covers all 4 supported targets (process, Docker, Compose, Kubernetes) with production-ready configuration examples.

## Assumptions

- The documentation site generator will be a standard static site tool (e.g., Hugo, MkDocs, Docusaurus). The specific tool choice is a planning-phase decision.
- Code examples will be extracted from or aligned with the existing `examples/` directory in the repository.
- Architecture diagrams will use text-based formats (Mermaid or ASCII) to remain version-controllable and avoid external image dependencies.
- The GitHub Pages deployment will use the `gh-pages` branch or GitHub Actions artifact-based deployment.
- The documentation will be written in Markdown as the source format.
- The automated validation of `.ias` examples assumes the `agentspec` CLI binary can be built during the documentation CI pipeline.
- SDK documentation will be written manually (not auto-generated from code comments) for the initial version, since SDKs are template-generated and may evolve.
- The site will initially be a single version (latest). Versioned documentation is out of scope for the initial release but can be added later.
