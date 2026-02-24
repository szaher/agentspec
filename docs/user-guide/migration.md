# IntentLang 1.0 to 2.0 Migration Guide

## Overview

IntentLang 2.0 introduces several breaking changes from 1.0, including renamed keywords, a new file extension, updated tooling, and required schema definitions for skills. This guide walks you through the migration process step by step so you can update your existing AgentSpec files with minimal disruption.

If you are starting a new project, you can skip this guide and go straight to the [Getting Started](getting-started/index.md) section.

## Key Changes Summary

| Area | IntentLang 1.0 | IntentLang 2.0 |
|------|----------------|----------------|
| File extension | `.az` | `.ias` |
| CLI binary | `agentz` | `agentspec` |
| State file | `.agentz.state.json` | `.agentspec.state.json` |
| Plugin directory | `~/.agentz/plugins/` | `~/.agentspec/plugins/` |
| Package header | `package "name" version "x.y.z"` | `package "name" version "x.y.z" lang "2.0"` |
| Execution blocks | `execution` keyword | `agent` keyword |
| Action blocks | `action` keyword | `skill` keyword |
| Pipeline steps | `step` blocks reference `execution` | `step` blocks reference `agent` |
| Skill schemas | Input/output optional | `input`/`output` blocks required on skills |
| New resource types | N/A | `type`, `server`, `client`, `environment`, `policy`, `plugin` |

## Using the Migration Tool

The `agentspec migrate` command automates most of the migration work. It renames keywords, updates the package header, renames files, and adjusts references throughout your project.

### Preview Changes (Dry Run)

Run a dry run first to see what the tool will change without modifying any files:

```bash
agentspec migrate --to-v2 --dry-run my-agent.az
```

### Apply Migration

When you are satisfied with the preview, apply the migration:

```bash
agentspec migrate --to-v2 my-agent.az
```

### Migrate All Files in a Directory

To migrate every `.az` file in a directory (and its subdirectories):

```bash
agentspec migrate --to-v2 agents/
```

!!! note
    The migration tool handles keyword renames and file extension changes automatically, but you will need to manually add `input` and `output` blocks to your skills. The tool inserts placeholder schemas that you should review and update.

## Before and After Examples

### Before (IntentLang 1.0)

```text
package "my-agent" version "1.0.0"

prompt "system" {
  content "You are a helpful assistant."
}

action "search" {
  description "Search for information"
  tool command {
    binary "search-tool"
  }
}

execution "assistant" {
  uses prompt "system"
  uses action "search"
  model "claude-sonnet-4-20250514"
}

deploy "local" target "process" {
  default true
}
```

### After (IntentLang 2.0)

```ias
package "my-agent" version "2.0.0" lang "2.0"

prompt "system" {
  content "You are a helpful assistant."
}

skill "search" {
  description "Search for information"
  input {
    query string required
  }
  output {
    results string
  }
  tool command {
    binary "search-tool"
  }
}

agent "assistant" {
  uses prompt "system"
  uses skill "search"
  model "claude-sonnet-4-20250514"
}

deploy "local" target "process" {
  default true
}
```

Notice the following changes in the 2.0 version:

- The `lang "2.0"` field is added to the package header.
- The `action` block is renamed to `skill`.
- The `skill` block now includes required `input` and `output` schemas.
- The `execution` block is renamed to `agent`.
- The `agent` block references `skill` instead of `action`.

## Step-by-Step Migration Checklist

1. **Rename `.az` files to `.ias`** -- Update all file extensions in your project. If you use the `agentspec migrate` tool, this is handled automatically.

2. **Add `lang "2.0"` to the package header** -- Every `.ias` file must declare the language version in its package statement:
   ```ias
   package "my-agent" version "2.0.0" lang "2.0"
   ```

3. **Rename `execution` blocks to `agent`** -- Replace all occurrences of the `execution` keyword with `agent`.

4. **Rename `action` blocks to `skill`** -- Replace all occurrences of the `action` keyword with `skill`. Update any `uses action` references to `uses skill`.

5. **Add `input`/`output` schemas to skills** -- In IntentLang 2.0, every `skill` block requires explicit `input` and `output` definitions. Define the fields, types, and whether they are required or optional.

6. **Update CI scripts referencing the `agentz` binary to `agentspec`** -- Search your CI/CD configuration files (GitHub Actions workflows, Makefiles, shell scripts) and replace `agentz` with `agentspec`.

7. **Rename `.agentz.state.json` to `.agentspec.state.json`** -- If your project has an existing state file, rename it. The CLI performs auto-migration on first run, but renaming it explicitly avoids ambiguity.

8. **Run `agentspec validate` to verify** -- After completing all changes, validate every file to confirm correctness:
   ```bash
   agentspec validate *.ias
   ```

## New Features in 2.0

After migrating, you gain access to several new capabilities:

- **Custom type definitions** -- Define reusable types with the `type` keyword for structured data shared across skills and agents.
- **MCP server/client blocks** -- Declare `server` and `client` resources for Model Context Protocol integrations.
- **Environment overlays** -- Use `environment` blocks to define configuration overlays for different deployment targets (development, staging, production).
- **Policy enforcement** -- Attach `policy` blocks to agents and skills to enforce guardrails, rate limits, and access controls.
- **WASM plugin system** -- Extend AgentSpec with custom plugins written in any language that compiles to WebAssembly, loaded via the `plugin` keyword.
- **Prompt template variables** -- Use variable interpolation in prompt content with `{{ .variable }}` syntax for dynamic prompt generation.

## Troubleshooting

### `unknown keyword "execution"`

You have an `.ias` file that still uses the 1.0 `execution` keyword. Rename it to `agent`:

```text
# Before
execution "my-exec" { ... }

# After
agent "my-exec" { ... }
```

### `unknown keyword "action"`

Similar to above -- rename `action` to `skill` and update any `uses action` references to `uses skill`.

### `missing required field "lang" in package`

IntentLang 2.0 requires the `lang` field in the package header. Add `lang "2.0"` to your package statement:

```ias
package "my-agent" version "2.0.0" lang "2.0"
```

### `skill "X" missing input block`

In 2.0, every skill must declare an `input` block. Even if the skill takes no input, you must provide an empty block:

```ias novalidate
skill "no-input-skill" {
  description "A skill with no input"
  input {}
  output {
    result string
  }
  tool command {
    binary "my-tool"
  }
}
```

### `skill "X" missing output block`

Same as above but for `output`. Provide at least an empty `output {}` block.

### State file not found after migration

If `agentspec` cannot find the state file, it may still be named `.agentz.state.json`. Rename it:

```bash
mv .agentz.state.json .agentspec.state.json
```

Alternatively, run any `agentspec` command and the CLI will auto-migrate the state file on first run.

### Plugins not loading after migration

The plugin directory has moved from `~/.agentz/plugins/` to `~/.agentspec/plugins/`. Move your plugins:

```bash
mv ~/.agentz/plugins/* ~/.agentspec/plugins/
```

### CI pipeline failures after migration

Search your CI configuration for references to the old binary name:

```bash
grep -r "agentz" .github/workflows/
```

Replace all occurrences of `agentz` with `agentspec`.

## See Also

- [Language Reference Index](language/index.md) -- Complete IntentLang 2.0 language reference.
- [CLI `migrate` Command](cli/migrate.md) -- Detailed documentation for the `agentspec migrate` command and its options.
