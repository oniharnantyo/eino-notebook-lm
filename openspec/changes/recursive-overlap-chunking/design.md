## Context

The current document chunking pipeline uses a custom `pkg/transformer/element/` package that implements semantic chunking — grouping Kreuzberg-parsed elements (titles, paragraphs, lists, tables) by section headers. This package has two critical issues:

1. **Dead code path**: The `knowledgeUseCase` accepts a `document.Transformer` via constructor but never calls `Transform()`. Extracted documents are passed directly to the indexer as-is, meaning all documents are indexed whole (per-page from Kreuzberg).
2. **Complexity**: The element transformer manages element type filtering, section tracking, title extraction, and a markdown header fallback — all for a strategy that's fundamentally fragile when structured elements are unavailable.

The existing pipeline flow:
```
Kreuzberg Extract → per-page docs → (transformer unused) → indexer.Store()
```

The target pipeline flow:
```
Kreuzberg Extract → per-page docs → concatenate → recursive.Splitter.Transform() → chunks → indexer.Store()
```

## Goals / Non-Goals

**Goals:**
- Replace custom element transformer with Eino's `recursive.NewSplitter`
- Actually invoke the transformer in the knowledge ingestion pipeline
- Simplify config from element-specific types to chunk_size / overlap_size
- Remove dead code (`pkg/transformer/element/`)

**Non-Goals:**
- Changing the retrieval system (retriever is chunking-strategy-agnostic)
- Changing the Kreuzberg parser or extractor layer
- Adding parent-child chunk relationships or neighborhood expansion
- Supporting multiple chunking strategies simultaneously

## Decisions

### 1. Use Eino's `recursive.NewSplitter` over custom implementation

**Choice**: Import `github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive`

**Rationale**: Eino provides a standard `document.Transformer` implementation for recursive text splitting. It's the same interface the knowledge usecase already depends on. Using a framework component eliminates custom maintenance burden and ensures consistency with the Eino ecosystem.

**Alternative considered**: Build a custom recursive splitter — rejected because it duplicates framework functionality and adds maintenance overhead.

### 2. Transform in `knowledgeUseCase.Create()`, not in source usecase

**Choice**: Call `transformer.Transform()` inside `knowledgeUseCase.Create()` after `prepareDocuments()`.

**Rationale**: The knowledge usecase already owns the transformer dependency and is responsible for preparing documents before indexing. Adding the transform call here keeps the responsibility clear — source usecase extracts raw content, knowledge usecase transforms and indexes.

**Alternative considered**: Transform in source usecase before passing docs — rejected because it would require source usecase to know about chunking, violating separation of concerns.

### 3. Concatenate page-documents before splitting

**Choice**: In `knowledgeUseCase.Create()`, concatenate all page-level documents into a single `schema.Document` before calling `transformer.Transform()`.

**Rationale**: Recursive splitting works best on contiguous text. Kreuzberg returns per-page documents, but chunk boundaries should be determined by content size/structure, not page boundaries. Concatenating first gives the splitter the full document context to produce optimal overlapping chunks.

### 4. Config: single transformer type with chunk parameters

**Choice**: Replace the `element` / `markdown` type switch in config with a single recursive splitter config (`chunk_size`, `overlap_size`).

**Rationale**: Since we're only supporting recursive splitting, the type switch is unnecessary. A simple config with two numeric parameters is cleaner and more intuitive.

## Risks / Trade-offs

- **Losing section-level metadata** → The current element transformer produces `chunk_title`, `page_start`, `page_end` metadata. Recursive splitter only produces content-based chunks without section awareness. This is acceptable because the retriever doesn't use this metadata for filtering — it relies on `reference_id`, `sub_indexes`, and vector/BM25 search. Mitigation: if needed later, metadata can be re-added via a post-transform enrichment step.

- **Chunk boundaries may split mid-sentence across pages** → The recursive splitter uses paragraph → line → word separator hierarchy, so it prefers natural boundaries. With 20% overlap, context loss at boundaries is minimal.

- **Existing indexed chunks become incompatible** → Any previously indexed documents (stored as per-page whole documents) will coexist with new smaller overlapping chunks. This is fine because the retriever's vector search scores by similarity regardless of chunk size. Old chunks will naturally score lower if they're too large and diffuse.