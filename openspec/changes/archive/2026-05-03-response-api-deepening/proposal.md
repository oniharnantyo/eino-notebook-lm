## Why

The response usecase (`response_usecase.go`) is 965 lines of untested orchestration code. It coordinates 8+ services (agent factory, tool factory, retrievers, repositories, chat model) with complex control flow. The code is hard to understand, test, and modify.

## What Changes

- Split `response_usecase.go` into focused modules (orchestration, retrieval, generation)
- Extract agent/tool factory logic into separate package
- Consolidate three retrievers into single interface (prepare for unified-retrieval-seam)
- Simplify the response generation flow with clear stages
- Add tests for the core orchestration logic

**BREAKING**: Internal structure changes; HTTP API remains stable.

## Capabilities

### New Capabilities
- `response-orchestration`: Clean separation of response generation into testable stages

### Modified Capabilities
- Implementation changes only—no spec requirement changes

## Impact

**Affected files**:
- `internal/core/application/usecases/response/response_usecase.go` (965 lines → ~200 lines)
- `internal/core/application/usecases/response/agent/` (refactored)
- `internal/core/application/usecases/response/agent/tools/` (refactored)
- `internal/interfaces/http/handlers/response.go` (minimal updates)
