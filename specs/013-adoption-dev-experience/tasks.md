# Tasks: Adoption & Developer Experience

**Input**: Design documents from `/specs/013-adoption-dev-experience/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Not explicitly requested. Integration validation via CI (`agentspec validate` on all templates and examples).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: No new project initialization needed — this feature modifies an existing Go CLI project. Setup phase covers only the template system restructure that blocks multiple user stories.

- [x] T001 Restructure template embedding from flat files to directories: change `//go:embed files/*.ias` to `//go:embed files/*` in `internal/templates/templates.go`, then move each existing `.ias` file into its own subdirectory (e.g., `internal/templates/files/customer-support/customer-support.ias`)
- [x] T002 Update `Template` struct in `internal/templates/templates.go`: add `Category string` field (values: `beginner`, `intermediate`, `advanced`) and `Dir string` field; update `All()` to use directory-based paths; update `Content()` to read from `files/<dir>/<filename>`; add new `ScaffoldDir()` func that copies all files from a template directory to a target path
- [x] T003 Update `cmd/agentspec/init.go` to use directory-based scaffolding: replace single-file write with `ScaffoldDir()` call that creates `<name>/` directory containing `.ias` file + `README.md`; update "next steps" output to show new file paths

**Checkpoint**: `go build ./...` and `go test ./... -count=1` pass; existing `agentspec init --template customer-support` still works (now scaffolds a directory instead of a single file)

---

## Phase 2: User Story 1 — One-Command Install (Priority: P1)

**Goal**: Enable installing AgentSpec via `brew install` on macOS and downloading pre-built binaries on all platforms.

**Independent Test**: Run `goreleaser release --snapshot --clean` locally and verify archives for all 5 platform/arch combos are produced; verify Homebrew formula is generated.

### Implementation for User Story 1

- [x] T004 [US1] Add `brews` section to `.goreleaser.yaml` targeting repository `szaher/homebrew-agentspec` with formula name `agentspec`, homepage, description, license `Apache-2.0`, and test block `system "#{bin}/agentspec", "version"` per `specs/013-adoption-dev-experience/contracts/homebrew-tap.md`
- [x] T005 [US1] Verify GoReleaser config is valid by running `goreleaser check` and `goreleaser release --snapshot --clean` locally; confirm archives for darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64 are produced and Homebrew formula is generated in `dist/`

**Checkpoint**: `goreleaser check` passes; snapshot release produces correct archives and Homebrew formula

---

## Phase 3: User Story 2 — Scaffold and Run First Agent in Under 5 Minutes (Priority: P1)

**Goal**: `agentspec init` presents an interactive template selector, scaffolds a project directory with `.ias` + README, and the scaffolded project validates immediately.

**Independent Test**: Run `agentspec init` (no flags) → select template 1 → verify directory created with `.ias` + README → run `agentspec validate` on the `.ias` file.

### Implementation for User Story 2

- [x] T006 [US2] Add interactive template selector to `cmd/agentspec/init.go`: when no `--template` flag is provided, print numbered list grouped by category (Beginner/Intermediate/Advanced) per `specs/013-adoption-dev-experience/contracts/cli-init.md`, read selection from stdin via `bufio.Scanner`, map number to template name
- [x] T007 [US2] Add overwrite detection to `cmd/agentspec/init.go`: before scaffolding, check if target directory or `.ias` file already exists; if so, print warning and prompt `Overwrite? [y/N]:` reading from stdin; abort on `N` or empty input (FR-016)
- [x] T008 [US2] Rename `--list` flag to `--list-templates` in `cmd/agentspec/init.go` for consistency with spec (FR-008); update the output format to show category labels per `specs/013-adoption-dev-experience/contracts/cli-init.md`
- [x] T009 [US2] Update "next steps" output in `cmd/agentspec/init.go`: after scaffolding, print the created files list, required environment variables extracted from template, and validate/run commands with correct paths per `specs/013-adoption-dev-experience/contracts/cli-init.md`

**Checkpoint**: `agentspec init` (interactive), `agentspec init --template basic-chatbot`, `agentspec init --list-templates` all work correctly; overwrite prompt works; scaffolded `.ias` files pass `agentspec validate`

---

## Phase 4: User Story 3 — Run AgentSpec via Docker (Priority: P2)

**Goal**: Official Docker image on ghcr.io, under 50 MB compressed, supporting all CLI commands.

**Independent Test**: Build image locally with `docker build -t agentspec:test .` → verify `docker run agentspec:test version` works → check image size.

### Implementation for User Story 3

- [x] T010 [P] [US3] Create `Dockerfile` at repository root: `FROM scratch`, `COPY agentspec /agentspec`, `WORKDIR /work`, `ENTRYPOINT ["/agentspec"]` per `specs/013-adoption-dev-experience/contracts/docker-image.md`
- [x] T011 [P] [US3] Add `dockers` section to `.goreleaser.yaml`: two entries for `linux/amd64` and `linux/arm64`, each using the root `Dockerfile`, image templates `ghcr.io/szaher/agentspec:{{ .Version }}-amd64` (and arm64), build flag `--platform linux/amd64` (and arm64), extra files: none (binary only)
- [x] T012 [US3] Add `docker_manifests` section to `.goreleaser.yaml`: create multi-arch manifest with image templates `ghcr.io/szaher/agentspec:{{ .Version }}`, `ghcr.io/szaher/agentspec:{{ .Major }}.{{ .Minor }}`, `ghcr.io/szaher/agentspec:{{ .Major }}`, `ghcr.io/szaher/agentspec:latest`; each manifest references both amd64 and arm64 images
- [x] T013 [US3] Update `.github/workflows/release.yaml`: add `packages: write` permission to the release job for ghcr.io push; add `DOCKER_CLI_EXPERIMENTAL: enabled` env var if needed for manifest support; add `docker login ghcr.io` step using `GITHUB_TOKEN`

**Checkpoint**: `goreleaser check` passes with docker config; local `docker build` produces working image under 50 MB

---

## Phase 5: User Story 4 — Browse and Run Curated Examples (Priority: P2)

**Goal**: At least 6 example projects in `examples/` with comprehensive READMEs, all passing validation.

**Independent Test**: Run `agentspec validate examples/<name>/<name>.ias` for each of the 6 required examples; verify each has a README with Description, Prerequisites, Run Instructions.

### Implementation for User Story 4

- [x] T014 [P] [US4] Rename `examples/customer-support/` to `examples/support-bot/` and rename `customer-support.ias` to `support-bot.ias` inside it (aligns example name with template name `support-bot`); rewrite `README.md` with standardized format — Description, Architecture Overview (agent + skills diagram), Prerequisites (`ANTHROPIC_API_KEY`), Step-by-Step Run Instructions, Customization Tips
- [x] T015 [P] [US4] Rename `examples/rag-chatbot/` to `examples/rag-assistant/` and rename `rag-chatbot.ias` to `rag-assistant.ias` inside it (aligns example name with template name `rag-assistant`); rewrite `README.md` with standardized format — Description, Architecture Overview (agent + MCP + vector search), Prerequisites, Step-by-Step Run Instructions, Customization Tips
- [x] T016 [P] [US4] Create `examples/research-swarm/research-swarm.ias`: multi-agent example using existing multi-agent DSL features with a coordinator agent and 2-3 specialist research agents; use env var references for API keys; add inline comments explaining multi-agent coordination pattern
- [x] T017 [P] [US4] Create `examples/research-swarm/README.md`: Description (multi-agent research coordination), Architecture Overview (coordinator + specialists), Prerequisites, Step-by-Step Run Instructions
- [x] T018 [P] [US4] Create `examples/incident-response/incident-response.ias`: single agent with escalation logic, alert triage skills, runbook execution tool; use env var references for API keys; add inline comments explaining incident-response pattern
- [x] T019 [P] [US4] Create `examples/incident-response/README.md`: Description, Architecture Overview, Prerequisites, Step-by-Step Run Instructions
- [x] T020 [P] [US4] Create `examples/gpu-batch/gpu-batch.ias`: agent that dispatches batch inference tasks to an external GPU API endpoint (no local GPU required); demonstrate async task dispatch pattern with result collection skill; use env var references
- [x] T021 [P] [US4] Create `examples/gpu-batch/README.md`: Description (batch inference dispatch to external GPU service), Architecture Overview, Prerequisites (API endpoint env var), Step-by-Step Run Instructions
- [x] T022 [P] [US4] Create `examples/multi-agent-router/multi-agent-router.ias`: router agent that classifies input and delegates to specialized sub-agents (e.g., billing, technical, general) using existing control-flow DSL features; use env var references
- [x] T023 [P] [US4] Create `examples/multi-agent-router/README.md`: Description (request routing across specialized agents), Architecture Overview, Prerequisites, Step-by-Step Run Instructions
- [x] T024 [US4] Update `examples/README.md`: add entries for all 4 new examples (research-swarm, incident-response, gpu-batch, multi-agent-router) with one-line descriptions matching the standardized format of existing entries
- [x] T025 [US4] Validate all 6 required examples pass: run `agentspec validate` on `support-bot.ias`, `rag-assistant.ias`, `research-swarm.ias`, `incident-response.ias`, `gpu-batch.ias`, `multi-agent-router.ias`; run `agentspec fmt --check` on each

**Checkpoint**: All 6 required examples validate, are canonically formatted, and have comprehensive READMEs

---

## Phase 6: User Story 5 — Starter Template Library (Priority: P2)

**Goal**: 6 bundled starter templates with READMEs and inline comments, selectable via `agentspec init`.

**Independent Test**: Run `agentspec init --template <name>` for each of the 6 templates → verify scaffolded project validates with `agentspec validate`.

### Implementation for User Story 5

- [x] T026 [P] [US5] Create `internal/templates/files/basic-chatbot/basic-chatbot.ias`: minimal single-agent template with system prompt, one skill, env var reference `${ANTHROPIC_API_KEY}`; add inline comments explaining each section (agent, prompt, skill, config)
- [x] T027 [P] [US5] Create `internal/templates/files/basic-chatbot/README.md`: Description, Prerequisites (`ANTHROPIC_API_KEY`), Configuration, Run Instructions (`agentspec run basic-chatbot.ias`)
- [x] T028 [P] [US5] Rename `internal/templates/files/customer-support/` to `internal/templates/files/support-bot/` and rename the `.ias` file inside to `support-bot.ias`; update env var references to use `${ANTHROPIC_API_KEY}` format; add inline comments explaining each section
- [x] T029 [P] [US5] Create `internal/templates/files/support-bot/README.md`: Description, Prerequisites, Configuration, Run Instructions
- [x] T030 [P] [US5] Rename `internal/templates/files/rag-chatbot/` to `internal/templates/files/rag-assistant/` and rename the `.ias` file inside to `rag-assistant.ias`; update env var references; add inline comments
- [x] T031 [P] [US5] Create `internal/templates/files/rag-assistant/README.md`: Description, Prerequisites, Configuration, Run Instructions
- [x] T032 [P] [US5] Create `internal/templates/files/research-swarm/research-swarm.ias`: multi-agent template with coordinator + 2 research agents; env var references; inline comments explaining multi-agent pattern and customization points
- [x] T033 [P] [US5] Create `internal/templates/files/research-swarm/README.md`: Description, Prerequisites, Configuration, Run Instructions
- [x] T034 [P] [US5] Create `internal/templates/files/incident-response/incident-response.ias`: incident triage agent with escalation skills; env var references; inline comments
- [x] T035 [P] [US5] Create `internal/templates/files/incident-response/README.md`: Description, Prerequisites, Configuration, Run Instructions
- [x] T036 [P] [US5] Create `internal/templates/files/multi-agent-router/multi-agent-router.ias`: router agent with input classification and delegation to sub-agents; env var references; inline comments
- [x] T037 [P] [US5] Create `internal/templates/files/multi-agent-router/README.md`: Description, Prerequisites, Configuration, Run Instructions
- [x] T038 [US5] Update `All()` in `internal/templates/templates.go`: replace existing 6 entries with the 6 required templates — `basic-chatbot` (beginner), `support-bot` (beginner), `rag-assistant` (intermediate), `incident-response` (intermediate), `research-swarm` (advanced), `multi-agent-router` (advanced); keep `data-extraction`, `code-review-pipeline`, `validated-agent` as additional templates
- [x] T039 [US5] Validate all 6 required templates: build binary, run `agentspec init --template <name>` for each into a temp directory, run `agentspec validate` and `agentspec fmt --check` on each scaffolded `.ias` file; time each `init` invocation to confirm < 3 seconds (SC-007)

**Checkpoint**: `agentspec init --list-templates` shows 6+ templates with categories; all scaffolded projects validate

---

## Phase 7: CLI Polish & Version Check (Cross-Cutting)

**Goal**: Helpful no-args message and version update notification.

**Independent Test**: Run `agentspec` with no args → see welcome message; run `agentspec version` → see version + update notice (or just version if up to date / offline).

### Implementation

- [x] T040 [US2] Update root command in `cmd/agentspec/main.go`: set `RunE` on root cobra command to display a welcome message when no subcommand given — print tool name, point to `agentspec --help`, `agentspec init`, and quickstart URL (FR-015); do NOT use cobra's default "unknown command" behavior
- [x] T041 Enhance `cmd/agentspec/version.go`: add `checkLatestVersion()` func that makes HTTP GET to `https://api.github.com/repos/szaher/agentspec/releases/latest` with 2-second timeout, parses JSON `tag_name` field, strips leading `v`, compares with compiled `version` using semver comparison; print update notice with release URL and install commands if outdated; silently skip on any error per `specs/013-adoption-dev-experience/contracts/cli-version.md` (FR-018)

**Checkpoint**: `agentspec` (no args) shows welcome message; `agentspec version` shows version info (update check works when network available, silent fallback when not)

---

## Phase 8: User Story 6 — Quickstart Guide in Documentation (Priority: P3)

**Goal**: Documentation site quickstart page with platform-specific install instructions and zero-to-running-agent walkthrough.

**Independent Test**: Run `mkdocs build` → verify quickstart page renders; follow instructions verbatim on a clean environment.

### Implementation for User Story 6

- [x] T042 [US6] Create `docs/quickstart.md`: platform-tabbed install instructions (macOS via `brew install szaher/agentspec/agentspec`, Linux via binary download from GitHub Releases, Windows via zip download, Docker via `docker pull ghcr.io/szaher/agentspec:latest`); scaffold walkthrough (`agentspec init`); configure step (set `ANTHROPIC_API_KEY`); run step (`agentspec run`); troubleshooting section covering missing API key, wrong architecture binary, port conflicts
- [x] T043 [US6] Update `mkdocs.yml`: add `Quickstart` entry to the `nav` section, positioned as the second item after the overview/intro page

**Checkpoint**: `mkdocs build` succeeds; quickstart page is accessible in the built site

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: CI integration, final validation, cleanup

- [x] T044 [P] Add template scaffolding validation step to `.github/workflows/ci.yml`: after building the binary, loop through all templates via `agentspec init --template <name> --output-dir /tmp/template-test-<name>` and run `agentspec validate` on each scaffolded `.ias` file
- [x] T045 [P] Verify existing CI example validation in `.github/workflows/ci.yml` picks up all 4 new example directories (research-swarm, incident-response, gpu-batch, multi-agent-router); adjust glob pattern if needed
- [x] T046 Run pre-commit checks: `golangci-lint run ./...` reports zero errors, `gofmt -l .` produces no output, `go build ./...` succeeds, `go test ./... -count=1` passes

**Checkpoint**: CI pipeline passes with all new templates and examples validated

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **US1 (Phase 2)**: No dependency on Phase 1 — GoReleaser config is independent
- **US2 (Phase 3)**: Depends on Phase 1 (template restructure) — needs directory-based scaffolding
- **US3 (Phase 4)**: No dependency on Phase 1 — Docker config is independent
- **US4 (Phase 5)**: No dependencies — example projects are standalone files
- **US5 (Phase 6)**: Depends on Phase 1 (template restructure) — needs directory structure
- **CLI Polish (Phase 7)**: Depends on Phase 3 (init enhancements must be done first)
- **US6 (Phase 8)**: Depends on Phase 2 + Phase 4 (needs final install commands and Docker image reference)
- **Polish (Phase 9)**: Depends on Phases 2-8

### User Story Dependencies

- **US1 (P1)**: Independent — can start immediately
- **US2 (P1)**: Depends on Phase 1 setup only
- **US3 (P2)**: Independent — can start immediately
- **US4 (P2)**: Independent — can start immediately
- **US5 (P2)**: Depends on Phase 1 setup only
- **US6 (P3)**: Depends on US1 + US3 (references install commands)

### Parallel Opportunities

- **Phase 1** + **Phase 2 (US1)** + **Phase 4 (US3)** + **Phase 5 (US4)** can all start in parallel
- Within **Phase 5 (US4)**: All T014-T023 are parallelizable (different directories)
- Within **Phase 6 (US5)**: All T026-T037 are parallelizable (different template directories)
- **Phase 3 (US2)** starts after Phase 1 completes
- **Phase 6 (US5)** starts after Phase 1 completes (can run parallel with Phase 3)

---

## Parallel Example: User Story 4 (Examples)

```bash
# Launch all example creation tasks together (all different directories):
Task: "Create examples/research-swarm/research-swarm.ias"
Task: "Create examples/research-swarm/README.md"
Task: "Create examples/incident-response/incident-response.ias"
Task: "Create examples/incident-response/README.md"
Task: "Create examples/gpu-batch/gpu-batch.ias"
Task: "Create examples/gpu-batch/README.md"
Task: "Create examples/multi-agent-router/multi-agent-router.ias"
Task: "Create examples/multi-agent-router/README.md"
Task: "Enhance examples/customer-support/README.md"
Task: "Enhance examples/rag-chatbot/README.md"
```

## Parallel Example: User Story 5 (Templates)

```bash
# Launch all template creation tasks together (all different directories):
Task: "Create internal/templates/files/basic-chatbot/"
Task: "Create internal/templates/files/support-bot/README.md"
Task: "Create internal/templates/files/rag-assistant/README.md"
Task: "Create internal/templates/files/research-swarm/"
Task: "Create internal/templates/files/incident-response/"
Task: "Create internal/templates/files/multi-agent-router/"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Template restructure (T001-T003)
2. Complete Phase 2: Homebrew tap config (T004-T005)
3. Complete Phase 3: Init command enhancements (T006-T009)
4. Complete Phase 7: CLI polish + version check (T040-T041)
5. **STOP and VALIDATE**: `agentspec init` works end-to-end; `goreleaser check` passes
6. This delivers: one-command install + scaffold-and-run experience

### Incremental Delivery

1. MVP (US1 + US2) → Core install + init experience
2. Add US3 (Docker) → Container-based workflows enabled
3. Add US4 (Examples) → Real-world use cases browsable
4. Add US5 (Templates) → Full template library with 6 templates
5. Add US6 (Quickstart) → Documentation complete
6. Polish (Phase 9) → CI validation of all artifacts

### Parallel Team Strategy

With multiple developers:

1. **Developer A**: Phase 1 → Phase 3 (US2) → Phase 7 (CLI polish)
2. **Developer B**: Phase 2 (US1) → Phase 4 (US3) → Phase 9 (CI)
3. **Developer C**: Phase 5 (US4 examples) → Phase 6 (US5 templates) → Phase 8 (US6 docs)

---

## Phase Mapping (tasks → plan)

| Tasks Phase | Plan Phase | Description |
|-------------|------------|-------------|
| Phase 1 (Setup) | Phase B.1-B.2 | Template system restructure |
| Phase 2 (US1) | Phase A.1 | Homebrew tap via GoReleaser |
| Phase 3 (US2) | Phase B.5 | Init command enhancements |
| Phase 4 (US3) | Phase A.2-A.4 | Docker image + CI permissions |
| Phase 5 (US4) | Phase C | Example projects |
| Phase 6 (US5) | Phase B.3-B.4 | Template content creation |
| Phase 7 (CLI Polish) | Phase D | No-args message + version check |
| Phase 8 (US6) | Phase E | Quickstart documentation |
| Phase 9 (Polish) | Phase B.6 + cross-cutting | CI validation + pre-commit |

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- All `.ias` files in templates and examples MUST use `${ENV_VAR}` references for secrets, never plaintext
- All `.ias` files MUST pass `agentspec validate` and `agentspec fmt --check`
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- **Integration testing**: CI steps T044-T045 validate all templates and examples end-to-end via `agentspec validate`, satisfying the constitution's integration test quality gate. No separate `integration_tests/` additions are needed since this feature adds content (templates, examples, docs) and CLI enhancements — not new DSL semantics or engine behavior.
- **Templates vs examples**: Templates (US5, `internal/templates/files/`) are minimal scaffolds with inline comments for learning. Examples (US4, `examples/`) are full reference implementations with architecture overviews. Names are aligned (e.g., `support-bot` in both) to avoid user confusion.
- **FR-001 coverage**: Pre-built binary releases are already handled by the existing GoReleaser configuration (assumption from 012-production-readiness). T005 implicitly verifies this via `goreleaser release --snapshot`.
