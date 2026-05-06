# Delta Spec: Agent Retrieval

## MODIFIED Requirements

### Requirement: Universal agent instruction prompt
The system SHALL use a single universal instruction prompt template for the agent across all notebooks. The prompt SHALL instruct the agent to use the four retrieval tools iteratively, to read chunks for full context after finding relevant snippets, to search for images when visual information may be relevant, and to provide a final answer when sufficient evidence is gathered. The prompt template SHALL contain a `{catalog}` placeholder that is replaced per-request with the available sources catalog.

#### Scenario: Agent uses the universal prompt template
- **WHEN** an agent is created for any notebook
- **THEN** the same instruction prompt template is used regardless of notebook identity
- **AND** the prompt template contains a `{catalog}` placeholder for dynamic source catalog injection
- **AND** the prompt describes the four available tools and the iterative retrieval strategy

#### Scenario: Catalog placeholder is replaced per-request
- **WHEN** the agent executes a query with specific source IDs
- **THEN** the `{catalog}` placeholder is replaced with the formatted catalog string
- **AND** the catalog is passed via ADK session values as `map[string]any{"catalog": catalog}`
- **AND** the agent receives the complete system prompt with catalog injected

#### Scenario: Agent receives only selected sources in catalog
- **WHEN** a user selects specific sources for a query
- **THEN** the catalog includes only those selected sources with their IDs, titles, and status
- **AND** the agent is aware of only those sources for tool execution
