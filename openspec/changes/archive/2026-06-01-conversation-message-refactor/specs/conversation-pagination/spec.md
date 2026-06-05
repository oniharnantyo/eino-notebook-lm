## ADDED Requirements

### Requirement: Reverse cursor pagination for conversation messages
The system SHALL support reverse cursor-based pagination for conversation messages, enabling chat interfaces to display the latest messages first and load older messages on user interaction.

#### Scenario: Load latest messages
- **WHEN** user requests conversation messages without providing a cursor
- **THEN** system SHALL return the most recent N messages (default: 10)
- **AND** system SHALL include pagination metadata with `has_more` flag
- **AND** system SHALL include `oldest_sequence` for next page cursor

#### Scenario: Load older messages with cursor
- **WHEN** user provides `before_sequence` cursor parameter
- **THEN** system SHALL return messages with `sequence_num` less than the cursor value
- **AND** system SHALL order messages by `sequence_num DESC` (newest first)
- **AND** system SHALL return the specified number of messages (limit)

#### Scenario: Pagination for specific conversation
- **WHEN** user provides `conversation_id` parameter
- **THEN** system SHALL return messages for that specific conversation
- **AND** system SHALL use the conversation's `conversation_id` for the query

#### Scenario: Default to latest conversation
- **WHEN** user requests messages without providing `conversation_id`
- **THEN** system SHALL return messages for the most recent conversation
- **AND** system SHALL identify the latest conversation by `created_at DESC`

#### Scenario: Empty conversation result
- **WHEN** conversation has no messages
- **THEN** system SHALL return empty messages array
- **AND** system SHALL set `has_more` to false

### Requirement: Efficient pagination queries
The system SHALL use indexed queries for pagination to ensure performance with conversations containing 50-500 messages.

#### Scenario: Pagination query uses index
- **WHEN** executing a pagination query
- **THEN** system SHALL use the composite index on `(conversation_id, sequence_num DESC)`
- **AND** system SHALL filter by `sequence_num < cursor` for cursor-based pagination
- **AND** system SHALL NOT use `LIMIT OFFSET` for large page numbers
