## Context

**Current State**: Conversation persistence is tightly coupled to the application layer (`ResponseUseCase` → `HistorySavingReader` → `HistoryStage.Save`). This works for a single agent but isn't reusable across different agents. Response metadata (token usage, finish reason) is buried inside `response_message` JSONB, making analytics queries slow and complex.

**Constraints**: 
- Must work with Eino ADK framework (`github.com/cloudwego/eino/adk`)
- Must not break existing conversation storage format
- Must preserve semantic message structure (not streaming artifacts)
- Must support per-notebook conversation scoping

## Goals / Non-Goals

**Goals:**
- Create reusable ADK middleware for automatic conversation persistence
- Extract response metadata into separate database columns for efficient analytics
- Enable per-notebook conversation threading and retrieval
- Async saving with minimal overhead (log only on failure)

**Non-Goals:**
- Complex error handling (DLQ, retry logic - defer to later if needed)
- Semantic search over conversations (just raw history retrieval)
- Streaming artifact preservation (keep semantic structure)
- Cross-notebook conversations

## Decisions

### 1. Middleware Architecture

**Decision**: Use Eino ADK Handler interface with BeforeModel/AfterAgent hooks

**Rationale**: 
- Eino ADK already provides `adk.Handler` interface for middleware
- `BeforeModel` hook allows injecting conversation history before model sees input
- `AfterAgent` hook allows saving after response is complete
- Works with ANY agent using ADK Runner, not just RetrievalAgent

**Alternatives considered**:
- Application-level wrapper: Would be agent-specific, less reusable
- Separate service: Over-engineering, adds network hops

### 2. Metadata Column Strategy

**Decision**: Add 4 separate columns instead of keeping metadata in JSONB

**Rationale**:
- Enables fast analytics queries (`SUM(total_tokens)`, `GROUP BY finish_reason`)
- Allows indexing on finish_reason and token counts
- No JSONB parsing needed for common queries
- Small storage overhead (4 INT/TEXT columns vs nested JSONB)

**Trade-off**: Schema changes required, but one-time cost

### 3. Async Saving with Simple Error Handling

**Decision**: Save in background goroutine, log only on failure (no DLQ)

**Rationale**:
- User doesn't wait for DB save (better latency)
- Simple implementation (~150 lines vs ~500 lines with DLQ)
- Logs provide observability for debugging
- Can add DLQ later if logs show frequent failures

**Trade-off**: If save fails, conversation is lost but user experience isn't impacted (response already sent)

### 4. Namespace Strategy: Per-Notebook

**Decision**: Scope conversations by `notebook_id` (already exists in schema)

**Rationale**:
- Aligns with existing domain model (notebooks contain conversations)
- `notebook_id` column already indexed
- Simple WHERE clause for history retrieval

### 5. No Retry Logic (Initially)

**Decision**: Single save attempt with timeout, no retry

**Rationale**:
- Start simple, add complexity if logs show it's needed
- Transient DB errors are rare with connection pooling
- If retries are needed, they can be added in one function

## Risks / Trade-offs

**[Risk] Data loss if save fails**
- **Mitigation**: Logs provide visibility; can add DLQ later if logs show frequent failures
- **Monitoring**: Alert on error log spikes

**[Risk] Conversation not found on load**
- **Mitigation**: Graceful degradation (continue without history) + warning log
- **User impact**: Next request starts fresh, but conversation isn't lost

**[Risk] Metadata extraction depends on ADK event structure**
- **Mitigation**: Validate event structure in implementation, add nil checks
- **Fallback**: Set metadata fields to 0/empty string if extraction fails

**[Trade-off] Schema migration required**
- **Cost**: One-time migration to add 4 columns
- **Benefit**: Permanent query performance improvement

## Migration Plan

### Phase 1: Database Migration
```sql
ALTER TABLE conversations 
ADD COLUMN finish_reason TEXT,
ADD COLUMN prompt_tokens INT,
ADD COLUMN completion_tokens INT,
ADD COLUMN total_tokens INT;
```

### Phase 2: Entity & Repository Updates
- Add 4 fields to `entities.Conversation`
- Update `ConversationRepository.Save()` INSERT/UPDATE queries
- Update `NewConversation()` constructor

### Phase 3: Middleware Implementation
- Create `internal/adk/middleware/conversation_memory.go`
- Implement `Handle()`, `handleBeforeModel()`, `handleAfterAgent()`, `saveAsync()`
- Extract metadata from `adk.AgentEvent.MessageOutput.ResponseMeta`

### Phase 4: DI Wiring (Parallel)
```go
// In serve.go, keep both running temporarily
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent: agent,
    Handlers: []adk.Handler{
        middleware.NewConversationMemory(conversationRepo, logger),
    },
})
```

### Phase 5: Cleanup (After Verification)
- Remove `HistorySavingReader` from ResponseUseCase
- Simplify `HistoryStage` to read-only (Load method only)
- Delete old accumulation code

### Rollback Strategy
- Middleware is additive (doesn't break existing code)
- If issues: Remove handler from ADK Runner config
- Database migration is additive (safe to rollback)

## Open Questions

**None** - Design is straightforward based on existing codebase patterns.
