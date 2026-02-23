# Plugin Usage

An AgentSpec that references a WASM plugin, extending IntentLang with custom resource types.

## What This Demonstrates

- **Plugin declaration** with name and version pinning
- **Plugin resolution** from local and global plugin directories
- **Extension model** for adding custom resource types, validators, transforms, and hooks to IntentLang

## AgentSpec Structure

### Plugin Reference

```
plugin "monitor" version "1.0.0"
```

This declares a dependency on the `monitor` plugin at version `1.0.0`. The tool resolves plugins from:
1. `./plugins/<name>/manifest.json` (local, project-scoped)
2. `~/.agentz/plugins/<name>/manifest.json` (global, user-scoped)

The version is pinned for reproducibility. If the installed plugin version doesn't match, the tool reports an error.

### Plugin Manifest

The monitor plugin's manifest (`plugins/monitor/manifest.json`) declares:
- **Resource types**: `Monitor` (a custom resource type not in the base IntentLang)
- **Validators**: `threshold` (validates monitor-specific attributes)
- **Transforms**: `monitor-to-alerts` (converts Monitor resources during compilation)
- **Hooks**: `preflight` (runs at pre-apply stage)

### Using Plugin Resources

Once the plugin is loaded, you can use its custom resource types in your AgentSpec files. The plugin's validators and transforms run automatically during the validate and plan stages.

## How to Run

```bash
# Validate (plugin manifest is loaded and checked)
./agentspec validate examples/plugin-usage.ias

# Plan
./agentspec plan examples/plugin-usage.ias

# Apply
./agentspec apply examples/plugin-usage.ias --auto-approve
```

## Resources Created

| Kind | Name | Description |
|------|------|-------------|
| Prompt | ops | Operations assistant instructions |
| Skill | check-health | Health check capability |
| Agent | ops-bot | Operations agent |

## Plugin System Architecture

```
.ias file
  |
  v
Parser --> loads plugin manifest
  |
  v
Validator --> dispatches to plugin validators
  |
  v
IR Lower --> plugin transforms modify resources
  |
  v
Plan/Apply --> lifecycle hooks execute (pre-apply, post-apply)
```

Plugins run in a WASM sandbox (via wazero) with:
- Memory isolation
- No filesystem access beyond declared capabilities
- Execution timeouts
- Deterministic output

## Plugin Directory Structure

```
plugins/
  monitor/
    manifest.json    # Declares capabilities, version, resource types
    plugin.wasm      # Compiled WASM module (optional, for active plugins)
```

## Next Steps

- Build your own plugin: see the plugin manifest at `plugins/monitor/manifest.json`
- Learn about the adapter interface: see [multi-binding](../multi-binding/)
