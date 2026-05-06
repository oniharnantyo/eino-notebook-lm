## Why

The retrieval agent operates without awareness of what documents it's searching. Users select sources for each query, but the agent receives no information about those sources — it must blindly guess keywords and search terms. This leads to inefficient queries and missed information.

## What Changes

- **Dynamic agent instruction**: `NewRetrievalAgent` accepts an `instruction string` parameter instead of using a static const
- **Source catalog injection**: `response_usecase` fetches selected sources and builds a catalog string (title, type, chunk count) that's injected into the agent's system prompt
- **`list_sources` tool**: New agent tool that returns full source metadata (URI, status, errors) for the selected scope
- **SourceRepository injection**: `response_usecase` gains `SourceRepository` dependency to fetch source metadata

## Capabilities

### New Capabilities
- `agent-source-awareness`: Retrieval agent receives context about available sources before searching, including a catalog in the system prompt and a tool for detailed metadata

### Modified Capabilities
- None (implementation-level change only)

## Impact

**Affected Files:**
- `internal/core/application/agent/agent.go` — `NewRetrievalAgent` signature adds `instruction string` parameter
- `internal/core/application/agent/instruction.go` — `AgentInstruction` const becomes base template
- `internal/core/application/usecases/response/response_usecase.go` — Adds `SourceRepository` dependency, source fetching, and instruction building logic
- `internal/core/application/agent/tools/factory.go` — Adds `list_sources` tool creation
- `internal/core/application/agent/tools/list_sources.go` — New file

**New Dependencies:**
- `response_usecase` constructor requires `repositories.SourceRepository`

**Breaking Changes:**
- `NewRetrievalAgent` function signature changes (all call sites must update)