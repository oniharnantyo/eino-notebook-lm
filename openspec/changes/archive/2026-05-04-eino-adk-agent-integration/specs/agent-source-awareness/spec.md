## ADDED Requirements

### Requirement: Agent receives source catalog in system prompt

The system SHALL inject a formatted source catalog into the agent's instruction before each response generation. The catalog MUST include source ID, title, content type, chunk count, and status for each selected source.

#### Scenario: Catalog included with multiple sources
- **WHEN** user selects 3 sources (PDF, markdown, text) for a query
- **THEN** agent instruction contains a catalog with all 3 sources formatted as markdown list items
- **AND** each catalog entry includes source ID in brackets, title in quotes, content type in parentheses, chunk count, and status indicator if applicable

#### Scenario: Catalog includes processing status indicators
- **WHEN** a selected source has status "processing" or "failed"
- **THEN** the catalog entry for that source includes the status in brackets (e.g., "[processing]" or "[failed]")

#### Scenario: Catalog includes total summary
- **WHEN** the source catalog is built with 2 sources containing 15 total chunks
- **THEN** the catalog ends with a summary line "Total: 2 sources, 15 chunks"

#### Scenario: Empty catalog when no sources selected
- **WHEN** user makes a query without selecting any sources
- **THEN** the agent instruction contains an empty catalog or a default message indicating no sources available

### Requirement: Agent can query source metadata via list_sources tool

The system SHALL provide a `list_sources` tool that returns full source metadata (ID, title, content type, chunk count, status, URI, error) for all sources within the current request scope.

#### Scenario: Tool returns all selected sources
- **WHEN** agent calls `list_sources` tool during a request scoped to 3 source IDs
- **THEN** the tool returns metadata for all 3 sources
- **AND** each source includes ID, title, content type, chunk count, status, and optionally URI and error

#### Scenario: Tool returns empty result for no sources
- **WHEN** agent calls `list_sources` tool during a request with zero source IDs
- **THEN** the tool returns an empty list without error

#### Scenario: Tool respects source scope
- **WHEN** agent calls `list_sources` tool during a request scoped to source IDs [A, B, C]
- **THEN** the tool returns only sources A, B, C
- **AND** the tool MUST NOT return sources outside the scoped IDs even if they exist in the notebook

### Requirement: Agent autonomously executes retrieval tools during ReAct loop

The system SHALL enable the agent to autonomously decide when and which retrieval tools to call (`keyword_search`, `semantic_search`, `image_search`, `chunk_read`) based on the query and available sources.

#### Scenario: Agent calls semantic_search for relevant document retrieval
- **WHEN** user asks "What does the paper say about climate change?"
- **THEN** agent autonomously calls `semantic_search` tool with relevant query terms
- **AND** agent uses retrieved context to formulate the answer

#### Scenario: Agent calls keyword_search for specific term lookup
- **WHEN** user asks for all mentions of "TensorFlow" in the documents
- **THEN** agent autonomously calls `keyword_search` tool with "TensorFlow" as the query term

#### Scenario: Agent calls chunk_read to read full context
- **WHEN** agent needs more detail after initial search results
- **THEN** agent autonomously calls `chunk_read` tool with specific chunk IDs to retrieve full content

#### Scenario: Agent calls multiple tools in sequence
- **WHEN** user asks a complex query requiring both semantic search and specific term lookup
- **THEN** agent autonomously calls `semantic_search` followed by `keyword_search`
- **AND** agent synthesizes results from both tools in the final response

#### Scenario: Agent calls list_sources before searching
- **WHEN** agent needs to understand what sources are available before deciding how to search
- **THEN** agent autonomously calls `list_sources` tool first
- **AND** agent uses the source metadata to inform subsequent tool calls (e.g., filtering by content type)

### Requirement: Agent instruction combines base template with source catalog

The system SHALL build the complete agent instruction by concatenating the `BaseAgentInstruction` template with the dynamically generated source catalog for each request.

#### Scenario: Base instruction followed by catalog
- **WHEN** agent instruction is built for a request with 2 sources
- **THEN** the instruction contains `BaseAgentInstruction` followed by the source catalog
- **AND** the catalog is appended to the base instruction without modification

#### Scenario: Catalog varies per request
- **WHEN** two concurrent requests have different selected sources (Request A: sources [1,2], Request B: sources [3,4])
- **THEN** Request A's agent instruction includes catalog for sources [1,2] only
- **AND** Request B's agent instruction includes catalog for sources [3,4] only
- **AND** the two instructions are independent and do not interfere

### Requirement: Agent stage consumes ADK events and converts to SSE format

The system SHALL consume `AsyncIterator[*AgentEvent]` from the ADK Runner and convert events to the Responses API SSE format for streaming responses.

#### Scenario: Streaming chunks are emitted as delta events
- **WHEN** agent generates a streaming response during ReAct loop
- **THEN** each content chunk is emitted as a `response.output_text.delta` event
- **AND** delta events include the sequence number and content fragment

#### Scenario: Tool calls are not exposed to client
- **WHEN** agent calls a retrieval tool during ReAct loop
- **THEN** the tool call is NOT exposed to the HTTP client as a separate event
- **AND** tool calls are captured internally for observability (e.g., Langfuse tracing)

#### Scenario: Final response completes the stream
- **WHEN** agent completes its ReAct loop and generates the final answer
- **THEN** a `response.completed` event is emitted with the complete response content
- **AND** the completed event includes all accumulated text in the output

### Requirement: Pipeline removes RetrievalStage to avoid redundant retrieval

The system SHALL remove the `RetrievalStage` from the response pipeline, delegating all retrieval operations to the agent via tools.

#### Scenario: No pre-fetching of documents
- **WHEN** a response request begins processing
- **THEN** the pipeline does NOT execute a retrieval stage before the agent
- **AND** all document retrieval happens only when the agent calls tools

#### Scenario: Agent controls retrieval method
- **WHEN** agent needs to retrieve documents
- **THEN** agent decides whether to use `semantic_search`, `keyword_search`, or `image_search`
- **AND** the pipeline does not impose a default retrieval method

### Requirement: Factory method provides type-safe tool calling model

The system SHALL provide a `CreateToolCallingChatModel()` factory method that returns `model.ToolCallingChatModel`, failing early if the model doesn't support tool calling.

#### Scenario: Factory returns tool-calling model for Gemini
- **WHEN** `CreateToolCallingChatModel()` is called with Gemini model configuration
- **THEN** the method returns a `model.ToolCallingChatModel` instance without error

#### Scenario: Factory fails for non-tool-calling model
- **WHEN** `CreateToolCallingChatModel()` is called with a model that doesn't support tool calling
- **THEN** the method returns an error indicating the model doesn't support tool calling
- **AND** the error occurs during initialization, not during request processing

### Requirement: Response usecase accepts source repository for catalog building

The system SHALL update `NewResponseUseCase` to accept `repositories.SourceRepository` as a dependency and pass it to the pipeline for source catalog building.

#### Scenario: Source repository injected at construction
- **WHEN** `NewResponseUseCase()` is called with all required dependencies including `sourceRepo`
- **THEN** the usecase stores the `sourceRepo` for later use in catalog building

#### Scenario: Pipeline fetches sources by IDs
- **WHEN** a response request includes source IDs [src1, src2, src3]
- **THEN** the pipeline calls `sourceRepo.ListSourceSummariesByID()` with those IDs
- **AND** the results are used to build the source catalog
