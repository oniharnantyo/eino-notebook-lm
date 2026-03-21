# Langfuse Observability Integration

This application uses [Langfuse](https://langfuse.com) for tracing and observability of LLM operations.

## What Gets Traced

When Langfuse is enabled, the following Eino components are automatically traced:
- **ChatModel**: All Gemini chat model calls (generate and stream)
- **Embedder**: Text embedding operations
- **Retriever**: Vector search queries to pgvector
- **Chain**: End-to-end RAG pipeline execution

## Configuration

Enable Langfuse by setting environment variables:
```bash
LANGFUSE_ENABLED=true
LANGFUSE_HOST=https://cloud.langfuse.com
LANGFUSE_PUBLIC_KEY=pk-lf-...
LANGFUSE_SECRET_KEY=sk-lf-...
LANGFUSE_SAMPLE_RATE=1.0
LANGFUSE_RELEASE=v1.0.0
```

## Viewing Traces

After enabling Langfuse, traces appear in your Langfuse dashboard automatically.

## Architecture

The Langfuse integration follows Eino's tracing pattern:

1. **Initialization** (`cmd/serve.go`): Langfuse handler is created during application startup if `LANGFUSE_ENABLED=true`

2. **Component Wrapping**: Eino components are wrapped with tracing capabilities:
   - `eino/components/model/wrappert.NewChatModel` - Wraps Gemini chat model
   - `eino/components/embedding/wrappert.NewEmbedder` - Wraps text embedder
   - `eino/components/retriever/wrappert.NewRetriever` - Wraps pgvector retriever
   - `eino/components/chain/wrappert.NewChain` - Wraps RAG chain

3. **Shutdown Handler**: Langfuse flushes pending traces before application termination

### Trace Hierarchy

```
Chain (RAG Pipeline)
├── Retriever (pgvector search)
├── Embedder (query embedding)
└── ChatModel (generation)
```

This hierarchy allows you to see the full request flow from user query to LLM response.
