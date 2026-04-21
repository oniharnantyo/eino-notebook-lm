# S3 Storage Capability

## Purpose

Provide S3-compatible object storage for binary data, supporting images and other large objects with automatic bucket management.

## Requirements

### Requirement: S3-compatible storage configuration
The system SHALL support S3-compatible object storage configuration via environment variables: `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`.

#### Scenario: Configure MinIO for local development
- **WHEN** the application starts with `S3_ENDPOINT=http://localhost:9000`, `S3_ACCESS_KEY=minioadmin`, `S3_SECRET_KEY=minioadmin`, `S3_BUCKET=eino-notebook`
- **THEN** the system SHALL connect to the MinIO instance and use the `eino-notebook` bucket for image storage

#### Scenario: Missing S3 configuration
- **WHEN** the application starts without required S3 config values
- **THEN** the system SHALL fail to start with a validation error indicating which S3 config values are missing

### Requirement: Automatic bucket creation
The system SHALL create the configured S3 bucket if it does not exist on startup.

#### Scenario: Bucket does not exist
- **WHEN** the application starts and the configured bucket `eino-notebook` does not exist
- **THEN** the system SHALL create the bucket automatically

#### Scenario: Bucket already exists
- **WHEN** the application starts and the configured bucket already exists
- **THEN** the system SHALL proceed without error

### Requirement: S3 upload with structured key
The system SHALL upload image data using a structured key format: `{source_id}/{image_id}.{format}`.

#### Scenario: Upload generates correct key
- **WHEN** an image with ID `abc-123` and format `png` is uploaded for source `src-456`
- **THEN** the S3 object key SHALL be `src-456/abc-123.png`

### Requirement: S3 object deletion
The system SHALL delete S3 objects when the corresponding image entity is deleted.

#### Scenario: Delete image removes S3 object
- **WHEN** an image record with `s3_key` = `src-456/abc-123.png` is deleted
- **THEN** the system SHALL remove the object `src-456/abc-123.png` from the S3 bucket

#### Scenario: S3 deletion failure does not block
- **WHEN** an S3 object deletion fails (e.g., object already deleted, permission error)
- **THEN** the system SHALL log the error and proceed with the database deletion
- **AND** not return an error to the caller

### Requirement: MinIO docker-compose setup
The project SHALL include a docker-compose configuration with a MinIO service for local development.

#### Scenario: Start MinIO locally
- **WHEN** a developer runs `docker-compose up`
- **THEN** MinIO SHALL be available at `http://localhost:9000` with console at `http://localhost:9001`
- **AND** default credentials SHALL be `minioadmin` / `minioadmin`
