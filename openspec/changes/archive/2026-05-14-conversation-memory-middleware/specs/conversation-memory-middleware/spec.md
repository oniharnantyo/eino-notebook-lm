## ADDED Requirements

### Requirement: Middleware automatically loads conversation history

The system SHALL automatically load conversation history before model invocation when a `previous_response_id` is provided.

#### Scenario: Load conversation history on follow-up request
- **GIVEN** a notebook has an existing conversation with `response_id` "conv-123"
- **WHEN** user sends a follow-up request with `previous_response_id` = "conv-123"
- **THEN** the middleware SHALL load the conversation from the database
- **AND** the middleware SHALL inject all messages from that conversation into the model input
- **AND** the model SHALL receive the conversation history as part of its context

#### Scenario: No history on first request
- **GIVEN** a new conversation with no `previous_response_id`
- **WHEN** user sends a request without `previous_response_id`
- **THEN** the middleware SHALL NOT attempt to load any history
- **AND** the model SHALL receive only the current user message

#### Scenario: Graceful degradation on load failure
- **GIVEN** a `previous_response_id` is provided
- **WHEN** loading the conversation from the database fails
- **THEN** the middleware SHALL log a warning
- **AND** the middleware SHALL continue processing without history
- **AND** the model SHALL receive only the current user message

### Requirement: Middleware saves conversations asynchronously

The system SHALL save completed conversations to the database asynchronously after agent execution completes.

#### Scenario: Successful async save
- **GIVEN** an agent completes execution with a response
- **WHEN** the agent's `AfterAgent` event fires
- **THEN** the middleware SHALL build a conversation entity from the event
- **AND** the middleware SHALL save the conversation to the database in a background goroutine
- **AND** the middleware SHALL NOT block the response from being sent to the client

#### Scenario: Save failure is logged
- **GIVEN** the async save operation fails
- **WHEN** the database returns an error
- **THEN** the middleware SHALL log an error with `response_id`, `notebook_id`, and error details
- **AND** the middleware SHALL NOT affect the already-sent response to the client

#### Scenario: Save includes all message types
- **GIVEN** a conversation with user messages, assistant responses, and tool calls
- **WHEN** saving the conversation
- **THEN** the middleware SHALL preserve the complete message structure including:
  - User messages with content
  - Assistant messages with reasoning content, tool calls, and response text
  - Tool responses with tool name and output
  - Message metadata (request IDs, timestamps)

### Requirement: Middleware extracts response metadata

The system SHALL extract response metadata into separate database columns for analytics.

#### Scenario: Extract finish reason
- **GIVEN** the model completes execution with `finish_reason` = "stop"
- **WHEN** saving the conversation
- **THEN** the middleware SHALL store "stop" in the `finish_reason` column

#### Scenario: Extract token usage
- **GIVEN** the model response includes token usage (prompt: 1306, completion: 569, total: 1875)
- **WHEN** saving the conversation
- **THEN** the middleware SHALL store 1306 in `prompt_tokens` column
- **AND** the middleware SHALL store 569 in `completion_tokens` column
- **AND** the middleware SHALL store 1875 in `total_tokens` column

#### Scenario: Missing metadata handled gracefully
- **GIVEN** the model response does not include token usage metadata
- **WHEN** extracting metadata
- **THEN** the middleware SHALL store 0 for all token count columns
- **AND** the middleware SHALL store empty string for `finish_reason`

### Requirement: Middleware scopes conversations per notebook

The system SHALL scope conversations by notebook to isolate conversation history.

#### Scenario: Conversations are isolated by notebook
- **GIVEN** Notebook A has conversation "conv-123"
- **GIVEN** Notebook B has conversation "conv-456"
- **WHEN** loading history for Notebook A using `previous_response_id` = "conv-123"
- **THEN** the middleware SHALL load only conversations associated with Notebook A
- **AND** the middleware SHALL NOT load conversations from Notebook B

#### Scenario: Save includes notebook association
- **GIVEN** a conversation is being saved
- **WHEN** building the conversation entity
- **THEN** the middleware SHALL associate the conversation with the `notebook_id` from the request context

### Requirement: Middleware preserves semantic message structure

The system SHALL preserve the semantic message structure (not streaming artifacts) when saving conversations.

#### Scenario: Merged reasoning content
- **GIVEN** the model streams reasoning in multiple chunks
- **WHEN** saving the conversation
- **THEN** the middleware SHALL merge all reasoning chunks into a single `reasoning_content` field

#### Scenario: Merged response content
- **GIVEN** the model streams response text in multiple chunks
- **WHEN** saving the conversation
- **THEN** the middleware SHALL merge all content chunks into a single `content` field

#### Scenario: Tool calls preserved as discrete events
- **GIVEN** the model makes tool calls during execution
- **WHEN** saving the conversation
- **THEN** the middleware SHALL preserve tool calls as discrete events with function name and arguments
- **AND** the middleware SHALL maintain chronological order of tool calls

### Requirement: Middleware handles conversation threading

The system SHALL support conversation threading via `previous_response_id` for multi-turn conversations.

#### Scenario: Thread links conversations
- **GIVEN** a first conversation with `response_id` = "conv-001"
- **WHEN** a second conversation is created with `previous_response_id` = "conv-001"
- **THEN** the middleware SHALL save the second conversation with `previous_response_id` = "conv-001"
- **AND** the second conversation SHALL include all messages from the first conversation in its `messages` array

#### Scenario: Load retrieves full thread
- **GIVEN** a conversation with `previous_response_id` = "conv-002"
- **WHEN** the middleware loads this conversation
- **THEN** the middleware SHALL retrieve all messages up to and including this response
- **AND** the loaded messages SHALL include the complete conversation thread

### Requirement: Middleware times out save operations

The system SHALL timeout save operations to prevent hanging goroutines.

#### Scenario: Save completes within timeout
- **GIVEN** a conversation save operation starts
- **WHEN** the save completes within 10 seconds
- **THEN** the middleware SHALL complete the save successfully

#### Scenario: Save times out
- **GIVEN** a conversation save operation takes longer than 10 seconds
- **WHEN** the 10-second timeout expires
- **THEN** the middleware SHALL cancel the save operation
- **AND** the middleware SHALL log an error with timeout details
