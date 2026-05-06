## MODIFIED Requirements

### Requirement: Sentence creation from knowledge chunks
The system SHALL split each knowledge chunk's content into sentences using `wikimedia/sentencex-go` for multilingual sentence boundary detection, and store each sentence with its text embedding in the `sentences` pgvector table.

#### Scenario: Split knowledge chunk into sentences
- **WHEN** a knowledge chunk is created with content "Dr. Smith went to the U.S.A. He enjoyed the trip there."
- **THEN** the system SHALL use sentencex-go with the detected language (default "en") to split into sentences
- **AND** create 2 sentence records, correctly preserving "Dr." and "U.S.A."
- **AND** each sentence SHALL have a `knowledge_id` FK referencing the parent knowledge chunk
- **AND** each sentence SHALL have a `position` integer indicating its order within the chunk

#### Scenario: Sentence embedding generation
- **WHEN** a sentence is created with content "The quick brown fox jumps over the lazy dog"
- **THEN** the system SHALL generate an embedding vector using the configured text embedding model
- **AND** store the embedding in the `embedding` column of the `sentences` table

#### Scenario: Filter short sentences
- **WHEN** a knowledge chunk splits into sentences with lengths [150, 5, 200, 8]
- **THEN** the system SHALL only create sentence records for sentences with content length > 10 characters
- **AND** position indices SHALL be reassigned sequentially after filtering

#### Scenario: Empty knowledge chunk produces no sentences
- **WHEN** a knowledge chunk is created with empty or whitespace-only content
- **THEN** the system SHALL not create any sentence records
- **AND** the system SHALL not fail or return an error

#### Scenario: Language detection from Kreuzberg
- **WHEN** Kreuzberg returns DetectedLanguages containing "fr"
- **THEN** the system SHALL use sentencex-go with language "fr"
- **AND** no DetectedLanguages SHALL default to "en"
