# Turn-based Message Storage

## Purpose
Store all messages within a single agent turn (input, output, tool results) under a single database row to enable clean conversation sequence tracking and pagination.

## Requirements

### Requirement: Per-turn message grouping
The system SHALL store all messages from a single agent turn (same `response_id`) as one row in the `messages` table, with messages stored as a JSONB array in the `messages` column.

#### Scenario: Single agent turn with tool calls
- **WHEN** an agent turn produces a user message, 3 assistant messages, 2 tool results, and a final assistant response
- **THEN** the system SHALL save exactly 1 row in the `messages` table
- **AND** the `messages` JSONB column SHALL contain 7 `StoredMessage` objects in chronological order

#### Scenario: Multiple turns in one conversation
- **WHEN** a user sends a second message in an existing conversation
- **THEN** the system SHALL create a new row with an incremented `sequence_num`
- **AND** the new row SHALL contain only the messages from the second turn

### Requirement: Entity field change to array
The `entities.Message` struct SHALL use `Messages []*StoredMessage` instead of `Message *StoredMessage`.

#### Scenario: NewMessage constructor accepts array
- **WHEN** creating a new message entity via `NewMessage()`
- **THEN** the constructor SHALL accept `[]*StoredMessage` as the message content parameter
- **AND** store it in the `Messages` field

### Requirement: Database column rename
The `messages` table SHALL rename the `message` column to `messages` and change the default to `'[]'::jsonb`.

#### Scenario: Migration applies cleanly
- **WHEN** migration 018 is applied after truncating existing data
- **THEN** the `messages` column SHALL accept JSONB arrays
- **AND** the column default SHALL be `'[]'::jsonb`

### Requirement: History loading flattens turn arrays
The `GetMessages` consumers SHALL iterate over turn arrays and flatten them into a single `[]*schema.Message` list for model input.

#### Scenario: BeforeModelRewriteState loads history
- **WHEN** conversation history is loaded before a model call
- **THEN** the middleware SHALL iterate over each turn's `Messages` array
- **AND** convert each `StoredMessage` to `*schema.Message` via `ToEinoMessage()`
- **AND** prepend the full flattened list to the model input

#### Scenario: History stage loads history
- **WHEN** the response pipeline's history stage loads previous conversation
- **THEN** the stage SHALL flatten all turn arrays into a single `[]*schema.Message`
- **AND** apply history trimming as before

### Requirement: Token metadata from final model call
The row-level token fields (`prompt_tokens`, `completion_tokens`, `total_tokens`) SHALL reflect the last model call's usage in the turn.

#### Scenario: Multi-call turn stores last call tokens
- **WHEN** an agent turn makes 3 model calls with different token counts
- **THEN** the `prompt_tokens`, `completion_tokens`, and `total_tokens` SHALL match the 3rd (last) model call's usage

### Requirement: HTTP API returns messages array
The message endpoint response SHALL use `messages` (array) instead of `message` (single object).

#### Scenario: GetMessages endpoint response
- **WHEN** a client calls the messages endpoint
- **THEN** each message item in the response SHALL contain a `messages` field with an array of message objects
- **AND** each object SHALL have `role` and `content` fields

### Requirement: Repository persistence handles arrays
The PostgreSQL repository SHALL marshal/unmarshal `[]*StoredMessage` arrays for the `messages` column.

#### Scenario: Save marshals array
- **WHEN** saving a message entity with `Messages []*StoredMessage`
- **THEN** the repository SHALL marshal the array to JSONB
- **AND** insert it into the `messages` column

#### Scenario: GetMessages unmarshals array
- **WHEN** loading messages from the database
- **THEN** the repository SHALL unmarshal the JSONB array into `[]*StoredMessage`
- **AND** assign it to the entity's `Messages` field
