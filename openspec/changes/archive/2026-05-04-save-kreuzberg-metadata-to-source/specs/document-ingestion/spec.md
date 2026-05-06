## MODIFIED Requirements

### Requirement: Storage Stage
The storage stage SHALL persist chunks and embeddings to the database. The stage SHALL support knowledge, sentence, and image storage types. The storage stage SHALL also persist Kreuzberg document-level metadata to the Source entity's metadata field.

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
