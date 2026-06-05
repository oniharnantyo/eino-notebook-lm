## ADDED Requirements

### Requirement: Conversation session management
The system SHALL manage conversation sessions as separate entities from message storage, with conversations representing chat sessions and messages representing individual messages within those sessions.

#### Scenario: Create conversation session
- **WHEN** user initiates a fresh chat
- **THEN** system SHALL create a new conversation session with unique `id`
- **AND** system SHALL link conversation to a `notebook_id`
- **AND** system SHALL set `created_at` timestamp

#### Scenario: Conversation contains multiple message turns
- **WHEN** user continues a conversation with follow-up messages
- **THEN** system SHALL associate all messages with the same `conversation_id`
- **AND** system SHALL maintain separate `response_id` for each message turn
- **AND** system SHALL link turns via `previous_response_id` for threading

#### Scenario: Conversation metadata storage
- **WHEN** storing conversation session information
- **THEN** system SHALL only store session-level metadata: id, notebook_id, metadata, created_at
- **AND** system SHALL NOT store message content in conversations table

### Requirement: Message turn management
The system SHALL organize messages into turns identified by `response_id`, with each turn containing one or more messages (user prompt and assistant response).

#### Scenario: Turn identification
- **WHEN** messages belong to the same turn
- **THEN** system SHALL assign the same `response_id` to all messages in that turn
- **AND** system SHALL increment `sequence_num` for each message within the turn

#### Scenario: Turn threading
- **WHEN** a new turn follows a previous turn
- **THEN** system SHALL set `previous_response_id` to the previous turn's `response_id`
- **AND** system SHALL maintain null `previous_response_id` for the first turn

#### Scenario: Turn metadata storage
- **WHEN** storing message metadata
- **THEN** system SHALL store turn-level metadata (model, tokens, finish_reason) with each message
- **AND** system SHALL NOT store turn metadata in conversations table

### Requirement: Message content storage
The system SHALL store message content as JSONB containing the full StoredMessage structure with role, content, extra fields, and timestamp.

#### Scenario: Simple text message
- **WHEN** message contains simple text content
- **THEN** system SHALL store content as string in `message` JSONB field

#### Scenario: Structured message
- **WHEN** message contains multimodal content, tool calls, or reasoning
- **THEN** system SHALL store content as structured object in `message` JSONB field
- **AND** system SHALL include tool_calls array if present
- **AND** system SHALL include reasoning_content if present

### Requirement: Unique message ordering
The system SHALL enforce unique ordering of messages within each conversation through sequence numbers.

#### Scenario: Sequence numbering
- **WHEN** adding messages to a conversation
- **THEN** system SHALL assign incrementing `sequence_num` starting from 1
- **AND** system SHALL maintain UNIQUE constraint on (conversation_id, sequence_num)