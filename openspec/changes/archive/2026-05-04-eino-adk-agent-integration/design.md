## Context

**Current State**: The response pipeline (`internal/core/application/usecases/response/pipeline.go`) executes retrieval upfront via `RetrievalStage` and dumps results into the system prompt. Tools (`keyword_search`, `semantic_search`, `list_sources`, `chunk_read`) are wired to the chat model in `GenerationStage`, but the model only gets a single `Generate()` or `Stream()` call — there's no ReAct loop for autonomous tool calling.

The agent has no awareness of what sources are available. Users select sources per query, but the agent receives no metadata about those documents (titles, types, chunk counts, status). It must blindly guess search terms.

**Constraints**:
- HTTP API must remain stable (Responses API contract unchanged)
- Eino framework abstractions must be preserved
- Existing streaming behavior must work
- Pipeline architecture should be maintained

**Stakeholders**:
- API consumers: expect stable streaming responses
- End users: expect relevant, context-aware responses
- Platform: needs observability (Langfuse integration)

## Goals / Non-Goals

**Goals**:
1. Agent receives source catalog (title, type, chunk count, status) in its system prompt
2. Agent can query detailed source metadata via `list_sources` tool
3. Agent autonomously executes retrieval tools during ReAct loop (decides what/when to search)
4. Instruction building happens in pipeline layer, agent package remains domain-agnostic
5. No breaking changes to tool scoping behavior or HTTP API

**Non-Goals**:
- Modifying existing tool behaviors (keyword_search, semantic_search, chunk_read, list_sources already exist)
- Changing HTTP API contracts
- Multi-turn source catalog updates (catalog is rebuilt per request)
- Source selection UI/UX changes

## Decisions

### 1. Use Eino ADK ChatModelAgent Instead of Custom Agent

**Decision**: Use `github.com/cloudwego/eino/adk`'s `ChatModelAgent` with ReAct pattern, not a custom agent implementation.

**Rationale**:
- ADK provides battle-tested ReAct orchestration (reason → act → observe → loop)
- Built-in streaming support via `AsyncIterator[*AgentEvent]`
- Tool calling is handled by the framework (no manual loop management)
- Middleware support for future enhancements (summarization, tool reduction, etc.)
- Aligns with existing Eino infrastructure (chat models, tools already use Eino)

**Alternatives Considered**:
- *Custom agent with manual tool loop*: Rejected because ADK already solves this problem well. Custom implementation would be maintenance burden.
- *DeepAgent from ADK*: Rejected because we don't need planning (WriteTodos) or filesystem access. ChatModelAgent is simpler and sufficient.

### 2. Remove RetrievalStage, Agent Does All Retrieval via Tools

**Decision**: Remove `RetrievalStage` from the pipeline. The agent retrieves context exclusively through tools (`keyword_search`, `semantic_search`, `image_search`, `chunk_read`).

**Rationale**:
- **Cleaner separation**: Pipeline handles orchestration (history, tools, agent), agent handles retrieval logic
- **No redundant retrieval**: Avoids fetching documents upfront that the agent might not need
- **Agent autonomy**: Agent decides which retrieval method to use (semantic vs keyword vs hybrid) based on the query
- **Tools are already there**: All retrieval tools exist and are tested — just need to be wired to the agent

**Trade-off**: First response latency increases because the agent must reason + search instead of getting pre-fetched context. However, this is acceptable because:
- Agent typically makes 1-2 tool calls (not infinite loops)
- Better relevance outweighs slight latency increase
- Can be optimized later with caching or prediction

**Alternatives Considered**:
- *Keep RetrievalStage for background context*: Rejected because it creates redundant retrieval and token waste. Agent might ignore the pre-fetched context and search anyway.
- *Hybrid (RetrievalStage provides top-K, agent does deeper searches)*: Considered but adds complexity. Agent can request "top 5 relevant chunks" via tools if needed.

### 3. Per-Request Agent Creation

**Decision**: Create a new `ChatModelAgent` for each request, not reuse a cached agent.

**Rationale**:
- **Tools are scoped per-request**: Different requests have different `SourceIDs`, so the agent's `ToolsConfig.Tools` must change
- **Instruction changes per-request**: The source catalog is injected into the instruction, which varies by selected sources
- **ADK agents are lightweight**: Creating an agent is cheap (just a config struct), no expensive initialization
- **No shared state**: Per-request creation avoids concurrency issues and state leakage

**Alternatives Considered**:
- *Create agent once, reconfigure per request*: Rejected because ADK's `ChatModelAgentConfig` is immutable after creation. Would require cloning the entire agent.
- *Pool of agents*: Rejected because over-engineering. Per-request creation is fast enough.

### 4. Factory Method: CreateToolCallingChatModel

**Decision**: Add a new factory method `CreateToolCallingChatModel()` in `pkg/model/chat_factory.go` that returns `model.ToolCallingChatModel`, separate from the existing `CreateChatModel()` that returns `model.BaseChatModel`.

**Rationale**:
- **Compile-time type safety**: Agent stage gets `ToolCallingChatModel` directly, no runtime assertion needed
- **Explicit intent**: Callers that need tools use the dedicated method
- **No breaking changes**: Existing consumers (mindmap, etc.) continue using `CreateChatModel()`
- **Fails fast at startup**: If model doesn't support tool calling, error happens during initialization, not during first request

**Alternatives Considered**:
- *Type-assert at point of use*: Rejected because it pushes runtime assertion to every call site. Factory method centralizes the check.
- *Change CreateChatModel to return ToolCallingChatModel*: Rejected because it's a breaking change and forces all consumers to carry the larger interface.

### 5. Source Catalog Format: Concise Inline List

**Decision**: Inject source catalog into agent instruction as a concise markdown-formatted list.

**Format**:
```
## Available Sources

- [src_abc1] "Paper Title" (PDF, 15 chunks) [processing]
- [src_abc2] "Meeting Notes" (markdown, 3 chunks)

Total: 2 sources, 18 chunks
```

**Rationale**:
- **Minimal token overhead**: ~50 tokens per source, acceptable for typical 1-5 source selections
- **Human-readable**: Format is clear if user inspects agent instruction
- **Status indicators**: Only `[processing]` or `[failed]` when applicable — keeps prompt lean
- **Markdown parsing**: LLMs understand markdown well, easy to parse

**Alternatives Considered**:
- *JSON format*: Rejected because more verbose and harder for LLM to parse naturally
- *Full metadata dump*: Rejected because token-prohibitive for large source catalogs

### 6. Streaming: AsyncIterator to SSE Conversion

**Decision**: Consume ADK's `AsyncIterator[*AgentEvent]` in `AgentStage` and map events to the existing SSE format defined in `dtos/chat.go`.

**Rationale**:
- **Preserves API contract**: HTTP clients continue receiving the same event types (`response.created`, `response.in_progress`, etc.)
- **ADK events are richer**: We filter/transform ADK events to the Responses API format
- **Single streaming path**: No need for separate streaming paths in the pipeline

**Mapping**:
- `AgentEvent.Output.MessageOutput.IsStreaming == true` → emit `response.output_text.delta` events
- Tool calls during ReAct → emitted as internal events (not exposed to client, or logged only)
- Final assistant message → `response.completed`

**Trade-off**: Tool calling intermediate steps are not exposed to the client in real-time. This is acceptable because:
- Responses API focuses on final output, not intermediate reasoning
- Can expose later via a different event type if needed
- Langfuse captures tool calls for observability

**Alternatives Considered**:
- *Expose all ADK events to client*: Rejected because it would break the Responses API contract and expose implementation details.
- *Skip ADK streaming, use model streaming directly*: Rejected because we lose agent events (tool calls, iterations) needed for observability.

### 7. Error Handling: Fail Fast with Clear Messages

**Decision**: If model doesn't support tool calling (type assertion fails), fail fast with clear error at startup.

**Rationale**:
- **Configuration errors should be caught early**: Better to fail at startup than during first user request
- **Clear error message**: "model X does not support tool calling, required for agent functionality"
- **Forces correct config**: Users can't accidentally deploy a non-tool-calling model

**Alternatives Considered**:
- *Graceful degradation to non-agent mode*: Rejected because it creates two code paths and defeats the purpose of the change.

## Risks / Trade-offs

### Risk 1: Agent Loop Latency

**Risk**: Agent may make multiple tool calls (semantic search → keyword search → chunk read), increasing response time compared to single-pass retrieval.

**Mitigation**:
- Set `MaxIterations: 10` to prevent infinite loops
- Use `list_sources` tool first to get source metadata, then targeted searches
- Monitor tool call patterns via Langfuse to optimize
- Consider caching frequent queries

### Risk 2: Tool Calling Reliability

**Risk**: Model may fail to call tools correctly (hallucinates parameters, skips tools, loops).

**Mitigation**:
- Use clear tool descriptions and JSON schema
- Test with various queries during development
- Set `MaxIterations` as safety net
- Add middleware (e.g., `PatchToolCallsMiddleware`) if needed

### Risk 3: Streaming Complexity

**Risk**: Converting `AsyncIterator[*AgentEvent]` to SSE events is more complex than direct model streaming.

**Mitigation**:
- Create a dedicated `eventTranslator` function in `AgentStage`
- Write unit tests for event mapping
- Test streaming with real agent before deploying
- Use non-streaming mode as fallback if issues arise

### Risk 4: Prompt Token Bloat

**Risk**: Large source catalogs (50+ documents) consume significant prompt tokens on every request.

**Mitigation**:
- Users select sources per query, so typical scope is 1-5 documents
- Catalog is ~50 tokens per source (acceptable for small selections)
- If problematic, can add `max_sources_in_catalog` limit or truncate

### Risk 5: Model Compatibility

**Risk**: Not all models support tool calling (e.g., some local models, older Gemini versions).

**Mitigation**:
- Factory method fails fast at startup if model doesn't implement `ToolCallingChatModel`
- Clear error message guides users to use compatible model
- Documentation lists supported models

## Migration Plan

### Phase 1: Add Dependencies and Factory (Non-Breaking)
1. Add `github.com/cloudwego/eino/adk` to `go.mod`
2. Add `CreateToolCallingChatModel()` to `pkg/model/chat_factory.go`
3. Update `cmd/serve.go` to use new factory method for response usecase
4. **Risk**: Low (new code, no behavior change)

### Phase 2: Create Agent Package (Non-Breaking)
1. Create `internal/core/application/agent/agent.go` with `NewRetrievalAgent()`
2. Create `internal/core/application/agent/instruction.go` with `BaseAgentInstruction`
3. Write unit tests for agent creation
4. **Risk**: Low (new code, not wired to pipeline yet)

### Phase 3: Create AgentStage (Non-Breaking)
1. Create `internal/core/application/usecases/response/stages/agent_stage.go`
2. Implement ADK Runner integration and AsyncIterator consumption
3. Write unit tests for AgentStage execution
4. **Risk**: Medium (new stage, but not replacing GenerationStage yet)

### Phase 4: Pipeline Integration (Breaking)
1. Remove `RetrievalStage` from pipeline
2. Replace `GenerationStage` with `AgentStage` in `ResponsePipeline`
3. Add source catalog building logic to pipeline
4. Update `response_usecase.go` to accept `sourceRepo`
5. Update `cmd/serve.go` wiring to pass `sourceRepo`
6. **Risk**: Medium (orchestration changes, thorough testing required)

### Phase 5: Streaming Conversion (Breaking)
1. Implement AsyncIterator to SSE event mapping in `AgentStage`
2. Update `CreateResponseStream` to use new streaming path
3. Test streaming with real agent and tools
4. **Risk**: High (streaming is critical path, validate thoroughly)

### Phase 6: Testing and Validation
1. Add integration tests for full agent flow
2. Test with various source selections (empty, single, many, mixed statuses)
3. Verify Langfuse tracing captures tool calls
4. Load test streaming endpoint
5. **Risk**: Low (testing only)

### Rollback Strategy
- Each phase is a separate commit
- Phase 1-3 can be reverted independently (new code)
- Phase 4-5 require reverting pipeline changes (keep old code during migration)
- Feature flag can gate agent usage if needed

## Open Questions

None — design is self-contained within existing architecture patterns. All key decisions have been made with clear rationale.
