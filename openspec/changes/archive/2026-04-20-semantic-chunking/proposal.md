# Semantic Chunking Proposal

## Why

Current document chunking uses markdown header-based splitting, which ignores document structure from the parser. This results in chunks that may split semantic sections mid-topic and lose important context about where content appears in the original document (page numbers, section titles).

The Kreuzberg parser already provides structured `elements[]` data with semantic information (titles, narrative text, page numbers). We should leverage this to create more meaningful, context-aware chunks that improve retrieval quality.

## What Changes

- **New**: Element-based transformer package at `pkg/transformer/element/` that chunks documents by semantic sections (grouping content under each title)
- **New**: Rich chunk metadata including `chunk_title`, `page_start`, `page_end`, `element_count`, and `chunk_type`
- **New**: Support for all Kreuzberg element types: `title`, `narrative_text`, `list_item`, `table`, `image`, `page_break`, `heading`, `code_block`, `block_quote`, `header`, `footer`
- **New**: Automatic fallback to markdown splitter when elements are unavailable (e.g., plain text uploads)
- **New**: Configurable transformer selection via `TRANSFORMER_TYPE` environment variable
- **Modified**: Kreuzberg parser to preserve and expose `elements[]` array in document metadata
- **Modified**: Server initialization to use element-based transformer by default

## Retrieval Enhancement

The retriever (`pkg/retriever/pgvector/`) already returns all metadata fields from the `metadata` JSONB column. No retriever changes are required - it automatically benefits from richer chunk metadata:

### Immediate Benefits (No Code Changes)

| Metadata Field | Retrieval Use Case |
|----------------|-------------------|
| `chunk_title` | Display section context in search results (e.g., "Found in: Introduction") |
| `page_start` / `page_end` | Show source page numbers for citation |
| `element_count` | Filter out very small chunks if needed |
| `chunk_type` | Distinguish semantic vs markdown chunks in analytics |

### Future Filtering Options (Optional)

The retriever supports custom WHERE clauses via `WithWhereClause()`. This enables filtering by new metadata without code changes:

```go
// Filter by page range
retriever.WithWhereClause("(metadata->>'page_end')::int >= 5 AND (metadata->>'page_start')::int <= 10")

// Filter by chunk title
retriever.WithWhereClause("metadata->>'chunk_title' = 'Introduction'")
```

Later, dedicated filter options (`WithFilterPageRange`, `WithFilterChunkTitle`) can be added for convenience.

## Capabilities

### New Capabilities

- `element-transformer`: Semantic document chunking that groups content by title sections using the Kreuzberg parser's elements[] array, with automatic markdown fallback and rich metadata (chunk_title, page_start, page_end, element_count)

### Modified Capabilities

None - this is a new capability that doesn't change existing spec-level behavior.

## Impact

### Code Changes

| File/Package | Change |
|--------------|--------|
| `pkg/transformer/element/` | **New package** - element-based transformer implementation |
| `pkg/parser/kreuzberg/kreuzberg.go` | Add `Elements` field to response struct |
| `internal/infrastructure/config/config.go` | Add `TransformerConfig` for configuration |
| `cmd/serve.go` | Use configurable transformer instead of hardcoded markdown |
| `.env.example` | Add transformer configuration examples |

### Database

No schema migrations required. New metadata fields (`chunk_title`, `page_start`, `page_end`, `element_count`, `chunk_type`) are stored in the existing `metadata` JSON column.

### Dependencies

- Uses existing `github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown` for fallback
- No new external dependencies

### Backwards Compatibility

- Existing chunks remain valid
- New uploads get enhanced semantic metadata
- Markdown fallback ensures non-PDF documents still process correctly
- Default transformer type is `element` (can be changed to `markdown` if needed)
