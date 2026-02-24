# Server

A **server** block defines an MCP (Model Context Protocol) server that exposes one
or more skills over a transport protocol. Servers act as the bridge between agent
skills and external systems -- they run tool implementations and make them available
to clients over stdio, SSE, or streamable HTTP.

---

## Syntax

```ias novalidate
server "<name>" {
  transport "<transport-type>"

  # stdio transport attributes
  command "<executable>"
  args    ["<arg1>", "<arg2>", ...]

  # sse / streamable-http transport attributes
  url  "<server-url>"
  auth "<secret-ref>"

  # common attributes
  exposes skill "<skill-ref>"
  env {
    <KEY> "<value>"
  }
}
```

---

## Attributes

| Attribute       | Type        | Required    | Condition                             | Description                                                       |
|:----------------|:------------|:------------|:--------------------------------------|:------------------------------------------------------------------|
| `transport`     | string      | Yes         | --                                    | Transport protocol: `"stdio"`, `"sse"`, or `"streamable-http"`.   |
| `command`       | string      | Conditional | Required when transport is `"stdio"`  | The executable command to launch the MCP server process.          |
| `args`          | string list | No          | Only with `"stdio"` transport         | Arguments passed to the command.                                  |
| `url`           | string      | Conditional | Required when transport is `"sse"` or `"streamable-http"` | The URL of the remote MCP server.             |
| `auth`          | secret ref  | No          | Only with `"sse"` or `"streamable-http"` | Reference to a secret used for authentication.                 |
| `exposes skill` | ref         | No          | Repeatable                            | A skill that this server exposes. Declare once per skill.         |
| `env`           | map         | No          | --                                    | Environment variables passed to the server process.               |

---

## Transport Types

### stdio

The server runs as a **local child process**. The AgentSpec runtime launches the
command, communicates over stdin/stdout, and manages the process lifecycle.

Best for:

- Local development
- CLI tools and scripts
- Self-contained server binaries

!!! info "Process management"
    When transport is `"stdio"`, AgentSpec starts the server process on
    `agentspec apply` and stops it on shutdown. The process inherits the
    environment variables defined in the `env` block.

### sse

The server is a **remote HTTP endpoint** that uses Server-Sent Events for
streaming responses. The client connects to the URL and receives tool results
as an event stream.

Best for:

- Shared development servers
- Staging environments
- Services behind a load balancer

### streamable-http

The server is a **remote HTTP endpoint** that uses chunked transfer encoding
for streaming. This is the recommended transport for production deployments
where SSE is not suitable.

Best for:

- Production environments
- High-throughput workloads
- Services requiring bidirectional streaming

---

## Rules

- Server names must be **unique within the package**.
- When transport is `"stdio"`, the `command` attribute is required and `url` must not be set.
- When transport is `"sse"` or `"streamable-http"`, the `url` attribute is required and `command` must not be set.
- Each `exposes skill` directive must reference a skill defined in the same package.
- The `auth` attribute must reference a `secret` block defined in the same package.

!!! warning "Transport and attribute mismatch"
    `agentspec validate` rejects server blocks where `command` is set on an
    `"sse"` transport, or where `url` is set on a `"stdio"` transport. Make sure
    the attributes match the chosen transport.

---

## Examples

### stdio Server

A local MCP server that exposes file-system skills.

```ias
package "file-tools" version "0.1.0" lang "2.0"

prompt "assistant" {
  content "You are a file management assistant."
}

skill "file-read" {
  description "Read files from disk"
  input  { path string required }
  output { content string }
  tool command { binary "file-reader" }
}

skill "file-write" {
  description "Write content to a file"
  input {
    path string required
    content string required
  }
  output { bytes_written int }
  tool command { binary "file-writer" }
}

server "file-server" {
  transport "stdio"
  command "file-mcp-server"
  args ["--verbose", "--root", "/data"]
  exposes skill "file-read"
  exposes skill "file-write"
  env {
    LOG_LEVEL "debug"
  }
}

client "local-client" {
  connects to server "file-server"
}

agent "file-bot" {
  uses prompt "assistant"
  uses skill "file-read"
  uses skill "file-write"
  model "claude-sonnet-4-20250514"
  connects to client "local-client"
}

deploy "local" target "process" {
  default true
}
```

### SSE Server with Authentication

A remote MCP server accessed over SSE with secret-based authentication.

```ias fragment
secret "api-token" {
  env(MCP_API_TOKEN)
}

skill "search-docs" {
  description "Search the documentation index"
  input  { query string required }
  output { results string }
  tool command { binary "doc-search" }
}

skill "index-docs" {
  description "Index new documents"
  input  { path string required }
  output { indexed_count int }
  tool command { binary "doc-indexer" }
}

server "docs-server" {
  transport "sse"
  url "https://mcp.example.com/docs"
  auth "api-token"
  exposes skill "search-docs"
  exposes skill "index-docs"
}
```

### Streamable HTTP Server

A production MCP server using the streamable-http transport.

```ias novalidate
secret "service-key" {
  store(production/mcp/service-key)
}

server "inference-server" {
  transport "streamable-http"
  url "https://inference.prod.example.com/mcp"
  auth "service-key"
  exposes skill "classify"
  exposes skill "summarize"
}
```

!!! tip "Choosing a transport"
    Use `"stdio"` for local development and testing. Use `"sse"` or
    `"streamable-http"` when the server runs as a shared service. Prefer
    `"streamable-http"` for production workloads that require reliable
    bidirectional communication.

---

## See Also

- [Client](client.md) -- connects agents to servers
- [Skill](skill.md) -- the skills exposed by a server
- [Secret](secret.md) -- authentication credentials referenced by `auth`
