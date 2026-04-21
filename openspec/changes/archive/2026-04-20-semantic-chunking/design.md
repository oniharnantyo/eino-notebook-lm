# Semantic Chunking Design

## Context

### Current State

Document chunking is performed by `markdown.NewHeaderSplitter` in `cmd/serve.go`, which splits documents based on markdown headers (`#`, `##`, `###`). This approach:

- Works well for markdown files
- Ignores document structure from PDF/DOCX parsers
- Loses page number information
- Cannot distinguish between different content types (titles, lists, tables)

### Available Data

The Kreuzberg parser (`pkg/parser/kreuzberg/kreuzberg.go`) receives structured `elements[]` from the Kreuzberg API containing:

```json
{
  "element_id": "...",
  "element_type": "title|narrative_text|list_item|table|image|page_break|heading|code_block|block_quote|header|footer",
  "text": "...",
  "metadata": {
    "page_number": 5,
    "coordinates": {...},
    "element_index": 42
  }
}
```

**Supported Element Types:**
| Type | Description | Default Inclusion |
|------|-------------|-------------------|
| `title` | Document titles | ✅ Yes |
| `narrative_text` | Body paragraphs | ✅ Yes |
| `list_item` | Bulleted/numbered items | ✅ Yes |
| `heading` | Section headings | ✅ Yes |
| `table` | Table content | ✅ Yes |
| `image` | Image references | ❌ No (metadata only) |
| `page_break` | Page separators | ❌ No (page tracking) |
| `code_block` | Code snippets | ❌ No (configurable) |
| `block_quote` | Quoted content | ❌ No (configurable) |
| `header` | Page headers | ❌ No (typically noise) |
| `footer` | Page footers | ❌ No (typically noise) |

This data is currently discarded after text extraction.

### Constraints

- Must maintain backwards compatibility with existing chunks
- Must handle documents without element data (plain text/markdown uploads)
- No database schema changes
- Must integrate with existing Eino document transformer interface

---

## Goals / Non-Goals

**Goals:**

- Leverage existing `elements[]` data from Kreuzberg parser for semantic chunking
- Preserve page number metadata for source attribution
- Group related content by document sections (title-based grouping)
- Provide seamless fallback to markdown splitting when elements unavailable
- Make transformer type configurable

**Non-Goals:**

- Changing the retriever or indexer interfaces
- Adding new database columns or migrations
- Modifying existing chunk content or metadata
- Cross-document chunking or merging

---

## Decisions

### D1: Transformer Package Location

**Decision:** Create new package at `pkg/transformer/element/`

**Rationale:**
- Follows existing pattern (`pkg/parser/`, `pkg/retriever/`, `pkg/indexer/`)
- Keeps transformer logic isolated and testable
- Allows independent versioning if needed

**Alternatives Considered:**
- `internal/infrastructure/transformer/` - rejected: this is a reusable component, not infrastructure
- `pkg/document/transformer/element/` - rejected: unnecessary nesting

---

### D2: Chunking Strategy - Title-Based Grouping

**Decision:** Group elements by title sections (all content between titles forms one chunk)

**Rationale:**
- Titles naturally represent semantic boundaries in documents
- Simpler than more sophisticated approaches (topic modeling, semantic similarity)
- Preserves document structure as intended by author
- Works well with existing RAG retrieval patterns

**Algorithm:**
```
for each element:
  if element is title:
    emit current section as chunk
    start new section with this title
  else if element type is included:
    add to current section
emit final section as chunk
```

**Alternatives Considered:**
- Fixed-size chunking - rejected: splits mid-sentence, loses semantic meaning
- Paragraph-based - rejected: too granular for retrieval
- Semantic embedding clustering - rejected: adds complexity, latency, and cost

---

### D3: Fallback Mechanism

**Decision:** Embed markdown splitter as fallback within element transformer

**Rationale:**
- Single transformer instance handles both cases
- Transparent to calling code
- Consistent interface regardless of input type

**Implementation:**
```go
type elementTransformer struct {
    config   *Config
    fallback document.Transformer  // markdown.NewHeaderSplitter
}

func (t *elementTransformer) Transform(ctx context.Context, docs []*schema.Document) {
    for _, doc := range docs {
        if !hasElements(doc) {
            // Use markdown fallback
            return t.fallback.Transform(ctx, docs)
        }
        // Use semantic chunking
        return t.createSemanticChunks(doc)
    }
}
```

**Alternatives Considered:**
- Separate transformers chosen at startup - rejected: more complex wiring, harder to test
- No fallback (fail on missing elements) - rejected: breaks markdown uploads

---

### D4: Metadata Schema

**Decision:** Store chunk metadata in existing `metadata` JSONB column

**New Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `chunk_title` | string | Section title (empty if none) |
| `page_start` | int | First page number |
| `page_end` | int | Last page number |
| `element_count` | int | Number of elements in chunk |
| `chunk_type` | string | "semantic" or "markdown" |

**Rationale:**
- No schema migration required
- JSONB flexible for future additions
- Consistent with existing metadata patterns
- Queryable via PostgreSQL JSON operators

**Alternatives Considered:**
- Dedicated columns - rejected: requires migration, less flexible
- Separate chunk_metadata table - rejected: adds join complexity

**Retrieval Integration:**

The existing retriever (`pkg/retriever/pgvector/`) automatically returns all metadata fields. No retriever changes required:

| Use Case | Implementation |
|----------|----------------|
| Display section context | Read `chunk_title` from returned document metadata |
| Show source citation | Read `page_start`/`page_end` from metadata |
| Filter by page range | `WithWhereClause("(metadata->>'page_end')::int >= 5")` |
| Filter by section | `WithWhereClause("metadata->>'chunk_title' = 'Introduction'")` |

Future: Add dedicated `WithFilterPageRange(start, end int)` option for convenience.

---

### D5: Configuration Approach

**Decision:** Environment-based configuration via existing config pattern

```bash
TRANSFORMER_TYPE=element
TRANSFORMER_ELEMENT_INCLUDED_TYPES=title,narrative_text,list_item
TRANSFORMER_ELEMENT_MAX_CHUNK_SIZE=0
```

**Rationale:**
- Consistent with existing configuration patterns
- Easy to change without code deployment
- Supports both development and production scenarios

**Alternatives Considered:**
- Config file only - rejected: harder to override in deployments
- Feature flags - rejected: overkill for this use case

---

## Risks / Trade-offs

### Risk: Large Sections Create Oversized Chunks

**Impact:** Very long sections (e.g., entire document with no titles) produce single large chunks that may exceed embedding model limits.

**Mitigation:**
- `MaxChunkSize` configuration option to split large sections
- Default is unlimited (0) - assumes documents have reasonable structure
- Can be tuned per deployment based on document characteristics

---

### Risk: Element Data Quality from Parser

**Impact:** Poor element detection by Kreuzberg (missing titles, wrong types) results in suboptimal chunks.

**Mitigation:**
- Markdown fallback ensures processing completes even with bad data
- `chunk_type` metadata allows filtering/analysis of chunk quality
- Can adjust `included_types` to exclude unreliable element types

---

### Risk: Title Detection Varies by Document Type

**Impact:** PDFs may use font size/weight for titles rather than explicit title elements.

**Mitigation:**
- Kreuzberg handles title detection heuristics
- Fallback ensures markdown documents work correctly
- Future: could add `heading` element type support

---

### Trade-off: Chunk Granularity vs Context

**Decision:** Favor larger semantic chunks over smaller ones.

**Rationale:**
- Larger chunks provide more context for LLM generation
- Reduces total chunk count (lower storage/retrieval cost)
- Trade-off: may include some irrelevant content per chunk

**Alternative:** Smaller chunks with overlap - rejected: more complex, higher storage cost

---

## Migration Plan

### Phase 1: Deploy Transformer (Zero Impact)

1. Deploy new `pkg/transformer/element/` package
2. Update Kreuzberg parser to preserve elements in metadata
3. Add configuration options (default: element transformer)
4. Existing documents unchanged

### Phase 2: New Uploads Use Semantic Chunking

1. New document uploads automatically use element transformer
2. Elements preserved in metadata, chunks created with rich metadata
3. Existing chunks remain as-is

### Rollback Strategy

1. Set `TRANSFORMER_TYPE=markdown` in environment
2. Restart server
3. New uploads revert to markdown splitting
4. Existing chunks unaffected (no data migration to undo)

---

## Open Questions

None - design is complete based on implementation plan.
