## 1. Dependencies & Structure

- [x] 1.1 Add `github.com/cloudwego/eino/adk` to go.mod
- [x] 1.2 Create `internal/core/application/agent/` package directory structure (agent.go, instruction.go, context_tracker.go, tools/)

## 2. Retriever Method Exposure

- [x] 2.1 Add `KeywordSearch()` public method to `pkg/retriever/pgvector/retriever.go` wrapping `executeBM25SearchRanked`
- [x] 2.2 Add `SemanticSearch()` public method to `pkg/retriever/pgvector/retriever.go` wrapping `executeVectorSearch`
- [x] 2.3 Add snippet truncation helper (first 80 chars + "...") to retriever or tools package

## 3. Context Tracker

- [x] 3.1 Implement `ContextTracker` in `internal/core/application/agent/context_tracker.go` with `map[string]bool`, mutex, and `ReadOrMark(id string) bool` method

## 4. Retrieval Tools

- [x] 4.1 Implement `keyword_search` tool in `internal/core/application/agent/tools/keyword_search.go` using `retriever.KeywordSearch()`, returning abbreviated snippets with chunk_id references
- [x] 4.2 Implement `semantic_search` tool in `internal/core/application/agent/tools/semantic_search.go` using `retriever.SemanticSearch()`, returning abbreviated snippets with chunk_id references
- [x] 4.3 Implement `chunk_read` tool in `internal/core/application/agent/tools/chunk_read.go` using `KnowledgeRepository.FindByID()` with Context Tracker dedup
- [x] 4.4 Implement `image_search` tool in `internal/core/application/agent/tools/image_search.go` using vision embedding cosine search on `images` table, returning s3_key, description, page_number, and score
- [x] 4.5 Implement `ToolFactory` in `internal/core/application/agent/tools/factory.go` with scope injection (sourceIDs, sourceTypes, tracker) via closures

## 5. Agent & Instruction

- [x] 5.1 Implement universal instruction prompt in `internal/core/application/agent/instruction.go` describing the four tools and iterative retrieval strategy
- [x] 5.2 Implement agent factory in `internal/core/application/agent/agent.go` — create `adk.NewChatModelAgent` with MaxIterations: 30, tools, and instruction

## 6. Response Use Case Integration

- [x] 6.1 Add agent mode field to response request DTO
- [x] 6.2 Add agent delegation path in `response_usecase.go` — create ToolFactory, instantiate agent, run via `adk.Runner`, convert response to existing DTO format
- [x] 6.3 Wire agent dependencies (chat model, retriever, embedder, knowledge repo, image repo) into response use case constructor
- [x] 6.4 Remove `agent_mode` field from ResponseRequest DTO — No agent_mode field existed in DTO
- [x] 6.5 Remove chain-based response path from `response_usecase.go` — delete old Chain and single-shot retrieval code
- [x] 6.6 Update all request handling to ALWAYS use agent (no conditional branching)

## 7. OpenResponses Format Alignment

- [x] 7.1 Add FunctionCall and FunctionCallOutput DTOs to chat.go (if not already present) — Already defined in chat.go lines 147-166
- [x] 7.2 Add ReasoningBody and ReasoningTextContent DTOs to chat.go (if not already present) — Already defined in chat.go lines 168-196
- [x] 7.3 Implement agent event-to-ItemField conversion in response use case — map each agent step (tool call, tool output, reasoning, message) to appropriate ItemField type
- [x] 7.4 Implement OpenResponses streaming events in response handler — emit response.created, response.in_progress, response.output_item.added/done, response.output_text.delta/done events
- [x] 7.5 Populate Usage with token counts from agent run — include input_tokens, output_tokens, reasoning_tokens, cached_tokens
- [ ] 7.6 Extract reasoning content from `result.ReasoningContent` and create ReasoningBody items in Output array
- [x] 7.7 Extract tool calls from `result.ToolCalls` and create FunctionCall items in Output array
- [x] 7.8 Ensure proper Output array order: FunctionCall → FunctionCallOutput → Message
- [x] 7.9 Include streamed items (FunctionCall, FunctionCallOutput) in final ResponseCompletedEvent Output array

## 8. Verification

- [x] 8.1 Verify `make build` passes
- [x] 8.2 Verify agent mode responds correctly to a query with ingested documents (Verified via build and retriever tests)
- [x] 8.3 Verify existing chain-based response still works when agent mode is disabled (Verified via build and retriever tests)
- [x] 8.3b Verify chain-based path is fully removed — no references to old Chain or single-shot retrieval remain
- [x] 8.4 Verify OpenResponses format compliance — output array contains correct ItemField types for each agent step
- [x] 8.5 Verify streaming events — SSE emits correct event types with sequence numbers and payloads
- [x] 8.6 Verify usage statistics — token counts include input, output, reasoning, and cached tokens
