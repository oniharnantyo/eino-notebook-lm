## 1. KWIC Extraction Utility

- [x] 1.1 Implement `ExtractKeywordContexts(content string, keywords []string, window int) []string` in `pkg/retriever/pgvector/kwic.go` — case-insensitive matching, ±window char extraction, overlapping window merge, dedup, cap at 5 snippets
- [x] 1.2 Add unit tests for KWIC: single keyword, multiple keywords, overlapping windows, multi-word phrase, keyword not found, snippet cap, case-insensitive matching

## 2. Knowledge Retriever Instance

- [x] 2.1 Add `knowledgeRetriever` creation in `cmd/serve.go` — `pgvector.Retriever` configured for `knowledges` table with `BM25IndexName: "knowledges_bm25_idx"`, `ReferenceIDColumn: "source_id"`, `KnowledgeIDColumn: ""`
- [x] 2.2 Wire `knowledgeRetriever` into `ToolFactory` constructor and `NewResponseUseCase`

## 3. Tool Refactoring

- [x] 3.1 Refactor `keyword_search.go` — new input type `KeywordSearchInput{Keywords []string, TopK int}`, use knowledge retriever, join keywords with spaces for BM25, apply KWIC extraction on results, return `KeywordMatchResult{ChunkID, Snippets}`
- [x] 3.2 Refactor `semantic_search.go` — add `TopK int` to `SemanticSearchInput`, pass top_k via `retriever.WithTopK()` option
- [x] 3.3 Refactor `chunk_read.go` — new input type `ChunkReadInput{ChunkIDs []string}`, iterate IDs through ContextTracker, batch-fetch unread IDs via repository, return array of chunk contents with per-ID status
- [x] 3.4 Update `factory.go` — add `knowledgeRetriever *pgvector.Retriever` to `ToolFactory`, pass to `NewKeywordSearchTool` instead of sentence retriever

## 4. Agent Instruction Update

- [x] 4.1 Update `instruction.go` — reflect new tool signatures: `keyword_search` accepts keywords array, `chunk_read` accepts multiple IDs, emphasize that search tools only return snippets and agent MUST call chunk_read for full content

## 5. Cleanup

- [x] 5.1 Remove `TruncateSnippet` from `pkg/retriever/pgvector/retriever.go` if no longer referenced
- [x] 5.2 Remove debug `fmt.Printf` statements from tool files
- [x] 5.3 Remove `SearchResult.SentenceID` field if no longer used by any tool

## 6. Verification

- [x] 6.1 Run `make build` and verify no compilation errors
- [x] 6.2 Run `make test` and verify all tests pass including new KWIC tests
- [ ] 6.3 Manual verification: agent uses keyword_search with keywords array, receives KWIC snippets, calls chunk_read with multiple IDs, receives full content
