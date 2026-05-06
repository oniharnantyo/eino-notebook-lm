## Context

Currently, the retrieval agent is created with a static `AgentInstruction` constant. The agent receives scoped tools filtered by `SourceIDs`, but has no awareness of what those sources actually are. The agent must blindly guess search terms without knowing document titles, types, or content scope.

The `response_usecase` already receives `SourceIDs` in the request and resolves them to UUIDs, but doesn't fetch the actual source metadata. Tools are created via `ToolFactory.NewScopedTools()` which accepts `ScopeConfig{SourceIDs, SourceTypes, Tracker}`.

## Goals / Non-Goals

**Goals:**
- Agent receives source catalog (title, type, chunk count, status) in its system prompt
- Agent can query detailed source metadata via `list_sources` tool
- Instruction building happens in usecase layer, agent package remains domain-agnostic
- No breaking changes to tool scoping behavior

**Non-Goals:**
- Multi-turn source awareness (catalog is rebuilt per request)
- Source selection UI/UX changes
- Modifying existing tool behaviors (keyword_search, semantic_search, chunk_read)

## Decisions

### Instruction Building: Template String in Usecase

The agent package will export `BaseAgentInstruction` (const) and `response_usecase` will concatenate the catalog. This keeps the agent package decoupled from domain entities — it receives a final instruction string, not source objects.

**Alternative considered:** Agent package accepts `[]*entities.Source` and builds catalog internally. Rejected because it creates an unnecessary dependency from `agent` to `entities`.

**Format:**
```go
// agent/instruction.go
const BaseAgentInstruction = `You are a Retrieval Agent...`

// response_usecase.go
func buildSourceCatalogInstruction(sources []*entities.Source) string
```

### Source Fetching: Batch GetByIDs

`SourceRepository` needs a new method `GetByIDs(ctx, []uuid.UUID)` to fetch all selected sources in a single query. Current code resolves `SourceIDs` to UUIDs but doesn't look up metadata.

**Trade-off:** Adding repository method vs. iterating `GetByID`. Batch query is more efficient for N sources and matches existing `GetByNotebookID` patterns.

### Catalog Format: Concise Inline List

Catalog injected into prompt follows format:
```
## Available Sources

- [src_abc1] "Paper Title" (PDF, 15 chunks) [processing]
- [src_abc2] "Meeting Notes" (markdown, 3 chunks)

Total: 2 sources, 18 chunks
```

Status indicators are minimal: `[processing]` or `[failed]` only when applicable. This keeps prompt overhead low while informing the agent about unavailable content.

### `list_sources` Tool: Full Metadata Exposure

Tool returns complete source details including URI, status, errors, and metadata. This enables the agent to diagnose why a source might be empty or report processing issues to the user.

**Tool schema:**
```go
type ListSourcesOutput struct {
    Sources []SourceDetail `json:"sources"`
}

type SourceDetail struct {
    ID          string                 `json:"id"`
    Title       string                 `json:"title"`
    ContentType string                 `json:"content_type"`
    ChunkCount  int                    `json:"chunk_count"`
    Status      string                 `json:"status"`
    URI         string                 `json:"uri,omitempty"`
    Error       string                 `json:"error,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

Tool is scoped by `ScopeConfig` — only returns sources within the user's selection.

### Error Handling: Silent Omission

Sources that fail to load or are deleted are omitted from the catalog rather than causing agent creation to fail. The agent can still operate with the remaining sources. `list_sources` tool will include failed sources with error details if the user asks.

### Constructor Changes: Minimal Signature Shift

`NewRetrievalAgent` gains `instruction string` parameter. Single call site in `response_usecase.go` updates to pass built instruction.

```go
// Before
func NewRetrievalAgent(ctx, model, tools) (*Runner, error)

// After
func NewRetrievalAgent(ctx, model, tools, instruction) (*Runner, error)
```

`NewResponseUseCase` gains `sourceRepo repositories.SourceRepository` parameter.

## Risks / Trade-offs

### Prompt Token Bloat
**Risk:** Large source catalogs (50+ documents) consume significant prompt tokens on every request.

**Mitigation:** User selects sources per query, so typical scope is 1-5 documents. Catalog is ~50 tokens per source. For notebooks with many sources, UI already filters selection.

### Source Fetch Latency
**Risk:** Fetching source metadata adds a database query before agent creation.

**Mitigation:** Batch `GetByIDs` is single round-trip. Sources are indexed and query is fast (PK lookup). If problematic, could cache source metadata in `Notebook` entity.

### Stale Catalog in Multi-Turn
**Risk:** Source catalog reflects state at request time, not mid-conversation updates.

**Mitigation:** Acceptable for current UX — each response rebuilds catalog. If sources change during conversation, user can re-select. Future work could support incremental updates.

### Instruction Versioning
**Risk:** `BaseAgentInstruction` and catalog format are coupled. Changes to instruction structure require coordinated updates.

**Mitigation:** Keep catalog format stable and append-only. Use clear section headers (`## Available Sources`) that won't conflict with future instruction additions.

## Migration Plan

1. Add `GetByIDs` method to `SourceRepository` interface and implementation
2. Update `NewRetrievalAgent` signature to accept `instruction string`
3. Add `SourceRepository` to `NewResponseUseCase` constructor
4. Create `list_sources.go` tool file
5. Update `ToolFactory` to accept `sourceRepo` and create `list_sources` tool
6. Add `buildSourceCatalogInstruction` function in `response_usecase.go`
7. Update `CreateResponseStream` to fetch sources and build instruction
8. Update `cmd/serve.go` wiring to pass `sourceRepo` to usecase
9. Test with varied source selections (empty, single, many, mixed statuses)

**Rollback:** Revert constructor changes and `AgentInstruction` can remain static. Tools silently ignored if not registered.

## Open Questions

None — design is self-contained within existing architecture patterns.