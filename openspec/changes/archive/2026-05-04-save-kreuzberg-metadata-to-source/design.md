## Context

The ingestion pipeline has 6 stages: Extraction → Parsing → Chunking → Embedding → Storage → StatusUpdate. Kreuzberg document-level metadata (title, authors, page_count, etc.) is carried from `ExtractionResult.Metadata` through `ParsingStage` into each `schema.Document.MetaData`, then preserved through `ChunkingStage` (Eino's recursive splitter does `deepCopyMap(doc.MetaData)` on child chunks). The metadata reaches `StorageStage` on every chunk but is never written to the Source entity.

The Source entity has a `Metadata map[string]interface{}` field persisted to a JSONB column. It's currently only populated with HTTP handler data (filename, size, content_type) at creation time.

## Goals / Non-Goals

**Goals:**
- Persist Kreuzberg document-level metadata to `sources.metadata` during ingestion
- Keep chunk-level metadata (first_page, last_page, heading_context) on knowledge entities unchanged

**Non-Goals:**
- Changing the Kreuzberg response structure
- Adding new API endpoints
- Re-ingesting existing documents
- Modifying the knowledge or sentence metadata schema

## Decisions

### 1. Merge metadata in StorageStage

**Decision:** Extract document-level metadata from the first chunk's `MetaData` and merge it into `Source.Metadata` during the existing source update in `StorageStage.Execute()`.

**Rationale:** StorageStage already reads the source (`sourceRepo.GetByID`) and updates it (`sourceRepo.Update`) to set `ChunkCount`. Adding metadata merge here avoids a new pipeline stage or a separate repository call. The source update happens once per ingestion, so there's no performance concern.

**Alternative considered:** Create a new `MetadataStage` between ExtractionStage and ParsingStage. Rejected because it adds a stage for a single field update that fits naturally into the existing source update step.

### 2. Define document-level metadata keys explicitly

**Decision:** Use an explicit list of document-level keys to extract from chunk metadata: `title`, `authors`, `created_by`, `format_type`, `pdf_version`, `producer`, `is_encrypted`, `width`, `height`, `page_count`, `output_format`, `quality_score`, `pages`.

**Rationale:** Explicit keys prevent accidentally merging chunk-level metadata (first_page, last_page, heading_context, embedding) into the source. The list matches Kreuzberg's documented metadata structure.

**Alternative considered:** Merge all metadata and exclude chunk-level keys. Rejected because it's fragile — any new chunk-level key would leak into the source.

### 3. Use first chunk's metadata as source

**Decision:** Read document-level metadata from `docs[0].MetaData` (the first chunk).

**Rationale:** Document-level metadata is identical across all chunks (ParsingStage copies `result.Metadata` to every doc, and ChunkingStage deep-copies it to every child). Using the first chunk is deterministic and avoids iterating all chunks.

## Risks / Trade-offs

**[Risk] Metadata keys may evolve** → Mitigation: Explicit key list is a single constant in `StorageStage`, easy to update when Kreuzberg adds new fields.

**[Risk] Source update grows slightly** → Negligible. One JSONB column update on an existing UPDATE statement.

**[Trade-off] StorageStage knows about metadata key names** → Acceptable. The stage already knows about chunk-level keys (first_page, last_page, heading_context) for knowledge entity creation.
