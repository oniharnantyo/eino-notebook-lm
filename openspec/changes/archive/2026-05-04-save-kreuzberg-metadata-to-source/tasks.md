## 1. Define document-level metadata keys

- [x] 1.1 Add `documentMetadataKeys` constant in `storage_stage.go` listing: `title`, `authors`, `created_by`, `format_type`, `pdf_version`, `producer`, `is_encrypted`, `width`, `height`, `page_count`, `output_format`, `quality_score`, `pages`
- [x] 1.2 Add `extractDocumentMetadata(docMeta map[string]any) map[string]any` helper that filters chunk metadata to only document-level keys

## 2. Merge metadata into Source entity

- [x] 2.1 In `StorageStage.Execute()`, after saving knowledge chunks and sentences, extract document metadata from the first doc's `MetaData` using `extractDocumentMetadata()`
- [x] 2.2 Merge extracted metadata into `source.Metadata` (preserving any existing keys from creation time)
- [x] 2.3 Ensure the source update call persists the merged metadata

## 3. Tests

- [x] 3.1 Add unit test for `extractDocumentMetadata`: returns only document-level keys, excludes chunk-level keys (first_page, last_page, heading_context, embedding)
- [x] 3.2 Add unit test for `extractDocumentMetadata`: returns empty map when no document-level keys present
- [x] 3.3 Verify `make build` and `make test` pass
