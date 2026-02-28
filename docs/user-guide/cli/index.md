# CLI Reference

The `agentspec` command-line interface provides tools for authoring, validating, deploying, and managing agents defined in IntentLang (`.ias`) files.

## Command Summary

| Command | Description |
|---------|-------------|
| [validate](validate.md) | Check an IntentLang file for syntax and semantic errors |
| [fmt](fmt.md) | Format an IntentLang file to canonical style |
| [plan](plan.md) | Preview changes that would be applied from a spec |
| [apply](apply.md) | Apply changes defined in a spec to the target environment |
| [run](run.md) | Execute an agent locally from a spec file |
| [compile](compile.md) | Compile .ias files into a deployable agent artifact |
| [package](package.md) | Package a compiled agent for deployment (Docker, Kubernetes, Helm) |
| [eval](eval.md) | Run evaluation test cases against agents |
| [install](install.md) | Install a package from a Git repository |
| [publish](publish.md) | Publish an AgentPack package to a Git remote |
| [dev](dev.md) | Start a local development server with live reload |
| [status](status.md) | Show the current state of deployed agents |
| [logs](logs.md) | Stream or retrieve logs from running agents |
| [destroy](destroy.md) | Tear down resources created by a spec |
| [init](init.md) | Scaffold a new IntentLang project |
| [migrate](migrate.md) | Migrate specs and state files to newer formats |
| [export](export.md) | Export a spec to JSON or YAML |
| [diff](diff.md) | Show differences between a spec and the current state |
| [sdk](sdk.md) | Generate typed SDK bindings from a spec |
| [version](version.md) | Print version, build date, and commit hash |

## Common Flags

These flags are accepted by every command.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--state-file` | | `.agentspec.state.json` | Path to the state file |
| `--verbose` | `-v` | `false` | Enable verbose output |
| `--no-color` | | `false` | Disable colored output |
| `--correlation-id` | | *(auto-generated)* | Set a correlation ID for tracing |

## Global Behavior

**State file.** Most commands that read or write deployment state use `.agentspec.state.json` in the current directory by default. Override this with `--state-file`. The CLI will automatically migrate legacy `.agentz.state.json` files on first access.

**Exit codes.** All commands follow a consistent exit code convention:

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error (validation failure, runtime error, etc.) |

**Verbose mode.** Pass `--verbose` (or `-v`) to any command to see detailed diagnostic output, including resolved configuration, plugin loading, and internal timings.

**Color output.** Output is colored by default when writing to a terminal. Use `--no-color` to disable colors, which is useful when piping output to files or other programs.
