## 1. SSE Formatter Package (Non-Breaking)

- [x] 1.1 Create `internal/interfaces/http/sse/` package directory
- [x] 1.2 Define `StreamMeta` struct with model name, notebook ID, and history context fields
- [x] 1.3 Implement `ResponsesAPIFormatter` struct with `WriteResponse(w io.Writer, stream *schema.StreamReader[*schema.Message], meta *StreamMeta) error` method
- [x] 1.4 Implement SSE event lifecycle: `response.created`, `response.in_progress`, `output_item.added`, `content_part.added` preamble events
- [x] 1.5 Implement delta event loop: consume stream chunks and emit `response.output_text.delta` events
- [x] 1.6 Implement closing events: `output_text.done`, `content_part.done`, `output_item.done`, `response.completed`
- [x] 1.7 Implement error handling: emit `response.failed` event on stream error (before io.EOF)
- [x] 1.8 Extract `sendStreamingEvent()` helper that formats `data: <json>\n\n`
- [x] 1.9 Write unit test for full event lifecycle with 3 chunks verifying event sequence and IDs
- [x] 1.10 Write unit test for stream error mid-response verifying `response.failed` event
- [x] 1.11 Write unit test for sequence number incrementing across events

## 2. Simplify AgentStage (Stream-Only)

- [x] 2.1 Remove non-streaming branch from `AgentStage.Execute()` (lines 91-169)
- [x] 2.2 Remove `GenerationOutput.Response` field usage — only `Stream` is populated
- [x] 2.3 Simplify `event_mapper.go` to only produce `*schema.Message` for the stream reader
- [x] 2.4 Write unit test verifying AgentStage always returns a non-nil `StreamReader`
- [x] 2.5 Write unit test verifying AgentStage stream closes with `io.EOF` on completion

## 3. Stream Wrapper for History Save

- [x] 3.1 Define `historySavingReader` struct wrapping `*schema.StreamReader[*schema.Message]` with `onSave` callback
- [x] 3.2 Implement `Recv()` method that triggers `onSave` on first error (including `io.EOF`)
- [x] 3.3 Implement `Close()` method that triggers `onSave` and delegates to inner stream
- [x] 3.4 Add `sync.Once` or `sync.Mutex` to ensure `onSave` fires exactly once
- [x] 3.5 Write unit test verifying `onSave` is called on `io.EOF`
- [x] 3.6 Write unit test verifying `onSave` is called on stream error
- [x] 3.7 Write unit test verifying `onSave` is not called twice

## 4. Simplify Response Usecase (Breaking)

- [x] 4.1 Update `chat.ResponseUseCase` interface to single `Stream()` method returning `(*schema.StreamReader[*schema.Message], *StreamMeta, error)`
- [x] 4.2 Remove `CreateResponse()` method from `responseUseCase`
- [x] 4.3 Remove `CreateResponseStream()` method from `responseUseCase`
- [x] 4.4 Implement new `Stream()` method: validate, build catalog, run pipeline, wrap with `historySavingReader`
- [x] 4.5 Fix catalog bug: pass real source catalog to pipeline for streaming (not empty string)
- [x] 4.6 Remove all SSE formatting code from usecase (`sendStreamingEvent`, `streamResponseContent`, `handleStreamingError`)
- [x] 4.7 Remove `fmt.Printf("[DEBUG]")` debug print from `Stream()`
- [x] 4.8 Remove `fmt.Printf("[ERROR]")` from `streamResponseContent` if still present
- [x] 4.9 Remove unused imports (`encoding/json`, `strings`, etc.) from usecase
- [x] 4.10 Update pipeline `Execute()` to remove `req.Stream` branching and async save logic
- [x] 4.11 Write unit test for `Stream()` with valid request verifying raw stream returned
- [x] 4.12 Write unit test for `Stream()` with invalid notebook verifying error

## 5. Update Handler (Breaking)

- [x] 5.1 Remove `req.Stream` check and non-streaming branch from `ResponseHandler.CreateResponse()`
- [x] 5.2 Update handler to call `useCase.Stream()` instead of `CreateResponseStream()`
- [x] 5.3 Replace `io.Copy` stream copy with `sse.ResponsesAPIFormatter.WriteResponse()`
- [x] 5.4 Remove handler's `handleStream()` method (replaced by SSE formatter)
- [x] 5.5 Write unit test for handler verifying SSE `Content-Type` header is always set
- [x] 5.6 Write unit test for handler verifying full SSE event sequence in response body

## 6. Update DI Wiring

- [x] 6.1 Update `cmd/serve.go` if `NewResponseUseCase` constructor signature changed
- [x] 6.2 Verify `ToolFactory` is still correctly passed to usecase constructor
- [x] 6.3 Verify handler initialization uses updated interface

## 7. Delete Dead Code

- [x] 7.1 Delete `internal/core/application/usecases/response/stages/generation_stage.go`
- [x] 7.2 Delete `internal/core/application/usecases/response/stages/generation_stage_test.go`
- [x] 7.3 Clean up unused fields in `stages/types.go` (remove `GenerationOutput.Response` if no longer referenced)
- [x] 7.4 Remove unused `buildSourceCatalog` from non-streaming path if duplicated

## 8. Verification

- [x] 8.1 Run `make build` and verify binary compiles
- [x] 8.2 Run `make test` and ensure all tests pass
- [x] 8.3 Run `make lint` and fix any issues
