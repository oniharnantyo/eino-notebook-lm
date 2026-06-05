## ADDED Requirements

### Requirement: Knowledge Mapping Stage Metadata Transfer
The KnowledgeMappingStage SHALL transfer document-level metadata from the ExtractionResult to each Knowledge entity's metadata field. Document-level metadata includes properties that describe the entire document (title, authors, page_count, pdf_version, producer, quality_score, etc.) as opposed to chunk-level properties (first_page, last_page, heading_context).

#### Scenario: Transfer document metadata to knowledge chunks
- **WHEN** Kreuzberg extraction returns document-level metadata in ExtractionResult.Metadata
- **THEN** KnowledgeMappingStage SHALL merge these metadata fields into each Knowledge entity's metadata
- **AND** chunk_type field SHALL be preserved
- **AND** all knowledge chunks SHALL contain the same document-level metadata

#### Scenario: Handle missing document metadata
- **WHEN** ExtractionResult.Metadata is empty or nil
- **THEN** KnowledgeMappingStage SHALL create knowledge entities with only chunk_type metadata
- **AND** pipeline SHALL continue without error

#### Scenario: Document metadata overrides chunk metadata
- **WHEN** a key exists in both ExtractionResult.Metadata and chunk metadata
- **THEN** document-level value from ExtractionResult SHALL take precedence
- **AND** chunk-level value SHALL be overwritten

## MODIFIED Requirements

### Requirement: Storage Stage
The storage stage SHALL persist Knowledge entities (no embeddings), Sentence entities (with embeddings), and Image entities (with description text embeddings) to the database. The storage stage SHALL also persist Kreuzberg document-level metadata to the Source entity's metadata field by extracting it from the first knowledge chunk.

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
- **THEN** storage stage SHALL extract these fields from the first knowledge chunk's metadata
- **AND** merge them into the Source entity's metadata field
- **AND** persist the updated metadata alongside the existing source update
- **AND** update source chunk_count with the number of knowledge entities

#### Scenario: Document metadata not present
- **WHEN** extraction produces no document-level metadata (e.g., plain text upload)
- **THEN** storage stage SHALL proceed without error
- **AND** source metadata SHALL remain unchanged from creation time

#### Scenario: Chunk-level metadata excluded from source
- **WHEN** chunk metadata contains first_page, last_page, heading_context, or embedding keys
- **THEN** storage stage SHALL NOT merge these into source metadata
- **AND** these keys SHALL only appear on knowledge entities
