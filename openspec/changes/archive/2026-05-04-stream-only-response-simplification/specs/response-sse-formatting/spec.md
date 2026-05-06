## ADDED Requirements

### Requirement: SSE formatter writes Responses API event lifecycle

The system SHALL provide an `internal/interfaces/http/sse/` package that writes the complete OpenAI Responses API SSE event lifecycle to an `io.Writer`. The formatter MUST consume a `*schema.StreamReader[*schema.Message]` and emit events in the correct sequence: `response.created`, `response.in_progress`, `response.output_item.added`, `response.content_part.added`, `response.output_text.delta` (per chunk), `response.output_text.done`, `response.content_part.done`, `response.output_item.done`, `response.completed`.

#### Scenario: Full event lifecycle for successful streaming response
- **WHEN** the formatter receives a stream containing 3 content chunks ["Hello", " world", "!"]
- **THEN** it emits `response.created` with status "in_progress" and a unique response ID
- **AND** it emits `response.in_progress`
- **AND** it emits `response.output_item.added` with a message item
- **AND** it emits `response.content_part.added` with empty text
- **AND** it emits 3 `response.output_text.delta` events with delta content "Hello", " world", "!"
- **AND** it emits `response.output_text.done` with accumulated text "Hello world!"
- **AND** it emits `response.content_part.done` with full text
- **AND** it emits `response.output_item.done` with completed message
- **AND** it emits `response.completed` with status "completed" and completed timestamp

#### Scenario: Event sequence numbers increment correctly
- **WHEN** the formatter writes a full event lifecycle with 3 delta events
- **THEN** each event has a `sequenceNumber` field starting from 0 and incrementing by 1
- **AND** the `response.created` event has sequence number 0

#### Scenario: Response and message IDs are consistent
- **WHEN** the formatter generates IDs for a streaming response
- **THEN** the same response ID appears in `response.created`, `response.in_progress`, and `response.completed`
- **AND** the same message ID appears in `output_item.added`, `output_text.done`, `output_item.done`, and all delta events

### Requirement: SSE formatter handles stream errors

The system SHALL emit a `response.failed` event when the stream returns an error before `io.EOF`.

#### Scenario: Stream error mid-response
- **WHEN** the stream returns an error after emitting 2 chunks
- **THEN** the formatter emits `response.failed` event with status "failed"
- **AND** the failed event includes an error object with code "internal_error" and the error message

#### Scenario: Stream ends normally with io.EOF
- **WHEN** the stream returns `io.EOF` after emitting all chunks
- **THEN** the formatter emits `response.completed` normally without error

### Requirement: SSE formatter uses Responses-API-specific format

The system SHALL format each event as `data: <json>\n\n` where `<json>` is the JSON-encoded event with `type` field matching the OpenAI Responses API event types.

#### Scenario: Event format matches SSE specification
- **WHEN** the formatter writes a `response.output_text.delta` event with delta "Hello"
- **THEN** the output line is `data: {"type":"response.output_text.delta",...}\n\n`
- **AND** the JSON payload includes `type`, `sequence_number`, `item_id`, `output_index`, `content_index`, and `delta` fields

### Requirement: Response usecase returns raw stream instead of SSE-formatted pipe

The system SHALL return a `*schema.StreamReader[*schema.Message]` and metadata from the `Stream()` method, without performing SSE formatting. SSE formatting is the responsibility of the interface layer (handler + SSE package).

#### Scenario: Usecase returns raw stream
- **WHEN** `Stream()` is called with a valid request containing 2 source IDs
- **THEN** the usecase returns a `*schema.StreamReader[*schema.Message]` containing the agent's response chunks
- **AND** the returned stream contains raw message content, not SSE-formatted bytes

#### Scenario: Usecase returns metadata alongside stream
- **WHEN** `Stream()` is called with a valid request
- **THEN** the usecase returns metadata including model name, notebook ID, and conversation history context needed for history saving
- **AND** the metadata does not contain SSE event types or sequence numbers

### Requirement: Response usecase interface is stream-only

The system SHALL define a single `Stream()` method on `ResponseUseCase` interface. The `CreateResponse()` method SHALL be removed.

#### Scenario: Interface has single method
- **WHEN** a type implements `ResponseUseCase`
- **THEN** it only needs to implement `Stream(ctx context.Context, req *dtos.ResponseRequest) (*schema.StreamReader[*schema.Message], *StreamMeta, error)`

#### Scenario: Handler always streams
- **WHEN** a POST request is made to `/v1/responses`
- **THEN** the handler always returns `Content-Type: text/event-stream`
- **AND** the handler never returns a JSON response body
