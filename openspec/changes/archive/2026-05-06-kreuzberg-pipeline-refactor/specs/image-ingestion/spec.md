## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Image resizing before embedding
**Reason**: Images are no longer embedded using vision embedder. Description text is embedded instead, so image resizing for embedding is unnecessary.
**Migration**: Full-size images are still uploaded to S3. No migration needed.

### Requirement: Image embedding from vision and description
**Reason**: Replaced by description-only text embedding. Images and sentences now share the same text embedder and vector space.
**Migration**: New ingestion will produce text embeddings. Existing vision embeddings remain functional but use a different vector space.
