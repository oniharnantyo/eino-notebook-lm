## Why

The retrieval agent's system prompt is currently assembled across multiple packages (`response` builds the catalog, `agent` holds the template, `AgentStage` combines them). This violates encapsulation—the agent should own its entire "brain" including prompt construction. Additionally, we need per-request catalog building as users can dynamically select/deselect sources.

## What Changes

- Move `buildSourceCatalog` from `response` package to `agent` package as `BuildCatalog`
- Change signature to support per-request source selection: `func BuildCatalog(ctx context.Context, sourceRepo repositories.SourceRepository, sourceIDs []uuid.UUID) string`
- Remove `PrepareInstruction` function (no longer needed)
- Update `AgentStage.Execute` to receive `sourceIDs []uuid.UUID` instead of `catalog string`
- Use Eino ADK's built-in placeholder replacement: pass catalog via session values, `{catalog}` placeholder in `BaseAgentInstruction` gets replaced automatically
- Restructure `BaseAgentInstruction` into 6 sections: Identity & Values, Safety & Ethics, Knowledge & Facts, Tools & Products, Behavioral Guidance, Style & Tone
- Delete `catalog_builder.go` (function moved to agent package)

## Capabilities

### New Capabilities
- `agent-system-prompt`: Agent owns its system prompt construction including catalog building and structured prompt sections

### Modified Capabilities
- `agent-retrieval`: Changes how the retrieval agent receives the catalog (via session values instead of pre-built string)

## Impact

**Modified Files:**
- `internal/core/application/agent/agent.go` - Add `BuildCatalog`, remove `PrepareInstruction`
- `internal/core/application/usecases/response/stages/agent_stage.go` - Change `Execute` signature, use `BuildCatalog`
- `internal/core/application/usecases/response/pipeline.go` - Update interface and `Execute` signature
- `internal/core/application/usecases/response/response_usecase.go` - Remove catalog building, pass sourceIDs

**Deleted Files:**
- `internal/core/application/usecases/response/catalog_builder.go` - Function moved to agent package
- `internal/core/application/usecases/response/catalog_builder_test.go` - Tests moved to agent package

**Test Updates:**
- All test files updated to match new signatures
