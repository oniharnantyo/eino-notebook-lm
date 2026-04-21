## 1. Fix Kreuzberg OCR Struct

- [x] 1.1 Update `KreuzbergOCRResult` struct in `pkg/parser/kreuzberg/kreuzberg.go` to match actual API: rename `Text` → `Content`, add `MimeType`, replace `Confidence`/`Language` with `OCRElements`
- [x] 1.2 Add `KreuzbergOCRElement`, `KreuzbergOCRGeometry`, `KreuzbergOCRConfidence` structs
- [x] 1.3 Update `internal/core/application/usecases/image/usecase.go` references from `img.OCRResult.Text` to `img.OCRResult.Content`
- [x] 1.4 Verify `make build` passes

## 2. VisionDescriber Interface and Factory

- [x] 2.1 Create `pkg/description/vision.go` with `VisionDescriber` interface
- [x] 2.2 Create `pkg/model/description_factory.go` with `CreateVisionDescriber` factory function
- [x] 2.3 Add vision description config fields to `internal/infrastructure/config/config.go` (`VisionDescriptionProvider`, `VisionDescriptionModel`, `VisionDescriptionAPIKey`, `VisionDescriptionBaseURL`)
- [x] 2.4 Update `.env.example` with vision description environment variables

## 3. Gemini VisionDescriber Provider

- [x] 3.1 Create `pkg/model/gemini_vision_describer.go` implementing `VisionDescriber` using Gemini SDK
- [x] 3.2 Implement structured prompt that passes OCR text as grounding context and requests factual + contextual description
- [x] 3.3 Register Gemini provider in the factory

## 4. LlamaCPP VisionDescriber Provider

- [x] 4.1 Create `pkg/model/llamacpp_vision_describer.go` implementing `VisionDescriber` using `/v1/chat/completions` with base64-encoded image
- [x] 4.2 Register LlamaCPP provider in the factory

## 5. Image Entity and UseCase Updates

- [x] 5.1 Update `internal/core/domain/entities/image.go`: add `Description` field, remove `OCRText` field, update `NewImage` constructor
- [x] 5.2 Update `internal/core/application/usecases/image/usecase.go`: add `VisionDescriber` dependency to constructor, remove `textEmbedder` dependency
- [x] 5.3 Rewrite `ProcessImages` with fail-fast logic: generate description → generate vision embedding → save entity
- [x] 5.4 Update `internal/core/application/usecases/source/usecase.go` to pass `VisionDescriber` when constructing `ImageUseCase`

## 6. Persistence and Migration

- [x] 6.1 Create migration: `ALTER TABLE images ADD COLUMN description TEXT`
- [x] 6.2 Update `internal/infrastructure/persistence/image.go` SQL queries to include `description` column and exclude `ocr_text`
- [x] 6.3 Update image DTO and mapper to reflect new entity fields

## 7. Image Resizing Before Embedding

- [x] 7.1 Create image resize utility (e.g., `pkg/imageutil/resize.go`) that proportionally resizes images to fit under 325 KB using Go's `image` + `image/jpeg`/`image/png` packages
- [x] 7.2 Integrate resize check in `imageUseCase.ProcessImages`: before calling `visionEmbedder.EmbedVision`, check byte size and resize if over 325 KB
- [x] 7.3 Ensure the original full-size image is uploaded to S3 (not the resized version)

## 8. Wiring and Verification

- [x] 8.1 Wire VisionDescriber initialization in the application startup (DI container / main)
- [x] 8.2 Run `make build` and `make lint` — fix any compilation or lint errors
- [x] 8.3 Test with a sample document containing images to verify end-to-end flow

