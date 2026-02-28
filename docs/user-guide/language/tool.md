# Tool

The `tool` block inside a `skill` defines how the skill is executed. IntentLang supports four tool variants, each designed for a different integration pattern: MCP server delegation, HTTP API calls, local command execution, and inline code.

Every skill must contain exactly one `tool` block.

---

## MCP Tool

The `mcp` variant delegates execution to a tool exposed by an MCP (Model Context Protocol) server. This is the preferred approach when you have MCP-compatible tool servers.

### Syntax

```ias
tool mcp "<server-name>/<tool-name>"
```

The argument is a string in the format `"server/tool"`, where `server` is the name of a `server` resource declared in the same package, and `tool` is the name of the tool exposed by that server.

### Attributes

The `mcp` variant takes no additional attributes -- the server and tool names are specified in the string argument.

| Name       | Type   | Required | Description                                                        |
|------------|--------|----------|--------------------------------------------------------------------|
| (argument) | string | Yes      | MCP server and tool name in `"server/tool"` format.                |

### Example

```ias
package "mcp-example" version "0.1.0" lang "2.0"

prompt "assistant" {
  content "You are a file management assistant."
}

skill "read-file" {
  description "Read the contents of a file from disk"
  input {
    path string required
  }
  output {
    content string
  }
  tool mcp "file-server/read-file"
}

server "file-server" {
  transport "stdio"
  command "file-mcp-server"
  exposes skill "read-file"
}

client "my-client" {
  connects to server "file-server"
}

agent "file-assistant" {
  uses prompt "assistant"
  uses skill "read-file"
  model "claude-sonnet-4-20250514"
  connects to client "my-client"
}

deploy "local" target "process" {
  default true
}
```

!!! tip "When to Use MCP"
    Use the `mcp` variant when tools are provided by external MCP-compatible servers. This decouples tool implementation from agent definition and enables tool reuse across multiple agents and packages.

---

## HTTP Tool

The `http` variant makes HTTP requests to external APIs. This is useful for integrating with REST APIs, webhooks, and third-party services.

### Syntax

```ias
tool http {
  method "<HTTP-method>"
  url "<endpoint-url>"
  headers {
    <Header-Name> "<value>"
  }
  body_template "<template>"
  timeout "<duration>"
}
```

### Attributes

| Name            | Type   | Required | Description                                                          |
|-----------------|--------|----------|----------------------------------------------------------------------|
| `method`        | string | Yes      | HTTP method. Valid values: `"GET"`, `"POST"`, `"PUT"`, `"PATCH"`, `"DELETE"`. |
| `url`           | string | Yes      | The endpoint URL. Supports `{{variable}}` placeholders from input fields. |
| `headers`       | block  | No       | HTTP headers as key-value pairs. Values support `{{variable}}` placeholders. |
| `body_template` | string | No       | Request body template. Supports `{{variable}}` placeholders. Typically JSON. |
| `timeout`       | string | No       | Request timeout as a duration string (e.g. `"10s"`, `"30s"`).        |

### Valid Method Values

| Value      | Description                  |
|------------|------------------------------|
| `"GET"`    | Retrieve a resource          |
| `"POST"`   | Create a resource            |
| `"PUT"`    | Replace a resource           |
| `"PATCH"`  | Partially update a resource  |
| `"DELETE"` | Delete a resource            |

### Example

```ias
package "http-tool" version "0.1.0" lang "2.0"

prompt "search-assistant" {
  content "You are a search assistant. Use the search API to find\nrelevant information and present results clearly."
}

skill "api-search" {
  description "Search an external API for information"
  input {
    query string required
    limit int
  }
  output {
    results string
  }
  tool http {
    method "POST"
    url "https://api.example.com/v1/search"
    headers {
      Authorization "Bearer {{API_KEY}}"
      Content-Type "application/json"
    }
    body_template "{\"query\": \"{{query}}\", \"limit\": {{limit}}}"
    timeout "10s"
  }
}

secret "api-key" {
  env(API_KEY)
}

agent "searcher" {
  uses prompt "search-assistant"
  uses skill "api-search"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 10
}

deploy "local" target "process" {
  default true
}
```

!!! warning "Sensitive Headers"
    Avoid hardcoding secrets in headers. Use `{{variable}}` placeholders that reference `secret` resources or environment variables. This keeps credentials out of your `.ias` files.

---

## Command Tool

The `command` variant executes a local binary or script. The binary receives input as JSON on stdin and must write output as JSON to stdout.

### Syntax

```ias
tool command {
  binary "<executable>"
  args ["<arg1>", "<arg2>"]
  timeout "<duration>"
  env {
    <VAR_NAME> "<value>"
  }
  secrets {
    <ENV_VAR> "<secret-name>"
  }
}
```

### Attributes

| Name      | Type     | Required | Description                                                                |
|-----------|----------|----------|----------------------------------------------------------------------------|
| `binary`  | string   | Yes      | Name or path of the executable to run. Resolved from `$PATH` if not absolute. |
| `args`    | array    | No       | Command-line arguments passed to the binary.                               |
| `timeout` | string   | No       | Maximum execution time (e.g. `"15s"`, `"1m"`). The process is killed if exceeded. |
| `env`     | block    | No       | Environment variables passed to the process as key-value pairs.            |
| `secrets` | block    | No       | Secret-to-environment-variable mappings as key-value pairs.                |

### Example

```ias
package "command-tool" version "0.1.0" lang "2.0"

prompt "ops" {
  content "You are an operations assistant. Use available tools to\ncheck system health and diagnose issues."
}

skill "check-health" {
  description "Check the health status of a service"
  input {
    service string required
    verbose bool
  }
  output {
    status string
    details string
  }
  tool command {
    binary "health-check"
    args ["--format", "json"]
    timeout "15s"
    env {
      CHECK_ENDPOINT "https://monitoring.internal"
    }
    secrets {
      MONITORING_API_KEY "monitoring-api-key"
    }
  }
}

secret "monitoring-api-key" {
  env(MONITORING_API_KEY)
}

agent "ops-bot" {
  uses prompt "ops"
  uses skill "check-health"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 10
}

deploy "local" target "process" {
  default true
}
```

!!! tip "Binary Resolution"
    The `binary` value is resolved from the system `$PATH`. For predictable deployments, consider using absolute paths or ensuring the binary is included in your AgentPack.

---

## Inline Tool

The `inline` variant executes embedded code in a sandboxed WebAssembly (WASM) environment. This is useful for lightweight transformations, formatting, and computations that do not warrant a separate binary.

### Syntax

```ias
tool inline {
  language "<language>"
  code "<source-code>"
  timeout "<duration>"
  memory "<megabytes>"
}
```

### Attributes

| Name       | Type   | Required | Description                                                             |
|------------|--------|----------|-------------------------------------------------------------------------|
| `language` | string | Yes      | Programming language of the inline code.                                |
| `code`     | string | Yes      | Source code to execute. Use `\n` for newlines. Input is available via `input` variable. |
| `timeout`  | string | No       | Maximum execution time (e.g. `"5s"`, `"10s"`). Default: `"10s"`.        |
| `memory`   | string | No       | Maximum memory allocation in megabytes (e.g. `"32"`, `"64"`). Default: `"64"`. |

### Valid Language Values

| Value        | Description               |
|--------------|---------------------------|
| `"python"`   | Python 3.x runtime        |
| `"javascript"` | JavaScript (ES2020+)    |

### Example

```ias
package "inline-tool" version "0.1.0" lang "2.0"

prompt "formatter" {
  content "You are a text formatting assistant. Use the formatting\ntool to process and transform text as requested."
}

skill "format-json" {
  description "Pretty-print and validate a JSON string"
  input {
    raw_json string required
  }
  output {
    formatted string
  }
  tool inline {
    language "python"
    code "import json\ndata = json.loads(input['raw_json'])\nresult = json.dumps(data, indent=2, sort_keys=True)\nprint(json.dumps({'formatted': result}))"
    timeout "5s"
    memory "32"
  }
}

agent "formatter" {
  uses prompt "formatter"
  uses skill "format-json"
  model "claude-sonnet-4-20250514"
  max_turns 5
}

deploy "local" target "process" {
  default true
}
```

!!! warning "Sandbox Limitations"
    Inline tools run in a WASM sandbox with no filesystem or network access. They are suitable for pure computation and transformation, not for I/O-bound tasks. Use `command` or `http` tools for tasks that require external access.

---

## Choosing a Tool Variant

| Requirement                                | Recommended Variant |
|--------------------------------------------|---------------------|
| Integrate with an MCP-compatible server     | `mcp`               |
| Call a REST API or webhook                  | `http`              |
| Run an existing binary or script            | `command`           |
| Lightweight computation without dependencies | `inline`           |

!!! tip "Start with Command"
    If you are prototyping, the `command` tool variant is the fastest way to get started. Write a small script that reads JSON from stdin and writes JSON to stdout, then reference it from your skill.
