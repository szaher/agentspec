# Plugin Host

The plugin system extends the AgentSpec toolchain with custom validators, transforms, and lifecycle hooks. Plugins are compiled to WebAssembly (WASM) and run in a sandboxed environment using the wazero runtime (v1.11.0).

## Package

| Package | Path | Purpose |
|---------|------|---------|
| `plugins` | `internal/plugins/` | WASM host, plugin loading, hook dispatch, validation, transforms |

Source files:

| File | Purpose |
|------|---------|
| `host.go` | WASM runtime initialization and plugin loading |
| `loader.go` | Plugin manifest, path resolution, conflict detection |
| `hooks.go` | Lifecycle hook dispatch |
| `validate.go` | Plugin-provided resource validation |
| `transform.go` | Plugin-provided resource transforms |

## Plugin Manifest

Every plugin declares its capabilities through a JSON manifest. The manifest is returned by the plugin's exported `manifest()` WASM function.

```go
type Manifest struct {
    Name         string       `json:"name"`
    Version      string       `json:"version"`
    Description  string       `json:"description"`
    Capabilities Capabilities `json:"capabilities"`
    WASM         WASMConfig   `json:"wasm"`
}

type Capabilities struct {
    ResourceTypes []ResourceType `json:"resource_types,omitempty"`
    Validators    []Validator    `json:"validators,omitempty"`
    Transforms    []Transform    `json:"transforms,omitempty"`
    Hooks         []Hook         `json:"hooks,omitempty"`
}
```

### Capability Types

**ResourceType** -- Declares a custom resource kind with an optional JSON schema:

```go
type ResourceType struct {
    Kind   string                 `json:"kind"`
    Schema map[string]interface{} `json:"schema,omitempty"`
}
```

**Validator** -- A custom validation rule applied to specific resource kinds:

```go
type Validator struct {
    Name        string   `json:"name"`
    AppliesTo   []string `json:"applies_to"` // Resource kinds, or "*" for all
    Description string   `json:"description"`
}
```

**Transform** -- Modifies resources at compile time:

```go
type Transform struct {
    Name        string `json:"name"`
    Stage       string `json:"stage"`       // "compile"
    InputKind   string `json:"input_kind"`  // Resource kind to transform
    Description string `json:"description"`
}
```

**Hook** -- Executes at specific lifecycle stages:

```go
type Hook struct {
    Stage       string `json:"stage"`
    Name        string `json:"name"`
    Description string `json:"description"`
}
```

## Hook Stages

Hooks can execute at these lifecycle stages:

| Stage | When |
|-------|------|
| `pre-validate` | Before built-in validation runs |
| `post-validate` | After validation completes |
| `pre-plan` | Before the plan engine computes diffs |
| `post-plan` | After the plan is computed |
| `pre-apply` | Before actions are dispatched to adapters |
| `post-apply` | After all actions complete |
| `pre-invoke` | Before an agent invocation (runtime) |
| `post-invoke` | After an agent invocation (runtime) |
| `runtime` | During runtime execution |

## WASM Host

The `Host` struct manages the wazero runtime and all loaded plugins:

```go
type Host struct {
    runtime wazero.Runtime
    plugins map[string]*LoadedPlugin
}

type LoadedPlugin struct {
    Manifest Manifest
    module   wazero.CompiledModule
}
```

### Initialization

```go
func NewHost(ctx context.Context) (*Host, error) {
    rt := wazero.NewRuntime(ctx)
    wasi_snapshot_preview1.MustInstantiate(ctx, rt)
    return &Host{
        runtime: rt,
        plugins: make(map[string]*LoadedPlugin),
    }, nil
}
```

The host initializes WASI (WebAssembly System Interface) preview 1 for basic I/O support.

### Plugin Loading

`LoadPlugin()` loads and validates a WASM module:

1. Reads the `.wasm` file from disk.
2. Compiles the module using the wazero runtime.
3. Instantiates the module to call its `manifest()` export.
4. Parses the returned JSON manifest.
5. Stores the compiled module and manifest for later use.

## Plugin Contract

A WASM plugin must export the following functions:

### Required Exports

| Function | Signature | Purpose |
|----------|-----------|---------|
| `manifest` | `() -> (ptr i32, len i32)` | Returns JSON manifest bytes |
| `alloc` | `(size i32) -> (ptr i32)` | Allocates memory in the WASM module |

### Optional Exports (by capability)

| Function | Signature | Purpose |
|----------|-----------|---------|
| `validate` or `validate_<name>` | `(ptr i32, len i32) -> (err_ptr i32, err_len i32)` | Validates a resource (JSON in, error list out) |
| `transform` or `transform_<name>` | `(ptr i32, len i32) -> (out_ptr i32, out_len i32)` | Transforms a resource (JSON in, JSON out) |
| `hook_<name>` or `<stage>` | `(ptr i32, len i32) -> (out_ptr i32, out_len i32)` | Executes a lifecycle hook |

### Memory Protocol

Data is exchanged between the host and plugin via shared linear memory:

1. **Host to Plugin** -- The host calls `alloc(size)` to get a memory pointer in the WASM module, then writes the input data at that pointer.
2. **Plugin to Host** -- The plugin function returns `(ptr, len)`. The host reads `len` bytes from `ptr` in the module's memory.

## Plugin Path Resolution

The `ResolvePluginPath()` function searches for plugin WASM files in order:

1. `./plugins/<name>/plugin.wasm` (local project directory)
2. `~/.agentspec/plugins/<name>/<version>/plugin.wasm` (user cache)
3. `~/.agentz/plugins/<name>/<version>/plugin.wasm` (deprecated fallback, emits a warning)

## Conflict Detection

Before running plugins, `CheckConflicts()` verifies that no two plugins declare the same custom resource type:

```go
func CheckConflicts(plugins []*LoadedPlugin) error
```

This prevents ambiguity when multiple plugins try to own the same resource kind.

## Validation Dispatch

`ValidateResources()` runs plugin validators against matching resources:

```go
func ValidateResources(host *Host, plugins []*LoadedPlugin, resources []ir.Resource) []error
```

For each plugin validator, the function checks if the resource's kind matches the validator's `applies_to` list (or `"*"` for all kinds). Matching resources are serialized to JSON and passed to the WASM validator function.

## Transform Dispatch

`TransformResources()` applies plugin transforms at the compile stage:

```go
func TransformResources(host *Host, plugins []*LoadedPlugin, resources []ir.Resource) ([]ir.Resource, error)
```

Transforms only run when `stage == "compile"` and the resource kind matches the transform's `input_kind`. The resource is serialized to JSON, sent to the WASM function, and the returned JSON is deserialized back into an `ir.Resource`.

## Hook Dispatch

`ExecuteHooks()` runs all hooks registered for a given stage:

```go
func ExecuteHooks(host *Host, plugins []*LoadedPlugin, stage HookStage,
                  hookCtx map[string]interface{}) ([]HookResult, error)
```

Each hook result includes the plugin name, hook name, output, and success/error status:

```go
type HookResult struct {
    Plugin  string
    Hook    string
    Stage   HookStage
    Output  string
    Success bool
    Error   string
}
```

## WASM Configuration

The manifest includes WASM runtime constraints:

```go
type WASMConfig struct {
    MinMemoryPages int      `json:"min_memory_pages"`
    MaxMemoryPages int      `json:"max_memory_pages"`
    Capabilities   []string `json:"capabilities"`
}
```

Memory pages are 64 KiB each. The `capabilities` list controls what the plugin can access (future: network, filesystem).
