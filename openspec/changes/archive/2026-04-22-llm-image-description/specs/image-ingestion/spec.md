# Image Ingestion Capability — Delta

## MODIFIED Requirements

### Requirement: Image extraction from Kreuzberg response
The system SHALL extract images from Kreuzberg's `images` array, generate an LLM description using the image bytes and OCR text as grounding, and store each as an image entity with the description and vision embedding.

#### Scenario: Image with OCR result
- **WHEN** Kreuzberg returns an image with `data` (byte array), `format`, `width`, `height`, `page_number`, and `ocr_result.content`
- **THEN** the system SHALL pass the image bytes and OCR content to the VisionDescriber to generate a description
- **AND** create an image record with `source_id` FK, `format`, `width`, `height`, `page_number`, and the generated `description`

#### Scenario: Image without OCR result
- **WHEN** Kreuzberg returns an image with no `ocr_result` or empty OCR content
- **THEN** the system SHALL pass the image bytes with empty OCR text to the VisionDescriber
- **AND** create an image record with the generated `description`

#### Scenario: Vision description fails
- **WHEN** the VisionDescriber returns an error for an image
- **THEN** the system SHALL return an error immediately and not continue processing that image
- **AND** the source ingestion SHALL be marked as failed

### Requirement: Image resizing before embedding
The system SHALL resize images exceeding 325 KB before passing them to the vision embedder. The original full-size image SHALL still be uploaded to S3. Only the resized version is used for embedding generation.

#### Scenario: Image under size limit
- **WHEN** an image byte size is 200 KB (under 325 KB)
- **THEN** the system SHALL pass the original image bytes to the vision embedder without resizing

#### Scenario: Image over size limit
- **WHEN** an image byte size is 1.2 MB (over 325 KB)
- **THEN** the system SHALL resize the image proportionally to fit within 325 KB
- **AND** pass the resized image bytes to the vision embedder
- **AND** upload the original full-size image to S3 (not the resized version)

#### Scenario: Image resize failure
- **WHEN** image resizing fails due to corrupt image data or unsupported format
- **THEN** the system SHALL return an error immediately and not continue processing that image

### Requirement: Image embedding from vision and description
The system SHALL generate vision embeddings using both the LLM description (as text prompt) and image pixel data (resized if over 325 KB), and store the resulting multimodal vector in the `images` table.

#### Scenario: Generate vision embedding with description
- **WHEN** an image has a generated description "A bar chart showing quarterly revenue with a peak in Q3"
- **THEN** the system SHALL pass the description as text prompt alongside the (potentially resized) image bytes to the vision embedder
- **AND** store the resulting vector in the `embedding` column

#### Scenario: Vision embedding fails
- **WHEN** the vision embedder returns an error
- **THEN** the system SHALL return an error immediately and not create the image record
- **AND** the source ingestion SHALL be marked as failed

### Requirement: Image storage schema
The system SHALL store images in an `images` table with columns: `id` (UUID PK), `source_id` (FK to sources), `s3_key` (TEXT), `format` (TEXT), `width` (INT), `height` (INT), `description` (TEXT), `page_number` (INT), `embedding` (vector), `metadata` (JSONB), `created_at` (TIMESTAMPTZ).

#### Scenario: Image record structure
- **WHEN** an image is persisted
- **THEN** the record SHALL contain a UUID primary key, the parent source_id, the S3 object key, image metadata, the LLM-generated description, vision embedding vector, and timestamps

### Requirement: Image vector search
The system SHALL support cosine similarity search on the `images` table using the `embedding` column, which contains multimodal vectors combining description text and image pixel data.

#### Scenario: Search images by description
- **WHEN** a user query "quarterly revenue chart" is used to search images
- **THEN** the system SHALL return matching image records ordered by cosine similarity on the vision embedding
- **AND** each result SHALL include the `source_id` and `description`

## REMOVED Requirements

### Requirement: Image embedding from OCR text
**Reason**: Replaced by vision embedding that combines LLM-generated description with image pixel data for richer vector representation.
**Migration**: Images are now embedded using VisionEmbedder with the LLM description as text prompt instead of text-only OCR embedding.

### Requirement: OCR text storage (part of original schema requirement)
**Reason**: OCR text is no longer stored as a first-class field. It is only used as grounding context for the LLM during description generation. The `ocr_text` column is replaced by the `description` column.
**Migration**: Existing `ocr_text` data will be lost. Re-ingest documents to generate LLM descriptions.