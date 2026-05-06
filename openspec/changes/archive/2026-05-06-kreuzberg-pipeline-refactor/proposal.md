## Why

The current ingestion pipeline ignores Kreuzberg's document-aware chunks and re-chunks content with Eino's recursive splitter. This is wasteful (double chunking) and loses Kreuzberg's structural metadata (heading context, page boundaries). Additionally, Kreuzberg extracts images that are never processed, and sentence splitting uses a naive recursive splitter instead of proper sentence boundary detection.

## What Changes

- **BREAKING**: Replace ParsingStage and ChunkingStage with KnowledgeMappingStage that uses Kreuzberg chunks directly as Knowledge entities
- Add SentenceSplittingStage using `wikimedia/sentencex-go` for multilingual sentence boundary detection (replaces Eino recursive splitter for sentences)
- Add ImageProcessingStage to the pipeline (uploads to S3, generates LLM description, embeds description as text)
- Embed sentences instead of knowledge chunks — sentences become the primary retrieval unit
- Images are embedded via description text only (same text embedder as sentences), removing dependency on vision embedder
- Use PipelineFactory pattern to select stage composition based on source type (Kreuzberg vs URL/text)

## Capabilities

### New Capabilities
- `kreuzberg-pipeline`: Pipeline stages that use Kreuzberg chunks directly for knowledge mapping, multilingual sentence splitting via sentencex-go, and image processing within the pipeline

### Modified Capabilities
- `document-ingestion`: Pipeline composition changes — removes ParsingStage and ChunkingStage, adds KnowledgeMappingStage and SentenceSplittingStage
- `sentence-embedding`: Sentences are now split by sentencex-go instead of Eino recursive transformer, and become the sole embedding target (knowledge chunks are no longer embedded)
- `image-ingestion`: Images are now processed as a pipeline stage, embedded via description text only (no vision embedder)

## Impact

**Files removed:**
- `internal/core/application/usecases/pipeline/parsing_stage.go`
- `internal/core/application/usecases/pipeline/chunking_stage.go`

**Files added:**
- `internal/core/application/usecases/pipeline/knowledge_mapping_stage.go`
- `internal/core/application/usecases/pipeline/sentence_splitting_stage.go`
- `internal/core/application/usecases/pipeline/image_processing_stage.go`

**Files modified:**
- `internal/core/application/usecases/pipeline/factory.go` — new stage composition, PipelineFactory pattern
- `internal/core/application/usecases/pipeline/embedding_stage.go` — embed sentences instead of documents
- `internal/core/application/usecases/pipeline/storage_stage.go` — save knowledge + sentences + images, remove internal sentence splitting
- `internal/core/application/usecases/extractor/factory.go` — ExtractionResult carries chunks/images through pipeline

**Dependencies:**
- Add `github.com/wikimedia/sentencex-go` for sentence segmentation

**Database:**
- Knowledge table: embedding column no longer populated (sentences-only embedding)
- Sentence table: embedder changes from Eino recursive to sentencex-go output
- Image table: embedding changes from vision embedder to text embedder on description

**Infrastructure:**
- `visionEmbedder` dependency can be removed from image processing
