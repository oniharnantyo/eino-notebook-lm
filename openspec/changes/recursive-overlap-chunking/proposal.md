## Why

The custom element transformer (`pkg/transformer/element/`) implements semantic chunking that groups Kreuzberg-parsed elements by section titles. This approach is brittle — it depends on structured element data that may not exist for all content types, requires complex fallback logic, and is never actually invoked in the pipeline (the `knowledgeUseCase` receives the transformer but never calls `Transform()`). Meanwhile, Eino provides a battle-tested recursive overlap splitter as a standard component that works uniformly on any text with configurable chunk size and overlap.

## What Changes

- **BREAKING**: Remove custom `pkg/transformer/element/` package entirely
- Replace element transformer with Eino's `recursive.NewSplitter` (`github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive`)
- Actually invoke the transformer in the knowledge ingestion pipeline (currently injected but unused)
- Simplify config: replace `ElementTransformerConfig` (included_types, max_chunk_size) with `RecursiveSplitterConfig` (chunk_size, overlap_size)
- Update `cmd/serve.go` wiring to create recursive splitter instead of element transformer

## Capabilities

### New Capabilities
- `recursive-chunking`: Document splitting using Eino's recursive overlap splitter with configurable chunk size and overlap, applied uniformly to all content types (files, URLs, text)

### Modified Capabilities

## Impact

- **Code**: `cmd/serve.go`, `internal/infrastructure/config/config.go`, `internal/core/application/usecases/knowledge/usecase.go`
- **Removed**: `pkg/transformer/element/` (entire package)
- **Dependency**: Add `github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive`
- **Config**: Replace `TRANSFORMER_TYPE`, `TRANSFORMER_ELEMENT_INCLUDED_TYPES`, `TRANSFORMER_ELEMENT_MAX_CHUNK_SIZE` with `TRANSFORMER_CHUNK_SIZE` (default 4000), `TRANSFORMER_OVERLAP_SIZE` (default 800)
- **Retrieval**: No changes — pgvector retriever is chunking-strategy-agnostic
- **API**: No external API changes