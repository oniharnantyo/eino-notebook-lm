# Image Ingestion Capability

## Purpose

Extract, store, and enable vector search for images from ingested documents, including multimodal vision embedding generation with LLM-generated descriptions.

## Requirements

### Requirement: Image extraction from Kreuzberg response
The system SHALL extract images from Kreuzberg's `images` array as a pipeline stage (ImageProcessingStage), generate an LLM description using the image bytes and OCR text as grounding, and store each as an image entity with a text embedding of the description.

#### Scenario: Image with OCR result
- **WHEN** Kreuzberg returns an image with `data` (byte array), `format`, `width`, `height`, `page_number`, and `ocr_result.content`
- **THEN** the system SHALL upload the image binary to S3
- **AND** pass the image bytes and OCR content to the VisionDescriber to generate a description
- **AND** generate a text embedding of the description using the text embedder (same embedder as sentences)
- **AND** create an image record with `source_id` FK, `s3_key`, `format`, `width`, `height`, `page_number`, `description`, and text `embedding`

#### Scenario: Image without OCR result
- **WHEN** Kreuzberg returns an image with no `ocr_result` or empty OCR content
- **THEN** the system SHALL pass the image bytes with empty OCR text to the VisionDescriber
- **AND** continue with description text embedding and storage

#### Scenario: Individual image failure does not fail pipeline
- **WHEN** image processing fails (S3 upload, description generation, or embedding)
- **THEN** the system SHALL log the error and skip that image
- **AND** continue processing remaining images
- **AND** the pipeline SHALL NOT fail

### Requirement: Image storage in S3
The system SHALL upload image binary data to S3-compatible storage and store only the `s3_key` reference in the database.

#### Scenario: Upload image to S3
- **WHEN** an image is extracted with binary data
- **THEN** the system SHALL upload the raw image bytes to the configured S3 bucket
- **AND** store the returned object key as `s3_key` in the `images` table

#### Scenario: S3 upload failure
- **WHEN** an S3 upload fails due to connection or permission error
- **THEN** the system SHALL return an error and not create the image record in the database
- **AND** the source ingestion SHALL be marked as failed

### Requirement: Image storage schema
The system SHALL store images in an `images` table with columns: `id` (UUID PK), `source_id` (FK to sources), `s3_key` (TEXT), `format` (TEXT), `width` (INT), `height` (INT), `description` (TEXT), `page_number` (INT), `embedding` (vector), `metadata` (JSONB), `created_at` (TIMESTAMPTZ).

#### Scenario: Image record structure
- **WHEN** an image is persisted
- **THEN** the record SHALL contain a UUID primary key, the parent source_id, the S3 object key, image metadata, the LLM-generated description, text embedding vector, and timestamps

### Requirement: Image vector search
The system SHALL support cosine similarity search on the `images` table using the `embedding` column, which contains text embeddings of the LLM-generated descriptions.

#### Scenario: Search images by description
- **WHEN** a user query "quarterly revenue chart" is used to search images
- **THEN** the system SHALL return matching image records ordered by cosine similarity on the text embedding of the description
- **AND** each result SHALL include the `source_id` and `description`

### Requirement: Image deletion cascades from source
The system SHALL delete all images belonging to a source when the source is deleted, and remove the corresponding objects from S3.

#### Scenario: Delete source cascades to images and S3
- **WHEN** a source with 2 images is deleted
- **THEN** both image records SHALL be deleted from the `images` table
- **AND** both S3 objects SHALL be removed from the bucket
