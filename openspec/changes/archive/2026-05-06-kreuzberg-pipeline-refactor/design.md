## Context

The ingestion pipeline currently runs 6 stages: Extraction → Parsing → Chunking → Embedding → Storage → StatusUpdate. The ParsingStage uses `parser.Parse()` which returns full document content as a single `schema.Document`, discarding Kreuzberg's pre-computed chunks. The ChunkingStage then re-chunks with Eino's recursive splitter (4000 chars, 800 overlap). StorageStage splits chunks again into sentences using another recursive splitter (200 chars, 20 overlap). This means:

1. Kreuzberg chunks (1000 chars, document-aware with heading context) are computed but discarded
2. Content is re-chunked by Eino with no awareness of document structure
3. Sentences are re-split from chunks using a naive recursive splitter
4. Kreuzberg images are extracted but never processed (dead code)
5. Knowledge chunks get embeddings, but sentences also get separate embeddings — doubling embedding cost

The codebase uses Clean Architecture with pipeline stages implementing the `Stage` interface (`Name() string`, `Execute(ctx, StageInput) (StageOutput, error)`).

## Goals / Non-Goals

**Goals:**
- Use Kreuzberg chunks directly as Knowledge entities, preserving document-aware metadata
- Use `wikimedia/sentencex-go` for multilingual sentence boundary detection
- Process images as a pipeline stage with text-only description embedding
- Make sentences the sole embedding target (remove knowledge-level embeddings)
- Support PipelineFactory pattern for different source types

**Non-Goals:**
- Modifying the retrieval/query pipeline (agent tools, response pipeline)
- Changing the database schema (tables stay the same, only which columns get populated changes)
- Supporting URL/text source types in the new pipeline (keep existing flow for those)
- Migrating existing ingested data

## Decisions

### Decision 1: Use Kreuzberg Chunks as Knowledge Entities

Create `KnowledgeMappingStage` that converts `[]KreuzbergChunk` → `[]*entities.Knowledge`, preserving `heading_context`, `first_page`, `last_page`, and `chunk_type` metadata.

**Alternative considered:** Use Kreuzberg Elements as Knowledge. Rejected because Elements are paragraph-level (multiple sentences), while Chunks are document-section-level with better size consistency.

**Rationale:** Kreuzberg chunks respect document structure (headings, page boundaries). Re-chunking loses this information and duplicates work.

### Decision 2: sentencex-go for Sentence Splitting

Use `github.com/wikimedia/sentencex-go` for sentence boundary detection with language parameter derived from Kreuzberg's `DetectedLanguages` field, defaulting to `"en"`.

**Alternative considered:** `sentencizer/sentencizer` (better accuracy on English but limited language support). `regexp.Split()` (zero accuracy on abbreviations).

**Rationale:** 244+ language support with fallback chain. Built by Wikimedia for production use. Non-destructive splitting preserves original text. Adequate accuracy (74% GRS) is acceptable for RAG — over-splitting is worse than under-splitting.

### Decision 3: Embed Sentences Only

EmbeddingStage embeds sentences. Knowledge entities store content and metadata but no embedding vector.

**Alternative considered:** Embed both knowledge and sentences. Embed knowledge only.

**Rationale:** Sentences provide the finest retrieval granularity. Knowledge serves as structural grouping (heading context, page range) returned alongside sentence matches. Single embedding target reduces cost and simplifies the pipeline.

### Decision 4: Image Processing as Pipeline Stage

Create `ImageProcessingStage` that processes Kreuzberg images within the pipeline: upload to S3 → LLM description → embed description text (not vision embedding) → create Image entity.

**Alternative considered:** Post-pipeline fire-and-forget. Parallel text+image pipelines.

**Rationale:** Pipeline stage gives unified progress tracking and error handling. Using description text embedding (same embedder as sentences) puts images and sentences in the same vector space, enabling unified search across both.

### Decision 5: PipelineFactory Pattern

`PipelineFactory.Create()` selects stages based on source type. Kreuzberg sources get the new pipeline. URL/text sources keep the existing flow.

**Rationale:** Gradual migration. URL/text sources can be refactored later without risk to the Kreuzberg path.

## Risks / Trade-offs

**[sentencex-go accuracy is lower than pySBD]** → Acceptable for RAG. Over-splitting (splitting mid-sentence) is worse for retrieval than under-splitting (combining two sentences). sentencex-go's philosophy of preferring false negatives aligns with this.

**[Kreuzberg chunk boundaries may split mid-sentence]** → Kreuzberg chunks already have 200-char overlap configured. Sentences that span chunk boundaries will appear in both chunks. During storage, deduplicate by content hash if needed, or accept minor duplication.

**[Removing vision embedder from image processing]** → Description text embedding is less precise than multimodal vision embedding for image content. Trade-off: simpler architecture and unified vector space vs. richer image understanding. Can re-add vision embedding later if needed.

**[Pipeline divergence between Kreuzberg and URL/text sources]** → Two different stage compositions to maintain. Mitigated by shared Stage interface and shared stages (EmbeddingStage, StorageStage, StatusUpdateStage).

**[No data migration plan]** → Existing sources will have knowledge-level embeddings but no sentence-level data in the new format. New ingestion produces different data shape. This is acceptable for current scale; migration can be added later by re-ingesting sources.
