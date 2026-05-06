## Context

The current RAG pipeline in `response_usecase.go` uses a linear Eino Chain: system template → history placeholder → user input → ChatModel. Context retrieval happens once before generation via `pkg/retriever/pgvector`, which runs BM25 and vector search in parallel, merges with RRF, and returns full sentence content as `[]*schema.Document`.

The retriever's `executeVectorSearch` and `executeBM25SearchRanked` are unexported methods on the `Retriever` struct. The `sentences` table links to `knowledges` via `knowledge_id`, providing the two-level hierarchy A-RAG needs: sentences for search, knowledge chunks for full reads.

The project uses Clean Architecture with DDD — agent orchestration belongs in the application layer, not `pkg/`.

## Goals / Non-Goals

**Goals:**
- Replace single-shot retrieval with an iterative ReAct agent loop driven by the LLM (DEFAULT behavior)
- Provide three tools: `keyword_search`, `semantic_search`, `chunk_read`
- Deduplicate chunk reads via a per-request Context Tracker
- Search tools return abbreviated snippets, forcing explicit `chunk_read` for full context
- Remove the old chain-based response path entirely
- Stream agent events to the client via SSE

**Non-Goals:**
- Agent execution persistence/tracking (no new entities or migrations)
- Multi-agent orchestration or sub-agents
- Custom middleware beyond what Eino ADK provides
- Tuning snippet truncation heuristics beyond initial implementation
- Per-notebook customizable instruction prompts

## Decisions

### 1. Agent placement: `internal/core/application/agent/`

The agent is an application-level orchestration concern. It composes domain components (retriever, repositories) into an agentic workflow. Placing it under `application/agent/` keeps the dependency direction clean — `usecases/response` depends on `agent/`, which depends on domain interfaces and infrastructure.

**Alternative**: `pkg/agent/` — rejected because agents are application-specific, not reusable shared packages.

### 2. Eino ADK ChatModelAgent instead of custom ReAct loop

Eino's `adk.NewChatModelAgent` provides the full ReAct loop natively: LLM reasons → generates tool calls → executes tools → feeds results back → repeats until no more tool calls or `MaxIterations` reached. Building a custom loop would duplicate this and risk inconsistencies with Eino's tool calling protocol.

The agent is configured with:
- `MaxIterations: 30` (user-specified)
- `Instruction`: universal system prompt (same for all notebooks)
- `Tools`: four retrieval tools (keyword_search, semantic_search, chunk_read, image_search), scoped to the request's notebook/sources
- `Model`: existing `model.ToolCallingChatModel` from the chat model factory

### 3. Retriever split: expose unexported methods as public

Add `KeywordSearch()` and `SemanticSearch()` as public methods on the existing `pgvector.Retriever`. These wrap the current `executeBM25SearchRanked` and `executeVectorSearch` respectively. The original `Retrieve()` hybrid method is preserved for backward compatibility.

**Alternative**: Create new thin retriever structs — rejected because it duplicates connection pool management and SQL query logic.

### 4. Snippet truncation: first 80 chars + chunk ID reference

Search tools return abbreviated snippets: first 80 characters of the sentence content + `"..."` + the `knowledge_id` as a reference. This is a fixed heuristic for initial implementation. The agent uses the `knowledge_id` with `chunk_read` to get full content.

```
Search result format per sentence:
{
  "sentence_id": "abc-123",
  "snippet": "The mitochondria is the powerhouse of the cell, generating ATP through...",
  "chunk_id": "knowledge-456",
  "score": 0.89
}

Image search result format:
{
  "image_id": "img-789",
  "source_id": "source-456",
  "s3_key": "images/doc1-page3-image1.png",
  "description": "A bar chart showing quarterly revenue with a peak in Q3",
  "page_number": 3,
  "score": 0.92
}
```

### 5. Context Tracker: per-request `map[string]bool` with mutex

A simple goroutine-safe map keyed by `knowledge_id`. Created per agent run, passed to `chunk_read` via closure. If the agent reads a chunk it's already seen, the tool returns `"Chunk {id} has already been read. Explore other areas."` — zero tokens wasted on re-reading.

No persistence needed. The tracker lives only for the duration of a single agent run.

### 6. Conversation history: injected as messages into Runner

The Eino `Runner.Run()` accepts `[]adk.Message` (aliased `[]*schema.Message`). History is loaded via `ConversationRepository`, trimmed by `HistoryManager`, and the new user message is appended. The agent sees conversation context as ambient messages — it doesn't need a history tool.

This keeps the existing history management (sliding window, token limits) unchanged. The agent's ReAct loop is purely about retrieval.

### 7. Tool scoping via factory closure

A `ToolFactory` creates scoped tools per request, baking in `notebookID`, `sourceIDs`, and `sourceTypes` as closures. The agent never sees these parameters — they're applied to the underlying retriever calls automatically.

**Server provides RAG tools automatically**: The four retrieval tools (`keyword_search`, `semantic_search`, `chunk_read`, `image_search`) are ALWAYS included by the server based on the request's RAG scope. Clients do NOT need to specify these tools.

**Client tools are optional**: Clients MAY provide additional custom tools via the `tools` request parameter. These are merged with the server's RAG tools.

```go
// Server always provides these
ragTools := factory.NewScopedTools(agent.ScopeConfig{
    SourceIDs:   req.SourceIDs,
    SourceTypes: req.SourceTypes,
    Tracker:     agent.NewContextTracker(),
})

// Client may provide custom tools
allTools := append(ragTools, req.Tools...)

factory := agent.NewToolFactory(retriever, knowledgeRepo, embedder, imageRepo)
tools := factory.NewScopedTools(agent.ScopeConfig{
    SourceIDs:   req.SourceIDs,
    SourceTypes: req.SourceTypes,
    Tracker:     agent.NewContextTracker(),
})
```

### 8. Response use case integration: agent-only pattern

`response_usecase.go` uses the agent for ALL requests. The `agent_mode` field is removed — agent-driven retrieval is the default and only behavior. The agent returns the final message, which is converted to the existing `ResponseResource` DTO format. The old chain-based response path is removed entirely.

**Alternative**: Keep chain as fallback — rejected because agent retrieval provides strictly better quality on multi-faceted queries and the performance overhead is acceptable.

## Risks / Trade-offs

### 9. OpenResponses contract alignment

The response DTOs in `internal/core/application/dtos/chat.go` already implement the OpenResponses API format. The agent's intermediate steps (tool calls, reasoning) are encoded as `ItemField` in the `output` array:

- **FunctionCall** - when agent calls a retrieval tool
- **FunctionCallOutput** - when tool returns results
- **ReasoningBody** with **ReasoningTextContent** - if the model exposes chain-of-thought
- **Message** with **OutputTextContent** - final answer

Streaming events follow OpenResponses event types:
- `response.created`, `response.in_progress`, `response.completed` / `response.failed`
- `response.output_item.added`, `response.output_item.done`
- `response.content_part.added`, `response.content_part.done`
- `response.output_text.delta`, `response.output_text.done`

**Current Implementation Gaps**:
- Reasoning content from `result.ReasoningContent` is not extracted to create `ReasoningBody` items
- Tool calls from `result.ToolCalls` are not included in the final `Output` array (only Message items)
- Streaming events for FunctionCall/FunctionCallOutput are sent but NOT included in final `ResponseCompletedEvent`
- Proper Output array order not enforced: Reasoning → Message → FunctionCall → FunctionCallOutput

**Alternative**: Custom agent event format — rejected because OpenResponses is emerging as the standard for agentic response streaming.

## Risks / Trade-offs

**[High iteration count → slow responses]** → `MaxIterations: 30` is generous. Most queries should converge in 5-10 iterations. Monitor average iterations and adjust the default down if latency becomes an issue. The limit prevents infinite loops.

**[Agent refuses to use tools → poor retrieval]** → The instruction prompt must be clear about tool usage expectations. If the LLM answers from its own knowledge, retrieval is skipped. This is acceptable for general knowledge questions but undesirable for notebook-specific queries. The prompt should emphasize using tools for notebook content.

**[Snippet truncation loses critical context]** → 80 chars is a heuristic. Some important sentences may be cut mid-phrase. This is an acceptable tradeoff — the agent can always `chunk_read` for full context. Tuning can happen after initial deployment based on observed agent behavior.

**[Token cost increase vs. single-shot retrieval]** → An iterative agent loop uses more tokens than a single retrieval + generation. The tradeoff is better answer quality on complex queries. The Context Tracker mitigates redundant token usage from re-reading chunks.

**[Eino ADK stability]** → `github.com/cloudwego/eino/adk` is relatively new. If it has bugs or breaking changes, the agent layer is isolated enough to swap to a custom ReAct loop without affecting the tools or tracker.

## Open Questions

- Should `chunk_read` support reading adjacent chunks (e.g., `chunk_read` with `offset: +1`/`-1`)? Deferred to post-initial implementation.
- Should the agent have an explicit "I have enough information" exit tool, or rely on the LLM naturally stopping tool calls? Eino's ChatModelAgent already handles this — the loop ends when the LLM doesn't generate tool calls.
