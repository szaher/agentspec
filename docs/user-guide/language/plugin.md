# plugin

The `plugin` block declares a dependency on a sandboxed WebAssembly (WASM) module that extends AgentSpec's behavior. Plugins run in an isolated wazero sandbox and can hook into validation, transformation, and pre-deployment stages of the toolchain pipeline.

---

## Syntax

```ias
plugin "<name>" version "<semver>"
```

The plugin declaration is a single-line statement (no block body). The name must match a WASM module installed in the plugin directory, and the version must be a valid semantic version string.

---

## Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | The plugin module name. Must match a `.wasm` file in the plugin directory. |
| `version` | string | Yes | Semantic version of the plugin (e.g., `"1.0.0"`, `"2.3.1"`). |

---

## Plugin Directory

Plugins are loaded from the following directories, checked in order:

1. `~/.agentspec/plugins/` (primary)
2. `~/.agentz/plugins/` (fallback, for backward compatibility)

Each plugin is a `.wasm` file named after the plugin. For example, a plugin declared as `plugin "monitor" version "1.0.0"` is loaded from `~/.agentspec/plugins/monitor.wasm`.

!!! info "WASM Sandbox"
    All plugins execute inside a [wazero](https://wazero.io/) sandbox. They cannot access the filesystem, network, or any host resources beyond the AgentSpec plugin API. This ensures that untrusted plugins cannot compromise the host system.

---

## Hook Types

Plugins can implement one or more hook types:

| Hook Type | Stage | Description |
|-----------|-------|-------------|
| `validator` | `agentspec validate` | Runs custom validation rules against the parsed AST. Can report errors and warnings. |
| `transform` | `agentspec plan` | Transforms the intermediate representation (IR) before plan generation. Can modify, add, or remove resources. |
| `pre_deploy` | `agentspec apply` | Executes logic immediately before deployment. Can perform pre-flight checks or resource preparation. |

A single plugin can implement multiple hooks. The hook type is determined by the exported functions in the WASM module, not by the IntentLang declaration.

---

## Examples

### Basic Plugin Usage

```ias
package "plugin-demo" version "0.1.0" lang "2.0"

plugin "monitor" version "1.0.0"

prompt "ops" {
  content "You are an operations assistant."
}

skill "check-health" {
  description "Check service health"
  input {
    service string required
  }
  output {
    status string
  }
  tool command {
    binary "health-check"
  }
}

agent "ops-bot" {
  uses prompt "ops"
  uses skill "check-health"
  model "claude-sonnet-4-20250514"
}

deploy "local" target "process" {
  default true
}
```

### Multiple Plugins

A package can declare multiple plugin dependencies:

```ias
plugin "security-scanner" version "2.1.0"
plugin "cost-estimator" version "1.3.0"
plugin "compliance-checker" version "1.0.0"
```

!!! warning "Version pinning"
    Plugin versions are exact matches. If version `"1.0.0"` is declared but only version `"1.1.0"` is installed, validation will fail. Always ensure the installed plugin version matches the declared version.

---

## Plugin Resolution

When `agentspec validate`, `agentspec plan`, or `agentspec apply` is run, the toolchain:

1. Reads all `plugin` declarations from the `.ias` file.
2. Locates each plugin's `.wasm` file in the plugin directory.
3. Validates the plugin version against the declared version.
4. Loads the WASM module into a wazero sandbox.
5. Invokes the appropriate hook functions based on the current command.

If a declared plugin cannot be found or its version does not match, the command fails with an error before any processing begins.

---

## Writing Plugins

For a detailed guide on developing custom WASM plugins, including the plugin API, exported function signatures, and build instructions, see the [WASM Plugins developer guide](../../developer-guide/extending/plugins.md).

---

## See Also

- [WASM Plugins Developer Guide](../../developer-guide/extending/plugins.md) -- Writing custom plugins
- [Plugin Host Architecture](../../developer-guide/architecture/plugins.md) -- How the plugin host works internally
