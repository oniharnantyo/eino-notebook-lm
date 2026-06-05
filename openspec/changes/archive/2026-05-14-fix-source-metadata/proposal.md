## Why

After successful document ingestion through the Kreuzberg pipeline, the source entity's `content` and `metadata` fields remain empty. This prevents displaying document information (title, authors, page count) in the UI and breaks document search/filtering functionality. The data exists in Kreuzberg's response but is lost during pipeline processing.

## What Changes

- Modify `KnowledgeMappingStage` to transfer document-level metadata from `ExtractionResult.Metadata` to knowledge chunks
- Ensure `StorageStage` can extract and persist document metadata to the source entity
- Fix metadata population for keys: `title`, `authors`, `page_count`, `pdf_version`, `producer`, `quality_score`, etc.

## Capabilities

### Modified Capabilities
- `source-ingestion`: Fix document metadata persistence to source entity (currently broken—metadata is lost during pipeline processing)

## Impact

**Affected Code:**
- `internal/core/application/usecases/pipeline/knowledge_mapping_stage.go` (transfer metadata from ExtractionResult to knowledge chunks)
- `internal/core/application/usecases/pipeline/storage_stage.go` (already has extraction logic, just needs data)

**Database:**
- `sources` table `metadata` column will be populated (currently empty `{}`)

**No API changes**—internal fix only
