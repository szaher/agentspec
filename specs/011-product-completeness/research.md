# Research: Product Completeness & UX

**Feature**: 011-product-completeness
**Date**: 2026-03-17

## R1: Compiler Target Tool Generation Patterns

**Decision**: Extend all 4 targets (CrewAI, LangGraph, LlamaIndex, LlamaStack) to generate functional tool implementations for HTTP, command, and inline tool types.

**Rationale**: CrewAI already generates functional HTTP (`urllib.request`) and command (`subprocess.run`) tool code when URL/binary are configured. The other 3 targets need similar treatment. The same pattern (check tool type → emit framework-appropriate code) applies to all.

**Alternatives considered**:
- Shared codegen helper across targets: Rejected — each framework has different idioms (LangChain `Tool` class vs CrewAI `@tool` decorator vs LlamaIndex `FunctionTool`).
- Runtime adapter approach: Rejected — spec requires generated code to be a standalone starting point without AgentSpec runtime dependency.

**Current state by target**:
- CrewAI (`crewai.go`): HTTP and command implemented; inline falls to "not implemented" stub; unknown types fall to stub.
- LangGraph (`langgraph.go`): Need to verify tool generation — likely has similar stubs.
- LlamaIndex (`llamaindex.go`): Need to verify tool generation.
- LlamaStack (`llamastack.go`): Need to verify tool generation.

**Pattern to follow**: CrewAI's `generateTools()` method at `internal/compiler/targets/crewai.go:157-247` is the reference. Each target needs:
1. `case "http"`: Generate HTTP client call with URL and method
2. `case "command"`: Generate subprocess execution with binary and args
3. `case "inline"`: Generate inline script execution (new for all targets)
4. Default: Generate TODO comment explaining limitation

## R2: Event-Based File Watching

**Decision**: Use `fsnotify/fsnotify` library for event-based file watching, with polling fallback.

**Rationale**: `fsnotify` is the de facto standard for Go file watching. It wraps OS-specific APIs (inotify on Linux, kqueue on macOS/BSD, ReadDirectoryChangesW on Windows). Already well-tested and widely used (Docker, Hugo, Kubernetes).

**Alternatives considered**:
- `github.com/radovskyb/watcher`: Pure polling — doesn't solve the latency issue.
- Custom inotify/kqueue: Too much platform-specific code to maintain.
- `os.File` polling with shorter interval: Still wastes CPU and doesn't achieve <500ms reliably.

**Implementation**: Replace `time.NewTicker(2*time.Second)` + `filepath.Walk` in `cmd/agentspec/dev.go:120-158` with fsnotify watcher filtered to `.ias` files. Debounce changes by 100ms to avoid rapid-fire reloads.

## R3: Live Eval Command

**Decision**: Add `--live` flag to eval command that creates a real LLM client and invokes agents through the agentic loop.

**Rationale**: The current `stubInvoker` at `cmd/agentspec/eval.go:140-144` always returns an error. The `--live` flag should reuse the same `loop.ReActStrategy` and `llm.NewClientForModel()` pattern from the `run` command.

**Alternatives considered**:
- Start a full runtime server and connect to it: Overcomplicated for eval — a direct loop invocation is simpler.
- Use a mock LLM for testing: Doesn't test real agent behavior as spec requires.

**Implementation**: Create a `liveInvoker` struct that accepts an IR document, creates LLM client per agent, and invokes via `loop.ReActStrategy.Execute()`. Replace `stubInvoker` when `--live` flag is set.

## R4: Command Rename Strategy (run ↔ dev)

**Decision**: Swap `run` and `dev` semantics. `run` starts the server; `dev` does one-shot invocation. Provide deprecation aliases.

**Rationale**: Users expect `run` to start a long-running process (like `docker run`, `npm run`). Current `run` does one-shot and `dev` starts server — opposite of conventions.

**Implementation**:
1. Rename `newRunCmd()` → serves as server command (current `dev` behavior)
2. Rename `newDevCmd()` → serves as one-shot command (current `run` behavior)
3. Add deprecation aliases that print warnings and delegate to the new commands
4. Update all documentation, help text, and examples

**Migration path**: Old `run` behavior → now `dev`. Old `dev` behavior → now `run`. Aliases maintained for one release cycle.

## R5: CLI Flag Audit

**Decision**: One-time audit of all 20 CLI commands for stub flags.

**Known stubs found**:
- `publish --sign`: Prints warning and continues (publish.go:32-34)
- Need to audit remaining commands systematically

**Audit approach**: For each command, check all `Flags()` registrations and trace whether the flag value is actually used in the RunE function body.

## R6: Frontend State Improvements

**Decision**: Add loading indicator, error banner with retry, and improved empty state to the existing vanilla JS frontend.

**Rationale**: The current `app.js` fetches agents on init but shows no loading indicator. On fetch failure, it calls `addSystemMessage()` — a non-actionable text message with no retry. The welcome message exists but is minimal.

**Current state**:
- Loading: No indicator during `fetchAgents()` call
- Error: `addSystemMessage("Failed to connect: " + err.message)` — no retry button
- Empty state: `addSystemMessage("Welcome to AgentSpec...")` — minimal
- Markdown: Already implemented via `renderMarkdown()` function

**Implementation**: Modify `internal/frontend/web/app.js` and `internal/frontend/web/index.html` to add CSS and JS for loading spinner, error banner with retry button, and enhanced welcome state.

## R7: README Documentation Completeness

**Decision**: Document all 20 CLI commands in README.md. Add CI validation script.

**Current gap**: README documents 12 commands. Missing 8: init, compile, publish, install, eval, status, logs, destroy. Also `pkg` (package) command is registered but not documented.

**Also**: README incorrectly describes `run` as "Start the agent runtime server" — this will be corrected as part of the command rename.

**CI validation**: Add a Go test or shell script that extracts registered commands from `main.go` and compares against README content.
