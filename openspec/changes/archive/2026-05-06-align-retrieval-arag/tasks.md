## 1. UnifiedRetriever Source Scoping

- [x] 1.1 Add `sourceIDs []string` parameter to `UnifiedRetriever.SemanticSearch` method; generate `WHERE source_id IN (...)` clause for knowledges/images tables and `WHERE metadata->>'source_id' IN (...)` for sentences table
- [x] 1.2 Add `sourceIDs []string` parameter to `UnifiedRetriever.KeywordSearch` method with same source filtering logic per table type
- [x] 1.3 Write unit tests for source-scoped retrieval on knowledges, sentences, and images table types

## 2. Semantic Search Tool Fix

- [x] 2.1 Update `semantic_search` tool to call `UnifiedRetriever.SemanticSearch` on "sentences" table with source scoping
- [x] 2.2 Implement SQL-level sentence-to-chunk aggregation: auto-aggregate by `knowledge_id` when table is "sentences" using `GROUP BY` + `MAX(similarity)` + `ARRAY_AGG` for snippet
- [x] 2.3 Update `SemanticSearchResult` struct to include `chunk_id` (knowledge_id) and `snippet` (matched sentence content) instead of placeholder text
- [x] 2.4 Write tests verifying semantic search returns chunk IDs that resolve via `chunk_read`

## 3. Keyword Search Tool Fix

- [x] 3.1 Update `keyword_search` tool to call `UnifiedRetriever.KeywordSearch` on "knowledges" table with source scoping
- [x] 3.2 After BM25 search, fetch full knowledge content via `KnowledgeRepository.FindByIDs` and apply `kwic.ExtractKeywordContexts()` to generate real KWIC snippets
- [x] 3.3 Update `KeywordMatchResult` struct to include actual KWIC snippets instead of placeholder text
- [x] 3.4 Write tests verifying keyword search returns real KWIC snippets with keyword highlighting

## 4. Image Search Tool Wiring

- [x] 4.1 Update `image_search` tool to accept `embedding.Embedder` and `UnifiedRetriever` as dependencies
- [x] 4.2 Implement tool body: embed query text, call `UnifiedRetriever.SemanticSearch` on "images" table with sourceIDs, return s3_key, description, page_number, and score
- [x] 4.3 Write tests verifying image search returns actual results from the images table

## 5. ToolFactory Migration

- [x] 5.1 Replace `SentencesRetriever`, `KnowledgesRetriever`, `ImagesRetriever` dependencies in `ToolFactory` with single `UnifiedRetriever`
- [x] 5.2 Update `NewScopedTools` to pass `ScopeConfig.SourceIDs` (as strings) to each tool for source scoping
- [x] 5.3 Update `NewToolFactory` constructor signature to accept `UnifiedRetriever` instead of three separate retrievers
- [x] 5.4 Update DI wiring in `serve.go` to inject `UnifiedRetriever` into `ToolFactory`
- [x] 5.5 Verify build passes with `make build`

## 6. Catalog Enrichment & Agent Instruction

- [x] 6.1 Update `BuildCatalog` to include content_type, chunk_count, and status for each source (per `agent-source-awareness` spec)
- [x] 6.2 Update `BaseAgentInstruction` to describe A-RAG progressive disclosure pattern: search returns snippets, use `chunk_read` for full content
- [x] 6.3 Update `list_sources` tool to return content_type, chunk_count, and status in `SourceDetail`
- [x] 6.4 Write tests verifying catalog format matches `agent-source-awareness` spec scenarios

## 7. Dead Code Cleanup

- [x] 7.1 Merge `SemanticSearchAggregated` into `SemanticSearch`: auto-aggregate when `tableType == "sentences"`, standard search otherwise; delete `SemanticSearchAggregated` method
- [x] 7.2 Remove `HybridRetrieve`, `semanticSearchRanked`, `keywordSearchRanked`, `rankedListToDocuments` from `unified.go`
- [x] 7.3 Delete dead files: `retriever.go`, `bm25.go`, `options.go`, `rrf.go`, `rrf_test.go`, `knowledges.go`, `images.go`, `sentences.go`, `doc.go`
- [x] 7.4 Trim `config.go` to only `DistanceFunction` and `operator()`; remove dead `Config` struct, `Reranker` interface, `setDefaults`
- [x] 7.5 Remove `DefaultK` and `DefaultTopK` from `UnifiedConfig` (only used by deleted `HybridRetrieve`)
- [x] 7.6 Update `serve.go`: remove `SentencesRetriever` creation, update `ResponseUseCase` constructor to accept `UnifiedRetriever` instead of `retriever.Retriever`
- [x] 7.7 Fix stale `catalog_test.go` to match enriched `BuildCatalog` format (content_type, chunk_count, status)
- [x] 7.8 Fix `tools_test.go` to match updated tool constructors (accepting `UnifiedRetriever` + sourceIDs)
- [x] 7.9 Update `unified_test.go` to remove references to deleted `DefaultK`/`DefaultTopK` fields
- [x] 7.10 Verify build passes with `make build`
