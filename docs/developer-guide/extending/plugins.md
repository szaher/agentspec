# WASM Plugin Guide

This guide covers building, testing, and deploying WASM plugins for the AgentSpec toolchain. Plugins extend the platform with custom validators, resource transforms, and lifecycle hooks.

## Plugin Contract

Every WASM plugin must export a `manifest()` function that returns a JSON-encoded manifest describing the plugin's capabilities. The plugin must also export an `alloc()` function for memory management.

### Required Exports

| Export | Signature | Purpose |
|--------|-----------|---------|
| `manifest` | `() -> (ptr i32, len i32)` | Returns JSON manifest bytes |
| `alloc` | `(size i32) -> (ptr i32)` | Allocates WASM memory for host data |

### Manifest Schema

```json
{
  "name": "my-plugin",
  "version": "1.0.0",
  "description": "Description of what the plugin does",
  "capabilities": {
    "resource_types": [],
    "validators": [],
    "transforms": [],
    "hooks": []
  },
  "wasm": {
    "min_memory_pages": 1,
    "max_memory_pages": 16,
    "capabilities": []
  }
}
```

## Hook Interfaces

Plugins can register hooks at various lifecycle stages. See the [Plugin Architecture page](../architecture/plugins.md) for the full list of stages.

### Validator Hook

A validator checks resources for custom rules. Declare it in the manifest:

```json
{
  "validators": [
    {
      "name": "check_model_allowed",
      "applies_to": ["agent"],
      "description": "Ensures agents use only approved LLM models"
    }
  ]
}
```

Export the corresponding function:

| Export Name | Input | Output |
|-------------|-------|--------|
| `validate_check_model_allowed` or `validate` | JSON-encoded `ir.Resource` | JSON array of error strings (empty = pass) |

### Transform Hook

A transform modifies resources at compile time:

```json
{
  "transforms": [
    {
      "name": "inject_metadata",
      "stage": "compile",
      "input_kind": "agent",
      "description": "Injects standard metadata into agent resources"
    }
  ]
}
```

Export:

| Export Name | Input | Output |
|-------------|-------|--------|
| `transform_inject_metadata` or `transform` | JSON-encoded `ir.Resource` | JSON-encoded modified `ir.Resource` |

### Lifecycle Hook

A lifecycle hook runs at a specific stage:

```json
{
  "hooks": [
    {
      "stage": "post-apply",
      "name": "notify_slack",
      "description": "Sends a Slack notification after apply"
    }
  ]
}
```

Export:

| Export Name | Input | Output |
|-------------|-------|--------|
| `hook_notify_slack` or `post-apply` | JSON-encoded context map | JSON string (output message) |

## Building a Plugin with TinyGo

TinyGo compiles Go to WASM with a small binary size.

### Prerequisites

```bash
# Install TinyGo
brew install tinygo  # macOS
# or see https://tinygo.org/getting-started/install/
```

### Example: Model Allowlist Validator

Create the plugin source:

```go
// plugin.go
package main

import (
    "encoding/json"
    "unsafe"
)

// Manifest returned to the host
var manifestJSON = []byte(`{
  "name": "model-allowlist",
  "version": "1.0.0",
  "description": "Validates that agents use only approved LLM models",
  "capabilities": {
    "validators": [
      {
        "name": "check_model",
        "applies_to": ["agent"],
        "description": "Checks agent model against allowlist"
      }
    ]
  },
  "wasm": {
    "min_memory_pages": 1,
    "max_memory_pages": 8
  }
}`)

var allowedModels = map[string]bool{
    "claude-sonnet-4-20250514":   true,
    "claude-haiku-4-20250414":    true,
}

//export manifest
func manifest() (uint32, uint32) {
    ptr := &manifestJSON[0]
    return uint32(uintptr(unsafe.Pointer(ptr))), uint32(len(manifestJSON))
}

//export alloc
func alloc(size uint32) uint32 {
    buf := make([]byte, size)
    return uint32(uintptr(unsafe.Pointer(&buf[0])))
}

//export validate_check_model
func validateCheckModel(ptr, size uint32) (uint32, uint32) {
    // Read input from WASM memory
    data := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)

    var resource struct {
        Kind       string                 `json:"kind"`
        Name       string                 `json:"name"`
        Attributes map[string]interface{} `json:"attributes"`
    }

    if err := json.Unmarshal(data, &resource); err != nil {
        return writeErrors([]string{"failed to parse resource: " + err.Error()})
    }

    model, _ := resource.Attributes["model"].(string)
    if model == "" {
        return writeErrors([]string{"agent " + resource.Name + ": no model specified"})
    }

    if !allowedModels[model] {
        return writeErrors([]string{
            "agent " + resource.Name + ": model " + model + " is not in the allowlist",
        })
    }

    // No errors -- return empty
    return 0, 0
}

func writeErrors(errs []string) (uint32, uint32) {
    data, _ := json.Marshal(errs)
    ptr := &data[0]
    return uint32(uintptr(unsafe.Pointer(ptr))), uint32(len(data))
}

func main() {}
```

### Build

```bash
tinygo build -o plugin.wasm -target wasi plugin.go
```

### Verify

Check the binary size and exports:

```bash
ls -lh plugin.wasm
wasm-objdump -x plugin.wasm | grep "Export"
```

## Building a Plugin with Rust

Rust produces compact, high-performance WASM modules.

### Setup

```bash
rustup target add wasm32-wasip1
cargo new --lib model-allowlist
cd model-allowlist
```

Add to `Cargo.toml`:

```toml
[lib]
crate-type = ["cdylib"]

[dependencies]
serde = { version = "1", features = ["derive"] }
serde_json = "1"
```

### Implementation

```rust
// src/lib.rs
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

static MANIFEST: &str = r#"{
  "name": "model-allowlist",
  "version": "1.0.0",
  "description": "Validates that agents use only approved LLM models",
  "capabilities": {
    "validators": [{
      "name": "check_model",
      "applies_to": ["agent"],
      "description": "Checks agent model against allowlist"
    }]
  },
  "wasm": {"min_memory_pages": 1, "max_memory_pages": 8}
}"#;

#[no_mangle]
pub extern "C" fn manifest() -> u64 {
    let ptr = MANIFEST.as_ptr() as u32;
    let len = MANIFEST.len() as u32;
    ((ptr as u64) << 32) | (len as u64)
}

#[no_mangle]
pub extern "C" fn alloc(size: u32) -> u32 {
    let layout = std::alloc::Layout::from_size_align(size as usize, 1).unwrap();
    unsafe { std::alloc::alloc(layout) as u32 }
}

#[derive(Deserialize)]
struct Resource {
    kind: String,
    name: String,
    attributes: HashMap<String, serde_json::Value>,
}

#[no_mangle]
pub extern "C" fn validate_check_model(ptr: u32, size: u32) -> u64 {
    let data = unsafe { std::slice::from_raw_parts(ptr as *const u8, size as usize) };

    let resource: Resource = match serde_json::from_slice(data) {
        Ok(r) => r,
        Err(e) => return write_errors(&[format!("parse error: {}", e)]),
    };

    let allowed = ["claude-sonnet-4-20250514", "claude-haiku-4-20250414"];

    if let Some(model) = resource.attributes.get("model").and_then(|v| v.as_str()) {
        if !allowed.contains(&model) {
            return write_errors(&[format!(
                "agent {}: model {} is not allowed",
                resource.name, model
            )]);
        }
    }

    0 // No errors
}

fn write_errors(errors: &[String]) -> u64 {
    let json = serde_json::to_vec(errors).unwrap();
    let ptr = json.as_ptr() as u32;
    let len = json.len() as u32;
    std::mem::forget(json);
    ((ptr as u64) << 32) | (len as u64)
}
```

### Build

```bash
cargo build --release --target wasm32-wasip1
cp target/wasm32-wasip1/release/model_allowlist.wasm plugin.wasm
```

## Testing Plugins

### Unit Testing (in the Plugin Language)

Test the plugin logic in its native language before compiling to WASM:

```go
// plugin_test.go (TinyGo -- run with standard go test, not tinygo)
func TestCheckModel_Allowed(t *testing.T) {
    resource := map[string]interface{}{
        "kind": "agent",
        "name": "assistant",
        "attributes": map[string]interface{}{
            "model": "claude-sonnet-4-20250514",
        },
    }
    data, _ := json.Marshal(resource)
    // Test the validation logic directly
    // ...
}
```

### Integration Testing with the Host

Test the compiled WASM module against the AgentSpec plugin host:

```go
func TestPluginIntegration(t *testing.T) {
    ctx := context.Background()
    host, err := plugins.NewHost(ctx)
    if err != nil {
        t.Fatal(err)
    }
    defer host.Close(ctx)

    plugin, err := host.LoadPlugin(ctx, "testdata/model_allowlist.wasm")
    if err != nil {
        t.Fatal(err)
    }

    if plugin.Manifest.Name != "model-allowlist" {
        t.Errorf("unexpected name: %s", plugin.Manifest.Name)
    }

    // Test validation
    resources := []ir.Resource{
        {
            Kind: "agent",
            Name: "test-agent",
            Attributes: map[string]interface{}{
                "model": "disallowed-model",
            },
        },
    }

    errs := plugins.ValidateResources(host, []*plugins.LoadedPlugin{plugin}, resources)
    if len(errs) == 0 {
        t.Error("expected validation error for disallowed model")
    }
}
```

## Deployment

### Local Development

Place the compiled `.wasm` file in the project's `plugins/` directory:

```text
plugins/
  model-allowlist/
    plugin.wasm
```

### User-Level Installation

Install to the user plugin cache:

```bash
mkdir -p ~/.agentspec/plugins/model-allowlist/1.0.0/
cp plugin.wasm ~/.agentspec/plugins/model-allowlist/1.0.0/plugin.wasm
```

### Declaring in IntentLang

Reference the plugin in your `.ias` file:

```text
plugin "model-allowlist" version "1.0.0"
```

The plugin loader will search for the WASM module using the resolution order described in the [Plugin Architecture page](../architecture/plugins.md).

## Debugging

### Stdout/Stderr

Plugins can write to stdout and stderr via WASI. Output appears in the AgentSpec CLI output:

```go
//export validate_check_model
func validateCheckModel(ptr, size uint32) (uint32, uint32) {
    fmt.Fprintf(os.Stderr, "validating resource at ptr=%d size=%d\n", ptr, size)
    // ...
}
```

### Binary Inspection

Inspect the WASM module's exports and imports:

```bash
wasm-objdump -x plugin.wasm
```

Verify the required exports are present: `manifest`, `alloc`, and your hook functions.
