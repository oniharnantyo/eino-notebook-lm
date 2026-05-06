## Why

Kreuzberg returns rich document-level metadata (title, authors, page_count, format_type, pdf_version, producer, quality_score, etc.) that currently flows through the ingestion pipeline to each knowledge chunk's `metadata` JSONB column, but is never persisted to the `sources.metadata` column. The Source entity — the logical owner of document metadata — only stores what the HTTP handler sends (filename, size, content_type). This makes source-level document information invisible to API consumers and impossible to query at the source level.

## What Changes

- `StorageStage` will extract document-level metadata from the first chunk's `MetaData` and merge it into `Source.Metadata` during the existing source update step
- Document-level fields (title, authors, created_by, format_type, pdf_version, producer, is_encrypted, width, height, page_count, output_format, quality_score, pages) will be saved to `sources.metadata`
- Chunk-level fields (first_page, last_page, heading_context, chunk_index) remain on `knowledges.metadata` unchanged

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `document-ingestion`: Storage stage requirement expanded to persist Kreuzberg document-level metadata to the Source entity

## Impact

**Modified files:**
- `internal/core/application/usecases/pipeline/storage_stage.go` — add metadata merge logic before source update

**No new files, no schema changes, no API changes.**
