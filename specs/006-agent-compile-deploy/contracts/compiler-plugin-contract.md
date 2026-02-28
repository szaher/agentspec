# Compiler Plugin Contract

**Feature**: 006-agent-compile-deploy

## Overview

Compilation plugins transform AgentSpec IR into framework-specific source code. They extend the existing WASM plugin system with a new capability: `compile`.

## Plugin Manifest Extension

Existing plugin manifest format (from 001-agent-packaging-dsl) extended with compilation capability:

```json
{
  "name": "crewai-compiler",
  "version": "1.0.0",
  "description": "Compile AgentSpec IR to CrewAI Python projects",
  "capabilities": {
    "compile": {
      "target_name": "crewai",
      "output_type": "source_code",
      "output_language": "python",
      "supported_features": [
        "agent",
        "skill",
        "prompt",
        "pipeline_sequential",
        "pipeline_hierarchical",
        "validation_rules",
        "config_params"
      ],
      "unsupported_features": [
        "loop_reflexion",
        "loop_map_reduce",
        "token_budget",
        "inline_tools"
      ]
    }
  },
  "wasm_module": "crewai-compiler.wasm",
  "min_agentspec_version": "0.3.0"
}
```

## WASM Module Exports

Compilation plugins must export these functions:

### `compile(ir_json: string) -> CompileResult`

Transform IR document into framework-specific source code.

**Input**: JSON-serialized IR Document (same schema as `internal/ir/ir.go`)

**Output**: JSON-serialized CompileResult:

```json
{
  "status": "success",
  "files": [
    {
      "path": "main.py",
      "content": "from crewai import ...",
      "mode": "0644"
    },
    {
      "path": "requirements.txt",
      "content": "crewai>=1.9.0\n...",
      "mode": "0644"
    },
    {
      "path": "config/agents.yaml",
      "content": "researcher:\n  role: ...",
      "mode": "0644"
    }
  ],
  "warnings": [
    {
      "feature": "loop_reflexion",
      "message": "CrewAI does not support reflexion loops. Using sequential process instead.",
      "suggestion": "Consider using LangGraph target for reflexion support."
    }
  ],
  "metadata": {
    "framework": "crewai",
    "framework_version": ">=1.9.0",
    "python_version": ">=3.10",
    "run_command": "python main.py"
  }
}
```

### `feature_support() -> FeatureMap`

Return the complete feature support matrix for this target.

**Output**: JSON map of AgentSpec feature → support level:

```json
{
  "agent": "full",
  "skill": "full",
  "prompt": "partial",
  "pipeline_sequential": "full",
  "pipeline_hierarchical": "full",
  "pipeline_conditional": "none",
  "loop_react": "full",
  "loop_plan_execute": "partial",
  "loop_reflexion": "none",
  "loop_router": "none",
  "loop_map_reduce": "none",
  "validation_rules": "emulated",
  "eval_cases": "emulated",
  "config_params": "full",
  "control_flow_if": "emulated",
  "control_flow_foreach": "emulated",
  "streaming": "full",
  "sessions": "partial"
}
```

Support levels:
- `full`: Feature maps directly to framework equivalent
- `partial`: Feature is supported with limitations (documented in warnings)
- `emulated`: Feature is implemented as generated application code, not a framework primitive
- `none`: Feature cannot be represented in this framework

### `version() -> string`

Return the plugin version string.

## Safe Zones for Recompilation

When recompiling over existing generated code, plugins must respect safe zones — regions of generated files where users may have made manual edits.

Safe zone markers:

```python
# --- AGENTSPEC GENERATED START ---
# Do not edit between these markers; changes will be overwritten on recompile
from crewai import Agent, Crew, Process, Task
# --- AGENTSPEC GENERATED END ---

# --- USER CODE START ---
# Your custom code here is preserved across recompilations
def custom_preprocessing(input_data):
    return input_data.strip()
# --- USER CODE END ---
```

Plugins must:
1. Detect existing safe zone markers in output files
2. Preserve content between `USER CODE START` and `USER CODE END`
3. Only overwrite content between `AGENTSPEC GENERATED START` and `AGENTSPEC GENERATED END`
4. If the file doesn't exist, generate with both marker types

## Plugin Discovery

Compilation plugins are discovered in the same locations as existing plugins:
1. `./plugins/<name>/plugin.wasm` (project-local)
2. `~/.agentspec/plugins/<name>/<version>/plugin.wasm` (user-global)

The compiler CLI's `--target` flag maps to the `target_name` field in the plugin manifest.

## Built-in Targets

The `standalone` target is built into the compiler (not a WASM plugin). It uses `go:embed` to produce a native Go binary. All framework targets (crewai, langgraph, llamastack, llamaindex) are implemented as WASM plugins.
