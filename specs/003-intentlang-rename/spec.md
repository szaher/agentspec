# Feature Specification: IntentLang Rename

**Feature Branch**: `003-intentlang-rename`
**Created**: 2026-02-23
**Status**: Draft
**Input**: User description: "change the language name to IntentLang or ilang for short. also, let's name the project as AgentSpec or AgentPack. use .ias as the agent extension."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Rename File Extension from .az to .ias (Priority: P1)

When a developer works with agent definition files, the file extension is `.ias` (IntentLang AgentSpec) instead of `.az`. All tooling (CLI, parser, formatter, validator) recognizes and processes `.ias` files. Existing `.az` files are detected and a migration hint is provided.

**Why this priority**: The file extension is the most visible user-facing change. Every developer interaction starts with a file, and using the wrong extension causes immediate friction. This must be the first thing updated.

**Independent Test**: Rename an existing `.az` file to `.ias`. Run `agentz validate`, `agentz fmt`, `agentz plan`, and `agentz apply` against it. All commands succeed. Run the same commands against a `.az` file — the CLI emits a deprecation warning suggesting migration to `.ias`.

**Acceptance Scenarios**:

1. **Given** a developer has a valid agent definition file with `.ias` extension, **When** they run any CLI command against it, **Then** the command processes it successfully.
2. **Given** a developer has a valid agent definition file with the old `.az` extension, **When** they run any CLI command against it, **Then** the command still processes it but emits a deprecation warning recommending the `.ias` extension.
3. **Given** a developer runs `agentz migrate` in a directory containing `.az` files, **When** the migration runs, **Then** all `.az` files are renamed to `.ias` and a summary is displayed.

---

### User Story 2 - Rename All Example Files and Documentation (Priority: P2)

All example files in the repository are renamed from `.az` to `.ias`. All README files, documentation, and references throughout the codebase are updated to use the new naming conventions: IntentLang (language name), AgentSpec (individual definition file), AgentPack (distributable bundle).

**Why this priority**: Examples and documentation are the primary learning materials. Inconsistent naming between the spec and the actual files creates confusion for new users.

**Independent Test**: Browse the `examples/` directory — all files use `.ias` extension. Open any README — all references say IntentLang, AgentSpec, and `.ias`. Run `agentz validate` and `agentz fmt --check` against all examples — all pass.

**Acceptance Scenarios**:

1. **Given** the repository has been updated, **When** a developer lists files in `examples/*/`, **Then** all definition files have the `.ias` extension.
2. **Given** the repository has been updated, **When** a developer reads any README or documentation, **Then** the language is referred to as IntentLang (or ilang), individual files as AgentSpec files, and bundles as AgentPack.
3. **Given** all examples have been renamed, **When** the CI pipeline runs, **Then** all validation and format-check steps pass on the `.ias` files.

---

### User Story 3 - Update CLI Binary Name and Internal References (Priority: P3)

The CLI binary name, internal string references, help text, error messages, and state file naming are updated to reflect the new branding. The state file changes from `.agentz.state.json` to `.agentspec.state.json`. Plugin directories change from `~/.agentz/plugins/` to `~/.agentspec/plugins/`.

**Why this priority**: Internal references and binary naming are less visible to new users than file extensions and documentation, but must be updated for consistency. The binary rename is the most disruptive change and should be done after the file format changes are stable.

**Independent Test**: Build the CLI — the binary is named `agentspec`. Run `agentspec version` — output shows the new project name. Run `agentspec plan` — the state file created is `.agentspec.state.json`. Install a plugin — it goes to `~/.agentspec/plugins/`.

**Acceptance Scenarios**:

1. **Given** the rename is complete, **When** a developer builds the project, **Then** the output binary is named `agentspec`.
2. **Given** the rename is complete, **When** a developer runs `agentspec apply`, **Then** the state file is written as `.agentspec.state.json`.
3. **Given** a developer has an old `.agentz.state.json` file, **When** they run `agentspec plan`, **Then** the CLI detects the old state file and migrates it to `.agentspec.state.json` with a notification.
4. **Given** the rename is complete, **When** a developer runs `agentspec --help`, **Then** all help text references IntentLang, AgentSpec, and `.ias`.

---

### Edge Cases

- What happens when both `.az` and `.ias` versions of the same file exist in a directory? The CLI should report a conflict error and refuse to process either file until the ambiguity is resolved.
- What happens when a user has an old `.agentz.state.json` and runs the renamed CLI? The CLI should auto-migrate the state file to `.agentspec.state.json` and print a notification.
- What happens when a user has plugins in `~/.agentz/plugins/` and runs the renamed CLI? The CLI should check both old and new plugin directories during a transition period, with a deprecation warning for the old path.
- What happens when CI glob patterns reference `.az` files after the rename? CI must be updated to use `.ias` globs — the CI workflow file must be part of this rename.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CLI MUST accept `.ias` as the primary file extension for IntentLang AgentSpec files.
- **FR-002**: The CLI MUST continue to accept `.az` files with a deprecation warning recommending migration to `.ias`.
- **FR-003**: The `migrate` command MUST rename all `.az` files to `.ias` in the current directory and subdirectories.
- **FR-004**: The `migrate` command MUST update internal references within files if the file extension is referenced in content (e.g., import paths).
- **FR-005**: All example files in `examples/*/` MUST use the `.ias` extension.
- **FR-006**: All README files and documentation MUST use the canonical naming: IntentLang (language), AgentSpec (definition file), AgentPack (bundle).
- **FR-007**: The CLI binary MUST be renamed from `agentz` to `agentspec`.
- **FR-008**: The state file MUST be renamed from `.agentz.state.json` to `.agentspec.state.json`.
- **FR-009**: The CLI MUST auto-migrate old state files (`.agentz.state.json`) to the new name on first run.
- **FR-010**: The plugin directory MUST change from `~/.agentz/plugins/` to `~/.agentspec/plugins/`.
- **FR-011**: The CLI MUST check both old and new plugin directories during a transition period, emitting a deprecation warning for the old path.
- **FR-012**: All CLI help text, error messages, and output MUST reference the new naming (IntentLang, AgentSpec, `.ias`).
- **FR-013**: The CI workflow (`.github/workflows/ci.yml`) MUST be updated to use `.ias` glob patterns and the new binary name.
- **FR-014**: The `.gitignore` MUST be updated to ignore the new binary name (`/agentspec`) instead of `/agentz`.

### Naming Conventions

- **IntentLang** (or **ilang** for short): The declarative language used to write agent definitions
- **AgentSpec**: An individual definition file written in IntentLang (file extension: `.ias`)
- **AgentPack**: A distributable bundle containing one or more AgentSpec files

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All 10 example files use `.ias` extension and pass validation and format-check.
- **SC-002**: The CLI binary is named `agentspec` and all help text references the new naming conventions.
- **SC-003**: Running old `.az` files produces a clear deprecation warning with migration instructions.
- **SC-004**: The `migrate` command successfully renames all `.az` files to `.ias` in a test directory.
- **SC-005**: CI pipeline passes with the updated glob patterns and binary name.
- **SC-006**: Old state files are auto-migrated on first run without data loss.

## Assumptions

- The Go module path (`github.com/szaher/designs/agentz`) remains unchanged — only the binary name changes. A module path rename is a separate, more disruptive change.
- The `cmd/agentz/` directory is renamed to `cmd/agentspec/` as part of the binary rename.
- The `.az` extension remains supported with deprecation warnings for at least one release cycle before removal.
- The constitution file and speckit tooling do not need to be renamed — they are project infrastructure, not user-facing.
