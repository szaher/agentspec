# Feature Specification: Adoption & Developer Experience

**Feature Branch**: `013-adoption-dev-experience`
**Created**: 2026-03-17
**Status**: Draft
**Input**: User description: "Focus on making AgentSpec installable and usable in minutes. Provide packaged releases, Homebrew install, Docker images, and starter templates. Deliver runnable examples like support bot, RAG assistant, research swarm, incident-response agent, GPU batch agent, multi-agent router."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - One-Command Install (Priority: P1)

A developer discovers AgentSpec and wants to install it immediately on their machine. They run a single command (e.g., `brew install` on macOS or download a binary from the releases page) and have a working `agentspec` CLI within seconds. No Go toolchain, no source compilation, no manual PATH configuration required.

**Why this priority**: Installation is the first interaction any new user has. If installation is difficult or requires compiling from source, the majority of potential adopters will abandon the tool before ever trying it.

**Independent Test**: Can be fully tested by running the install command on a clean machine and verifying `agentspec --version` returns the correct version.

**Acceptance Scenarios**:

1. **Given** a macOS machine without AgentSpec or Go installed, **When** the user runs the Homebrew install command, **Then** the `agentspec` CLI is available on their PATH and `agentspec --version` prints the installed version.
2. **Given** a Linux machine without AgentSpec, **When** the user downloads the release binary for their architecture, **Then** the binary runs without additional dependencies.
3. **Given** a Windows machine without AgentSpec, **When** the user downloads the release archive from the releases page, **Then** the binary runs without additional dependencies.
4. **Given** any supported platform, **When** the user runs `agentspec --help` after install, **Then** they see a summary of all available commands with descriptions.

---

### User Story 2 - Scaffold and Run a First Agent in Under 5 Minutes (Priority: P1)

A new user who just installed AgentSpec wants to create and run their first agent immediately. They run a scaffolding command that generates a complete, runnable project from a starter template. They then run the agent and interact with it — all within 5 minutes of their first install.

**Why this priority**: Time-to-first-success is the strongest predictor of tool adoption. A user who gets a working agent in minutes becomes an advocate; one who spends 30 minutes reading docs before anything works moves on.

**Independent Test**: Can be tested by timing a new user from `agentspec init` to a working agent conversation on a clean install.

**Acceptance Scenarios**:

1. **Given** AgentSpec is installed, **When** the user runs `agentspec init`, **Then** they are prompted to choose a starter template from a list (e.g., "basic chatbot", "support bot", "RAG assistant").
2. **Given** the user selects a template, **When** the scaffolding completes, **Then** a directory is created containing a valid `.ias` file, a README explaining the project, and any required configuration using environment variable references (e.g., `${ANTHROPIC_API_KEY}`) resolved at runtime.
3. **Given** a scaffolded project with an API key configured, **When** the user runs `agentspec run <file>.ias`, **Then** the agent starts and the user can interact with it immediately.
4. **Given** a scaffolded project, **When** the user runs `agentspec validate <file>.ias`, **Then** validation passes with zero errors.

---

### User Story 3 - Run AgentSpec via Docker (Priority: P2)

A developer or platform team wants to run AgentSpec in a containerized environment (CI/CD, Kubernetes, serverless) without installing the binary directly. They pull an official Docker image and use it to validate, plan, apply, or run agents.

**Why this priority**: Container-based workflows are the standard for CI/CD and cloud deployments. Without a Docker image, teams must build custom images, which adds friction and delays adoption in enterprise environments.

**Independent Test**: Can be tested by pulling the Docker image and running `docker run agentspec validate <file>` against a sample `.ias` file.

**Acceptance Scenarios**:

1. **Given** Docker is installed, **When** the user pulls the official AgentSpec image, **Then** the image downloads successfully and is under 50 MB compressed.
2. **Given** the Docker image is available, **When** the user runs a validation command mounting a local `.ias` file, **Then** the validation runs against the mounted file and outputs results to stdout.
3. **Given** the Docker image, **When** the user runs the version command, **Then** the version matches the latest release.
4. **Given** a CI/CD pipeline, **When** the pipeline uses the AgentSpec Docker image as a build step, **Then** all CLI commands (validate, fmt, plan, apply) work without additional setup.

---

### User Story 4 - Browse and Run Curated Examples (Priority: P2)

A developer evaluating AgentSpec wants to see real-world use cases before committing to the tool. They browse a curated set of examples in the repository and documentation, pick one that matches their use case, and run it locally to see it in action.

**Why this priority**: Examples are the most effective documentation for developers. Seeing a working agent for their specific use case (support bot, RAG, multi-agent) removes doubt and demonstrates capability.

**Independent Test**: Can be tested by cloning the repo, navigating to any example directory, and running it end-to-end with a test model.

**Acceptance Scenarios**:

1. **Given** the repository is cloned, **When** the user lists the `examples/` directory, **Then** they find at least 6 examples covering distinct use cases, each with a descriptive name and README.
2. **Given** an example directory, **When** the user reads the example's README, **Then** they find a description, prerequisites, and step-by-step instructions to run the example.
3. **Given** an example project, **When** the user runs `agentspec validate <example>.ias`, **Then** validation passes with zero errors.
4. **Given** an example project with required credentials configured, **When** the user runs `agentspec run <example>.ias`, **Then** the agent starts and behaves as described in the README.

---

### User Story 5 - Starter Template Library (Priority: P2)

A developer starting a new project wants to choose from a variety of starter templates that match common agent patterns. The templates cover progressively complex scenarios — from a simple chatbot to a multi-agent research swarm — so the developer can pick the right starting complexity.

**Why this priority**: Templates reduce boilerplate and teach best practices by example. They bridge the gap between "hello world" and production-ready agents.

**Independent Test**: Can be tested by running `agentspec init --template <name>` for each template and verifying the scaffolded project validates and runs.

**Acceptance Scenarios**:

1. **Given** AgentSpec is installed, **When** the user runs `agentspec init --list-templates`, **Then** they see a list of available templates with short descriptions.
2. **Given** the template list, **When** the user runs `agentspec init --template support-bot`, **Then** a project is scaffolded with a support bot agent, relevant skills, and a system prompt.
3. **Given** the template list, **When** the user runs `agentspec init --template research-swarm`, **Then** a project is scaffolded with multiple coordinating agents, demonstrating multi-agent patterns.
4. **Given** any scaffolded template, **When** the user opens the generated `.ias` file, **Then** it contains inline comments explaining each section and how to customize it.

---

### User Story 6 - Quickstart Guide in Documentation (Priority: P3)

A new user visits the AgentSpec documentation site and finds a prominent quickstart guide that walks them from zero to a running agent. The guide covers install, scaffold, configure, run, and next steps — with copy-pasteable commands for each platform.

**Why this priority**: Documentation is the fallback for every user who hits a snag. A clear quickstart guide reduces support burden and accelerates time-to-success for users who prefer reading over exploration.

**Independent Test**: Can be tested by following the quickstart guide verbatim on a clean machine and verifying each step succeeds.

**Acceptance Scenarios**:

1. **Given** the documentation site, **When** a user navigates to the quickstart page, **Then** they see platform-specific install instructions (macOS, Linux, Windows) with copy-pasteable commands.
2. **Given** the quickstart guide, **When** a user follows all steps in order, **Then** they have a running agent within 5 minutes (excluding download time).
3. **Given** the quickstart guide, **When** a user encounters an error at any step, **Then** the guide includes a troubleshooting section covering common issues (missing API key, wrong architecture, port conflicts).

---

### Edge Cases

- What happens when a user runs `agentspec init` in a directory that already contains an `.ias` file? The system should warn and ask for confirmation before overwriting.
- What happens when the Homebrew formula or Docker image version is behind the latest release? The CLI should be able to check for updates and notify the user.
- What happens when a user selects a template that requires skills or tools not available on their platform? The scaffolded README should clearly list prerequisites.
- What happens when a user tries to run a scaffolded project without configuring required environment variables (API keys)? The runtime should print a clear error message identifying which environment variables are missing and how to set them.
- What happens when `agentspec init` is run without network access? Templates are bundled with the binary, so scaffolding should work fully offline.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide pre-built release binaries for macOS (amd64, arm64), Linux (amd64, arm64), and Windows (amd64) via GitHub Releases.
- **FR-002**: System MUST provide a Homebrew tap that installs the `agentspec` CLI with a single `brew install` command on macOS.
- **FR-003**: System MUST provide an official Docker image published to GitHub Container Registry (ghcr.io) with each release, tagged with semver versions (e.g., `1.2.3`, `1.2`, `1`) and a `latest` tag pointing to the most recent stable release.
- **FR-004**: The Docker image MUST be minimal (under 50 MB compressed) and contain only the `agentspec` binary.
- **FR-005**: System MUST provide an `agentspec init` command that scaffolds a new project from a starter template.
- **FR-006**: The `init` command MUST present an interactive template selector when no `--template` flag is provided.
- **FR-007**: The `init` command MUST support a `--template <name>` flag for non-interactive template selection.
- **FR-008**: The `init` command MUST support a `--list-templates` flag to display available templates with descriptions.
- **FR-009**: System MUST bundle at least 6 starter templates: basic chatbot, support bot, RAG assistant, research swarm, incident-response agent, and multi-agent router.
- **FR-010**: Each starter template MUST produce a project that passes `agentspec validate` without modification (except secret placeholders).
- **FR-011**: Each starter template MUST include a README with description, prerequisites, configuration steps, and run instructions.
- **FR-012**: The `examples/` directory MUST contain at least 6 runnable example projects covering distinct use cases: `support-bot`, `rag-assistant`, `research-swarm`, `incident-response`, `gpu-batch` (dispatches work to external GPU API/service — no local GPU required), and `multi-agent-router`. Example directory names MUST match their corresponding template names where both exist.
- **FR-013**: Each example MUST include a README with description, architecture overview, prerequisites, and step-by-step run instructions.
- **FR-014**: The documentation site MUST include a quickstart guide with platform-specific install instructions and a complete "zero to running agent" walkthrough.
- **FR-015**: The CLI MUST display a helpful getting-started message when run with no arguments, pointing users to `--help`, `init`, and the quickstart guide URL.
- **FR-016**: The `init` command MUST detect existing `.ias` files in the target directory and prompt for confirmation before overwriting.
- **FR-017**: The Homebrew tap MUST be maintained in a dedicated repository following Homebrew conventions.
- **FR-018**: The `agentspec --version` command MUST check for newer releases (via GitHub Releases API) and print an update notice if the installed version is outdated.

### Key Entities

- **Release Artifact**: A pre-built binary for a specific platform and architecture, distributed via GitHub Releases and Homebrew.
- **Docker Image**: A container image containing the `agentspec` binary, published to GitHub Container Registry (ghcr.io) with version tags.
- **Starter Template**: A project scaffold including `.ias` file(s), README, and configuration placeholders. Bundled with the CLI binary for offline use.
- **Example Project**: A complete, runnable agent project in the `examples/` directory demonstrating a specific use case.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new user on macOS, Linux, or Windows can install AgentSpec and run their first agent within 5 minutes (excluding download time and API key setup).
- **SC-002**: All 6 starter templates produce projects that pass validation and can be run with a single command after secret configuration.
- **SC-003**: All example projects in the repository validate successfully and include READMEs that a non-expert user can follow to run them.
- **SC-004**: The Docker image size is under 50 MB compressed and supports all CLI commands without additional setup.
- **SC-005**: The Homebrew install works on both Intel and Apple Silicon Macs with a single command.
- **SC-006**: The quickstart guide in documentation covers all three platforms (macOS, Linux, Windows) with copy-pasteable commands verified to work on clean installs.
- **SC-007**: The `agentspec init` command completes scaffolding in under 3 seconds with no network access required.

## Clarifications

### Session 2026-03-18

- Q: Which container registry should the Docker image be published to? → A: GitHub Container Registry (ghcr.io) only
- Q: What should the "GPU batch agent" example demonstrate? → A: An agent that dispatches batch tasks to an external GPU API/service (no local GPU required)
- Q: Should the CLI include an update-check/notification mechanism? → A: Yes, check only on `agentspec --version` and print a notice if outdated
- Q: What format should template placeholder values use? → A: Environment variable references (`${ANTHROPIC_API_KEY}`) resolved at runtime
- Q: What Docker image tagging strategy should be used on ghcr.io? → A: Semver tags (e.g., `1.2.3`, `1.2`, `1`) plus `latest` tag pointing to most recent stable release

## Assumptions

- GoReleaser is already configured for cross-platform binary builds (added in 012-production-readiness).
- The existing `examples/` directory contains several examples (basic-agent, customer-support, rag-chatbot, etc.) that can be enhanced with better READMEs rather than rebuilt from scratch.
- Templates will be embedded in the CLI binary (using Go's `embed` package) so `agentspec init` works offline.
- The Homebrew tap will be a separate repository that GoReleaser can update automatically on release.
- Docker images will be built as part of the CI/CD release pipeline, not as a separate manual process.
- The "research swarm" and "multi-agent router" examples will use the existing multi-agent and pipeline DSL features.

## Scope Boundaries

**In scope**:
- Pre-built binary releases (macOS, Linux, Windows)
- Homebrew tap and formula
- Docker image published to container registry
- `agentspec init` command with template scaffolding
- 6 starter templates bundled in the CLI
- 6 example projects in the repository (new or enhanced from existing)
- Quickstart documentation page
- Helpful CLI output for new users

**Out of scope**:
- Package managers beyond Homebrew (apt, yum, scoop, chocolatey — future work)
- GUI installer for Windows
- Web-based playground or sandbox environment
- VS Code extension or IDE integration
- Template marketplace or community template registry
- Video tutorials or interactive walkthroughs
