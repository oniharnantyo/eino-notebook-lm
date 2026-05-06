## Context

Images are extracted from documents via Kreuzberg and stored with vision embeddings, but have no searchable text representation. The retriever only searches the `sentences` table, so images are invisible to queries. OCR quality varies (confidence 0.0–0.96) and many images (diagrams, photos) have no meaningful OCR at all.

The current `KreuzbergOCRResult` struct doesn't match the actual Kreuzberg API response — it expects `text` but the API returns `content`, so OCR data was never actually captured.

The codebase uses a factory pattern for providers (chat, embedding) with environment-based configuration. This design follows the same pattern for the new VisionDescriber component.

## Goals / Non-Goals

**Goals:**
- Generate rich, contextual image descriptions using multimodal LLMs
- Use descriptions as the primary text representation for image retrieval
- Support multiple providers (Gemini, LlamaCPP) via factory pattern
- Store a single multimodal embedding (description + image pixels) per image
- Fix Kreuzberg OCR struct to match the actual API response

**Non-Goals:**
- Modifying the retriever or chat pipeline (future work)
- Re-ingesting existing documents (manual action, not automated)
- Image-to-image search (relies on future retriever changes)
- Storing per-word OCR elements in the database (kept as metadata count only)

## Decisions

### 1. VisionDescriber as a separate package (`pkg/description`)

**Decision:** Create `pkg/description/vision.go` with the `VisionDescriber` interface, separate from the embedding package.

**Rationale:** Description generation is conceptually different from embedding — it produces text for human/machine consumption, not vectors for similarity search. Following the codebase pattern where interfaces live in dedicated packages (`pkg/embedding`, `pkg/parser`).

**Alternative considered:** Put VisionDescriber inside `pkg/embedding/vision.go`. Rejected because it conflates two different concerns (text generation vs. vector encoding).

### 2. Single multimodal embedding instead of separate text + vision embeddings

**Decision:** Generate one vision embedding using the LLM description as text prompt + image pixel data. No separate text-only embedding.

**Rationale:** The vision embedder already accepts both text and image. Using the richer LLM description (instead of raw OCR) as the text prompt produces a better multimodal vector. A separate text embedding is redundant — the description text is stored in the database and can be searched with BM25 later.

**Alternative considered:** Store both a vision embedding and a text embedding of the description. Rejected as over-engineering — one good multimodal vector + a searchable text column covers both retrieval strategies.

### 3. OCR text as grounding only, not stored

**Decision:** Pass OCR text to the VisionDescriber as context but don't store it in the Image entity.

**Rationale:** OCR serves as grounding to reduce LLM hallucination, but the LLM description is strictly superior as a text representation. Storing both adds complexity with no clear retrieval benefit. OCR element count is kept in metadata for debugging.

**Alternative considered:** Store OCR text alongside description. Rejected because it would require migration complexity and no retrieval strategy uses raw OCR over LLM description.

### 4. Fail-fast error handling

**Decision:** Return errors immediately on description or embedding failure. No fallback to OCR text or text-only embeddings.

**Rationale:** Silent degradation produces incomplete data that's hard to detect. Fail-fast makes problems visible immediately. The user can re-ingest the source after fixing the underlying issue.

**Alternative considered:** Graceful degradation with OCR fallback. Rejected because it hides provider failures and produces inconsistent data quality.

### 5. Provider implementations in `pkg/model/`

**Decision:** `gemini_vision_describer.go` and `llamacpp_vision_describer.go` in `pkg/model/`, with a `description_factory.go` following the existing factory pattern.

**Rationale:** Consistent with how `chat_factory.go`, `embedding_factory.go`, and `provider.go` are organized in `pkg/model/`.

**Alternative considered:** Put providers in `pkg/description/providers/`. Rejected to match existing codebase convention of keeping providers in `pkg/model/`.

### 6. LlamaCPP uses `/v1/chat/completions` with base64 image

**Decision:** LlamaCPP VisionDescriber sends requests to the OpenAI-compatible chat endpoint with base64-encoded image in the message content array.

**Rationale:** LlamaCPP exposes an OpenAI-compatible API. The chat/completions endpoint supports multimodal messages with image_url content type when using a vision model (e.g., llava, bakllava).

### 7. Image resizing before embedding (325 KB limit)

**Decision:** Resize images exceeding 325 KB before passing to the vision embedder. Original image is uploaded to S3 unchanged. Only the embedding uses the resized version.

**Rationale:** The vision embedding model has a maximum input size of 325 KB for image data. Large images from Kreuzberg (e.g., high-DPI PDFs at 300 DPI) can be several MB. Resizing ensures embedding always succeeds within the model's constraints.

**Alternative considered:** Reject large images. Rejected because high-DPI PDFs commonly produce large images that still contain valuable content.

**Implementation:** Use Go's `image` + `image/jpeg` / `image/png` packages. Resize proportionally, iteratively reducing quality/dimensions until under 325 KB. Keep aspect ratio.

## Risks / Trade-offs

**[Cost]** Every image requires a vision LLM call during ingestion → Mitigation: Use cost-efficient models (Gemini Flash, small LlamaCPP models). Cost is amortized over many future queries.

**[Latency]** Image processing is slower with description generation → Mitigation: Description and embedding run after S3 upload in the same goroutine. Parallelism across images is preserved in the source processing pipeline.

**[Quality]** LLM descriptions may hallucinate details not in the image → Mitigation: Pass OCR text as grounding context. Use structured prompts that favor factual observation over speculation.

**[Breaking change]** `Image` entity loses `OCRText` field → Mitigation: Migration drops column. Existing data requires re-ingestion. Document this clearly.

**[Dependency on LLM availability]** Ingestion fails if the vision description provider is down → Mitigation: Fail-fast with clear error message. User retries after provider recovers. Consistent with fail-fast philosophy.

## Migration Plan

1. Deploy migration: `ALTER TABLE images ADD COLUMN description TEXT`
2. Deploy code changes (entity, usecase, repository, config)
3. Deploy migration: `ALTER TABLE images DROP COLUMN ocr_text`
4. Re-ingest any documents that had images (manual, per-source)

Rollback: Reverse migrations, revert code. No data loss for images ingested before this change (they still have their embeddings).
