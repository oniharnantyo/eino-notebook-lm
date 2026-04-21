## ADDED Requirements

### Requirement: Artifact entity and lifecycle
The system SHALL define an `Artifact` entity with fields: ID, SourceID, NotebookID, Type (enum), Status (enum), Result (JSONB), Error, CreatedAt, UpdatedAt. The entity SHALL enforce status transitions: pending → processing → completed | failed. An artifact in `completed` or `failed` status SHALL NOT transition to any other status.

#### Scenario: Create artifact with pending status
- **WHEN** a new artifact is created with a type and source reference
- **THEN** the artifact is initialized with `pending` status, a unique UUID, and current timestamps

#### Scenario: Transition to processing
- **WHEN** artifact generation begins
- **THEN** the artifact status is updated to `processing` and UpdatedAt is refreshed

#### Scenario: Transition to completed
- **WHEN** artifact generation succeeds
- **THEN** the artifact status is updated to `completed`, the Result field is populated with type-specific JSON, Error is cleared, and UpdatedAt is refreshed

#### Scenario: Transition to failed
- **WHEN** artifact generation fails
- **THEN** the artifact status is updated to `failed`, the Error field is populated with the failure message, Result is nil, and UpdatedAt is refreshed

#### Scenario: Completed artifact is immutable
- **WHEN** a status update is attempted on a `completed` artifact
- **THEN** the update SHALL be rejected and return an error

### Requirement: Artifact types
The system SHALL support the following artifact types via an `ArtifactType` enum: `mindmap`, `podcast`, `slides`. Each type SHALL have a dedicated generation use case. The system SHALL validate that the requested type is supported before creating an artifact.

#### Scenario: Create artifact with supported type
- **WHEN** an artifact is created with type `mindmap`
- **THEN** the artifact is created successfully

#### Scenario: Reject unsupported type
- **WHEN** an artifact is created with type `unknown_type`
- **THEN** the system returns a validation error

### Requirement: Artifact repository
The system SHALL provide an `ArtifactRepository` interface with operations: Create, GetByID, ListByNotebook (with type filter and pagination), Update.

#### Scenario: List artifacts filtered by type
- **WHEN** artifacts are listed for a notebook with type filter `mindmap`
- **THEN** only artifacts of type `mindmap` are returned

#### Scenario: List artifacts with pagination
- **WHEN** artifacts are listed with page=2 and limit=10
- **THEN** the system returns artifacts 11-20 and the total count

### Requirement: List artifacts API
The system SHALL expose `GET /api/v1/notebooks/{notebookId}/artifacts` returning a paginated list of artifacts. The endpoint SHALL support query parameters: `type` (filter by artifact type), `page`, `limit`.

#### Scenario: List all artifacts for notebook
- **WHEN** `GET /api/v1/notebooks/{nbId}/artifacts` is called
- **THEN** the system returns all artifacts for that notebook with pagination metadata

#### Scenario: Filter artifacts by type
- **WHEN** `GET /api/v1/notebooks/{nbId}/artifacts?type=mindmap` is called
- **THEN** only mindmap artifacts are returned

#### Scenario: Notebook not found
- **WHEN** the notebook ID does not exist
- **THEN** the system returns 404 Not Found

### Requirement: Get artifact by ID API
The system SHALL expose `GET /api/v1/notebooks/{notebookId}/artifacts/{id}` returning a single artifact including its result JSON.

#### Scenario: Get existing artifact
- **WHEN** `GET /api/v1/notebooks/{nbId}/artifacts/{id}` is called with a valid artifact ID
- **THEN** the system returns the artifact with all fields including result

#### Scenario: Artifact not found
- **WHEN** the artifact ID does not exist
- **THEN** the system returns 404 Not Found

### Requirement: Artifact database table
The system SHALL create an `artifacts` table with columns: id (UUID PK), source_id (UUID FK → sources), notebook_id (UUID FK → notebooks), type (VARCHAR), status (VARCHAR), result (JSONB, nullable), error (TEXT, nullable), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ). The table SHALL have indexes on notebook_id and type.

#### Scenario: Foreign key enforcement
- **WHEN** an artifact references a non-existent source_id
- **THEN** the database rejects the insert with a foreign key violation

#### Scenario: Query artifacts by notebook and type efficiently
- **WHEN** artifacts are queried by notebook_id and type
- **THEN** the query uses an index scan (composite index on notebook_id + type)
