# Client

A **client** block defines an MCP (Model Context Protocol) client that connects to
one or more MCP servers. Agents reference clients to gain access to the skills
exposed by those servers.

---

## Syntax

```ias novalidate
client "<name>" {
  connects to server "<server-ref>"
  # ... more server connections
}
```

`<name>` is a unique identifier for the client within the package. Each
`connects to server` directive establishes a connection to a named server.

---

## Attributes

| Attribute              | Type | Required | Repeatable | Description                                              |
|:-----------------------|:-----|:---------|:-----------|:---------------------------------------------------------|
| `connects to server`   | ref  | Yes      | Yes        | Reference to a server block defined in the same package. |

A client must connect to **at least one server**. Multiple `connects to server`
directives can be specified to aggregate skills from several servers into a
single client.

---

## Rules

- Client names must be **unique within the package**.
- Each `connects to server` directive must reference a `server` block in the same package.
- A client must have **at least one** server connection.
- An agent references a client with the `connects to client` directive.

!!! info "Client-server relationship"
    A client does not expose skills directly. It acts as a connection point.
    The skills available through a client are determined by the servers it
    connects to and the skills those servers expose.

---

## Examples

### Single Server Connection

The simplest case: a client that connects to one server.

```ias
package "mcp-basic" version "0.1.0" lang "2.0"

prompt "assistant" {
  content "You are a helpful assistant with file access."
}

skill "file-read" {
  description "Read files from disk"
  input  { path string required }
  output { content string }
  tool command { binary "file-reader" }
}

server "file-server" {
  transport "stdio"
  command "file-mcp-server"
  exposes skill "file-read"
}

client "my-client" {
  connects to server "file-server"
}

agent "file-assistant" {
  uses prompt "assistant"
  uses skill "file-read"
  model "claude-sonnet-4-20250514"
  connects to client "my-client"
}

deploy "local" target "process" {
  default true
}
```

The agent `file-assistant` connects to `my-client`, which connects to
`file-server`. This gives the agent access to the `file-read` skill served
over stdio.

### Multiple Server Connections

A client that aggregates skills from several servers.

```ias fragment
server "db-server" {
  transport "stdio"
  command "db-mcp-server"
  exposes skill "query-db"
  exposes skill "list-tables"
}

server "search-server" {
  transport "sse"
  url "https://search.example.com/mcp"
  exposes skill "full-text-search"
}

client "data-client" {
  connects to server "db-server"
  connects to server "search-server"
}

agent "data-analyst" {
  uses prompt "analyst"
  uses skill "query-db"
  uses skill "list-tables"
  uses skill "full-text-search"
  model "claude-sonnet-4-20250514"
  connects to client "data-client"
}
```

Here, `data-client` connects to both `db-server` (stdio) and `search-server`
(SSE). The `data-analyst` agent gains access to all three skills through a
single client reference.

!!! tip "One client per agent"
    While you can define multiple client blocks, a common pattern is to create
    one client per agent, connecting it to exactly the servers that agent needs.
    This keeps the relationship between agents and their tool backends explicit.

### Multi-Agent, Shared Server

Multiple agents can use separate clients that connect to the same server.

```ias fragment
server "review-server" {
  transport "stdio"
  command "review-mcp-server"
  exposes skill "read-diff"
  exposes skill "analyze-code"
  exposes skill "scan-security"
  exposes skill "post-review"
}

client "analyzer-client" {
  connects to server "review-server"
}

client "scanner-client" {
  connects to server "review-server"
}

agent "code-analyzer" {
  uses prompt "analyzer"
  uses skill "read-diff"
  uses skill "analyze-code"
  model "claude-sonnet-4-20250514"
  connects to client "analyzer-client"
}

agent "security-scanner" {
  uses prompt "security-reviewer"
  uses skill "read-diff"
  uses skill "scan-security"
  model "claude-sonnet-4-20250514"
  connects to client "scanner-client"
}
```

Both agents connect to the same underlying `review-server`, but through
separate client instances. This provides logical separation even when the
physical server is shared.

---

## See Also

- [Server](server.md) -- the MCP servers that clients connect to
- [Agent](agent.md) -- uses `connects to client` to access server skills
- [Skill](skill.md) -- the skills exposed through the server-client chain
