# MCP Server/Client

An agent connected to external tools through Model Context Protocol (MCP) transport.

## What This Demonstrates

- **MCP Server** resource that exposes skills over a transport protocol
- **MCP Client** resource that connects to a server
- **Agent-to-client** connectivity via `connects to client`
- **Transport configuration** with `stdio` protocol

## Definition Structure

### Server

```
server "file-server" {
  transport "stdio"
  command "file-mcp-server"
  exposes skill "file-read"
}
```

A server declares:
- `transport` — the protocol used (`stdio` for local subprocess communication)
- `command` — the binary that implements the MCP server
- `exposes skill` — which skills are available through this server (validated against declared skills)

### Client

```
client "my-client" {
  connects to server "file-server"
}
```

A client connects to a server by name. The validator ensures the referenced server exists.

### Agent Connectivity

```
agent "file-assistant" {
  uses prompt "assistant"
  uses skill "file-read"
  connects to client "my-client"
  model "claude-sonnet-4-20250514"
}
```

The agent uses skills and connects to clients. This creates a dependency chain: agent -> client -> server -> skill implementation.

## How to Run

```bash
# Validate
./agentz validate examples/mcp-server-client.az

# Plan
./agentz plan examples/mcp-server-client.az

# Apply
./agentz apply examples/mcp-server-client.az --auto-approve

# Export (generates mcp-servers.json and mcp-clients.json)
./agentz export examples/mcp-server-client.az --out-dir ./output
```

## Resources Created

| Kind | Name | Description |
|------|------|-------------|
| Prompt | assistant | System instructions |
| Skill | file-read | File reading capability |
| MCPServer | file-server | Server exposing file-read over stdio |
| MCPClient | my-client | Client connecting to file-server |
| Agent | file-assistant | Agent wired to the MCP client |

## Exported Artifacts

When exported via the `local-mcp` adapter, this produces:
- `mcp-servers.json` — server transport and skill mappings
- `mcp-clients.json` — client-to-server connection config
- `agents.json` — agent definitions with resolved references

## Next Steps

- Build a multi-agent pipeline with MCP: see [code-review-pipeline](../code-review-pipeline/)
- Add a RAG pipeline with MCP transport: see [rag-chatbot](../rag-chatbot/)
