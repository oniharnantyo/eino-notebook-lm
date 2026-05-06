## 1. Phase 1: Create Unified Module

- [x] 1.1 Create `pkg/retriever/pgvector/unified.go` with `Retriever` struct and `TableConfig` type
- [x] 1.2 Implement `NewRetriever` constructor with table configuration map (knowledges, sentences, images)
- [x] 1.3 Implement `TableConfig` struct with Name, BM25Index, and optional JoinClause fields
- [x] 1.4 Implement semantic search method using pgvector cosine distance operator
- [x] 1.5 Implement keyword search method using BM25 with table-specific indexes
- [x] 1.6 Implement RRF fusion function (extract existing `MergeByRRF` logic)
- [x] 1.7 Implement hybrid retrieval method that fuses semantic and keyword results
- [x] 1.8 Implement full document fetch method for fused results with metadata
- [x] 1.9 Create `unified_test.go` with table-driven tests for semantic search
- [x] 1.10 Add table-driven tests for keyword search with BM25 validation
- [x] 1.11 Add tests for hybrid retrieval with RRF fusion verification
- [x] 1.12 Add error handling tests (invalid dimension, nil pool, unknown table type)

## 2. Phase 2: Migrate Adapters

- [x] 2.1 Refactor `pkg/retriever/pgvector/knowledges.go` to embed unified `Retriever`
- [x] 2.2 Implement `KnowledgesRetriever` methods as delegation wrappers to unified retriever
- [x] 2.3 Refactor `pkg/retriever/pgvector/sentences.go` to embed unified `Retriever`
- [x] 2.4 Implement `SentencesRetriever` methods with JOIN clause delegation
- [x] 2.5 Refactor `pkg/retriever/pgvector/images.go` to embed unified `Retriever`
- [x] 2.6 Implement `ImagesRetriever` methods as delegation wrappers
- [x] 2.7 Update adapter constructors to use `NewRetriever` internally
- [x] 2.8 Run existing `retriever_test.go` to verify no regressions

## 3. Phase 3: Update Callers

- [x] 3.1 Update `response_usecase.go` to use unified `Retriever` directly
- [x] 3.2 Remove type assertions from `response_usecase.go` (none found - already using interfaces)
- [x] 3.3 Update constructor injection to pass `Retriever` instead of adapters
- [x] 3.4 Add deprecation comments to adapter types
- [x] 3.5 Update integration tests to use unified retriever interface
- [x] 3.6 Run full test suite to verify all changes work correctly