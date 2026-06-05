## Why

The ingestion pipeline was refactored to Kreuzberg (commit `ac06c5a`), producing a three-level hierarchy: knowledge chunks (BM25-indexed), sentences (vector-embedded), and images (vision-embedded). The retrieval tools are misaligned — `semantic_search` returns sentence IDs instead of chunk IDs, `keyword_search` returns placeholder snippets, `image_search` is an unwired stub, and no tool filters by source scope (cross-notebook data leakage). This breaks the A-RAG hierarchical retrieval pattern (keyword → sentence → chunk) that the agent relies on.

## What Changes

- Fix `semantic_search` to aggregate sentence hits by parent `knowledge_id`, returning chunk IDs + matched sentence snippets (A-RAG Section 3.2)
- Fix `keyword_search` to return actual KWIC snippets using the existing `kwic-keyword-search` utility instead of placeholder text
- Add `sourceIDs` filter parameter to `UnifiedRetriever.SemanticSearch` and `KeywordSearch` SQL queries for source-scoped retrieval
- Wire `image_search` tool: inject embedder, call `UnifiedRetriever.SemanticSearch` on images table with source scoping
- Enrich `BuildCatalog` to include content_type, chunk_count, and status per the `agent-source-awareness` spec
- Update agent instruction to reflect A-RAG progressive disclosure pattern (search → snippet → chunk_read)

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `unified-retrieval`: Add source scoping filter (`sourceIDs` parameter) to semantic and keyword retrieval SQL queries; add `SemanticSearchAggregated` for sentence-to-chunk aggregation

## Impact

- `pkg/retriever/pgvector/unified.go` — Add `sourceIDs` WHERE clause to query methods; add `SemanticSearchAggregated` method
- `pkg/retriever/pgvector/sentences.go` — Pass sourceIDs through adapter
- `pkg/retriever/pgvector/knowledges.go` — Pass sourceIDs through adapter
- `pkg/retriever/pgvector/images.go` — Pass sourceIDs through adapter
- `internal/core/application/agent/tools/semantic_search.go` — Use `SemanticSearchAggregated`, return chunk IDs + snippets
- `internal/core/application/agent/tools/keyword_search.go` — Use KWIC utility for real snippets
- `internal/core/application/agent/tools/image_search.go` — Wire embedder + retriever + source scoping
- `internal/core/application/agent/tools/factory.go` — Pass source IDs to tools, migrate to UnifiedRetriever
- `internal/core/application/agent/agent.go` — Enrich BuildCatalog with content_type, chunk_count, status
- `internal/core/application/agent/instruction.go` — Update to A-RAG progressive disclosure pattern
