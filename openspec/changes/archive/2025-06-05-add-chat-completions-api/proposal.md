## Why

The current API exposes LLM outputs exclusively through the `OpenResponses` format (`/v1/responses`), which wraps events in a complex lifecycle state machine. Standard OpenAI clients expect the standard `/chat/completions` API schema. Adding a native `/chat/completions` endpoint allows direct compatibility with the entire OpenAI ecosystem without requiring a proxy, making the agent more versatile.

## What Changes

- Add a new API endpoint at `POST /api/v1/chat/completions`.
- Create new DTOs (`ChatCompletionRequest`, `ChatCompletionMessage`, `ChatCompletionExtraBody`) to parse standard OpenAI payloads.
- Map the custom OpenAI `extra_body` payload to support context resolution (`notebook_id`, `conversation_id`, `source_id`).
- Create a dedicated `ChatCompletionsHandler` to validate and map the incoming payload to the ADK Agent pipeline.
- Introduce `ChatCompletionsFormatter` to translate ADK `schema.Message` chunks into `chat.completion` and `chat.completion.chunk` Server-Sent Events payloads.

## Capabilities

### New Capabilities
- `chat-completions-api`: Exposes the `/chat/completions` endpoint that streams standard OpenAI API responses and parses standard requests, leveraging `extra_body` for Eino-specific context.

### Modified Capabilities

## Impact

- **New Files**: 
  - `internal/core/application/dtos/chat_completions.go`
  - `internal/interfaces/http/handlers/chat_completions.go`
  - `internal/interfaces/http/sse/chat_completions_formatter.go`
- **Modified Files**: 
  - `internal/interfaces/http/routes/routes.go` (to wire the new handler)
- **APIs**: New `POST /api/v1/chat/completions` endpoint.
