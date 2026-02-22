# RAG Chatbot

A retrieval-augmented generation (RAG) chatbot that searches a vector store and generates grounded answers with source citations. Skills are exposed via MCP transport.

## What This Demonstrates

- **RAG pipeline** modeled as discrete skills (embed, search, rerank)
- **MCP server** exposing the full retrieval stack over stdio
- **Secret management** for vector database credentials
- **Grounded generation** prompt pattern with citation requirements

## Definition Structure

### RAG-Specific Prompt

```
prompt "rag" {
  content "You are a knowledge assistant. Answer questions using only
           the retrieved context. Always cite your sources with
           [Source: document_name] notation. If the context does not
           contain enough information, say so honestly rather than
           guessing."
}
```

The prompt enforces grounded generation — the agent must cite sources and avoid hallucination. This pattern is common in enterprise RAG applications.

### Retrieval Pipeline as Skills

```
skill "embed-query" { ... }    # Step 1: Convert query to vector
skill "vector-search" { ... }  # Step 2: Find similar documents
skill "rerank" { ... }         # Step 3: Rerank by relevance
```

The RAG pipeline is decomposed into three skills that execute in sequence:
1. **embed-query** — generates an embedding vector for the user's question
2. **vector-search** — queries the vector store for similar documents
3. **rerank** — reorders results by relevance to improve quality

### MCP Transport

```
server "rag-server" {
  transport "stdio"
  command "rag-mcp-server"
  exposes skill "embed-query"
  exposes skill "vector-search"
  exposes skill "rerank"
}

client "rag-client" {
  connects to server "rag-server"
}
```

All three retrieval skills are exposed through a single MCP server. The server binary (`rag-mcp-server`) implements the embedding, search, and reranking logic.

### Secret for Vector DB

```
secret "vector-db-key" {
  store "env"
  env "VECTOR_DB_API_KEY"
}
```

The vector database API key is stored as an environment variable and referenced by name.

## How to Run

```bash
# Validate
./agentz validate examples/rag-chatbot.az

# Plan
./agentz plan examples/rag-chatbot.az

# Apply
./agentz apply examples/rag-chatbot.az --auto-approve

# Export (produces mcp-servers.json with RAG server config)
./agentz export examples/rag-chatbot.az --out-dir ./output
```

## Resources Created

| Kind | Name | Description |
|------|------|-------------|
| Prompt | rag | Grounded generation instructions with citation rules |
| Skill | embed-query | Query embedding generation |
| Skill | vector-search | Vector similarity search |
| Skill | rerank | Result reranking |
| Agent | rag-bot | RAG chatbot agent |
| MCPServer | rag-server | Exposes retrieval skills over stdio |
| MCPClient | rag-client | Connects to the RAG server |
| Secret | vector-db-key | Vector database API key |

## RAG Flow

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
                                                              Agent (rag-bot)
                                                                    |
                                                                    v
                                                            Grounded Answer
                                                            with [Source: ...] citations
```

## Next Steps

- Add environment overlays (dev uses smaller embedding model): see [multi-environment](../multi-environment/)
- Deploy to Docker Compose for production: see [multi-binding](../multi-binding/)
- Add monitoring with a plugin: see [plugin-usage](../plugin-usage/)
