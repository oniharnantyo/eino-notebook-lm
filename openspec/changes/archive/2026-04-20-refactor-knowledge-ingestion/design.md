## Context

The current ingestion pipeline extracts documents via Kreuzberg but discards its pre-chunked output, rich elements (images, tables, heading context), and re-chunks everything locally using an Eino `document.Transformer`. Embeddings are generated at the chunk level (~1000 chars) and stored in a single `knowledges` pgvector table. Images are completely ignored.

Kreuzberg now returns structured output with `chunks` (112 segments with heading context and page ranges), `images` (raw pixel data + OCR results), `elements` (typed semantic units), and `tables`. The refactored pipeline leverages this structure directly.

The current schema has a single `knowledges` table serving dual purpose: relational metadata AND vector search. This coupling makes it hard to optimize for different query patterns.

## Goals / Non-Goals

**Goals:**
- Separate knowledge storage (relational) from search (vector) into distinct tables
- Enable sentence-level embedding for high-precision vector search with chunk-level context retrieval
- Extract and store images from documents with OCR-based embeddings
- Use Kreuzberg's pre-chunked output instead of local re-chunking
- Add S3-compatible storage for image binary data (MinIO for local dev)

**Non-Goals:**
- Hybrid search (BM25 + vector) — out of scope for this change
- Re-ingestion tooling for existing data — manual re-upload required after migration
- Image embedding using vision models — text embeddings on OCR output only
- Streaming sentence/embedding progress to client — fire-and-forget for now

## Decisions

### 1. Three-level hierarchy: Source → Knowledge → Sentence

**Decision**: Knowledge chunks store ~1000 chars from Kreuzberg with no embedding. Sentences are split from knowledge chunks and carry embeddings. Search hits a sentence, surfaces the parent knowledge chunk as LLM context.

**Rationale**: Sentence-level embeddings give higher precision for similarity search. The parent knowledge chunk provides enough surrounding context for the LLM without overloading it. This is a "small chunk search, big chunk context" pattern common in production RAG systems.

**Alternative considered**: Embedding at knowledge chunk level (current approach) — too coarse, noisy retrieval. Embedding at element level (Kreuzberg's `narrative_text`) — elements are paragraph-level, inconsistent sizing, and would require mapping elements back to chunks.

### 2. Kreuzberg chunks as knowledge, recursive splitter for sentences

**Decision**: Use Kreuzberg's `chunks[]` directly as knowledge records. Use the existing Eino recursive transformer (with smaller chunk_size) to split each knowledge chunk into sentences.

**Rationale**: Kreuzberg chunks already have heading context, page ranges, and byte offsets. Re-chunking locally would lose this metadata. The recursive splitter is already wired in the codebase — just needs config adjustment (chunk_size from 4000 → ~200 for sentence-level).

**Alternative considered**: Splitting by regex (`.` sentence boundaries) — fragile across languages and document types. Using Kreuzberg's `elements` for sentences — `narrative_text` elements are paragraph-level, not sentence-level.

### 3. Separate pgvector tables for sentences and images

**Decision**: Two independent pgvector tables: `sentences` and `images`, each with their own HNSW index. Not a single table with a type discriminator.

**Rationale**: Different query patterns. Sentence search is text→embedding→cosine similarity. Image search is OCR text→embedding→cosine similarity. Different column sets (sentences have `knowledge_id` FK, images have `s3_key`, `format`, `width`, `height`). Separate tables allow independent schema evolution and indexing tuning.

**Alternative considered**: Single `embeddings` table with `entity_type` discriminator — adds JOIN complexity, shared index can't be tuned per type, schema becomes a union of all fields.

### 4. S3-compatible storage for images (MinIO locally)

**Decision**: Store image binary data in S3-compatible object storage. Keep only metadata + embedding in PostgreSQL. MinIO for local development.

**Rationale**: Image data is large (345K+ bytes for a single PNG in the sample). Storing blobs in PostgreSQL bloats the database, hurts backup times, and can't leverage CDN/edge caching. S3 is the standard for binary asset storage. MinIO provides full S3 API compatibility for local development.

**Alternative considered**: Storing images as BYTEA in PostgreSQL — simpler infra but doesn't scale, poor separation of concerns.

### 5. Aggressive migration (drop and recreate `knowledges`)

**Decision**: Drop the existing `knowledges` table entirely and recreate as pure relational. No data migration path.

**Rationale**: The current table couples metadata with embeddings in a schema that doesn't map to the new entity model. Attempting to migrate (extract embeddings, split into sentences) would be complex and error-prone. Clean break is simpler and the system is early enough that re-ingestion is acceptable.

### 6. Image embedding via OCR text, not vision model

**Decision**: Generate embeddings from the OCR text that Kreuzberg already produces for each image. No vision model embedding.

**Rationale**: Kreuzberg already runs OCR on extracted images and returns the text. Using this avoids an additional model call and keeps the embedding pipeline text-only. Vision model embeddings can be added later as an enhancement without schema changes.

## Risks / Trade-offs

- **[Risk] Kreuzberg chunking quality may vary across document types** → Mitigation: Kreuzberg's markdown chunker with configurable `max_characters` and `overlap` handles most document types well. Heading context provides fallback structure. Can adjust config per content-type if needed.

- **[Risk] Sentence splitter may produce uneven sentence sizes** → Mitigation: Recursive splitter handles edge cases (code blocks, tables, formulas) by falling back to character-level splitting. Short sentences (< 10 chars) can be filtered out.

- **[Risk] MinIO adds infra dependency** → Mitigation: MinIO is a single Docker container with minimal resource usage. S3 interface is standard — switching to AWS S3 requires only config changes.

- **[Trade-off] Extra DB round-trip for retrieval** → Search hits a sentence, then fetches the parent knowledge chunk. One additional query vs. current single-table approach. Acceptable because the sentence→knowledge join is by UUID (indexed) and keeps the search index lean.

- **[Trade-off] OCR text embedding for images is lossy** → Complex diagrams, charts, and figures may have poor OCR output. Accepted for now — vision model embeddings can be layered on later.

## Migration Plan

1. **Deploy migration**: DROP `knowledges` table, CREATE new `knowledges` (relational), `sentences` (pgvector), `images` (pgvector) tables
2. **Deploy application**: New ingestion pipeline handles the three-level flow
3. **Re-ingest**: Existing sources must be re-uploaded to populate knowledge, sentences, and images
4. **Rollback**: If issues arise, revert to previous migration and application version. Old `knowledges` table is gone, so rollback requires re-ingestion regardless.

## Open Questions

- Should the recursive splitter's sentence-level config be per-source-type (e.g., smaller chunks for code, larger for prose)?
- Should we add a `sentences_count` or `images_count` column to `knowledges`/`sources` for progress tracking during async ingestion?
- MinIO bucket policy — public read for images or pre-signed URLs?
