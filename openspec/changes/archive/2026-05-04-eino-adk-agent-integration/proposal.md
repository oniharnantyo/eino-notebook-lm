## Why

The current response pipeline executes retrieval upfront and dumps results into the system prompt. Tools are wired to the chat model but never called autonomously—there's no ReAct loop. The agent operates without awareness of what sources are available, leading to inefficient queries and missed information. Users select sources per query, but the agent receives no metadata about those documents.

## What Changes

- **Add Eino ADK dependency** (`github.com/cloudwego/eino/adk`) for ChatModelAgent and Runner
- **Create `internal/core/application/agent/` package**:
  - `agent.go` with `NewRetrievalAgent()` that creates ChatModelAgent
  - `instruction.go` with `BaseAgentInstruction` template
- **Add `CreateToolCallingChatModel()` factory method** in `pkg/model/chat_factory.go` for type-safe agent creation
- **Create `AgentStage`** in `internal/core/application/usecases/response/stages/` to replace `GenerationStage`
- **Remove `RetrievalStage`** from pipeline—agent handles all retrieval via tools
- **Add source catalog building**: fetch sources via `sourceRepo.GetByIDs()`, build catalog string (title, type, chunk count, status), inject into agent instruction
- **Update `ResponsePipeline`** to use `AgentStage` with source catalog, tools, and history
- **Implement AsyncIterator to SSE conversion** for streaming responses from ADK events

## Capabilities

### New Capabilities
- `agent-source-awareness`: Retrieval agent receives source catalog in system prompt and can query detailed metadata via `list_sources` tool. Agent autonomously decides which tools to call during ReAct loop.

### Modified Capabilities
None (implementation-level change only)

## Impact

**Affected Files:**
- `pkg/model/chat_factory.go` — Add `CreateToolCallingChatModel()` method
- `internal/core/application/agent/agent.go` — New file
- `internal/core/application/agent/instruction.go` — New file
- `internal/core/application/usecases/response/stages/agent_stage.go` — New file
- `internal/core/application/usecases/response/stages/retrieval_stage.go` — **DELETE**
- `internal/core/application/usecases/response/pipeline.go` — Use AgentStage, remove RetrievalStage, add source catalog building
- `internal/core/application/usecases/response/response_usecase.go` — Accept `sourceRepo` parameter, pass to pipeline
- `cmd/serve.go` — Wire `sourceRepo` to usecase, use `CreateToolCallingChatModel()`

**New Dependencies:**
- `github.com/cloudwego/eino/adk` — Agent Development Kit (ChatModelAgent, Runner)

**Breaking Changes:**
- `RetrievalStage` removed (no longer needed with agent)
- `CreateResponseStream` event flow changes from direct model stream to ADK event iteration
