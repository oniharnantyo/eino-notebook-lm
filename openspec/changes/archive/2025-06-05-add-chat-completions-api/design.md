## Context

Currently, the Eino ADK output stream is exclusively translated into the `OpenResponses` API spec, which is heavily stateful and uses lifecycle events (`response.created`, `response.in_progress`, etc.). While correct for OpenResponses clients, standard OpenAI SDKs and third-party UIs (e.g., Chatbox, Cursor) expect the traditional `chat.completions` API format. We want to add this native endpoint without breaking existing pipelines.

## Goals / Non-Goals

**Goals:**
- Provide a `POST /api/v1/chat/completions` endpoint perfectly mimicking OpenAI's standard payload.
- Stream data directly from the ADK agent as `chat.completion.chunk` SSE events.
- Allow clients to pass `notebook_id` and `conversation_id` through the OpenAI SDK's built-in `extra_body` payload parameter.

**Non-Goals:**
- Removing or altering the existing `/v1/responses` endpoint.
- Supporting legacy features that the ADK does not handle (like `n > 1` completions).

## Decisions

- **Dedicated Request Structs (`ChatCompletionRequest`)**: Instead of shoehorning standard OpenAI fields into our `dtos.ResponseRequest`, we create dedicated DTOs that precisely model the OpenAI request (e.g. `messages` array instead of `input`). 
- **`extra_body` for custom Context**: Standard OpenAI clients allow passing arbitrary keys via `extra_body` (which get injected at the root of the JSON payload). We map this to extract `notebook_id`, `conversation_id`, and `source_id`. *Rationale*: Avoids requiring clients to send custom HTTP headers or modify bearer tokens.
- **Dedicated Formatter (`ChatCompletionsFormatter`)**: Instead of modifying the massive `ResponsesAPIFormatter`, we build a thin, simple formatter that wraps `schema.Message` output into `{"choices": [{"delta": {"content": "..."}}]}`. *Rationale*: The two formats are fundamentally different; keeping them separate reduces complexity and minimizes blast radius.

## Risks / Trade-offs

- [Risk] Diverging feature sets between the two endpoints. → Mitigation: Both handlers call the exact same `ResponseUseCase` pipeline internally. The only differences are the input mapping and output formatting.
