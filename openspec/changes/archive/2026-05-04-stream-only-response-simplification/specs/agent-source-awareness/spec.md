## MODIFIED Requirements

### Requirement: Agent stage consumes ADK events and converts to SSE format

The system SHALL consume `AsyncIterator[*AgentEvent]` from the ADK Runner and return a `*schema.StreamReader[*schema.Message]` containing the agent's response messages. SSE event formatting is no longer performed by the agent stage.

#### Scenario: Streaming chunks are returned as StreamReader messages
- **WHEN** agent generates a streaming response during ReAct loop
- **THEN** each content chunk is sent through the `*schema.StreamReader[*schema.Message]` as a `schema.Message` with content field populated
- **AND** the stream is consumed by the SSE formatter in the interface layer

#### Scenario: Tool calls are not exposed in stream
- **WHEN** agent calls a retrieval tool during ReAct loop
- **THEN** tool call events are NOT included in the returned `StreamReader`
- **AND** tool calls are captured internally for observability (e.g., Langfuse tracing)

#### Scenario: Final message closes the stream
- **WHEN** agent completes its ReAct loop and generates the final answer
- **THEN** the last message is sent through the stream reader
- **AND** the stream reader closes with `io.EOF`

### Requirement: Agent instruction combines base template with source catalog

The system SHALL build the complete agent instruction by concatenating the `BaseAgentInstruction` template with the dynamically generated source catalog for each request, regardless of whether the request is streaming or non-streaming.

#### Scenario: Base instruction followed by catalog
- **WHEN** agent instruction is built for a request with 2 sources
- **THEN** the instruction contains `BaseAgentInstruction` followed by the source catalog
- **AND** the catalog is appended to the base instruction without modification

#### Scenario: Catalog varies per request
- **WHEN** two concurrent requests have different selected sources (Request A: sources [1,2], Request B: sources [3,4])
- **THEN** Request A's agent instruction includes catalog for sources [1,2] only
- **AND** Request B's agent instruction includes catalog for sources [3,4] only
- **AND** the two instructions are independent and do not interfere

#### Scenario: Streaming request receives source catalog
- **WHEN** a streaming request includes source IDs [src1, src2]
- **THEN** the pipeline builds the source catalog from those IDs
- **AND** the catalog is passed to the agent stage for instruction building
- **AND** the catalog is NOT empty or omitted
