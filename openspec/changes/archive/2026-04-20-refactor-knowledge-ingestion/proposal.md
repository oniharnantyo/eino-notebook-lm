## Why

The current knowledge ingestion pipeline ignores Kreuzberg's pre-chunked output and rich elements (images, tables, heading context), re-chunking everything locally with a generic transformer. Embeddings live at the chunk level (~1000 chars), which is too coarse for precise retrieval. Images are discarded entirely. A three-level hierarchy (Source → Knowledge → Sentence) with embeddings at the sentence level enables high-precision vector search while preserving chunk-level context for the LLM.

## What Changes

- **BREAKING**: Recreate `knowledges` table as pure relational (no embedding column). Existing data will be dropped and must be re-ingested.
- Introduce `sentences` pgvector table — embeddings at sentence level for high-precision search.
- Introduce `images` pgvector table — extracted images with OCR text embeddings and S3 references.
- Expand Kreuzberg response parsing to capture chunks, images, elements, tables, heading context, and quality scores.
- Replace Eino `document.Transformer` chunking with Kreuzberg's pre-chunked output for knowledge creation.
- Repurpose recursive transformer for sentence-level splitting within knowledge chunks.
- Add S3-compatible storage (MinIO for local dev) for image binary data.
- Refactor ingestion pipeline: Extract → Store Source → Create Knowledge (relational) → Split Sentences → Embed & Index (pgvector). Image extraction runs in parallel.

## Capabilities

### New Capabilities
- `sentence-embedding`: Sentence-level splitting of knowledge chunks, embedding generation, and pgvector indexing for high-precision vector search.
- `image-ingestion`: Image extraction from documents, S3 storage, OCR text embedding, and pgvector indexing for visual search.
- `s3-storage`: S3-compatible object storage integration for binary assets (images), with MinIO for local development.

### Modified Capabilities
<!-- No existing specs to modify -->

## Impact

- **Database**: Migration drops and recreates `knowledges` table (relational only). Creates new `sentences` and `images` tables with pgvector columns. Adds HNSW indexes on both.
- **Entities**: `Knowledge` entity refactored (new fields: chunk_index, heading_context, page_range; removed embedding concern). New `Sentence` and `Image` entities.
- **Repositories**: New `SentenceRepository` and `ImageRepository`. `KnowledgeRepository` simplified (no vector operations).
- **Use Cases**: `KnowledgeUseCase` split into knowledge storage (DB) + sentence splitting + embedding. New `ImageUseCase` for image pipeline.
- **Kreuzberg Parser**: `KreuzbergExtractResponse` expanded with typed fields for chunks, images, elements, tables, quality_score.
- **File Extractor**: Refactored to return structured extraction result (chunks + images) instead of flat `[]*schema.Document`.
- **pgvector**: Two separate indexer instances — one for sentences, one for images.
- **Config**: New `S3Config` section (endpoint, access_key, secret_key, bucket).
- **Infrastructure**: docker-compose with MinIO container + bucket setup.
- **Source UseCase**: `IngestContent` pipeline restructured for new three-level flow.
