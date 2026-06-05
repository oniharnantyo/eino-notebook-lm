## 1. StreamEvent Types

- [x] 1.1 Define `StreamEvent` interface and concrete event types (`TextDeltaEvent`, `ToolCallEvent`, `ReasoningEvent`, `UsageEvent`) in `internal/interfaces/http/sse/types.go`
- [x] 1.2 Add `FunctionCall` item type implementing `ItemField` in `internal/core/application/dtos/chat.go`
- [x] 1.3 Add `Reasoning` item type implementing `ItemField` and `SummaryTextContent` implementing `ContentPart` in `internal/core/application/dtos/chat.go`
- [x] 1.4 Add streaming event DTOs for function_call lifecycle (`response.function_call_arguments.delta`, etc.) in `internal/core/application/dtos/chat.go`
- [x] 1.5 Add streaming event DTOs for reasoning lifecycle in `internal/core/application/dtos/chat.go`

## 2. Agent Stage Refactor

- [x] 2.1 Change `AgentStage.Execute` to emit `StreamEvent` values through the pipe instead of raw `*schema.Message`, detecting tool calls from `msg.ToolCalls` and reasoning from `msg.ReasoningContent`
- [x] 2.2 Extract token usage from the final message's `ResponseMeta.Usage` and emit a `UsageEvent` before closing the pipe
- [x] 2.3 Update `GenerationOutput` to use `*schema.StreamReader[StreamEvent]` instead of `*schema.StreamReader[*schema.Message]`

## 3. Response UseCase Wiring

- [x] 3.1 Update `response_usecase.go` to pass `StreamEvent` reader to the formatter and carry usage metadata in `StreamMeta`
- [x] 3.2 Update `HistorySavingReader` to work with the new `StreamEvent` stream, accumulating text from `TextDeltaEvent` for history saving

## 4. Formatter Enrichment

- [x] 4.1 Add `data: [DONE]\n\n` terminal event after `response.completed` in `formatter.go`
- [x] 4.2 Refactor `WriteResponse` to type-switch on `StreamEvent` and dispatch to handler methods per event type
- [x] 4.3 Implement tool call SSE emission: emit `response.output_item.added` (function_call, in_progress) → `response.output_item.done` (function_call, completed) for each `ToolCallEvent`
- [x] 4.4 Implement reasoning SSE emission: emit full lifecycle (output_item.added → content_part.added → output_text.delta → output_text.done → content_part.done → output_item.done) for `ReasoningEvent`
- [x] 4.5 Populate `usage` field in `response.completed` from `UsageEvent` data

## 5. Tests

- [x] 5.1 Add unit test for `data: [DONE]` terminal event in `formatter_test.go`
- [x] 5.2 Add unit test for tool call SSE event emission
- [x] 5.3 Add unit test for reasoning SSE event emission
- [x] 5.4 Add unit test for usage metadata in `response.completed`
- [x] 5.5 Verify existing handler test still passes after type changes
