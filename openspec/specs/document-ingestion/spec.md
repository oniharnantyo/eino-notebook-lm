## Purpose
Define the requirements for the ingestion pipeline architecture to improve maintainability and testability.

## ADDED Requirements

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

### Requirement: Progress Streaming
The system SHALL stream ingestion progress via a read-only channel. Progress updates SHALL include stage name, status, and optional error details.

#### Scenario: Progress channel sends stage updates
- **WHEN** pipeline executes a stage
- **THEN** progress channel receives update with stage name
- **AND** status indicates "in_progress", "completed", or "failed"
- **AND** update includes timestamp

#### Scenario: Progress channel sends error details
- **WHEN** a stage fails with error
- **THEN** progress channel receives update with status "failed"
- **AND** update includes error message
- **AND** channel is closed after final update

### Requirement: Document Extraction Stage
The extraction stage SHALL download content from the source URL using the Kreuzberg service. The stage SHALL return raw content or a structured error.

#### Scenario: Successful extraction
- **WHEN** source URL is accessible
- **THEN** stage downloads content from Kreuzberg
- **AND** returns raw bytes or text
- **AND** sends progress update with status "completed"

#### Scenario: Extraction failure
- **WHEN** Kreuzberg service returns error
- **THEN** stage returns error with context
- **AND** sends progress update with status "failed"
- **AND** pipeline execution stops

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

### Requirement: Status Update Stage
The status update stage SHALL update the source entity status to reflect completion. The stage SHALL set status to "completed" on success or "failed" with error message on failure.

#### Scenario: Update status on success
- **WHEN** all previous stages complete successfully
- **THEN** stage updates source status to "completed"
- **AND** stores completion timestamp
- **AND** sends final progress update

#### Scenario: Update status on failure
- **WHEN** any stage fails
- **THEN** stage updates source status to "failed"
- **AND** stores error message in source entity
- **AND** sends final progress update

### Requirement: Pipeline Parallelism
The system SHALL support configurable parallelism for independent operations. The pipeline SHALL execute stages sequentially but allow parallel work within stages.

#### Scenario: Parallel embedding generation
- **WHEN** embedding stage is configured with parallelism=5
- **THEN** stage generates up to 5 embedding batches concurrently
- **AND** combines results preserving order
- **AND** sends progress updates per batch

#### Scenario: Sequential stage execution
- **WHEN** pipeline is configured with sequential mode
- **THEN** stages execute one after another
- **AND** each stage completes before next starts
- **AND** progress updates are sent in order

### Requirement: Pipeline Completion
The system SHALL close the progress channel and return final status when pipeline completes or fails. The pipeline SHALL guarantee no orphaned goroutines.

#### Scenario: Successful pipeline completion
- **WHEN** all stages complete successfully
- **THEN** progress channel receives final "completed" update
- **AND** channel is closed
- **AND** source status is "completed"
- **AND** no goroutines leak

#### Scenario: Pipeline failure
- **WHEN** any stage fails
- **THEN** progress channel receives final "failed" update
- **AND** channel is closed
- **AND** source status is "failed" with error
- **AND** no goroutines leak
