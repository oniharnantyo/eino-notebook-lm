# Capability: Agent System Prompt

## Purpose
The Agent System Prompt capability manages the construction and formatting of the agent's core instructions. It ensures a consistent structure across six logical sections and provides a mechanism for injecting dynamic, request-specific context (like the source catalog) into a universal template using a surgical placeholder replacement strategy. This encapsulation within the agent package ensures that the logic for prompt construction remains co-located with the agent's behavior.

## Requirements

### Requirement: Structured prompt sections
The agent's system prompt SHALL be organized into 6 named sections in the following order: Identity & Values, Safety & Ethics, Knowledge & Facts, Tools & Products, Behavioral Guidance, Style & Tone.

#### Scenario: Prompt contains all 6 sections
- **WHEN** the BaseAgentInstruction is read
- **THEN** it contains sections in order: Identity & Values, Safety & Ethics, Knowledge & Facts, Tools & Products, Behavioral Guidance, Style & Tone
- **AND** each section is clearly delimited with a `###` heading

#### Scenario: Identity section comes first
- **WHEN** the agent processes the system prompt
- **THEN** the first section is Identity & Values, establishing the agent's role and purpose
- **AND** it describes the agent as a Retrieval Agent with access to search and reading tools

#### Scenario: Safety section establishes boundaries
- **WHEN** the agent processes the Safety & Ethics section
- **THEN** it contains rules against fabricating information, speculating beyond evidence, and handling source status limitations

#### Scenario: Knowledge section contains catalog placeholder
- **WHEN** the Knowledge & Facts section is processed
- **THEN** it contains the `{catalog}` placeholder for dynamic source injection

### Requirement: Agent owns system prompt construction
The agent package SHALL own all aspects of its system prompt construction, including the instruction template and catalog building logic.

#### Scenario: Agent package contains BaseAgentInstruction
- **WHEN** the retrieval agent is initialized
- **THEN** the agent package provides `BaseAgentInstruction` constant containing the prompt template with `{catalog}` placeholder

#### Scenario: Agent package provides BuildCatalog function
- **WHEN** a catalog is needed for the agent's system prompt
- **THEN** the `agent.BuildCatalog()` function constructs the catalog string from provided source IDs

### Requirement: Per-request catalog building
The system SHALL build the source catalog per-request based on user-selected sources.

#### Scenario: Catalog built with selected sources
- **WHEN** user selects specific sources for a query
- **THEN** the catalog includes only those selected sources with their IDs, titles, and status

#### Scenario: Empty catalog when no sources selected
- **WHEN** no sources are provided or sourceIDs is empty
- **THEN** the catalog string returns "No sources available."

#### Scenario: Catalog handles repository errors
- **WHEN** source repository returns an error
- **THEN** the function returns an error to the caller

### Requirement: ADK session value placeholder replacement
The agent SHALL use Eino ADK's built-in placeholder replacement to inject the catalog into the system prompt.

#### Scenario: Catalog injected via session values
- **WHEN** the agent executes a query
- **THEN** the catalog is passed via `adk.WithSessionValues(map[string]any{"catalog": catalog})`
- **AND** the `{catalog}` placeholder in `BaseAgentInstruction` is automatically replaced

### Requirement: BuildCatalog function signature
The `BuildCatalog` function SHALL accept context, source repository, and source IDs.

#### Scenario: Function accepts required parameters
- **WHEN** `BuildCatalog` is called
- **THEN** it receives `ctx context.Context`, `sourceRepo repositories.SourceRepository`, and `sourceIDs []uuid.UUID`
- **AND** returns a formatted catalog string and an error

#### Scenario: Function fetches sources by IDs
- **WHEN** `BuildCatalog` receives source IDs
- **THEN** it calls `sourceRepo.ListSourceSummariesByID(ctx, sourceIDs)` to fetch source metadata
- **AND** formats each source as "- [{status}] ID: {id}, Title: {title}"
