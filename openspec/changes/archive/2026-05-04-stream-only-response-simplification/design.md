## Context

**Current State**: `response_usecase.go` (382 lines) delegates to a 3-stage pipeline (ToolPrep → Agent → History). However, it maintains two code paths — `CreateResponse` (non-streaming) and `CreateResponseStream` (streaming) — even though the HTTP handler only meaningfully exercises streaming. The SSE formatting (~166 lines) is embedded in the usecase, mixing presentation logic with business logic. Dead code (`generation_stage.go`) and debug prints remain from prior migrations. The streaming path has a catalog bug — it passes empty string instead of source data.

**Constraints**:
- OpenAI Responses API contract must remain stable (always stream SSE)
- Eino framework abstractions (`schema.StreamReader`, `model.ToolCallingChatModel`) preserved
- Pipeline stage pattern established in prior refactoring must be maintained

## Goals / Non-Goals

**Goals**:
1. Remove non-streaming code path — single execution mode through pipeline
2. Extract SSE formatting to interface layer — usecase returns raw stream
3. Delete dead code and debug prints
4. Fix catalog bug in streaming path
5. Reduce `response_usecase.go` to ~100 lines of orchestration

**Non-Goals**:
1. Changing the pipeline stage interface or adding new stages
2. Modifying AgentStage's agent logic or tool behavior
3. Changing the HTTP endpoint path or request/response DTOs
4. Adding new features — this is purely simplification

## Decisions

### 1. Stream-Only Interface

**Decision**: Replace two-method `ResponseUseCase` interface with single `Stream()` method returning `*schema.StreamReader[*schema.Message]`.

```go
// Before
type ResponseUseCase interface {
    CreateResponse(ctx, req) (*ResponseResource, error)
    CreateResponseStream(ctx, req) (io.ReadCloser, error)
}

// After
type ResponseUseCase interface {
    Stream(ctx context.Context, req *dtos.ResponseRequest) (*schema.StreamReader[*schema.Message], *StreamMeta, error)
}
```

`StreamMeta` carries metadata the SSE formatter needs (model name, catalog, notebook ID) without the usecase knowing about SSE.

**Rationale**: Dual-path interface forces every consumer to handle branching. A single return type eliminates this. The non-streaming JSON response can be achieved by consuming the stream and serializing — but we don't need it since the API is SSE-only.

**Alternatives Considered**:
- *Keep both, make CreateResponse call Stream internally*: Adds indirection without removing the branching.
- *Return interface{} from single method*: Loses type safety.

### 2. SSE Formatter Package Location

**Decision**: Create `internal/interfaces/http/sse/` package with Responses-API-specific formatting.

```go
package sse

type ResponsesAPIFormatter struct{}

func (f *ResponsesAPIFormatter) WriteResponse(w io.Writer, stream *schema.StreamReader[*schema.Message], meta *StreamMeta) error
```

**Rationale**: SSE formatting is presentation logic. In Clean Architecture, presentation belongs in the interfaces layer. The usecase (application layer) should not import `encoding/json` for SSE events or know about `response.output_text.delta` event types.

**Alternatives Considered**:
- *Keep SSE in usecase, just refactor*: Doesn't solve the layering violation.
- *Generic SSE package*: Over-engineering — we only have one SSE format (OpenAI Responses API).

### 3. History Save via Stream Wrapper

**Decision**: Wrap the returned `StreamReader` with a `historySavingReader` that calls `historyStage.Save` on stream close.

```go
type historySavingReader struct {
    inner  *schema.StreamReader[*schema.Message]
    onSave func()
    saved  bool
    mu     sync.Mutex
}

func (r *historySavingReader) Recv() (*schema.Message, error) {
    msg, err := r.inner.Recv()
    if err != nil {
        r.triggerSave()
        return msg, err
    }
    return msg, nil
}

func (r *historySavingReader) Close() error {
    r.triggerSave()
    return r.inner.Close()
}
```

**Rationale**: Currently history saves only for non-streaming (`if !req.Stream`). In stream-only mode, we need to save after stream completes. Wrapping the reader keeps the save logic co-located with the stream lifecycle without coupling the pipeline to history concerns.

**Alternatives Considered**:
- *Callback in SSE formatter*: Couples formatter to business logic.
- *Channel-based signal*: More complex, no clearer than wrapper.
- *Pipeline handles save after Execute*: Pipeline doesn't control when stream is consumed.

### 4. Simplify AgentStage — Remove Non-Streaming Branch

**Decision**: AgentStage.Execute always enables streaming and returns `GenerationOutput` with only the `Stream` field populated.

**Rationale**: The non-streaming branch (lines 91-169, ~80 lines) accumulates content server-side, which defeats the purpose of streaming. If a non-streaming response is ever needed, the consumer can drain the stream.

**Trade-off**: Removes the ability to get a fully-formed `ResponseResource` from AgentStage. The SSE formatter will construct the response object from stream content instead.

### 5. Delete GenerationStage Files

**Decision**: Delete `generation_stage.go` and `generation_stage_test.go`. `AgentStage` fully replaced `GenerationStage`.

**Rationale**: `GenerationStage` was superseded by `AgentStage` during the Eino ADK integration. No code references it. Dead code has no value.

## Risks / Trade-offs

### Risk 1: Breaking Change to ResponseUseCase Interface
**Risk**: Any code depending on `CreateResponse()` will fail to compile.
**Mitigation**: Only one implementation and one consumer exist (handler). Both are updated in the same change. No external consumers.

### Risk 2: History Save Timing
**Risk**: If stream consumer disconnects before draining, history may save with partial content.
**Mitigation**: `historySavingReader` triggers save on both `Recv()` error (including `io.EOF`) and `Close()`. Partial history is better than no history. History is non-critical — used for context, not billing.

### Risk 3: SSE Event Ordering Regression
**Risk**: Moving SSE formatting to a new package may change event sequence or format.
**Mitigation**: The formatter produces the exact same event types and order — the code moves, not the logic. Integration test validates event sequence.

## Migration Plan

### Phase 1: Create SSE Package (Non-Breaking)
1. Create `internal/interfaces/http/sse/` package
2. Move SSE event formatting logic from usecase
3. Add test for event sequence
4. **Risk**: Low (new code, no consumers yet)

### Phase 2: Simplify Usecase (Breaking)
1. Update `chat.ResponseUseCase` interface to single `Stream()` method
2. Remove `CreateResponse()` from usecase
3. Replace SSE formatting with raw stream return
4. Add `historySavingReader` wrapper
5. Fix catalog bug in streaming path
6. **Risk**: Medium (interface change, but single consumer)

### Phase 3: Update Handler (Breaking)
1. Update handler to use SSE formatter
2. Remove non-streaming handler branch
3. **Risk**: Low (handler is thin, follows usecase change)

### Phase 4: Cleanup
1. Delete `generation_stage.go` and test
2. Remove debug prints
3. Remove unused imports and fields from types.go
4. Run `make lint && make test`
5. **Risk**: Low (dead code removal)

### Rollback Strategy
- Each phase is a separate commit
- Phase 1 is independently revertible
- Phases 2-3 must be reverted together (interface change + consumer)
- Phase 4 is independently revertible
