# Config

The `config` block declares runtime configuration parameters for an agent. Parameters are resolved from environment variables, config files, or default values, and are available to the agent during execution.

---

## Syntax

<!-- novalidate -->
```ias
agent "<name>" {
  config {
    <param_name> <type> [default "<value>"] [required] [secret]
      "<description>"
  }
}
```

The `config` block is nested inside an `agent` block and contains one or more parameter declarations.

---

## Parameter Attributes

| Attribute   | Required | Description                                                        |
|-------------|----------|--------------------------------------------------------------------|
| name        | Yes      | Parameter name. Used as the key for resolution and access.          |
| type        | Yes      | Data type: `string`, `int`, `float`, or `bool`.                    |
| `default`   | No       | Default value (quoted string). Used when no override is provided.   |
| `required`  | No       | Marks the parameter as mandatory. Validation fails if not provided. |
| `secret`    | No       | Marks the parameter as sensitive. Values are masked in logs and output. |
| description | Yes      | Human-readable description (quoted string on the following line).   |

!!! note "Required vs Default"
    A parameter can have `default` or `required`, but not both. If neither is specified, the parameter is optional with a zero-value default for its type.

---

## Types

| Type     | Zero Value | Example Values            |
|----------|------------|---------------------------|
| `string` | `""`       | `"hello"`, `"production"` |
| `int`    | `0`        | `"10000"`, `"42"`         |
| `float`  | `0.0`      | `"0.8"`, `"3.14"`         |
| `bool`   | `false`    | `"true"`, `"false"`       |

All default values are specified as quoted strings regardless of type. The runtime converts them to the declared type.

---

## Resolution Order

Config parameters are resolved in the following order (first match wins):

1. **CLI flag** -- `--config <file>` loads a JSON or YAML config file.
2. **Environment variable** -- `AGENTSPEC_<AGENT>_<PARAM>` (uppercased, hyphens replaced with underscores).
3. **Default value** -- The `default` value from the config block.

For a parameter `api_key` in an agent named `support-agent`, the environment variable is:

```
AGENTSPEC_SUPPORT_AGENT_API_KEY
```

!!! tip "Secret Parameters"
    Parameters marked `secret` follow the same resolution order but their values are never printed in plan output, logs, or error messages.

---

## Config File

Pass a config file at runtime with the `--config` flag:

```bash
agentspec run --config config.json
```

The file maps agent names to parameter key-value pairs:

```json
{
  "support-agent": {
    "company_name": "Acme Corp",
    "max_response_length": "500",
    "api_key": "sk-abc123"
  }
}
```

---

## Examples

### Basic Config

<!-- novalidate -->
```ias
agent "coder" {
  uses prompt "coder-system"
  uses skill "list-files"
  model "ollama/llama3.1"
  strategy "react"
  config {
    working_dir string default "."
      "Working directory for file operations"
    max_file_size int default "10000"
      "Maximum file size to read in bytes"
  }
}
```

### Config with Required and Secret Parameters

<!-- novalidate -->
```ias
agent "support-agent" {
  uses prompt "support-system"
  uses skill "knowledge-search"
  model "claude-sonnet-4-5-20250514"
  strategy "react"
  config {
    company_name string default "Acme Corp"
      "Company name"
    max_response_length int default "500"
      "Max response length"
    api_key string required secret
      "API key for external services"
    support_tier string default "standard"
      "Support tier level"
  }
}
```

In this example, `api_key` must be provided via an environment variable or config file. Its value will be masked in all output.

---

## See Also

- [Agent](agent.md) -- The parent block that contains `config`
- [Validate](validate.md) -- Output validation rules within agents
- [Eval](eval.md) -- Evaluation test cases within agents
