# Semantic Chunking Implementation Tasks

## 1. Element Types and Constants

- [x] 1.1 Create `pkg/transformer/element/element_type.go` with ElementType constants for all Kreuzberg types
- [x] 1.2 Add `String()`, `IsValid()`, and `DefaultIncludedTypes()` methods to ElementType
- [x] 1.3 Write unit tests for ElementType validation

## 2. Kreuzberg Parser Updates

- [x] 2.1 Add `KreuzbergElement` struct to `pkg/parser/kreuzberg/kreuzberg.go` response types
- [x] 2.2 Add `ElementMetadata` struct with `page_number`, `filename`, `coordinates`, `element_index`, `additional` fields
- [x] 2.3 Update `KreuzbergExtractResponse` to include `Elements []KreuzbergElement` field
- [x] 2.4 Store `elements[]` in document metadata after parsing (if len > 0)
- [x] 2.5 Write unit tests for element preservation in parser

## 3. Element Transformer Core

- [x] 3.1 Create `pkg/transformer/element/config.go` with `Config` struct (`IncludedTypes`, `MaxChunkSize`)
- [x] 3.2 Create `pkg/transformer/element/transformer.go` with `elementTransformer` struct
- [x] 3.3 Add `section` struct to track current section state (`title`, `elements`, `startPage`, `endPage`)
- [x] 3.4 Implement `NewElementTransformer()` constructor with markdown fallback initialization
- [x] 3.5 Implement `hasElements()` helper to check for element data in document metadata

## 4. Transform Logic

- [x] 4.1 Implement `Transform()` method with fallback logic (elements → semantic, no elements → markdown)
- [x] 4.2 Implement `isIncluded()` helper for element type filtering
- [x] 4.3 Implement `createSemanticChunks()` method with title-based grouping algorithm
- [x] 4.4 Implement `sanitizeID()` helper for chunk ID generation (max 50 chars)
- [x] 4.5 Implement `createChunk()` method with metadata population (`chunk_title`, `page_start`, `page_end`, `element_count`, `chunk_type`)

## 5. Unit Tests

- [x] 5.1 Write test for semantic chunking with multiple title sections
- [x] 5.2 Write test for document with single title
- [x] 5.3 Write test for document with no titles (single chunk with empty title)
- [x] 5.4 Write test for markdown fallback when elements missing
- [x] 5.5 Write test for markdown fallback when elements empty
- [x] 5.6 Write test for custom included types configuration
- [x] 5.7 Write test for max chunk size splitting
- [x] 5.8 Write test for metadata inheritance and chunk-specific fields
- [x] 5.9 Run all tests: `go test ./pkg/transformer/element/... -v` (✓ 33 tests passed)

## 6. Configuration

- [x] 6.1 Add `TransformerConfig` struct to `internal/infrastructure/config/config.go`
- [x] 6.2 Add `ElementTransformerConfig` struct with `IncludedTypes []string` and `MaxChunkSize int`
- [x] 6.3 Add `GetIncludedElementTypes()` helper method to convert strings to ElementTypes
- [x] 6.4 Update `setDefaults()` to include transformer defaults (`type: element`, `included_types: title,narrative_text,list_item,heading,table`)
- [x] 6.5 Add validation for transformer type (must be "element" or "markdown")
- [x] 6.6 Run build verification: `go build ./cmd/...`

## 7. Server Integration

- [x] 7.1 Add `pkg/transformer/element` import to `cmd/serve.go`
- [x] 7.2 Replace hardcoded markdown transformer with configurable transformer switch
- [x] 7.3 Implement transformer initialization for `type: markdown` case
- [x] 7.4 Implement transformer initialization for `type: element` case with fallback to markdown on error
- [x] 7.5 Add logging for transformer initialization (type, included_types, max_chunk_size)
- [x] 7.6 Run build verification: `go build ./cmd/serve`

## 8. Documentation

- [x] 8.1 Add transformer configuration to `.env.example` (TRANSFORMER_TYPE, TRANSFORMER_ELEMENT_INCLUDED_TYPES, TRANSFORMER_ELEMENT_MAX_CHUNK_SIZE)
- [x] 8.2 Update any relevant README or deployment documentation

## 9. Integration Testing

- [ ] 9.1 Start server: `make run` - verify element transformer initialization in logs
- [ ] 9.2 Upload PDF document - verify chunks have semantic metadata in database
- [ ] 9.3 Upload plain markdown file - verify fallback to markdown splitter works
- [ ] 9.4 Query retriever - verify metadata fields (chunk_title, page_start, page_end) are returned
- [ ] 9.5 Test rollback: set `TRANSFORMER_TYPE=markdown`, restart, verify markdown splitter used

## 10. Final Verification

- [x] 10.1 All unit tests pass: `go test ./pkg/transformer/element/... -v` (✓ 33 tests passed)
- [x] 10.2 Build succeeds: `go build ./cmd/...`
- [ ] 10.3 Server starts without errors
- [ ] 10.4 PDF uploads create semantic chunks with rich metadata
- [ ] 10.5 Markdown uploads work with fallback
- [ ] 10.6 Existing chunks remain unchanged
