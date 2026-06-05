# Chat Completions API

## Purpose

Provides OpenAI-compatible chat completions endpoint with streaming support and RAG context injection.

## Requirements

### Requirement: Expose chat completions endpoint
The system SHALL expose a `POST /api/v1/chat/completions` endpoint that handles standard OpenAI payloads and streams standard `chat.completion.chunk` responses.

#### Scenario: User sends a streaming request
- **WHEN** the user sends a `POST` request with `messages` array and `stream: true`
- **THEN** the system responds with `text/event-stream` SSE events
- **AND** the chunks match the structure `{"id":"...","object":"chat.completion.chunk","choices":[{"delta":{...}}]}`
- **AND** the stream terminates with `data: [DONE]`

### Requirement: Support extra_body for notebook context
The system SHALL extract Eino-specific context fields (`notebook_id`, `conversation_id`, `source_id`) from the request payload to support standard OpenAI SDK `extra_body` injection.

#### Scenario: Client injects notebook_id via extra_body
- **WHEN** a client payload includes `"notebook_id": "123"`
- **THEN** the handler extracts this ID and assigns it to the internal pipeline request context
- **AND** the pipeline validates the notebook and proceeds without error
