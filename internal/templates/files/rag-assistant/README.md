# RAG Assistant

A retrieval-augmented generation (RAG) agent that answers questions by searching a document store before generating responses. Combines embedding-based semantic search with LLM generation to provide grounded, source-cited answers.

## Prerequisites

- AgentSpec CLI installed
- `ANTHROPIC_API_KEY` environment variable set
- A document store (e.g., Elasticsearch) running on `localhost:9200`
- An embedding service running on `localhost:8000`

## Configuration

Set required environment variables:

```bash
export ANTHROPIC_API_KEY="your-key-here"
```

## Run

```bash
agentspec validate rag-assistant.ias
agentspec run rag-assistant.ias
```

## Customization

- Update the `url` fields in the `search-documents` and `embed-query` skills to point to your actual services.
- Adjust `max_messages` in the `memory` block to control how much conversation history is retained.
- Lower `temperature` further (e.g., 0.1) for strictly factual use cases, or raise it for more creative responses.
- Add additional skills for document ingestion or metadata filtering.
