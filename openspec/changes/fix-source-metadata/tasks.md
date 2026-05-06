## 1. Core Implementation

- [x] 1.1 Modify `KnowledgeMappingStage.Execute()` to merge `ExtractionResult.Metadata` into each knowledge chunk's metadata
- [x] 1.2 Add nil check for `ExtractionResult.Metadata` to handle missing metadata gracefully
- [x] 1.3 Preserve existing `chunk_type` field when merging metadata
- [x] 1.4 Update knowledge mapping tests to verify document metadata transfer

## 2. Verification

- [x] 2.1 Run existing knowledge mapping stage tests to ensure no regressions
- [x] 2.2 Add test case for metadata transfer when `ExtractionResult.Metadata` is populated
- [x] 2.3 Add test case for graceful handling when `ExtractionResult.Metadata` is empty/nil
- [x] 2.4 Verify `StorageStage.extractDocumentMetadata()` logic works with transferred data

## 3. Integration Testing

- [ ] 3.1 Test full pipeline with a PDF file that has document metadata
- [ ] 3.2 Verify `source.metadata` is populated in database after successful ingestion
- [ ] 3.3 Test with plain text upload (no document metadata) to ensure no errors
- [ ] 3.4 Verify `chunk_count` is correctly updated in source entity

## 4. Documentation

- [x] 4.1 Update `KnowledgeMappingStage` code comments to describe metadata transfer behavior
- [x] 4.2 Add example of expected document metadata structure to design.md if needed
