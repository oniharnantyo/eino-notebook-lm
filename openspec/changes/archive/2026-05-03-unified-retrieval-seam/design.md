## Context

**Current State**: The retrieval layer in `pkg/retriever/pgvector/` has three separate retriever implementations:

- `KnowledgesRetriever` (120 lines) - queries `knowledges` table
- `SentencesRetriever` (70 lines) - queries `sentences` table with potential JOINs
- `ImagesRetriever` (67 lines) - queries `images` table

Each implements identical logic for semantic search (vector similarity), keyword search (BM25), and hybrid retrieval (RRF fusion). The only difference is the table name in SQL queries.

**Problem**: Callers in `response_usecase.go` must use type assertions to access specific retrievers:

```go
pgRetriever, ok := uc.retriever.(*pgvector.Retriever)
if !ok {
    return nil, fmt.Errorf("retriever must be *pgvector.Retriever")
}
```

This violates the dependency injection principle and makes testing difficult. Adding a new retriever type requires duplicating ~120 lines.

**Constraints**:
- Must maintain backward compatibility for existing callers
- Database schema cannot change (knowledges/sentences/images tables remain)
- RRF fusion algorithm must produce identical results
- Tests are limited (currently only `retriever_test.go` exists)

## Goals / Non-Goals

**Goals:**
1. Single `Retriever` module with unified interface for all table types
2. Table configuration via `TableConfig` struct (name, BM25 index, optional JOIN)
3. Thin adapter types for backward compatibility
4. Remove type assertions from `response_usecase.go`
5. Enable testing retrieval through a single interface

**Non-Goals:**
1. Changing external APIs or HTTP handlers
2. Modifying database schema or indexes
3. Changing the RRF fusion algorithm
4. Adding new retrieval capabilities (e.g., hybrid search parameters)

## Decisions

### 1. Unified Retriever Module Structure

**Decision**: Create a single `Retriever` struct with an internal `tables` map.

```go
type Retriever struct {
    config *Config
    tables map[string]TableConfig
}

type TableConfig struct {
    Name       string // "knowledges", "sentences", "images"
    BM25Index  string // "knowledges_bm25_idx", etc.
    JoinClause string // Optional JOIN for sentences
}
```

**Rationale**: A map-based configuration allows easy addition of new table types without code changes. The `Retriever` interface becomes simple: `Retrieve(tableType string, ...)` instead of three separate types.

**Alternatives Considered**:
- *Interface-based*: Define `Retriever` interface with three implementations. Rejected: duplicates logic, doesn't solve type assertion problem.
- *Database metadata*: Load table config from database. Rejected: over-engineering, adds startup latency.

### 2. Adapter Pattern for Backward Compatibility

**Decision**: Keep existing `KnowledgesRetriever`, `SentencesRetriever`, `ImagesRetriever` as thin wrappers.

```go
type KnowledgesRetriever struct {
    inner *Retriever
}

func (r *KnowledgesRetriever) Retrieve(...) {
    return r.inner.Retrieve("knowledges", ...)
}
```

**Rationale**: Zero breaking changes for callers. Existing code continues to work without modification. Adapters can be removed incrementally.

**Trade-off**: Adds ~20 lines per adapter, but enables safe migration.

### 3. Table-Specific JOIN Handling

**Decision**: `TableConfig.JoinClause` stores optional SQL fragments for queries requiring JOINs (e.g., sentences → knowledges).

**Rationale**: Sentences retrieval may need parent knowledge metadata. Hardcoding JOIN in the unified query builder is inflexible; storing it in config allows per-table customization.

**Alternatives Considered**:
- *Post-processing*: Fetch sentence data, then query knowledge separately. Rejected: N+1 query problem.
- *View*: Create database view joining sentences/knowledges. Rejected: schema change required.

### 4. Error Handling Strategy

**Decision**: Validate table configuration at construction time (`NewRetriever`). Unknown table types return error immediately, not at query time.

**Rationale**: Fail-fast prevents runtime surprises. Matches existing pattern (e.g., dimension validation in constructors).

**Trade-off**: Less dynamic than runtime validation, but more predictable.

### 5. Testing Approach (TDD)

**Decision**: Write tests before implementation following the spec scenarios. Use table-driven tests for multiple table types.

**Rationale**: The spec has 6 requirements with 14+ scenarios. TDD ensures the implementation satisfies all requirements before refactoring existing code.

**Test Structure**:
```go
func TestRetriever_SemanticSearch(t *testing.T) {
    tests := []struct {
        name      string
        tableType string
        // ... fields
    }{
        {"knowledges table", "knowledges", ...},
        {"sentences table", "sentences", ...},
        {"images table", "images", ...},
    }
    // ...
}
```

## Risks / Trade-offs

### Risk 1: JOIN Performance for Sentences
**Risk**: Sentences with JOIN clause may be slower than current implementation.
**Mitigation**: Benchmark before/after. If performance degrades, make JOIN optional via config flag.

### Risk 2: Adapter Confusion
**Risk**: Developers may accidentally use adapters directly instead of unified `Retriever`.
**Mitigation**: Add documentation comments directing to `Retriever`. Deprecate adapter constructors in future release.

### Risk 3: Limited Test Coverage
**Risk**: Current codebase has minimal test coverage. Changes may break untested edge cases.
**Mitigation**: TDD approach ensures new code is tested. Run existing tests to verify no regressions.

### Risk 4: RRF Fusion Correctness
**Risk**: Consolidating RRF logic may introduce subtle bugs in score calculation.
**Mitigation**: Extract existing RRF functions (`MergeByRRF`, `SortByScore`, `TopN`) unchanged. Add unit tests for fusion logic.

### Trade-off: Code Location
**Choice**: Keep unified `Retriever` in existing files vs. new `unified.go`.
**Decision**: Create new `unified.go` to minimize merge conflicts.
**Trade-off**: Additional file, but clearer separation during migration.

## Migration Plan

### Phase 1: Create Unified Module (Non-Breaking)
1. Create `unified.go` with `Retriever` and `TableConfig`
2. Implement semantic search, keyword search, hybrid retrieval
3. Write tests covering all spec scenarios
4. **Risk**: Low (new code, isolated)

### Phase 2: Migrate Adapters (Non-Breaking)
1. Refactor existing retrievers to embed `Retriever`
2. Delegate methods to unified implementation
3. Verify existing tests still pass
4. **Risk**: Low (delegation preserves behavior)

### Phase 3: Update Callers (Breaking)
1. Update `response_usecase.go` to use `Retriever` directly
2. Remove type assertions
3. Update constructor injection
4. Deprecate adapter types (add comments, not removal)
5. **Risk**: Medium (caller changes required)

### Rollback Strategy
- Each phase is a separate commit
- Phase 1-2 can be reverted independently
- Phase 3 requires updating callers back to adapters

## Open Questions

1. **Should `TableConfig` be publicly exported?**
   - **Decision**: Yes, allows callers to register custom tables without modifying code
   - **Resolved**: Export as `TableConfig`

2. **Should we support dynamic table registration at runtime?**
   - **Decision**: No, validate at construction time for fail-fast behavior
   - **Resolved**: Tables registered via constructor only

3. **Should JOIN results include all columns or specific ones?**
   - **Decision**: Keep existing behavior (fetch all columns via *)
   - **Open**: May need optimization later (projection)

4. **What happens if BM25 index doesn't exist for a table?**
   - **Decision**: Query fails with database error (same as current behavior)
   - **Open**: Could add validation at construction if schema introspection is acceptable