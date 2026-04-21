# Sentence Embedding Capability

## Purpose

Split knowledge chunks into sentences and generate embeddings for granular semantic search.

## Requirements

### Requirement: Sentence creation from knowledge chunks
The system SHALL split each knowledge chunk's content into sentences using a recursive transformer and store each sentence with its embedding in the `sentences` pgvector table.

#### Scenario: Split knowledge chunk into sentences
- **WHEN** a knowledge chunk is created with content "First sentence. Second sentence. Third sentence."
- **THEN** the system SHALL create 3 sentence records, each with a `knowledge_id` FK referencing the parent knowledge chunk
- **AND** each sentence SHALL have a `position` integer indicating its order within the chunk

#### Scenario: Sentence embedding generation
- **WHEN** a sentence is created with content "The quick brown fox jumps over the lazy dog"
- **THEN** the system SHALL generate an embedding vector using the configured embedding model
- **AND** store the embedding in the `embedding` column of the `sentences` table

#### Scenario: Empty knowledge chunk produces no sentences
- **WHEN** a knowledge chunk is created with empty or whitespace-only content
- **THEN** the system SHALL not create any sentence records
- **AND** the system SHALL not fail or return an error

### Requirement: Sentence storage schema
The system SHALL store sentences in a `sentences` table with columns: `id` (UUID PK), `knowledge_id` (FK to knowledges), `content` (TEXT), `embedding` (vector), `position` (INT), `metadata` (JSONB), `created_at` (TIMESTAMPTZ).

#### Scenario: Sentence record structure
- **WHEN** a sentence is persisted
- **THEN** the record SHALL contain a UUID primary key, the parent knowledge_id, the sentence text, the embedding vector, its position in the chunk, and a JSONB metadata field

### Requirement: Sentence vector search
The system SHALL support cosine similarity search on the `sentences` table using an HNSW index on the `embedding` column.

#### Scenario: Search returns matching sentences with parent knowledge
- **WHEN** a user query "what is agentic OCR" is embedded and searched against sentences
- **THEN** the system SHALL return matching sentence records ordered by cosine similarity
- **AND** each result SHALL include the `knowledge_id` for joining to the parent knowledge chunk

### Requirement: Sentence deletion cascades from knowledge
The system SHALL delete all sentences belonging to a knowledge chunk when the knowledge chunk is deleted.

#### Scenario: Delete knowledge cascades to sentences
- **WHEN** a knowledge chunk with 5 sentences is deleted
- **THEN** all 5 associated sentence records SHALL be deleted from the `sentences` table
