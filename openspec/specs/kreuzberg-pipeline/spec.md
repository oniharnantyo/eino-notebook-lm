# Kreuzberg Pipeline Capability

## Purpose
Define the stages and logic for the Kreuzberg-optimized ingestion pipeline, including document-aware chunk mapping, multilingual sentence splitting, and integrated image processing.

## Requirements

### Requirement: Knowledge Mapping Stage
The system SHALL convert Kreuzberg chunks into Knowledge entities, preserving document-aware metadata from Kreuzberg's chunk response.

#### Scenario: Map Kreuzberg chunks to Knowledge entities
- **WHEN** ExtractionStage returns an ExtractionResult with 5 Kreuzberg chunks
- **THEN** the system SHALL create 5 Knowledge entities
- **AND** each Knowledge SHALL have content from the Kreuzberg chunk's Content field
- **AND** each Knowledge SHALL preserve heading_context from Kreuzberg chunk metadata
- **AND** each Knowledge SHALL preserve first_page and last_page from Kreuzberg chunk metadata
- **AND** each Knowledge SHALL reference the source_id from the pipeline input
- **AND** each Knowledge SHALL have a position index matching the chunk order

#### Scenario: No chunks in extraction result
- **WHEN** ExtractionResult contains zero chunks
- **THEN** the stage SHALL return an error
- **AND** progress channel SHALL receive a "failed" status

#### Scenario: Non-file source produces no chunks
- **WHEN** the extraction result is from a URL or text source that does not produce Kreuzberg chunks
- **THEN** the stage SHALL return an error indicating chunks are required

### Requirement: Sentence Splitting Stage
The system SHALL split each Knowledge chunk's content into sentences using `wikimedia/sentencex-go` for multilingual sentence boundary detection, and filter out sentences shorter than 10 characters.

#### Scenario: Split English knowledge chunk into sentences
- **WHEN** a Knowledge chunk contains "Dr. Smith went to the U.S.A. He enjoyed the trip."
- **THEN** the system SHALL use sentencex-go with language "en" to split into sentences
- **AND** the result SHALL be 2 sentences, correctly handling "Dr." and "U.S.A." as non-sentence boundaries
- **AND** each sentence SHALL reference the parent Knowledge entity via knowledge_id
- **AND** each sentence SHALL have a sequential position index

#### Scenario: Split with detected language
- **WHEN** Kreuzberg returns DetectedLanguages containing "fr"
- **THEN** the system SHALL use sentencex-go with language "fr" for splitting
- **AND** apply French-specific sentence boundary rules

#### Scenario: Language fallback
- **WHEN** Kreuzberg returns no DetectedLanguages
- **THEN** the system SHALL default to language "en"

#### Scenario: Filter short sentences
- **WHEN** a Knowledge chunk splits into sentences with lengths [150, 5, 200, 8]
- **THEN** only sentences with length > 10 SHALL be kept (150 and 200)
- **AND** position indices SHALL be reassigned sequentially after filtering

#### Scenario: Knowledge chunk produces no valid sentences
- **WHEN** all split sentences are under 10 characters
- **THEN** the stage SHALL return an empty sentence list for that chunk
- **AND** the stage SHALL NOT return an error

### Requirement: Image Processing Stage
The system SHALL process Kreuzberg images as a pipeline stage: upload to S3, generate LLM description, embed description as text, and create Image entities.

#### Scenario: Process image with OCR result
- **WHEN** Kreuzberg returns an image with data, format, dimensions, page_number, and OCR content
- **THEN** the system SHALL upload the image binary to S3
- **AND** generate an LLM description using vision describer with OCR as grounding
- **AND** generate a text embedding of the description using the text embedder
- **AND** create an Image entity with s3_key, description, and text embedding

#### Scenario: Process image without OCR result
- **WHEN** Kreuzberg returns an image with no OCR content
- **THEN** the system SHALL pass image bytes with empty OCR text to the vision describer
- **AND** continue with description embedding and storage

#### Scenario: Individual image failure does not fail pipeline
- **WHEN** an image processing step fails (S3 upload, description, or embedding)
- **THEN** the system SHALL log the error and skip that image
- **AND** continue processing remaining images
- **AND** the pipeline SHALL NOT fail

#### Scenario: No images in extraction result
- **WHEN** ExtractionResult contains zero images
- **THEN** the stage SHALL pass through with no changes
- **AND** send progress update with status "completed"

### Requirement: Pipeline Factory for Source Types
The system SHALL compose pipeline stages based on source type, using different stage sequences for Kreuzberg sources versus URL/text sources.

#### Scenario: Kreuzberg file source pipeline
- **WHEN** source content type is file (PDF, DOCX, image)
- **THEN** PipelineFactory SHALL create pipeline with stages: Extraction → KnowledgeMapping → SentenceSplitting → Embedding → ImageProcessing → Storage → StatusUpdate

#### Scenario: URL or text source pipeline
- **WHEN** source content type is URL or text
- **THEN** PipelineFactory SHALL create pipeline with existing stages: Extraction → Parsing → Chunking → Embedding → Storage → StatusUpdate
