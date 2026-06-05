## 1. DTOs and Formatter

- [x] 1.1 Create `ChatCompletionRequest`, `ChatCompletionMessage`, and `ChatCompletionExtraBody` structs in `internal/core/application/dtos/chat_completions.go`
- [x] 1.2 Create `ChatCompletionsFormatter` struct and `WriteResponse` method in `internal/interfaces/http/sse/chat_completions_formatter.go`
- [x] 1.3 Implement mapping logic inside `ChatCompletionsFormatter` to translate ADK `schema.Message` chunks into `chat.completion.chunk` Server-Sent Events payloads

## 2. Handler Implementation

- [x] 2.1 Create `ChatCompletionsHandler` in `internal/interfaces/http/handlers/chat_completions.go`
- [x] 2.2 Implement payload parsing and basic validation in `CreateCompletion` method
- [x] 2.3 Extract `extra_body` from payload and map `notebook_id`, `conversation_id`, and `source_id` into the request context
- [x] 2.4 Wire `ChatCompletionsHandler` to call the existing `chat.ResponseUseCase` pipeline
- [x] 2.5 Ensure the handler utilizes the new `ChatCompletionsFormatter` to write the streamed response

## 3. Routing and Testing

- [x] 3.1 Update `internal/interfaces/http/routes/routes.go` to add `POST /v1/chat/completions` route and inject `ChatCompletionsHandler`
- [x] 3.2 Add unit tests for `ChatCompletionsFormatter` mapping logic
- [x] 3.3 Add unit tests for `ChatCompletionsHandler` payload parsing and `extra_body` extraction
