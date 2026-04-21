## ADDED Requirements

### Requirement: Recursive overlap splitting
The system SHALL use Eino's `recursive.NewSplitter` to split documents into chunks with configurable `chunk_size` (default 4000 chars) and `overlap_size` (default 800 chars).

#### Scenario: Document split into multiple chunks
- **WHEN** a document with 10,000 characters is ingested
- **THEN** the system SHALL produce at least 3 chunks, each no larger than 4000 characters
- **AND** adjacent chunks SHALL share overlapping content of approximately 800 characters

#### Scenario: Document smaller than chunk size
- **WHEN** a document with 2,000 characters is ingested
- **THEN** the system SHALL produce a single chunk containing the full document content

#### Scenario: Empty document
- **WHEN** an empty document is ingested
- **THEN** the system SHALL produce zero chunks and not fail

### Requirement: Transformer invocation in pipeline
The system SHALL call `transformer.Transform()` on concatenated documents during knowledge creation, before indexing.

#### Scenario: Multi-page PDF ingestion
- **WHEN** a PDF is extracted into 5 page-level documents
- **THEN** the system SHALL concatenate all pages into a single document
- **AND** pass it through the recursive splitter to produce overlapping chunks
- **AND** index each chunk with `reference_id` and source metadata

#### Scenario: Single text document ingestion
- **WHEN** a text content source is extracted as a single document
- **THEN** the system SHALL pass it through the recursive splitter
- **AND** index the resulting chunks

### Requirement: Recursive splitter configuration
The system SHALL support `TRANSFORMER_CHUNK_SIZE` and `TRANSFORMER_OVERLAP_SIZE` environment variables with sensible defaults.

#### Scenario: Default configuration
- **WHEN** no transformer env vars are set
- **THEN** chunk_size SHALL default to 4000 and overlap_size SHALL default to 800

#### Scenario: Custom configuration
- **WHEN** `TRANSFORMER_CHUNK_SIZE=2000` and `TRANSFORMER_OVERLAP_SIZE=400` are set
- **THEN** the splitter SHALL use those values

#### Scenario: Invalid overlap larger than chunk size
- **WHEN** `TRANSFORMER_OVERLAP_SIZE` is greater than `TRANSFORMER_CHUNK_SIZE`
- **THEN** the system SHALL fail to start with a validation error

### Requirement: Remove element transformer package
The system SHALL NOT include `pkg/transformer/element/` or any element-type-specific configuration.

#### Scenario: Element transformer removed
- **WHEN** the application starts
- **THEN** no code from `pkg/transformer/element/` SHALL be referenced
- **AND** config keys `TRANSFORMER_ELEMENT_INCLUDED_TYPES` and `TRANSFORMER_ELEMENT_MAX_CHUNK_SIZE` SHALL NOT be recognized

### Requirement: Chunk metadata preservation
Each chunk produced by the splitter SHALL inherit metadata from the parent document (reference_id, title, source_type, sub_indexes, created_at).

#### Scenario: Metadata on chunks
- **WHEN** a document with metadata `{reference_id: "abc", title: "My Doc"}` is split into 3 chunks
- **THEN** each chunk SHALL have `reference_id: "abc"` and `title: "My Doc"` in its metadata
