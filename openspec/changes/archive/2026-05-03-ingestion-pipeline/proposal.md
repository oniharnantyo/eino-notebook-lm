## Why

Content ingestion requires coordinating 7+ separate services across `source`, `knowledge`, `sentence`, and `image` usecases. The 627-line `source/usecase.go` has duplicated `processAsync` and `processSync` methods with the same pipeline logic. The concept of "ingesting a document" has no single owner—complexity is scattered across modules, making the pipeline hard to understand, test, and modify.

## What Changes

- Extract a deep `IngestionPipeline` module with interface `Ingest(ctx, source) <-chan Progress`
- Consolidate the 6-step pipeline (extract → parse → chunk → embed → store → update status) into one place
- Convert existing usecases (knowledge, sentence, image) to pipeline stage adapters
- Replace duplication between sync/async paths with configurable parallelism
- Enable streaming progress updates via channel instead of status polling

**BREAKING**: `processAsync` and `processSync` methods will be removed; callers migrate to `Ingest()`.

## Capabilities

### New Capabilities
- `document-ingestion`: Unified pipeline for ingesting documents with streaming progress and configurable parallelism

### Modified Capabilities
- Implementation changes only for existing ingestion—no spec requirement changes

## Impact

**Affected files**:
- `internal/core/application/usecases/source/usecase.go` (627 lines → ~150 lines coordinator)
- `internal/core/application/usecases/knowledge/` (becomes pipeline stage)
- `internal/core/application/usecases/sentence/` (becomes pipeline stage)
- `internal/core/application/usecases/image/` (becomes pipeline stage)
- `internal/core/application/usecases/extractor/` (used by pipeline)
- `internal/core/application/usecases/document/` (used by pipeline)
- `internal/interfaces/http/handlers/source.go` (update to use `Ingest()`)
