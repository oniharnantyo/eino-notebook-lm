## Why

The current RAG pipeline retrieves context in a single shot — a hybrid retriever (BM25 + vector) pulls top-K sentences, injects them into a prompt template, and the LLM generates a response in one pass. The LLM never drives retrieval, never refines its search, and never seeks additional context when results are insufficient. This limits answer quality on complex or multi-faceted queries where the relevant information is scattered across different parts of the corpus.

The A-RAG framework replaces this static retrieval with an agent-driven iterative loop: the LLM autonomously chooses between keyword search, semantic search, and chunk reading, iterating until it has sufficient evidence to answer confidently.

## What Changes

- Add a **retrieval agent** using Eino's `ChatModelAgent` with ReAct pattern and `MaxIterations: 30`
- Implement four retrieval tools as Eino `tool.BaseTool`: `keyword_search` (BM25), `semantic_search` (vector cosine), `chunk_read` (full chunk by ID), `image_search` (vision embedding cosine)
- Add a **per-request Context Tracker** that deduplicates chunk reads — returns "already read" instead of re-fetching, forcing the agent to explore new areas
- Search tools return **abbreviated sentence snippets** + chunk IDs instead of full content, requiring the agent to explicitly read chunks for complete context
- Modify the response use case to use the agent as the DEFAULT and ONLY response method (remove agent_mode toggle)
- Split the existing hybrid retriever's BM25 and vector search into separate tool-callable methods
- **Align response format with OpenResponses contract** — tool calls, reasoning, and streaming events follow https://www.openresponses.org/reference specification

## Capabilities

### New Capabilities
- `agent-retrieval`: Agent-driven iterative retrieval using ReAct pattern with keyword search, semantic search, and chunk read tools. Includes per-request context tracking for chunk deduplication.

### Modified Capabilities
<!-- Replaces the un-specified chain-based retrieval with agent-driven retrieval as the default behavior. -->

## Impact

**New files**:
- `internal/core/application/agent/` — agent factory, instruction prompt, context tracker, tool implementations (keyword_search, semantic_search, chunk_read, tool factory)

**Modified files**:
- `internal/core/application/usecases/response/response_usecase.go` — add agent mode delegation
- `pkg/retriever/pgvector/retriever.go` — expose BM25 and vector search as separate public methods
- `go.mod` — add `github.com/cloudwego/eino/adk` dependency

**API**: Breaking change — removes agent_mode toggle, agent is now the default and only response mechanism. RAG retrieval tools are automatically provided by the server; clients may optionally provide custom tools.

**Dependencies**: `github.com/cloudwego/eino/adk` (Agent Development Kit)
