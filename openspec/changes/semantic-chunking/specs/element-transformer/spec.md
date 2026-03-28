# Element Transformer Specification

## ADDED Requirements

### Requirement: Semantic chunking by title sections

The transformer SHALL group document elements by their semantic sections, where each section is defined by content following a title element until the next title or document end.

#### Scenario: Document with multiple title sections
- **WHEN** a document contains elements `[title: "Intro", text: "...", title: "Methods", text: "..."]`
- **THEN** the transformer SHALL produce 2 chunks: one for "Intro" section and one for "Methods" section

#### Scenario: Document with single title
- **WHEN** a document contains elements `[title: "Document", text: "para1", text: "para2"]`
- **THEN** the transformer SHALL produce 1 chunk containing both paragraphs under "Document" title

#### Scenario: Document with no titles
- **WHEN** a document contains only narrative text elements with no titles
- **THEN** the transformer SHALL produce 1 chunk with empty title containing all elements

---

### Requirement: Rich chunk metadata

Each chunk SHALL include metadata fields that enable precise source attribution and filtering.

#### Scenario: Metadata fields populated
- **WHEN** a chunk is created from elements spanning pages 5-7
- **THEN** the chunk metadata SHALL include:
  - `chunk_title`: the section title (empty string if none)
  - `page_start`: 5
  - `page_end`: 7
  - `element_count`: number of elements in chunk
  - `chunk_type`: "semantic"

#### Scenario: Title included in chunk content
- **WHEN** a section has a title "Introduction"
- **THEN** the chunk content SHALL start with `# Introduction\n\n` followed by element text

---

### Requirement: Element type support

The transformer SHALL support all Kreuzberg element types:

#### Scenario: All element types recognized
- **WHEN** the transformer processes elements
- **THEN** the following types SHALL be recognized:
  - `title` - Document titles and headings
  - `narrative_text` - Body paragraphs
  - `list_item` - Bulleted or numbered list items
  - `table` - Table content
  - `image` - Image references
  - `page_break` - Page separators
  - `heading` - Section headings
  - `code_block` - Code snippets
  - `block_quote` - Quoted content
  - `header` - Page headers
  - `footer` - Page footers

---

### Requirement: Configurable element type filtering

The transformer SHALL allow configuration of which element types to include in chunks.

#### Scenario: Default included types
- **WHEN** no configuration is provided
- **THEN** the transformer SHALL include `title`, `narrative_text`, `list_item`, `heading`, and `table` elements by default (content-bearing elements)

#### Scenario: Custom included types
- **WHEN** configuration specifies `included_types: ["title", "narrative_text"]`
- **THEN** the transformer SHALL exclude `list_item`, `table`, and other non-specified types from chunks

#### Scenario: Page breaks always excluded
- **WHEN** a document contains `page_break` elements
- **THEN** these elements SHALL NOT appear in chunk content (used only for page tracking)

#### Scenario: Headers and footers excluded by default
- **WHEN** a document contains `header` or `footer` elements
- **THEN** these elements SHALL NOT be included in chunks by default (typically noise for RAG)

---

### Requirement: Automatic markdown fallback

The transformer SHALL automatically fall back to markdown header-based splitting when no elements are available in the document metadata.

#### Scenario: Document without elements
- **WHEN** a document's metadata does not contain an `elements` array
- **THEN** the transformer SHALL use markdown header splitter as fallback

#### Scenario: Document with empty elements array
- **WHEN** a document's metadata contains `elements: []`
- **THEN** the transformer SHALL use markdown header splitter as fallback

#### Scenario: Fallback chunk metadata
- **WHEN** markdown fallback is used
- **THEN** chunks SHALL have `chunk_type: "markdown"` in metadata

---

### Requirement: Max chunk size limit

The transformer SHALL optionally limit the number of elements per chunk when configured.

#### Scenario: Max chunk size configured
- **WHEN** `max_chunk_size: 10` is configured and a section has 15 elements
- **THEN** the section SHALL be split into 2 chunks: first with 10 elements, second with 5 elements

#### Scenario: Max chunk size unlimited
- **WHEN** `max_chunk_size: 0` is configured (default)
- **THEN** sections SHALL NOT be split regardless of element count

#### Scenario: Continuation chunks preserve title
- **WHEN** a section is split due to max chunk size
- **THEN** both chunks SHALL have the same `chunk_title` and contiguous page ranges

---

### Requirement: Unique chunk identifiers

Each chunk SHALL have a unique identifier derived from the document ID and section title.

#### Scenario: Chunk ID format
- **WHEN** a chunk is created from document "doc-123" with title "Introduction"
- **THEN** the chunk ID SHALL be `doc-123-chunk-Introduction`

#### Scenario: Long title truncation
- **WHEN** a section title exceeds 50 characters
- **THEN** the chunk ID SHALL use only the first 50 characters of the title

---

### Requirement: Original metadata preservation

Chunks SHALL inherit all original document metadata while adding chunk-specific fields.

#### Scenario: Metadata inheritance
- **WHEN** a document has metadata `{source_type: "document", filename: "report.pdf"}`
- **THEN** each chunk SHALL contain both original metadata AND chunk-specific fields

#### Scenario: No metadata overwrite
- **WHEN** chunk-specific metadata fields are added
- **THEN** existing metadata fields with the same names SHALL be preserved (chunk fields added with different keys)
