## Context

The ingestion pipeline (commit `ac06c5a`) refactored to a Kreuzberg-based architecture producing three entity types: `Knowledge` (chunks, BM25-indexed, no embeddings), `Sentence` (fine-grained, vector-embedded, FK to knowledge), and `Image` (S3-stored, vision-embedded, FK to source). The retrieval agent uses a three-tool A-RAG pattern (keyword_search, semantic_search, chunk_read) but the tools don't align with this hierarchy — semantic search returns sentence IDs that chunk_read can't resolve, keyword search returns placeholder snippets, image search is unwired, and no retrieval is scoped to source IDs.

## Goals / Non-Goals

**Goals:**
- Align retrieval tools with the three-level A-RAG hierarchy: keyword → sentence → chunk
- Fix semantic_search to aggregate sentence hits by parent `knowledge_id` and return chunk-level results
- Fix keyword_search to return actual KWIC snippets using the existing `kwic-keyword-search` utility
- Add source-scoped filtering (`WHERE source_id IN (...)`) to `UnifiedRetriever` SQL queries
- Wire image_search to actually embed queries and search the images table with source scoping
- Enrich `BuildCatalog` with content_type, chunk_count, and status per the `agent-source-awareness` spec

**Non-Goals:**
- Replacing deprecated adapter retrievers (SentencesRetriever, KnowledgesRetriever, ImagesRetriever) — they still work via UnifiedRetriever internally
- Adding hybrid/RRF search as an agent tool — the A-RAG paper's three-tool design (keyword, semantic, chunk_read) is sufficient
- Changing the agent loop or ADK integration
- Modifying the ingestion pipeline
- Removing `HybridRetrieve` from `UnifiedRetriever` — it remains available as a library method for non-agent callers

## Decisions

### 1. Source scoping at UnifiedRetriever SQL level

**Decision**: Add `sourceIDs []string` parameter to `UnifiedRetriever.SemanticSearch` and `KeywordSearch` methods. Generate `WHERE source_id IN (...)` clauses in SQL.

**Alternatives considered**:
- Filter at tool level (post-retrieval): Simpler but inefficient, retrieves irrelevant rows, doesn't scale with corpus size.
- Filter via retriever options: Eino's `retriever.Option` doesn't support custom filters; would require wrapper logic.

**Rationale**: SQL-level filtering is correct and performant. The sentences table has `metadata->>'source_id'` and knowledges/images have `source_id` directly. This also prevents cross-notebook data leakage at the lowest layer.

### 2. Semantic search aggregation pattern

**Decision**: `semantic_search` performs the sentence-to-chunk aggregation at the SQL level using `GROUP BY knowledge_id` + `MAX(similarity)` + `ARRAY_AGG(content ORDER BY embedding <=> $1)[1]` for snippet extraction in a single query.

```sql
SELECT
    metadata->>'knowledge_id' as knowledge_id,
    MAX(1 - (embedding <=> $1)) as similarity,
    (ARRAY_AGG(content ORDER BY embedding <=> $1))[1] as snippet,
    metadata->>'source_id' as source_id
FROM sentences
WHERE metadata->>'source_id' IN ('src-1', 'src-2')  -- source scoping
GROUP BY metadata->>'knowledge_id', metadata->>'source_id'
ORDER BY similarity DESC
LIMIT $2
```

**Why MAX not AVG**: A single highly-relevant sentence is more valuable than many weakly-relevant ones. If a chunk has one sentence scoring 0.9 and ten scoring 0.3, MAX=0.9 correctly identifies it, while AVG=0.36 would incorrectly demote it.

**Why SQL-level not Go-level**: Without GROUP BY, a query returning top-5 sentences might yield `[chunk24, chunk24, chunk24, chunk172, chunk172]` — only 2 unique chunks. GROUP BY ensures we get 5 *unique* chunks, maximizing coverage. Pushing this to SQL avoids fetching redundant rows.

**Why ARRAY_AGG instead of two queries**: The original design used two queries (GROUP BY for top chunks, then a second fetch for snippets). The implementation uses `ARRAY_AGG` to extract the best-matching sentence content inline, eliminating the second round-trip while producing the same results.

**Rationale**: This matches A-RAG Section 3.2 exactly: *"We retrieve the top-ranked sentences and aggregate them by their parent chunks. Each chunk's relevance score is determined by its highest-scoring sentence."* The ingestion pipeline already stores `knowledge_id` on each sentence.

### 3. Keyword search uses existing KWIC utility

**Decision**: `keyword_search` tool fetches full knowledge content via `KnowledgeRepository.FindByIDs`, then applies `kwic.ExtractKeywordContexts()` to generate snippets. This matches the A-RAG paper's approach of extracting sentences containing at least one keyword.

**Alternatives considered**:
- SQL-level snippet extraction: pg_textsearch doesn't natively support KWIC windows.
- Returning full content: Wastes tokens, defeats progressive disclosure.

**Rationale**: The `kwic-keyword-search` spec already defines `ExtractKeywordContexts(content, keywords, window)` with merge/dedup/cap logic. Reuse it.

### 4. Image search uses text-to-image vector search

**Decision**: `image_search` tool embeds the query text using the same text embedder (not a vision model), then calls `UnifiedRetriever.SemanticSearch` on the images table. This works because the ingestion pipeline stores vision-generated text descriptions as embeddings in the images table.

**Rationale**: The `agent-retrieval` spec states: "The query SHALL be embedded using the configured text embedding model (NOT vision embedder — text-to-image retrieval)." The images table has embeddings derived from vision descriptions, so text-to-embedding cosine similarity finds images by description.

### 5. ToolFactory migration to UnifiedRetriever

**Decision**: Replace separate `SentencesRetriever`, `KnowledgesRetriever`, `ImagesRetriever` dependencies in `ToolFactory` with a single `UnifiedRetriever`. Tools call `UnifiedRetriever.SemanticSearch(ctx, tableType, ...)` or `UnifiedRetriever.SemanticSearchAggregated(ctx, tableType, ...)` directly.

**Rationale**: The adapter retrievers are deprecated wrappers. Going direct to `UnifiedRetriever` simplifies the factory and enables source scoping in one place. The adapters remain for any external callers.

## Risks / Trade-offs

- **[Source ID in sentence metadata]** → Sentences store `source_id` in `metadata` JSONB, not a dedicated column. `WHERE metadata->>'source_id' IN (...)` is slower than a native column. Mitigation: If performance becomes an issue, migrate to a `source_id` column with index.
- **[KWIC extraction at tool level]** → Requires fetching full knowledge content for each BM25 hit to extract snippets. Mitigation: Already fetching by IDs in batch; the KWIC function is pure Go, no DB overhead.
