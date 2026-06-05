## 1. DTO Definitions

- [x] 1.1 Add `ResponseReasoningDeltaEvent` to `internal/core/application/dtos/chat.go`
- [x] 1.2 Add `ResponseReasoningDoneEvent` to `internal/core/application/dtos/chat.go`
- [x] 1.3 Add `ResponseFunctionCallArgumentsDeltaEvent` to `internal/core/application/dtos/chat.go`
- [x] 1.4 Add `ResponseFunctionCallArgumentsDoneEvent` to `internal/core/application/dtos/chat.go`

## 2. Thinking Tag Parser

- [x] 2.1 Implement `<think>` tag parser logic in `AgentStage.Execute` to extract reasoning content
- [x] 2.2 Update `processChunk` and related mapping logic to pass parsed reasoning via `ReasoningContent`

## 3. SSE Formatter Compliance

- [x] 3.1 Update `internal/interfaces/http/sse/formatter.go` to emit `response.reasoning.delta`/`done`
- [x] 3.2 Update `internal/interfaces/http/sse/formatter.go` to emit `response.function_call_arguments.delta`/`done`
- [x] 3.3 Modify `WriteResponse` to include token `usage` metadata in `response.completed` event
- [x] 3.4 Modify `WriteResponse` to send `data: [DONE]\n\n` as the final terminal event

## 4. Testing & Validation

- [x] 4.1 Run unit tests in `internal/interfaces/http/sse/formatter_test.go` and fix any breakages
- [x] 4.2 Run unit tests in `internal/core/application/usecases/response/stages/agent_stage_test.go`
- [x] 4.3 Verify overall compliance against `response.log` expectations
