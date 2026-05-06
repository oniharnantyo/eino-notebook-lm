## 1. Dependencies

- [x] 1.1 Add `github.com/wikimedia/sentencex-go` dependency via `go get`
- [x] 1.2 Verify build passes with `make build`

## 2. Knowledge Mapping Stage

- [x] 2.1 Create `internal/core/application/usecases/pipeline/knowledge_mapping_stage.go` with `KnowledgeMappingStage` struct implementing the `Stage` interface
- [x] 2.2 Implement `Execute()` to convert `[]KreuzbergChunk` from `ExtractionResult` into `[]*entities.Knowledge`, preserving `heading_context`, `first_page`, `last_page`, and `chunk_type` metadata
- [x] 2.3 Write unit tests for KnowledgeMappingStage: success case with multiple chunks, empty chunks error case, metadata preservation

## 3. Sentence Splitting Stage

- [x] 3.1 Create `internal/core/application/usecases/pipeline/sentence_splitting_stage.go` with `SentenceSplittingStage` struct implementing the `Stage` interface
- [x] 3.2 Implement `Execute()` using `sentencex.Segment(lang, content)` with language from Kreuzberg's `DetectedLanguages` (default "en"), filtering sentences with `len <= 10`
- [x] 3.3 Define intermediate `Sentence` struct with fields: ID, KnowledgeID, Content, Position
- [x] 3.4 Write unit tests for SentenceSplittingStage: English text, abbreviation handling ("Dr.", "U.S.A."), short sentence filtering, language fallback to "en"

## 4. Image Processing Stage

- [x] 4.1 Create `internal/core/application/usecases/pipeline/image_processing_stage.go` with `ImageProcessingStage` struct implementing the `Stage` interface
- [x] 4.2 Implement `Execute()` to iterate Kreuzberg images: upload to S3, generate LLM description via VisionDescriber, embed description text via text embedder, create Image entity
- [x] 4.3 Handle individual image failures by logging and continuing (not failing pipeline)
- [x] 4.4 Write unit tests for ImageProcessingStage: success case, image failure skips gracefully, no images pass-through

## 5. Embedding Stage Refactor

- [x] 5.1 Modify `EmbeddingStage.Execute()` to accept `[]*Sentence` (intermediate struct) instead of `[]*schema.Document`
- [x] 5.2 Embed sentence content using text embedder, attach embedding to each Sentence
- [x] 5.3 Update unit tests to reflect new input/output types

## 6. Storage Stage Refactor

- [x] 6.1 Modify `StorageStage.Execute()` to accept Knowledge entities + Sentence entities (with embeddings) + Image entities
- [x] 6.2 Remove internal sentence splitting logic (recursive transformer) — sentences come pre-split from SentenceSplittingStage
- [x] 6.3 Remove knowledge-level embedding storage — Knowledge table no longer gets embedding column populated
- [x] 6.4 Save Knowledge batch, Sentence batch (with embeddings), and Image batch (with description embeddings)
- [x] 6.5 Preserve existing Kreuzberg document-level metadata merge logic for Source entity
- [x] 6.6 Update unit tests for new input types and storage behavior

## 7. Pipeline Factory Refactor

- [x] 7.1 Modify `PipelineFactory.Create()` to accept source type parameter
- [x] 7.2 Implement Kreuzberg pipeline composition: Extraction → KnowledgeMapping → SentenceSplitting → Embedding → ImageProcessing → Storage → StatusUpdate
- [x] 7.3 Preserve existing pipeline composition for non-Kreuzberg sources: Extraction → Parsing → Chunking → Embedding → Storage → StatusUpdate
- [x] 7.4 Update `cmd/serve.go` DI wiring to pass ImageUseCase dependencies to ImageProcessingStage via factory
- [x] 7.5 Write integration test for full Kreuzberg pipeline flow

## 8. Cleanup

- [x] 8.1 Remove unused `visionEmbedder` initialization from `cmd/serve.go` if no other consumers exist
- [x] 8.2 Verify all tests pass with `make test`
- [x] 8.3 Verify linter passes with `make lint`
- [x] 8.4 Verify build succeeds with `make build`
