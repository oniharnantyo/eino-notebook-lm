## Purpose
Define the requirements for the ingestion pipeline architecture to improve maintainability and testability.

## ADDED Requirements

### Requirement: Pipeline Initialization
The system SHALL initialize an ingestion pipeline with configurable stages. Each stage SHALL receive dependencies via constructor and validate readiness before execution.

#### Scenario: Pipeline created with all stages
- **WHEN** pipeline is created with extract, parse, chunk, embed, and store stages
- **THEN** all stages are initialized in order
- **AND** each stage validates its dependencies
- **AND** pipeline is ready for ingestion

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

### Requirement: Document Parsing Stage
The parsing stage SHALL convert extracted content into structured elements using the document parser. The stage SHALL handle multiple file formats (PDF, DOCX, images).

#### Scenario: Parse PDF document
- **WHEN** extracted content is PDF
- **THEN** stage parses PDF into structured elements
- **AND** returns elements with text, metadata
- **AND** sends progress update with status "completed"

#### Scenario: Parse image with OCR
- **WHEN** extracted content is image
- **THEN** stage uses OCR to extract text
- **AND** returns elements with extracted text
- **AND** sends progress update with status "completed"

#### Scenario: Parsing failure
- **WHEN** document parser cannot parse content
- **THEN** stage returns error with file type context
- **AND** sends progress update with status "failed"

### Requirement: Chunking Stage
The chunking stage SHALL split parsed elements into manageable chunks for embedding. The stage SHALL preserve metadata and element boundaries.

#### Scenario: Chunk text elements
- **WHEN** parsed elements contain text
- **THEN** stage splits text into chunks under token limit
- **AND** preserves element metadata in chunks
- **AND** returns chunk list with parent references
- **AND** sends progress update with chunk count

#### Scenario: Chunk empty elements
- **WHEN** parsed elements are empty
- **THEN** stage returns empty chunk list
- **AND** sends progress update with warning status

### Requirement: Embedding Stage
The embedding stage SHALL generate vector embeddings for all chunks using the configured embedding model. The stage SHALL support batch processing for efficiency.

#### Scenario: Generate embeddings for chunks
- **WHEN** chunking stage returns 100 chunks
- **THEN** stage generates embeddings in batches of 10
- **AND** returns chunks with vector data
- **AND** sends progress update with batch progress

#### Scenario: Embedding service failure
- **WHEN** embedding model returns error
- **THEN** stage returns error with batch index
- **AND** sends progress update with status "failed"
- **AND** pipeline execution stops

### Requirement: Storage Stage
The storage stage SHALL persist chunks and embeddings to the database. The stage SHALL support knowledge, sentence, and image storage types.

#### Scenario: Store knowledge chunks
- **WHEN** chunks are knowledge type
- **THEN** stage inserts into knowledges table
- **AND** stores embeddings in pgvector column
- **AND** returns stored entity IDs
- **AND** sends progress update with inserted count

#### Scenario: Storage failure
- **WHEN** database insert fails
- **THEN** stage returns error with table context
- **AND** sends progress update with status "failed"
- **AND** no partial data is committed (transaction rollback)

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
