## ADDED Requirements

### Requirement: Image extraction from Kreuzberg response
The system SHALL extract images from Kreuzberg's `images` array and store each as an image entity with metadata and embedding.

#### Scenario: Image with OCR result
- **WHEN** Kreuzberg returns an image with `data` (byte array), `format`, `width`, `height`, `page_number`, and `ocr_result.content`
- **THEN** the system SHALL create an image record with `source_id` FK, `format`, `width`, `height`, `page_number`, and the OCR text content

#### Scenario: Image without OCR result
- **WHEN** Kreuzberg returns an image with no `ocr_result` or empty OCR content
- **THEN** the system SHALL create an image record with `ocr_text` set to empty string
- **AND** the system SHALL NOT generate an embedding for that image

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

### Requirement: Image embedding from OCR text
The system SHALL generate text embeddings from the OCR result of each image and store them in the `images` pgvector table.

#### Scenario: Embed OCR text
- **WHEN** an image has OCR text content "Shanghai Artificial Intelligence Laboratory"
- **THEN** the system SHALL generate an embedding vector from the OCR text
- **AND** store it in the `embedding` column of the `images` table

### Requirement: Image storage schema
The system SHALL store images in an `images` table with columns: `id` (UUID PK), `source_id` (FK to sources), `s3_key` (TEXT), `format` (TEXT), `width` (INT), `height` (INT), `ocr_text` (TEXT), `page_number` (INT), `embedding` (vector), `metadata` (JSONB), `created_at` (TIMESTAMPTZ).

#### Scenario: Image record structure
- **WHEN** an image is persisted
- **THEN** the record SHALL contain a UUID primary key, the parent source_id, the S3 object key, image metadata, OCR text, embedding vector, and timestamps

### Requirement: Image vector search
The system SHALL support cosine similarity search on the `images` table using an HNSW index on the `embedding` column.

#### Scenario: Search images by OCR content
- **WHEN** a user query "laboratory logo" is embedded and searched against images
- **THEN** the system SHALL return matching image records ordered by cosine similarity
- **AND** each result SHALL include the `source_id` for joining to the parent source

### Requirement: Image deletion cascades from source
The system SHALL delete all images belonging to a source when the source is deleted, and remove the corresponding objects from S3.

#### Scenario: Delete source cascades to images and S3
- **WHEN** a source with 2 images is deleted
- **THEN** both image records SHALL be deleted from the `images` table
- **AND** both S3 objects SHALL be removed from the bucket
