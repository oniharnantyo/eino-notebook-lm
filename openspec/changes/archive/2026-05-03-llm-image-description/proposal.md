## Why

Images extracted from documents are stored with embeddings but have no searchable text representation. The retriever only searches the sentences table, making images invisible to queries. OCR text quality varies significantly (confidence ranges from 0.0 to 0.96), and many images (diagrams, charts, photos) have no meaningful OCR at all. A rich, LLM-generated description is needed as the primary text representation for each image.

## What Changes

- Add a `VisionDescriber` interface that sends images to a multimodal LLM (Gemini or LlamaCPP) to generate factual descriptions with contextual interpretation
- **BREAKING**: `KreuzbergOCRResult` struct updated to match actual Kreuzberg API response (`Content` instead of `Text`, new `OCRElements` with per-word geometry/confidence)
- **BREAKING**: `Image` entity replaces `OCRText` with `Description` field (LLM-generated text as primary representation)
- **BREAKING**: `ImageUseCase` constructor adds `VisionDescriber` dependency, removes `textEmbedder` fallback
- Image ingestion becomes fail-fast: no silent degradation on description or embedding failures
- Vision embedding uses LLM description as text prompt instead of raw OCR text
- Database migration: add `description` column, remove `ocr_text` column

## Capabilities

### New Capabilities
- `vision-description`: Multimodal LLM-based image description generation with multi-provider support (Gemini, LlamaCPP). Includes VisionDescriber interface, factory, and provider implementations.

### Modified Capabilities
- `image-ingestion`: Image processing pipeline now generates LLM descriptions before embedding. Entity schema changes (Description replaces OCRText). Fail-fast error handling replaces graceful degradation.

## Impact

**New files:**
- `pkg/description/vision.go` — VisionDescriber interface
- `pkg/model/description_factory.go` — Provider factory
- `pkg/model/gemini_vision_describer.go` — Gemini implementation
- `pkg/model/llamacpp_vision_describer.go` — LlamaCPP implementation

**Modified files:**
- `pkg/parser/kreuzberg/kreuzberg.go` — Fix OCRResult struct mapping
- `internal/core/domain/entities/image.go` — Add Description, remove OCRText
- `internal/core/application/usecases/image/usecase.go` — Add VisionDescriber, fail-fast logic
- `internal/infrastructure/persistence/image.go` — Update SQL for new schema
- `internal/infrastructure/config/config.go` — Add vision description config
- `internal/core/application/usecases/source/usecase.go` — Update ImageUseCase constructor call

**Database:** Migration to add `description` column and remove `ocr_text` column on `images` table.

**Dependencies:** None new — reuses existing chat provider infrastructure (Gemini SDK, LlamaCPP HTTP API).