# IR Schema Contract

## Overview

The Intermediate Representation (IR) is the canonical,
platform-neutral data format produced after parsing, lowering,
and validating DSL source. Adapters and SDKs consume IR, never
raw DSL. The IR schema is defined as JSON Schema and serves as
the single source of truth for SDK type generation.

## Top-Level Structure

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["ir_version", "lang_version", "package", "resources"],
  "properties": {
    "ir_version": { "type": "string", "pattern": "^\\d+\\.\\d+$" },
    "lang_version": { "type": "string", "pattern": "^\\d+\\.\\d+$" },
    "package": { "$ref": "#/$defs/Package" },
    "resources": {
      "type": "array",
      "items": { "$ref": "#/$defs/Resource" }
    },
    "policies": {
      "type": "array",
      "items": { "$ref": "#/$defs/Policy" }
    },
    "bindings": {
      "type": "array",
      "items": { "$ref": "#/$defs/Binding" }
    }
  }
}
```

## Resource Schema

```json
{
  "$defs": {
    "Resource": {
      "type": "object",
      "required": ["kind", "name", "fqn", "attributes", "hash"],
      "properties": {
        "kind": {
          "type": "string",
          "enum": ["Agent", "Prompt", "Skill", "MCPServer",
                   "MCPClient", "Environment", "Secret"]
        },
        "name": { "type": "string", "minLength": 1 },
        "fqn": {
          "type": "string",
          "pattern": "^[a-z0-9-]+/[A-Z][a-zA-Z]+/[a-z0-9-]+$"
        },
        "attributes": { "type": "object" },
        "references": {
          "type": "array",
          "items": { "type": "string" }
        },
        "hash": {
          "type": "string",
          "pattern": "^sha256:[a-f0-9]{64}$"
        },
        "metadata": { "type": "object" }
      }
    }
  }
}
```

## Determinism Rules

1. Resources MUST be serialized sorted by `kind` (alphabetical),
   then by `name` (alphabetical) within each kind.
2. All JSON object keys MUST be sorted alphabetically.
3. JSON output MUST use 2-space indentation, no trailing
   whitespace, and a final newline.
4. The `hash` field MUST be computed as SHA-256 of the
   canonical JSON serialization of `attributes` (with sorted
   keys, no whitespace).

## Versioning

- `ir_version`: Follows `major.minor` semver. Major increments
  indicate breaking schema changes.
- `lang_version`: Tracks the DSL language version independently.
- Both versions MUST be present in every IR document.

## Extension Points

Plugin-defined resource types extend the `kind` enum. Custom
kinds MUST be prefixed with the plugin name to prevent future
collisions with built-in types (e.g., `monitor.Threshold`
rather than `Threshold`).

## Validation

The IR JSON MUST validate against `spec/ir.schema.json` before
being passed to any adapter. The validator ensures:

- All required fields present
- All references (`fqn` values in `references`) resolve to
  existing resources in the same IR document
- All hashes are correctly computed
- No duplicate `fqn` values
- Resource ordering matches determinism rules
