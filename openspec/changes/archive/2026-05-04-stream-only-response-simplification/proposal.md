## Why

The response usecase mixes business logic (pipeline orchestration, history management) with presentation logic (SSE event formatting). It maintains two code paths (streaming and non-streaming) through the pipeline and AgentStage, but only streaming is used in practice. Dead code (`generation_stage.go`) and debug prints remain from the migration to agent-based generation. The `CreateResponseStream` path has a bug — it passes an empty catalog to the pipeline while `CreateResponse` passes the real one.

## What Changes

- **Remove `CreateResponse()` method** — stream-only response generation
- **Extract SSE formatting** (~166 lines) from usecase to `internal/interfaces/http/sse/` package (Responses-API-specific)
- **Delete dead code** — `generation_stage.go` and `generation_stage_test.go` (replaced by AgentStage)
- **Remove debug prints** — `fmt.Printf("[DEBUG]")` and `fmt.Printf("[ERROR]")` from production code
- **Fix catalog bug** — streaming path passes empty catalog instead of source data
- **Simplify `AgentStage`** — remove non-streaming branch (~80 lines)
- **Wrap stream for history save** — use `onClose` callback pattern instead of pipeline-level branching on `req.Stream`

**BREAKING**: `chat.ResponseUseCase` interface changes from two methods to one: `Stream(ctx, req) (StreamReader, error)`. HTTP handler no longer supports non-streaming JSON responses.

## Capabilities

### New Capabilities
- `response-sse-formatting`: Responses-API-specific SSE event formatting extracted from usecase to interface layer

### Modified Capabilities
- `agent-source-awareness`: Streaming path now receives source catalog (was empty string — bug fix)

## Impact

**Affected Files:**
- `internal/core/application/usecases/response/response_usecase.go` — Remove CreateResponse, remove SSE formatting, simplify to ~100 lines
- `internal/core/application/usecases/response/pipeline.go` — Remove stream/non-stream branching, remove async save logic
- `internal/core/application/usecases/response/stages/agent_stage.go` — Remove non-streaming branch
- `internal/core/application/usecases/response/stages/generation_stage.go` — **DELETE**
- `internal/core/application/usecases/response/stages/generation_stage_test.go` — **DELETE**
- `internal/core/application/usecases/response/stages/types.go` — Remove `GenerationOutput.Response` field
- `internal/core/application/usecases/chat/usecase.go` — Interface changes to single `Stream()` method
- `internal/interfaces/http/handlers/response.go` — Simplify to stream-only, use SSE formatter
- `internal/interfaces/http/sse/formatter.go` — **NEW** SSE formatting extracted from usecase
- `cmd/serve.go` — Update DI if constructor signature changes

**No new dependencies.**
