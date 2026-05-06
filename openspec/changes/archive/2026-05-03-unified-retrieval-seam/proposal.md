## Why

The retrieval layer has three separate retriever types (`KnowledgesRetriever`, `SentencesRetriever`, `ImagesRetriever`) with nearly identical implementations—the only difference is the table name in SQL queries. This shallow abstraction forces callers to use type assertions and makes the code harder to test. Adding a new retriever type requires duplicating ~120 lines of code across three methods.

## What Changes

- Introduce a deep `Retriever` module with a unified interface that accepts table configuration
- Consolidate semantic search, keyword search (BM25), and hybrid retrieval (RRF fusion) into single implementations
- Convert existing retrievers to thin adapter types that delegate to the unified module
- Remove dangerous type assertions in `response_usecase.go`

**BREAKING**: Internal `Retriever` facade struct will be replaced; external callers using the three specific retriever types remain unaffected via adapters.

## Capabilities

### New Capabilities
- `unified-retrieval`: Single retriever interface supporting multiple table types (knowledges, sentences, images) with pluggable configuration

### Modified Capabilities
- `agent-retrieval`: Implementation changes only—requirements unchanged. No delta spec needed.

## Impact

**Affected files**:
- `pkg/retriever/pgvector/knowledges.go` (120 lines → ~20 lines adapter)
- `pkg/retriever/pgvector/sentences.go` (70 lines → ~20 lines adapter)
- `pkg/retriever/pgvector/images.go` (67 lines → ~20 lines adapter)
- `pkg/retriever/pgvector/retriever.go` (18 lines facade → ~150 lines unified implementation)
- `pkg/retriever/pgvector/bm25.go` (refactored into unified module)
- `internal/core/application/usecases/response/response_usecase.go` (remove type assertions)