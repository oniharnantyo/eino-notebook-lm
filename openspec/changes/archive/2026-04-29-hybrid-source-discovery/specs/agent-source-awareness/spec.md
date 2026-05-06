## ADDED Requirements

### Requirement: Agent receives source catalog in system prompt

The system SHALL inject a catalog of selected sources into the retrieval agent's system prompt before each request. The catalog SHALL include source ID, title, content type, chunk count, and processing status (if not completed).

#### Scenario: Multiple selected sources
- **WHEN** user selects 3 sources for a query
- **THEN** agent's system prompt contains a "## Available Sources" section listing all 3 sources with their metadata
- **AND** each source entry includes ID, title, type, and chunk count
- **AND** the catalog includes a total count of sources and chunks

#### Scenario: Source with processing status
- **WHEN** a selected source has status "processing" or "failed"
- **THEN** the source entry includes a status indicator (e.g., "[processing]" or "[failed]")
- **AND** the agent is informed the source may not have complete content available

#### Scenario: Single source selection
- **WHEN** user selects exactly 1 source
- **THEN** the catalog lists that single source with full metadata
- **AND** the total count reflects 1 source

### Requirement: Agent can query source details via list_sources tool

The system SHALL provide a `list_sources` tool that returns detailed metadata for all sources in the current scope. The tool SHALL return source ID, title, content type, chunk count, status, URI, error messages (if failed), and metadata map.

#### Scenario: Agent calls list_sources
- **WHEN** agent invokes `list_sources` tool with no parameters
- **THEN** system returns an array of source details for all selected sources
- **AND** each source includes id, title, content_type, chunk_count, status
- **AND** sources with failed status include an error field
- **AND** sources include URI and metadata if available

#### Scenario: list_sources respects source scope
- **WHEN** user selected sources A and B, but source C exists in the notebook
- **THEN** `list_sources` returns only A and B
- **AND** source C is not included in the response

#### Scenario: Failed source includes error details
- **WHEN** a selected source has status "failed" with error message
- **THEN** `list_sources` includes the error field with the failure reason
- **AND** the agent can communicate this to the user

### Requirement: Source catalog omits sources that fail to load

The system SHALL handle source lookup failures gracefully by omitting problematic sources from the catalog rather than failing agent creation.

#### Scenario: Source deleted after selection
- **WHEN** a selected source ID no longer exists in the database
- **THEN** the catalog is built without that source
- **AND** agent creation succeeds with remaining sources
- **AND** a warning is logged for the missing source

#### Scenario: Source lookup timeout
- **WHEN** source metadata query times out
- **THEN** the catalog is built without the timed-out source
- **AND** agent creation succeeds
- **AND** an error is logged

### Requirement: Agent instruction is built per-request

The system SHALL rebuild the source catalog and agent instruction for each request, ensuring the agent always has current information about the selected sources.

#### Scenario: Same sources, different request
- **WHEN** user sends two requests with the same source selection
- **THEN** each request generates a fresh agent with a newly built instruction
- **AND** both agents receive identical source catalogs

#### Scenario: Different source selection
- **WHEN** user changes source selection between requests
- **THEN** the second request's agent receives a catalog reflecting only the new selection
- **AND** the first agent's catalog is not reused
