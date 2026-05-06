## Context

**Current State**: Document ingestion is scattered across multiple usecases:

- `internal/core/application/usecases/source/usecase.go` (627 lines) — orchestrates ingestion
- `internal/core/application/usecases/knowledge/` — stores knowledge chunks
- `internal/core/application/usecases/sentence/` — stores sentence chunks
- `internal/core/application/usecases/image/` — stores image chunks
- `internal/core/application/usecases/extractor/` — Kreuzberg content extraction
- `internal/core/application/usecases/document/` — document parsing

The `source/usecase.go` has two duplicated methods:
- `processAsync` — parallel processing with goroutines
- `processSync` — sequential processing

Both implement the same 6-step pipeline:
1. Extract content from URL
2. Parse document into elements
3. Chunk elements
4. Generate embeddings
5. Store to database
6. Update source status

**Problem**: No single module owns the "ingestion" concept. Complexity is scattered across 7+ services. The pipeline is hard to test because logic lives in orchestration code, not in isolated units.

**Constraints**:
- Must preserve existing behavior (sync/async modes)
- Kreuzberg service API cannot change
- Database transactions must be preserved
- HTTP handlers must continue to work
- Limited existing tests (only `source/usecase_test.go`)

## Goals / Non-Goals

**Goals**:
1. Deep `IngestionPipeline` module with small interface: `Ingest(ctx, source) <-chan Progress`
2. Pipeline stages as adapters behind a seam
3. Streaming progress via channel (eliminate status polling)
4. Configurable parallelism per stage
5. Testable pipeline in isolation

**Non-Goals**:
1. Changing Kreuzberg service API
2. Modifying database schema
3. Adding new ingestion capabilities (e.g., new file formats)
4. Changing HTTP handler contracts

## Decisions

### 1. Pipeline Architecture: Stage Interface

**Decision**: Define a `Stage` interface that all pipeline stages implement:

```go
type Stage interface {
    Name() string
    Execute(ctx context.Context, input StageInput) (StageOutput, error)
}

type IngestionPipeline struct {
    stages    []Stage
    parallelism int
}

func (p *IngestionPipeline) Ingest(ctx context.Context, source *entities.Source) <-chan Progress {
    // ... stream progress
}
```

**Rationale**: Interface enables testing stages in isolation. Stages can be mocked or replaced without modifying pipeline logic.

**Alternatives Considered**:
- *Function-based pipeline*: Use functions instead of interfaces. Rejected: harder to mock, no type safety for stage inputs/outputs.
- *Actor model*: Each stage as a goroutine with channels. Rejected: over-engineering, harder to debug, potential goroutine leaks.

### 2. Progress Streaming: Read-Only Channel

**Decision**: Return read-only channel from `Ingest()` method. Send progress updates after each stage. Close channel on completion or failure.

```go
type Progress struct {
    Stage    string
    Status   string // "in_progress", "completed", "failed"
    Error    error
    Metadata map[string]any
}

func (p *IngestionPipeline) Ingest(ctx context.Context, source *entities.Source) <-chan Progress {
    progress := make(chan Progress, 10)
    go func() {
        defer close(progress)
        // execute stages, send updates
    }()
    return progress
}
```

**Rationale**: Channels are idiomatic Go for streaming. Buffered channel prevents blocking if caller is slow. Guaranteed close prevents goroutine leaks.

**Alternatives Considered**:
- *Callback function*: Pass callback for progress updates. Rejected: less flexible, caller must handle concurrency.
- *Polling*: Store progress in database, poll for updates. Rejected: adds latency, database load.

### 3. Stage Communication: Typed Inputs/Outputs

**Decision**: Each stage receives typed `StageInput` and returns `StageOutput`:

```go
type StageInput struct {
    Source      *entities.Source
    Content     []byte
    Elements    []element.Element
    Chunks      []kreuzberg.KreuzbergChunk
    Embeddings  []float64
    // ... stage-specific data
}

type StageOutput struct {
    Result      interface{}
    Progress    Progress
}
```

**Rationale**: Type safety prevents passing wrong data between stages. Compiler catches errors.

**Trade-off**: More boilerplate than `interface{}`, but worth it for safety.

### 4. Error Handling: Fail-Fast with Context

**Decision**: Pipeline stops on first stage error. Context cancellation propagates to all running goroutines.

```go
for _, stage := range p.stages {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        output, err := stage.Execute(ctx, input)
        if err != nil {
            progress <- Progress{Status: "failed", Error: err}
            return err
        }
    }
}
```

**Rationale**: Fail-fast prevents wasted work. Context cancellation ensures clean shutdown.

**Alternatives Considered**:
- *Continue on error*: Log errors but continue pipeline. Rejected: could store partial data, harder to reason about.
- *Retry mechanism*: Retry failed stages. Rejected: adds complexity, could be added later as decorator.

### 5. Parallelism: Per-Stage Configuration

**Decision**: Each stage configures its parallelism. Pipeline supports sequential or parallel stage execution.

```go
type StageConfig struct {
    Parallelism int
}

type IngestionPipeline struct {
    stages []Stage
    mode   ExecutionMode // Sequential or Parallel
}
```

**Rationale**: Different stages have different parallelism needs. Embedding is I/O bound (high parallelism), parsing is CPU bound (low parallelism).

**Trade-off**: More configuration surface. Default to sequential for safety.

### 6. Database Transactions: Stage-Level Control

**Decision**: Storage stage manages its own transactions. Other stages are stateless.

```go
func (s *StorageStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
    tx, err := s.db.Begin(ctx)
    if err != nil {
        return StageOutput{}, err
    }
    defer tx.Rollback(ctx)
    
    // ... store data
    
    if err := tx.Commit(ctx); err != nil {
        return StageOutput{}, err
    }
    return StageOutput{Result: ids}, nil
}
```

**Rationale**: Storage stage is the only stage with stateful operations. Keeping transaction logic local is simpler than coordinating across pipeline.

**Trade-off**: Cannot rollback across multiple stages. Acceptable because only storage stage modifies database.

### 7. Stage Adapters: Wrap Existing UseCases

**Decision**: Create thin adapter types that wrap existing usecases:

```go
type ExtractionStage struct {
    extractor extractor.ContentExtractor
}

func (s *ExtractionStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
    content, err := s.extractor.ExtractContent(ctx, input.Source.URL)
    return StageOutput{Result: content}, err
}
```

**Rationale**: Minimizes rewriting logic. Existing usecases continue to work during migration.

**Migration Path**:
1. Create adapters that delegate to existing usecases
2. Migrate logic into stages incrementally
3. Deprecate old usecases

## Risks / Trade-offs

### Risk 1: Goroutine Leaks
**Risk**: Pipeline goroutines may not exit if context is mishandled or channels block.
**Mitigation**: 
- Always use `defer close()` for progress channel
- Always check `<-ctx.Done()` in loops
- Use buffered channels to prevent blocking
- Add tests for context cancellation

### Risk 2: Partial Data on Failure
**Risk**: If pipeline fails mid-execution, some data may be stored but status not updated.
**Mitigation**: Storage stage uses transactions. Status update runs in defer block to ensure it executes.

### Risk 3: Stage Input/Output Evolution
**Risk**: Adding new fields to `StageInput` requires updating all stages.
**Mitigation**: Use accessor methods instead of direct field access. Start with minimal interface, evolve as needed.

### Risk 4: Testing Complexity
**Risk**: Pipeline has many moving parts; tests may become complex.
**Mitigation**: Test each stage in isolation. Use table-driven tests for pipeline scenarios. Mock stages for integration tests.

### Risk 5: Performance Regression
**Risk**: Channel overhead and stage indirection may slow down ingestion.
**Mitigation**: Benchmark before/after. Use buffered channels. Pool allocations in hot paths.

### Trade-off: Complexity vs. Locality
**Choice**: More code (adapters, interfaces) for better locality.
**Decision**: Accept complexity. The long-term benefit of having all pipeline logic in one place outweighs short-term code increase.

## Migration Plan

### Phase 1: Create Pipeline Module (Non-Breaking)
1. Create `internal/core/application/usecases/pipeline/` package
2. Define `Stage` interface and `IngestionPipeline` struct
3. Implement progress channel logic
4. Write unit tests for pipeline orchestration
5. **Risk**: Low (new code)

### Phase 2: Create Stage Adapters (Non-Breaking)
1. Create adapter types for each existing usecase
2. Implement `Execute()` methods that delegate to existing logic
3. Write unit tests for each adapter
4. **Risk**: Low (delegation preserves behavior)

### Phase 3: Migrate HTTP Handlers (Breaking)
1. Update `source` handler to use `IngestionPipeline`
2. Replace `processAsync`/`processSync` with `Ingest()`
3. Update progress handling to use channel
4. Deprecate old methods (add comments)
5. **Risk**: Medium (caller changes)

### Phase 4: Deprecate Old UseCases (Future)
1. Move logic from old usecases into stages
2. Delete `processAsync`/`processSync` methods
3. Update callers
4. **Risk**: Low (isolated to usecase package)

### Rollback Strategy
- Each phase is a separate commit
- Phase 1-2 can be reverted independently
- Phase 3 requires reverting handler changes (keep old methods during migration)

## Open Questions

1. **Should progress channel be buffered or unbuffered?**
   - **Decision**: Buffered (size 10)
   - **Reason**: Prevents blocking if caller is slow to consume

2. **Should pipeline support retrying failed stages?**
   - **Decision**: Not in initial design
   - **Reason**: Adds complexity; can be added as stage decorator later

3. **Should stages be able to skip execution?**
   - **Decision**: Yes, stages can return `Skip` status
   - **Reason**: Some stages may not apply to all sources (e.g., OCR for text files)

4. **How should pipeline handle large files?**
   - **Decision**: Stream content through stages, not load entirely in memory
   - **Open**: May need streaming API for `StageInput`

5. **Should we support dynamic stage registration?**
   - **Decision**: No, stages are fixed at construction time
   - **Reason**: Simpler, type-safe; dynamic registration adds little value
