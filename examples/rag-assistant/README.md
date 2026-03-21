# RAG Assistant

A retrieval-augmented generation (RAG) assistant that searches a vector store and generates grounded answers with source citations. The retrieval pipeline is exposed as an MCP server for integration with external clients.

## Architecture Overview

The assistant decomposes the RAG workflow into three discrete skills executed in sequence:

```
User Query
    |
    v
embed-query  -->  vector  -->  vector-search  -->  documents
                                                        |
                                                        v
                                                    rerank  -->  top docs
                                                                    |
                                                                    v
                                                              rag-bot (agent)
                                                                    |
                                                                    v
                                                            Grounded Answer
                                                            with [Source: ...] citations
```

All three retrieval skills are exposed through an MCP server (`rag-server`) over stdio transport. A companion MCP client connects to the server for tool invocations.

## Prerequisites

1. Build the AgentSpec CLI from the repository root:

   ```bash
   go build -o agentspec ./cmd/agentspec
   ```

2. Set the required environment variable:

   ```bash
   export VECTOR_DB_API_KEY="your-vector-db-api-key"
   ```

3. Ensure the following tool binaries are available on `$PATH` (or stubbed for testing):
   - `embed-tool`
   - `vector-search-tool`
   - `rerank-tool`
   - `rag-mcp-server`

## Step-by-Step Run Instructions

```bash
# 1. Validate the AgentSpec
./agentspec validate examples/rag-assistant/rag-assistant.ias

# 2. Preview planned changes
./agentspec plan examples/rag-assistant/rag-assistant.ias

# 3. Apply the changes
./agentspec apply examples/rag-assistant/rag-assistant.ias --auto-approve

# 4. Export artifacts (produces mcp-servers.json with RAG server config)
./agentspec export examples/rag-assistant/rag-assistant.ias --out-dir ./output
```

## Customization Tips

- **Swap embedding models**: Replace the `embed-tool` binary with one that calls a different embedding API (OpenAI, Cohere, local sentence-transformers).
- **Add chunking**: Insert a `chunk-documents` skill before `embed-query` for ingestion workflows.
- **Add environment overlays**: Use `environment` blocks to switch between a smaller embedding model for dev and a larger one for production. See [multi-environment](../multi-environment/).
- **Deploy to Docker Compose**: Use the `multi-binding` deploy target for containerized production deployments. See [multi-binding](../multi-binding/).
- **Add monitoring**: Attach a WASM plugin for observability. See [plugin-usage](../plugin-usage/).
