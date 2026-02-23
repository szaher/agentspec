# Plugin Manifest Contract

## Overview

Plugins extend the AgentSpec toolchain with custom resource types,
validators, transforms, and lifecycle hooks. Plugins are
distributed as WASM modules with an embedded manifest.

## Manifest Format

```json
{
  "name": "monitor",
  "version": "1.0.0",
  "description": "Adds monitoring resource type",
  "capabilities": {
    "resource_types": [
      {
        "kind": "Monitor",
        "schema": { ... }
      }
    ],
    "validators": [
      {
        "name": "monitor-threshold",
        "applies_to": ["Monitor"],
        "description": "Validates threshold ranges"
      }
    ],
    "transforms": [
      {
        "name": "monitor-to-alerts",
        "stage": "compile",
        "input_kind": "Monitor",
        "description": "Expands Monitor into alert configs"
      }
    ],
    "hooks": [
      {
        "stage": "pre-apply",
        "name": "monitor-preflight",
        "description": "Checks monitoring endpoint reachability"
      }
    ]
  },
  "wasm": {
    "min_memory_pages": 16,
    "max_memory_pages": 256,
    "capabilities": ["wasi_snapshot_preview1"]
  }
}
```

## WASM Module Exports

The WASM module MUST export these functions:

| Export             | Signature             | Description                 |
|--------------------|-----------------------|-----------------------------|
| `manifest`         | `() → ptr, len`      | Returns JSON manifest       |
| `validate`         | `(ptr, len) → ptr, len` | Validate resource JSON   |
| `transform`        | `(ptr, len) → ptr, len` | Transform IR resource    |
| `hook`             | `(ptr, len) → ptr, len` | Execute lifecycle hook   |

All I/O is JSON over shared memory. The host allocates input
buffers; the plugin allocates output buffers.

## Host-Plugin Protocol

1. Host calls `manifest()` to discover capabilities.
2. During validation, host calls `validate(resource_json)` for
   each resource matching the plugin's `applies_to` types.
3. During compilation, host calls `transform(resource_json)` for
   matching resources.
4. At lifecycle stages, host calls `hook(context_json)` for
   registered hooks.

## Isolation Guarantees

- WASM sandbox: no filesystem, network, or OS access unless
  explicitly granted via WASI capabilities.
- Memory limit enforced via `max_memory_pages`.
- Execution timeout enforced by the host (default: 30 seconds).

## Plugin Resolution

Plugins are referenced in `.ias` files:

```
plugin "monitor" version "1.0.0"
```

The host resolves plugins from:
1. Local path: `./plugins/<name>/plugin.wasm`
2. Package cache: `~/.agentspec/plugins/<name>/<version>/plugin.wasm`

## Conflict Rules

- Two plugins MUST NOT declare the same `kind` in
  `resource_types`. The validator rejects this at load time.
- When multiple plugins register hooks at the same `stage`, the
  user MUST declare explicit ordering via `hooks_order` in the
  plugin reference.
