# CLI Contract: `agentspec`

## Commands

### `agentspec fmt [files...]`

Format `.ias` IntentLang source files to canonical style.

- **Input**: One or more `.ias` files, or current directory (recursive)
- **Output**: Formatted files written in-place; diff to stdout if
  `--check` flag is set
- **Exit codes**: 0 = all files formatted; 1 = files would change
  (with `--check`)
- **Flags**:
  - `--check`: Report whether files need formatting without writing
  - `--diff`: Print diff of changes to stdout

### `agentspec validate [files...]`

Validate `.ias` AgentSpec definitions (structural + semantic).

- **Input**: One or more `.ias` files, or current directory
- **Output**: Validation errors to stderr in format:
  `<file>:<line>:<col>: error: <message>\n  hint: <fix suggestion>`
- **Exit codes**: 0 = valid; 1 = validation errors
- **Flags**:
  - `--format`: Output format (`text` | `json`), default `text`

### `agentspec plan [--target <binding>] [--env <environment>]`

Show what changes would be made without applying.

- **Input**: `.ias` files in current directory + state file
- **Output**: Machine-diffable plan to stdout
- **Exit codes**: 0 = no changes; 2 = changes pending
- **Flags**:
  - `--target`: Binding name (default: default binding)
  - `--env`: Environment name (default: base)
  - `--out`: Write plan to file instead of stdout
  - `--format`: Output format (`text` | `json`), default `text`
- **Determinism**: Identical inputs MUST produce byte-identical
  output across machines and runs.

### `agentspec apply [--target <binding>] [--env <environment>]`

Apply desired state idempotently.

- **Input**: `.ias` files + state file + target adapter
- **Output**: Applied resource summary to stdout; structured events
  to stderr
- **Exit codes**: 0 = success (no changes or all applied);
  1 = partial failure (some resources failed)
- **Flags**:
  - `--target`: Binding name
  - `--env`: Environment name
  - `--auto-approve`: Skip confirmation prompt
  - `--plan-file`: Use a saved plan file instead of computing
- **Behavior**: On partial failure, records partial state
  accurately. Re-running retries only failed resources.
- **Events**: Emits structured events with correlation ID.

### `agentspec diff [--target <binding>]`

Show drift between desired state and actual state.

- **Input**: `.ias` files + state file
- **Output**: Drift report to stdout
- **Exit codes**: 0 = no drift; 2 = drift detected

### `agentspec export [--target <binding>] [--env <environment>]`

Export adapter-specific artifacts without applying.

- **Input**: `.ias` files + target adapter
- **Output**: Artifacts written to `--out-dir`
- **Exit codes**: 0 = success; 1 = error
- **Flags**:
  - `--target`: Binding name
  - `--env`: Environment name
  - `--out-dir`: Output directory (default: `./export/`)
- **Determinism**: Identical inputs MUST produce byte-identical
  artifacts.

### `agentspec sdk generate [--lang <language>]`

Generate SDK for a target language.

- **Input**: IR schema + state file
- **Output**: Generated SDK files to `--out-dir`
- **Flags**:
  - `--lang`: Target language (`python` | `typescript` | `go`)
  - `--out-dir`: Output directory (default: `./sdk/<lang>/`)

### `agentspec version`

Print version information.

- **Output**: `agentspec version <semver> (lang <lang-version>,
  ir <ir-version>)`

## Global Flags

- `--state-file`: Path to state file (default: `.agentspec.state.json`)
- `--verbose`: Enable verbose output
- `--no-color`: Disable colored output
- `--correlation-id`: Set explicit correlation ID (auto-generated
  if omitted)
