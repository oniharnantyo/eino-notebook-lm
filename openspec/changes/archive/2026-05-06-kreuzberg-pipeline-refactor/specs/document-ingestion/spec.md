## MODIFIED Requirements

### Requirement: Pipeline Initialization
The system SHALL initialize an ingestion pipeline with configurable stages selected by PipelineFactory based on source type. Each stage SHALL receive dependencies via constructor and validate readiness before execution.

#### Scenario: Pipeline created with Kreuzberg stages
- **WHEN** pipeline is created for a Kreuzberg file source
- **THEN** stages are initialized in order: Extraction, KnowledgeMapping, SentenceSplitting, Embedding, ImageProcessing, Storage, StatusUpdate
- **AND** each stage validates its dependencies
- **AND** pipeline is ready for ingestion

#### Scenario: Pipeline created with URL/text stages
- **WHEN** pipeline is created for a URL or text source
- **THEN** stages are initialized in order: Extraction, Parsing, Chunking, Embedding, Storage, StatusUpdate
- **AND** each stage validates its dependencies

#### Scenario: Pipeline created with missing dependency
- **WHEN** pipeline is created with a stage that has a nil dependency
- **THEN** constructor returns error
- **AND** error describes the missing dependency

### Requirement: Embedding Stage
The embedding stage SHALL generate vector embeddings for sentences using the configured text embedding model. The stage SHALL support batch processing for efficiency.

#### Scenario: Generate embeddings for sentences
- **WHEN** sentence splitting stage returns 200 sentences
- **THEN** stage generates embeddings for all sentence content using the text embedder
- **AND** returns sentences with vector data attached
- **AND** sends progress update with status "completed"

#### Scenario: Embedding service failure
- **WHEN** embedding model returns error
- **THEN** stage returns error with context
- **AND** sends progress update with status "failed"
- **AND** pipeline execution stops

### Requirement: Storage Stage
The storage stage SHALL persist Knowledge entities (no embeddings), Sentence entities (with embeddings), and Image entities (with description text embeddings) to the database. The storage stage SHALL also persist Kreuzberg document-level metadata to the Source entity's metadata field.

#### Scenario: Store knowledge and sentence entities
- **WHEN** pipeline provides Knowledge entities and Sentence entities with embeddings
- **THEN** stage batch inserts Knowledge entities into knowledges table (no embedding column populated)
- **AND** stage batch inserts Sentence entities into sentences table with embeddings in pgvector column
- **AND** stage batch inserts Image entities into images table with description text embeddings
- **AND** returns stored entity IDs
- **AND** sends progress update with inserted counts

#### Scenario: Storage failure
- **WHEN** database insert fails
- **THEN** stage returns error with table context
- **AND** sends progress update with status "failed"
- **AND** no partial data is committed (transaction rollback)

#### Scenario: Save document metadata to source
- **WHEN** Kreuzberg returns document-level metadata (title, authors, page_count, format_type, pdf_version, producer, is_encrypted, width, height, output_format, quality_score, pages)
- **THEN** storage stage SHALL extract these fields from the first chunk's metadata
- **AND** merge them into the Source entity's metadata field
- **AND** persist the updated metadata alongside the existing source update

#### Scenario: Document metadata not present
- **WHEN** extraction produces no document-level metadata (e.g., plain text upload)
- **THEN** storage stage SHALL proceed without error
- **AND** source metadata SHALL remain unchanged from creation time

#### Scenario: Chunk-level metadata excluded from source
- **WHEN** chunk metadata contains first_page, last_page, heading_context, or embedding keys
- **THEN** storage stage SHALL NOT merge these into source metadata
- **AND** these keys SHALL only appear on knowledge entities

## REMOVED Requirements

### Requirement: Document Parsing Stage
**Reason**: Kreuzberg sources no longer need a separate parsing stage. Kreuzberg chunks are used directly via KnowledgeMappingStage.
**Migration**: Non-Kreuzberg sources (URL/text) continue using ParsingStage through PipelineFactory.

### Requirement: Chunking Stage
**Reason**: Kreuzberg provides document-aware chunks. Re-chunking with Eino recursive splitter is redundant and loses structural metadata.
**Migration**: Non-Kreuzberg sources (URL/text) continue using ChunkingStage through PipelineFactory.
