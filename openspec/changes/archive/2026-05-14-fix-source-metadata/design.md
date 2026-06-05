## Context

The ingestion pipeline successfully processes documents through Kreuzberg and stores knowledge chunks and sentences, but the source entity's metadata field remains empty. Investigation reveals:

1. Kreuzberg returns document-level metadata (`title`, `authors`, `page_count`, etc.) in `KreuzbergExtractResponse.Metadata`
2. `FileContentExtractor` merges this into `ExtractionResult.Metadata`
3. `KnowledgeMappingStage` only adds `chunk_type` to knowledge metadata—**ignoring document metadata**
4. `StorageStage.extractDocumentMetadata()` expects document metadata in the first knowledge chunk

Current pipeline flow:
```
Kreuzberg → ExtractionResult → KnowledgeMappingStage → StorageStage → Source
     ↓              ↓                  ↓                  ↓            ↓
  Metadata    Metadata           Only chunk_type   Finds nothing   Empty!
```

## Goals / Non-Goals

**Goals:**
- Transfer document-level metadata from `ExtractionResult.Metadata` to knowledge chunks
- Enable `StorageStage` to populate `source.metadata` with document properties
- Maintain backward compatibility (no API changes)

**Non-Goals:**
- Changing the metadata schema or keys (already defined in `documentMetadataKeys`)
- Modifying how Kreuzberg returns data
- Populating `source.content` field (by design, content lives in chunks)

## Decisions

### 1. Transfer metadata in `KnowledgeMappingStage`

**Decision:** Add document-level metadata from `ExtractionResult.Metadata` to each knowledge chunk's metadata.

**Rationale:** 
- `StorageStage` already extracts document metadata from `data.Knowledges[0].Metadata`
- Minimal code change—leverages existing infrastructure
- Preserves document context with each chunk for potential future use

**Alternative considered:** Store document metadata separately in `PipelineData` and pass directly to `StorageStage`. Rejected because it requires changing the `PipelineData` structure and `StorageStage` signature.

### 2. Merge strategy: document metadata takes precedence

**Decision:** When merging, document-level keys from `ExtractionResult.Metadata` are added directly. If there's a conflict with chunk-level metadata, document metadata wins (source of truth).

**Rationale:** Document metadata comes from Kreuzberg's document analysis, which is more authoritative than chunk-level properties.

### 3. No validation of metadata keys

**Decision:** Copy all metadata from `ExtractionResult.Metadata` without validation. Let `StorageStage.extractDocumentMetadata()` filter to `documentMetadataKeys`.

**Rationale:** Single point of filtering (in `StorageStage`) is easier to maintain. Avoids duplication of the key list.

## Risks / Trade-offs

**Risk:** Metadata bloat—every knowledge chunk gets a copy of document metadata.
- **Mitigation:** Metadata is JSONB in PostgreSQL, compression handles repetition. Impact is minimal for typical documents (5-10 keys).

**Risk:** Missing keys if Kreuzberg changes response structure.
- **Mitigation:** `extractDocumentMetadata()` safely checks for key existence. Missing keys result in empty values, not errors.

**Trade-off:** Storing document metadata in every chunk vs. once per source.
- Chosen: Per-chunk for simplicity and potential future query optimizations (e.g., filtering chunks by document property).
