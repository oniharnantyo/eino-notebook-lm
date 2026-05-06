## ADDED Requirements

### Requirement: Response Orchestration Stages
The system SHALL split response generation into distinct stages: context retrieval, tool preparation, response generation, and streaming. Each stage SHALL have a single responsibility and be independently testable.

#### Scenario: Successful response through all stages
- **WHEN** a response request is received with valid input
- **THEN** system executes stages in order: retrieve → prepare → generate → stream
- **AND** each stage receives input from previous stage
- **AND** response is streamed back to client

#### Scenario: Stage failure stops pipeline
- **WHEN** any stage fails with error
- **THEN** subsequent stages are not executed
- **AND** error is returned to client
- **AND** no partial response is sent

### Requirement: Context Retrieval Stage
The context retrieval stage SHALL fetch relevant knowledge using retrievers. The stage SHALL support semantic, keyword, and hybrid search modes.

#### Scenario: Semantic retrieval
- **WHEN** retrieval mode is "semantic" and query vector is provided
- **THEN** stage performs vector similarity search
- **AND** returns top-K relevant documents
- **AND** includes relevance scores in context

#### Scenario: Hybrid retrieval
- **WHEN** retrieval mode is "hybrid"
- **THEN** stage performs both semantic and keyword search
- **AND** combines results using RRF fusion
- **AND** returns fused ranked documents

### Requirement: Tool Preparation Stage
The tool preparation stage SHALL initialize available tools based on retrieved context. The stage SHALL validate tool dependencies before use.

#### Scenario: Tools prepared with context
- **WHEN** context contains documents requiring tools
- **THEN** stage initializes relevant tools (e.g., knowledge lookup)
- **AND** validates tool dependencies are available
- **AND** returns prepared tool list

#### Scenario: Tool preparation failure
- **WHEN** a required tool dependency is unavailable
- **THEN** stage returns error with missing dependency details
- **AND** pipeline stops before generation

### Requirement: Response Generation Stage
The response generation stage SHALL invoke the chat model with context and tools. The stage SHALL support streaming and non-streaming modes.

#### Scenario: Generate streaming response
- **WHEN** generation mode is "streaming"
- **THEN** stage invokes chat model with stream option
- **AND** returns channel for response chunks
- **AND** chunks are sent as model generates

#### Scenario: Generate non-streaming response
- **WHEN** generation mode is "non-streaming"
- **THEN** stage invokes chat model normally
- **AND** waits for complete response
- **AND** returns full response text

### Requirement: History Management
The system SHALL manage conversation history across requests. History SHALL be loaded before retrieval and updated after response.

#### Scenario: Load conversation history
- **WHEN** request includes conversation ID
- **THEN** stage loads previous messages from database
- **AND** includes history in generation context
- **AND** respects token limits for history

#### Scenario: Update history after response
- **WHEN** response generation completes
- **THEN** stage saves user message and assistant response
- **AND** updates conversation metadata (timestamp, message count)
- **AND** returns updated conversation

### Requirement: Error Handling
Each stage SHALL return descriptive errors. Errors SHALL include context about which stage failed and why.

#### Scenario: Retrieval error
- **WHEN** retriever service is unavailable
- **THEN** stage returns error with service name and reason
- **AND** error message is suitable for client response

#### Scenario: Generation error
- **WHEN** chat model returns error
- **THEN** stage returns error with model details
- **AND** includes original model error message

### Requirement: Configuration
Orchestration stages SHALL be configurable via constructor parameters. Configuration SHALL include retrievers, chat model, history manager, and tool factory.

#### Scenario: Configure with custom retriever
- **WHEN** orchestrator is created with custom retriever
- **THEN** retrieval stage uses provided retriever
- **AND** no default retriever is instantiated

#### Scenario: Configure without history manager
- **WHEN** orchestrator is created without history manager
- **THEN** history stage is skipped
- **AND** response proceeds without history

### Requirement: Testing Support
Each stage SHALL be testable in isolation. Stages SHALL accept interfaces for dependencies to enable mocking.

#### Scenario: Mock retriever in test
- **WHEN** test provides mock retriever
- **THEN** retrieval stage uses mock
- **AND** test can verify retriever was called correctly
- **AND** test can control retriever responses

#### Scenario: Mock chat model in test
- **WHEN** test provides mock chat model
- **THEN** generation stage uses mock
- **AND** test can verify model inputs
- **AND** test can control model responses
