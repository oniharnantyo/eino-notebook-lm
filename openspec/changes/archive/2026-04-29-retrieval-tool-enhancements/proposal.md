## Why

The retrieval tools (keyword_search, semantic_search, chunk_read) were built as a first pass for the agentic RAG loop. keyword_search queries the sentences table and returns first-80-chars truncations, which wastes tokens on irrelevant text and doesn't show the agent *where* keywords matched. chunk_read accepts only a single ID, forcing N tool calls for N chunks. semantic_search ignores top_k from the agent. These limitations reduce retrieval quality and increase latency on multi-hop questions where the agent needs to efficiently scan results across many chunks.

## What Changes

- **keyword_search**: Move from sentences to knowledges table. Accept `keywords[]` array + `top_k`. Return keyword-in-context (KWIC) snippets — ±80 chars around each keyword match — instead of first-80-chars truncation. Requires a new retriever instance configured for the knowledges table with its existing BM25 index.
- **semantic_search**: Accept `top_k` parameter, pass through to retriever. Table stays sentences.
- **chunk_read**: Accept `chunk_ids[]` array instead of single `chunk_id`. Return array of chunks with per-ID dedup via ContextTracker.

## Capabilities

### New Capabilities
- `kwic-keyword-search`: Keyword search on knowledges table with keyword-in-context snippet extraction. Accepts keywords array, top_k, returns per-chunk KWIC snippets with ±80 char context windows and overlapping window merging.

### Modified Capabilities
- `agent-retrieval`: Tool input/output contracts change — keyword_search switches to keywords[] + knowledges table, semantic_search adds top_k, chunk_read accepts multiple IDs. ToolFactory needs an additional retriever instance.

## Impact

**Modified files**:
- `internal/core/application/agent/tools/keyword_search.go` — new input/output types, knowledge retriever, KWIC logic
- `internal/core/application/agent/tools/semantic_search.go` — add top_k to input
- `internal/core/application/agent/tools/chunk_read.go` — accept chunk_ids[], batch fetch
- `internal/core/application/agent/tools/factory.go` — accept knowledgeRetriever, wire to keyword_search
- `cmd/serve.go` — create knowledgeRetriever (knowledges table), pass to ToolFactory
- `internal/core/application/agent/instruction.go` — update tool descriptions

**New files**:
- `pkg/retriever/pgvector/kwic.go` — ExtractKeywordContexts function with overlapping window merge

**API**: Tool input schemas change — agents must send `keywords[]` instead of `query`, `chunk_ids[]` instead of `chunk_id`. Internal change only (tools are server-side, not client-facing).

**Dependencies**: None new. `knowledges_bm25_idx` already exists in migrations.