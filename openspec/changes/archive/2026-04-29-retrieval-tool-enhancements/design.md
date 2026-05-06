## Context

The agentic RAG system (`agentic-rag-retrieval` change) is implemented and working. The agent uses three retrieval tools: keyword_search (BM25 on sentences), semantic_search (vector on sentences), and chunk_read (single ID lookup). All three tools have limitations discovered during real usage:

1. keyword_search operates on the `sentences` table and returns first-80-chars truncations — the agent sees irrelevant text instead of where keywords matched
2. chunk_read accepts only one chunk ID, forcing N tool calls for N chunks
3. semantic_search ignores the agent's top_k preference

The retriever infrastructure is a `pgvector.Retriever` configured per table. The current sentence retriever joins through `knowledges` for source filtering. A `knowledges_bm25_idx` BM25 index already exists on the `knowledges` table from migration 000012.

## Goals / Non-Goals

**Goals:**
- Move keyword_search from sentences to knowledges table for chunk-level BM25
- Implement keyword-in-context (KWIC) extraction: ±80 char windows around each keyword match, overlapping windows merged
- Accept `keywords[]` array + `top_k` for keyword_search, `top_k` for semantic_search
- Accept `chunk_ids[]` array for chunk_read with per-ID dedup
- Enforce two-phase retrieval: search tools return only pointers/snippets, agent must call chunk_read for full content

**Non-Goals:**
- Changing semantic_search's table (stays on sentences)
- Changing image_search
- Adding adjacent-chunk reading (offset +1/-1)
- Reranking or score normalization changes
- Modifying the response streaming pipeline

## Decisions

### 1. Second retriever instance for knowledges table

Add a new `pgvector.Retriever` configured for the `knowledges` table with `BM25IndexName: "knowledges_bm25_idx"`. The ToolFactory accepts this as a third retriever alongside the existing sentence and image retrievers.

```go
// cmd/serve.go — new retriever
knowledgeRetriever, err := pgvectoretriever.NewRetriever(ctx, &pgvectoretriever.Config{
    Pool:              dbPool,
    TableName:         "knowledges",
    Dimension:         cfg.Embedding.Dimension,
    ReferenceIDColumn: "source_id",           // direct, no join needed
    AutoCreateBM25Extension: true,
    AutoCreateBM25Index:     true,
    BM25IndexName:           "knowledges_bm25_idx",
    KnowledgeIDColumn:       "",               // knowledges ARE the chunks
})
```

**Alternative**: Make the retriever table-agnostic with per-method table switching — rejected because the retriever config (indexes, join tables, column names) is table-specific. Separate instances are cleaner and match the existing pattern (sentenceRetriever, imageRetriever).

### 2. KWIC extraction as a pure function in pkg/retriever/pgvector/

New file `pkg/retriever/pgvector/kwic.go` with `ExtractKeywordContexts(content string, keywords []string, window int) []string`. This is a string-processing utility with no DB dependency. Placing it in `pkg/retriever/pgvector/` keeps it alongside the existing `TruncateSnippet` helper.

Algorithm:
1. Lowercase both content and keywords for case-insensitive matching
2. For each keyword, find all occurrences in the content
3. Extract `[pos - window, pos + len(keyword) + window]` for each
4. Sort windows by position, merge overlapping ones
5. Wrap each merged window with `"..."` prefix/suffix
6. Deduplicate identical snippets

### 3. keyword_search input: keywords[] joined for BM25

Agent sends `keywords: ["telephone", "invented"]` → tool joins with spaces to `"telephone invented"` → passes to `retriever.KeywordSearch()`. Individual keywords are preserved for KWIC extraction after BM25 returns matching chunks.

Multi-word phrases like `["Alexander Graham Bell", "born"]` are joined as `"Alexander Graham Bell born"`. BM25 tokenizes this naturally; the KWIC extractor searches for each keyword element (including multi-word ones) separately in the chunk content.

### 4. chunk_read batch with per-ID dedup

The tool accepts `chunk_ids[]`, iterates each ID through the ContextTracker, and fetches only unread IDs from the repository. Results are returned as an array. Already-read IDs get a status message instead of content.

**Alternative**: Fetch all then filter — rejected because it wastes DB queries on already-read chunks. Check tracker first, then batch-fetch only new IDs.

### 5. ToolFactory accepts knowledgeRetriever

The factory constructor adds a `knowledgeRetriever *pgvector.Retriever` parameter. keyword_search uses this instead of the sentence retriever. semantic_search and image_search continue using their respective retrievers unchanged.

## Risks / Trade-offs

**[BM25 scoring on chunks vs sentences]** → Chunk-level BM25 returns fewer, larger results. Score granularity may decrease for short queries. This is acceptable — the agent reads full chunks anyway, and chunk-level search gives better context per result.

**[KWIC window overlap]** → Two keywords close together produce overlapping windows. The merge algorithm handles this, but edge cases (keywords at chunk boundaries, very long chunks with many matches) need testing. The ±80 window size is a heuristic tunable after deployment.

**[Agent tool call overhead]** → Accepting `chunk_ids[]` reduces tool calls from N to 1 for batch reads. But if the agent sends many IDs, the response is larger. The ContextTracker naturally limits re-reads.

**[Retriever instance proliferation]** → Three retriever instances (sentences, knowledges, images) each hold a reference to the same connection pool. Memory overhead is negligible — each instance is just a config struct + pool pointer.

## Open Questions

- Should KWIC snippets have a max count per chunk (e.g., top 5 keyword matches) to avoid returning huge arrays for chunks where a keyword appears many times? Suggest capping at 5 snippets per chunk initially.