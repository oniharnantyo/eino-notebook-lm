## Context

Currently, the retrieval agent's system prompt is assembled across multiple packages:
- `response` package builds the catalog string via `buildSourceCatalog()`
- `agent` package holds the `BaseAgentInstruction` template with `{catalog}` placeholder
- `AgentStage` combines them using `PrepareInstruction()`
- The agent receives the full instruction via ADK session values

This violates encapsulation—the agent should own its entire "brain" including prompt construction. Additionally, we need per-request catalog building as users can dynamically select/deselect sources through the UI.

## Goals / Non-Goals

**Goals:**
- Encapsulate all system prompt construction within the `agent` package
- Enable per-request catalog building based on user-selected sources
- Leverage Eino ADK's built-in placeholder replacement for cleaner code
- Reduce coupling between `response` usecase and agent implementation

**Non-Goals:**
- Changing the content of the system prompt (instruction template stays the same)
- Modifying how tools are created or scoped
- Changing the agent's behavior or capabilities

## Decisions

### Use ADK's Session Value Placeholder Replacement

**Decision:** Pass catalog via ADK session values with `{catalog}` placeholder in `BaseAgentInstruction`.

**Rationale:**
- ADK's `ChatModelAgentConfig.Instruction` supports f-string placeholders for session values
- This is the framework's intended pattern for dynamic prompt content
- Eliminates need for manual string replacement (`PrepareInstruction` removed)
- Clean separation: agent owns template, stage provides data

**Alternatives considered:**
1. **Custom handler for catalog injection** - More complex, handler would need to intercept and modify instruction
2. **Pre-build instruction in stage** - Violates encapsulation (stage knows about prompt structure)
3. **Create new agent per request** - Inefficient, agents are reusable

### Move `BuildCatalog` to Agent Package

**Decision:** Relocate catalog building logic from `response` to `agent` package with signature `func BuildCatalog(ctx context.Context, sourceRepo repositories.SourceRepository, sourceIDs []uuid.UUID) string`.

**Rationale:**
- Agent package owns all aspects of its "brain" including catalog formatting
- Per-request building supports dynamic source selection
- Access to `SourceRepository` needed for fetching source metadata
- Function is pure (no side effects) and easily testable

**Alternatives considered:**
1. **Keep in response package** - Would violate encapsulation goal
2. **Make it a method on `SourceRepository`** - Blurs repository responsibility (data access vs presentation)
3. **Interface-based approach** - Over-engineering for a simple formatting function

### Change `AgentStage.Execute` Signature

**Decision:** Change from `Execute(ctx, input, catalog string, tools)` to `Execute(ctx, input, sourceIDs []uuid.UUID, tools)`.

**Rationale:**
- Stage now provides "what" (source IDs) rather than "how" (catalog string)
- Aligns with per-request catalog building
- Matches the user's selection model (checkboxes in UI → source IDs)

**Alternatives considered:**
1. **Keep catalog string parameter** - Would require pre-building in response usecase (violates encapsulation)
2. **Pass full Source objects** - Unnecessary data transfer, stage would need repository access anyway

### Delete `catalog_builder.go`

**Decision:** Remove `internal/core/application/usecases/response/catalog_builder.go` entirely.

**Rationale:**
- Function moved to `agent` package where it belongs
- No other consumers of this function
- Reduces package coupling

**Migration:** Tests moved to `internal/core/application/agent/catalog_test.go`

### Restructure BaseAgentInstruction into 6 Sections

**Decision:** Reorganize the system prompt into 6 clearly defined sections in this order:
1. **Identity & Values** — Who the agent is and its purpose
2. **Safety & Ethics** — Hard boundaries and constraints
3. **Knowledge & Facts** — The `{catalog}` placeholder with source context
4. **Tools & Products** — Descriptions of each available tool
5. **Behavioral Guidance** — Step-by-step retrieval workflow
6. **Style & Tone** — Output format and communication style

**Rationale:**
- LLMs weight earlier instructions more heavily (primacy effect) — identity-first grounds the agent's purpose before constraints
- Separating concerns into named sections makes the prompt easier to maintain and iterate
- Knowledge & Facts placed before Tools so the agent knows its data context before learning about capabilities
- Behavioral Guidance after Tools so the agent understands what it CAN do before learning HOW to use them

**Alternatives considered:**
1. **Safety-first ordering** — Valid approach (used by Anthropic), but for a retrieval agent, grounding identity first produces more focused behavior
2. **Single flat prompt** — Current structure; harder to maintain and less structured for the LLM

## Risks / Trade-offs

### Risk: Breaking existing tests

**Impact:** All test files that mock `AgentStage.Execute` will need signature updates.

**Mitigation:**
- Update all test files in this change
- Test changes are mechanical (signature updates, not logic changes)

### Risk: ADK placeholder behavior changes

**Impact:** If ADK's placeholder replacement behaves unexpectedly, agent prompts may be malformed.

**Mitigation:**
- Framework feature is well-documented in `ChatModelAgentConfig`
- Can be verified with integration tests
- Fallback: manual string replacement if needed (but unlikely)

### Trade-off: Agent package now depends on repositories

**Impact:** `agent` package imports `repositories.SourceRepository`.

**Rationale:**
- Necessary for fetching source metadata to build catalog
- Acceptable coupling: agent needs domain data to construct its prompt
- Repository is an interface, not concrete implementation (testable)

## Migration Plan

1. **Phase 1: Prepare**
   - Add `BuildCatalog` function to `agent` package
   - Update imports in `agent` package

2. **Phase 2: Update consumers**
   - Change `AgentStage.Execute` signature
   - Update `ResponsePipeline` to pass sourceIDs
   - Update `ResponseUseCase` to remove catalog building

3. **Phase 3: Cleanup**
   - Delete `catalog_builder.go`
   - Remove `PrepareInstruction` function
   - Move tests to `agent` package

4. **Phase 4: Verify**
   - Run full test suite
   - Manual testing with dynamic source selection

**Rollback strategy:** Git revert if issues arise (mechanical refactoring, low risk)

## Open Questions

None - the refactoring is straightforward with no architectural ambiguities.
