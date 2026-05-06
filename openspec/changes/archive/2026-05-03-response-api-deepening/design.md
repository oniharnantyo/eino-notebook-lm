## Context

**Current State**: The response usecase (`internal/core/application/usecases/response/response_usecase.go`) is a 965-line monolith that orchestrates:

- Agent creation (`agent.NewRetrievalAgent`)
- Tool factory initialization (`tools.NewToolFactory`)
- Three separate retrievers (knowledge, sentence, image)
- Chat model invocation via factory
- History management (`history_manager.go`)
- Repository access (knowledge, source, conversation, notebook)
- Streaming response handling

**Problem**: Understanding how a single response is generated requires bouncing between 8+ modules. The code is untested (no test file exists). Adding features or fixing bugs requires navigating complex control flow with nested conditionals.

**Constraints**:
- HTTP API must remain stable (external contract unchanged)
- Eino framework abstractions must be preserved
- Existing behavior (streaming, tools, retrieval modes) must work
- Limited test coverage to reference

## Goals / Non-Goals

**Goals**:
1. Split 965-line usecase into focused stages (~100-150 lines each)
2. Each stage independently testable with mockable dependencies
3. Clear data flow between stages (input → output)
4. Reduce cognitive load for understanding response generation
5. Enable adding tests for core orchestration logic

**Non-Goals**:
1. Changing HTTP API contracts
2. Modifying Eino framework integration
3. Changing retrieval implementations (deferred to unified-retrieval-seam)
4. Adding new features (focus is simplification)

## Decisions

### 1. Stage-Based Pipeline Architecture

**Decision**: Implement response generation as a pipeline with 4 stages:

```go
type Stage interface {
    Execute(ctx context.Context, input StageInput) (StageOutput, error)
}

type ResponsePipeline struct {
    retrievalStage   *RetrievalStage
    toolStage        *ToolPreparationStage
    generationStage  *GenerationStage
    historyStage     *HistoryStage
}
```

**Rationale**: Clear separation of concerns. Each stage has single responsibility. Stages can be tested in isolation. Pipeline orchestration is trivial.

**Alternatives Considered**:
- *Function-based refactoring*: Extract functions from monolith. Rejected: Still hard to test, no clear boundaries.
- *Actor model*: Each stage as goroutine with channels. Rejected: Over-engineering, harder to debug.

### 2. Stage Communication via Typed Structs

**Decision**: Define typed input/output for each stage:

```go
type RetrievalInput struct {
    Query        string
    QueryVector  []float64
    RetrievalConfig RetrievalConfig
}

type RetrievalOutput struct {
    Context      []*schema.Document
    Scores       map[string]float64
}
```

**Rationale**: Type safety prevents passing wrong data. Compiler catches errors. Self-documenting.

**Trade-off**: More boilerplate than `interface{}`, but worth it for safety and clarity.

### 3. History Stage Positioning

**Decision**: History stage runs twice—before retrieval (load) and after generation (save).

```go
func (p *ResponsePipeline) Execute(ctx context.Context, req Request) (<-chan ResponseChunk, error) {
    // Load history
    history, err := p.historyStage.Load(ctx, req.ConversationID)
    
    // Retrieve with history
    context, err := p.retrievalStage.Execute(ctx, RetrievalInput{History: history})
    
    // ... generate ...
    
    // Save history (in goroutine)
    go p.historyStage.Save(ctx, req, response)
}
```

**Rationale**: History is needed for retrieval context but must be saved after generation. Running save in goroutine avoids blocking response streaming.

**Alternatives Considered**:
- *Single history stage*: Load and save in one stage. Rejected: Violates single responsibility, stage would need to wait for generation.
- *History in pipeline wrapper*: Handle history outside pipeline. Rejected: History is part of response logic, should be visible in pipeline.

### 4. Streaming Response Handling

**Decision**: Generation stage returns read-only channel for streaming. Pipeline wraps this channel and adds metadata.

```go
func (s *GenerationStage) Execute(ctx context.Context, input GenerationInput) (<-chan Chunk, error) {
    stream, err := s.chatModel.Stream(ctx, input.Messages)
    // ... wrap and return
}
```

**Rationale**: Channels are idiomatic Go for streaming. Preserves Eino framework streaming contract.

### 5. Error Propagation

**Decision**: Each stage returns error. Pipeline stops on first error and returns it to caller.

```go
func (p *ResponsePipeline) Execute(ctx context.Context, req Request) (<-chan ResponseChunk, error) {
    context, err := p.retrievalStage.Execute(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("retrieval stage: %w", err)
    }
    // ... continue
}
```

**Rationale**: Fail-fast is appropriate for response generation. Partial responses are not useful. Error wrapping preserves context.

**Alternatives Considered**:
- *Continue on error*: Log error but continue pipeline. Rejected: Could produce garbage responses.
- *Error channel*: Send errors on response channel. Rejected: Mixing errors and responses complicates client handling.

### 6. Dependency Injection

**Decision**: Pipeline and stages receive dependencies via constructor. All dependencies are interfaces.

```go
func NewResponsePipeline(
    retriever Retriever,
    chatModel ChatModel,
    historyMgr HistoryManager,
    toolFactory ToolFactory,
) *ResponsePipeline
```

**Rationale**: Enables testing with mocks. Follows project convention (dependencies injected via constructors, never nil-checked).

### 7. Agent/Tool Simplification

**Decision**: Move agent/tool factory logic into `ToolPreparationStage`. Remove `agent` package indirection.

**Current**: `response_usecase` → `agent.NewRetrievalAgent` → `tools.NewToolFactory` → tools
**Proposed**: `response_usecase` → `ToolPreparationStage` → tools

**Rationale**: Agent package adds no value—it's just a wrapper around tool factory. Removing it reduces complexity.

**Trade-off**: Less abstraction, but the abstraction wasn't earning its keep (deletion test).

## Risks / Trade-offs

### Risk 1: Breaking Streaming Behavior
**Risk**: Refactoring might break streaming responses.
**Mitigation**: Preserve Eino streaming contract. Add integration test for streaming before refactoring. Compare responses before/after.

### Risk 2: Performance Regression
**Risk**: Additional stage boundaries might add overhead.
**Mitigation**: Stages are thin wrappers, no significant allocation. Benchmark before/after. Hot path remains chat model invocation.

### Risk 3: History Race Conditions
**Risk**: Saving history in goroutine might race with next request.
**Mitigation**: Use database-level constraints (conversation ID). History save is idempotent.

### Risk 4: Test Coverage Gaps
**Risk**: Existing code has no tests; hard to verify refactoring preserves behavior.
**Mitigation**: Add integration test for full pipeline first. Use this as regression test during refactoring.

### Trade-off: More Files vs. Clearer Code
**Choice**: Split into multiple files (one per stage) vs. keep in one file.
**Decision**: Split into stages. Each stage in its own file.
**Reason**: Clearer boundaries, easier to navigate, enables parallel work.

## Migration Plan

### Phase 0: Baseline Test (Safety Net)
1. Add integration test for current response flow
2. Test streaming, tools, history, retrieval modes
3. **Risk**: Low (test only)

### Phase 1: Extract Stage Interfaces (Non-Breaking)
1. Create `internal/core/application/usecases/response/stages/` package
2. Define stage interfaces and input/output types
3. Create empty stage implementations
4. **Risk**: Low (new code)

### Phase 2: Implement Stages (Non-Breaking)
1. Move retrieval logic into `RetrievalStage`
2. Move tool logic into `ToolPreparationStage`
3. Move generation logic into `GenerationStage`
4. Move history logic into `HistoryStage`
5. Write unit tests for each stage
6. **Risk**: Medium (moving code)

### Phase 3: Replace Orchestration (Breaking)
1. Create `ResponsePipeline` that wires stages together
2. Update `response_usecase.go` to use pipeline
3. Delete old orchestration code
4. Verify baseline test still passes
5. **Risk**: Medium (orchestration changes)

### Phase 4: Cleanup
1. Delete `agent` package (if unused)
2. Remove deprecated code
3. Update documentation
4. **Risk**: Low (cleanup)

### Rollback Strategy
- Each phase is a separate commit
- Baseline test ensures behavior preserved
- Phase 1-2 can be reverted independently
- Phase 3 requires reverting orchestration (keep old code during migration)

## Open Questions

1. **Should history save wait for completion or run in background?**
   - **Decision**: Run in goroutine (non-blocking)
   - **Reason**: History save is not critical for response; blocking would add latency

2. **Should stages support middleware (e.g., logging, metrics)?**
   - **Decision**: Not in initial design
   - **Reason**: Keep it simple. Can add stage decorators later if needed.

3. **Should pipeline support parallel stage execution?**
   - **Decision**: No, stages are sequential
   - **Reason**: Stages have dependencies (retrieval → tools → generation). Parallelism adds complexity for little gain.

4. **How to handle tool calls during generation?**
   - **Decision**: Generation stage manages tool calls via Eino framework
   - **Reason**: Tool calling is part of chat model behavior, not a separate stage

5. **Should we keep the agent package for backwards compatibility?**
   - **Decision**: No, delete if unused after refactoring
   - **Reason**: Dead code has no value. Can restore from git if needed.
