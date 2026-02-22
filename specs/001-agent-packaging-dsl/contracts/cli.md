# CLI Contract: `agentz`

## Commands

### `agentz fmt [files...]`

Format `.az` source files to canonical style.

- **Input**: One or more `.az` files, or current directory (recursive)
- **Output**: Formatted files written in-place; diff to stdout if
  `--check` flag is set
- **Exit codes**: 0 = all files formatted; 1 = files would change
  (with `--check`)
- **Flags**:
  - `--check`: Report whether files need formatting without writing
  - `--diff`: Print diff of changes to stdout

### `agentz validate [files...]`

Validate `.az` definitions (structural + semantic).

- **Input**: One or more `.az` files, or current directory
- **Output**: Validation errors to stderr in format:
  `<file>:<line>:<col>: error: <message>\n  hint: <fix suggestion>`
- **Exit codes**: 0 = valid; 1 = validation errors
- **Flags**:
  - `--format`: Output format (`text` | `json`), default `text`

### `agentz plan [--target <binding>] [--env <environment>]`

Show what changes would be made without applying.

- **Input**: `.az` files in current directory + state file
- **Output**: Machine-diffable plan to stdout
- **Exit codes**: 0 = no changes; 2 = changes pending
- **Flags**:
  - `--target`: Binding name (default: default binding)
  - `--env`: Environment name (default: base)
  - `--out`: Write plan to file instead of stdout
  - `--format`: Output format (`text` | `json`), default `text`
- **Determinism**: Identical inputs MUST produce byte-identical
  output across machines and runs.

### `agentz apply [--target <binding>] [--env <environment>]`

Apply desired state idempotently.

- **Input**: `.az` files + state file + target adapter
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

### `agentz diff [--target <binding>]`

Show drift between desired state and actual state.

- **Input**: `.az` files + state file
- **Output**: Drift report to stdout
- **Exit codes**: 0 = no drift; 2 = drift detected

### `agentz export [--target <binding>] [--env <environment>]`

Export adapter-specific artifacts without applying.

- **Input**: `.az` files + target adapter
- **Output**: Artifacts written to `--out-dir`
- **Exit codes**: 0 = success; 1 = error
- **Flags**:
  - `--target`: Binding name
  - `--env`: Environment name
  - `--out-dir`: Output directory (default: `./export/`)
- **Determinism**: Identical inputs MUST produce byte-identical
  artifacts.

### `agentz sdk generate [--lang <language>]`

Generate SDK for a target language.

- **Input**: IR schema + state file
- **Output**: Generated SDK files to `--out-dir`
- **Flags**:
  - `--lang`: Target language (`python` | `typescript` | `go`)
  - `--out-dir`: Output directory (default: `./sdk/<lang>/`)

### `agentz version`

Print version information.

- **Output**: `agentz version <semver> (lang <lang-version>,
  ir <ir-version>)`

## Global Flags

- `--state-file`: Path to state file (default: `.agentz.state.json`)
- `--verbose`: Enable verbose output
- `--no-color`: Disable colored output
- `--correlation-id`: Set explicit correlation ID (auto-generated
  if omitted)
