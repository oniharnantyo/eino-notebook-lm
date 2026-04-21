# ReAct-Style Agentic RAG Implementation Plan

**Plan Status:** Draft
**Created:** 2025-04-04
**Complexity:** HIGH
**Estimated Files:** 15 new files, 3 modifications

---

## Executive Summary

This plan details the implementation of ReAct-style (Reasoning + Acting) agentic RAG capabilities for the chat/conversation feature. The agent will be able to iteratively reason, retrieve context, refine queries, and decide when to use tools vs. when to respond directly.

**Key Approach:** Integrate `github.com/cloudwego/eino/adk` (Agent Development Kit) to create a ChatModelAgent with ReAct prompting, tool definitions for retrieval/search, and event-driven streaming support.

---

## Architecture Overview

### Current State
- ✅ pgvector hybrid retrieval (BM25 + Vector with RRF)
- ✅ Gemini embeddings and chat models
- ✅ Simple chain orchestration via `eino/compose`
- ✅ SSE streaming for responses
- ✅ Conversation history with sliding window
- ❌ No agent orchestration
- ❌ No tool/function calling
- ❌ No iterative reasoning

### Target State
```
User Query → ReAct Agent → Tool Executor → Retrieval/Context → Reasoning Loop → Final Response
                    ↓
              Event Stream (SSE)
```

---

## Dependencies

### Go Module Additions
```go
// Already in use: github.com/cloudwego/eino v0.8.1
// New imports needed:
import (
    "github.com/cloudwego/eino/adk"  // Agent Development Kit
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/adk/tool"
)
```

### External Dependencies
No new external dependencies required - all functionality available via `eino` ecosystem.

---

## File Structure

### New Files to Create

```
internal/core/application/usecases/agent/
├── agent_usecase.go                 # Main agent orchestration use case
├── tools/
│   ├── retrieval_tool.go           # Tool: Semantic search via pgvector
│   ├── search_tool.go              # Tool: BM25 keyword search
│   ├── context_tool.go             # Tool: Multi-source context aggregation
│   └── tools.go                    # Tool registration and configuration
├── prompts/
│   ├── react_prompt.go             # ReAct-style reasoning prompt template
│   └── system_prompts.go           # System prompt variations
├── events/
│   ├── agent_events.go             # Agent event types for SSE streaming
│   └── event_handler.go            # Event processing and streaming
└── agent.go                        # Agent factory and configuration

internal/core/domain/entities/
├── agent_execution.go              # Agent execution tracking entity
└── tool_invocation.go              # Tool invocation records

internal/core/domain/repositories/
└── agent_execution_repository.go   # Repository for agent execution logs

internal/infrastructure/persistence/
└── agent_execution.go              # Postgres implementation

internal/core/application/dtos/
└── agent.go                        # Agent-related DTOs (request/response)

internal/interfaces/http/handlers/
└── agent.go                        # HTTP handler for agent endpoints
```

### Files to Modify

```
internal/core/application/usecases/response/response_usecase.go
  - Add agent mode option to CreateResponse/CreateResponseStream
  - Delegate to agent_usecase when agent mode is enabled

internal/interfaces/http/routes/routes.go
  - Add agent endpoints (if separate from response endpoints)

internal/core/application/usecases/chat/usecase.go
  - Add AgentResponseUseCase interface
```

---

## Component Design

### 1. Agent Configuration (`internal/core/application/usecases/agent/agent.go`)

```go
type AgentConfig struct {
    // Model configuration
    ChatModel     model.BaseChatModel
    Embedder      embedding.Embedder

    // Retrieval configuration
    Retriever     retriever.Retriever
    MaxIterations int  // Default: 5

    // ReAct configuration
    ReasoningPrompt string  // ReAct-style prompt template
    ToolNames       []string // Available tools

    // Event configuration
    EnableStreaming bool
    EventHandler    AgentEventHandler
}

type ReActAgent struct {
    config *AgentConfig
    agent  *adk.ChatModelAgent
    runner *adk.AgentRunner
}
```

**Responsibilities:**
- Factory for creating ChatModelAgent instances
- Configure ReAct prompt templates
- Register available tools
- Manage agent lifecycle

---

### 2. Tool Definitions (`internal/core/application/usecases/agent/tools/`)

**Design Decision:** Using 2 tools instead of 3. The `get_context` hybrid tool is redundant since the existing retriever at `pkg/retriever/pgvector/` already implements RRF (Reciprocal Rank Fusion) merging of BM25 + vector search.

#### 2.1 Semantic Search Tool (`semantic_search_tool.go`)

```go
type SemanticSearchInput struct {
    Query      string   `json:"query"`
    Limit      int      `json:"limit,omitempty"` // Default: 5
    SourceIDs  []string `json:"source_ids,omitempty"`
    SourceTypes []string `json:"source_types,omitempty"`
}

type SemanticSearchOutput struct {
    Documents []DocumentContext `json:"documents"`
    Query     string            `json:"query"`
    Count     int               `json:"count"`
}

// Tool implementation
func NewSemanticSearchTool(retriever retriever.Retriever, embedder embedding.Embedder) tool.Tool {
    return tool.NewTool(
        "semantic_search",
        "Search for relevant documents using semantic vector similarity. Use this when you need to find information by meaning rather than exact keywords. Limits results to 5 by default.",
        &SemanticSearchInput{},
        func(ctx context.Context, input any) (string, error) {
            // Implementation
        },
    )
}
```

**Tool Behavior:**
- Performs vector similarity search
- Supports filtering by source IDs/types
- Returns formatted document snippets
- Default limit: 5 results

#### 2.2 Keyword Search Tool (`keyword_search_tool.go`)

```go
type KeywordSearchInput struct {
    Query      string   `json:"query"`
    Limit      int      `json:"limit,omitempty"` // Default: 5
    SourceIDs  []string `json:"source_ids,omitempty"`
    SourceTypes []string `json:"source_types,omitempty"`
}

type KeywordSearchOutput struct {
    Documents []DocumentContext `json:"documents"`
    Query     string            `json:"query"`
    Count     int               `json:"count"`
}

func NewKeywordSearchTool(retriever retriever.Retriever) tool.Tool {
    return tool.NewTool(
        "keyword_search",
        "Search for documents using BM25 keyword matching. Use this for exact term searches, names, or specific phrases. Limits results to 5 by default.",
        &KeywordSearchInput{},
        // Implementation
    )
}
```

**Tool Behavior:**
- Performs BM25 keyword search
- Supports filtering by source IDs/types
- Returns formatted document snippets
- Default limit: 5 results

#### 2.3 Tool Registration (`tools.go`)

```go
type ToolRegistry struct {
    tools map[string]tool.Tool
}

func NewToolRegistry(
    retriever retriever.Retriever,
    embedder embedding.Embedder,
) *ToolRegistry {
    registry := &ToolRegistry{tools: make(map[string]tool.Tool)}

    registry.Register(NewSemanticSearchTool(retriever, embedder))
    registry.Register(NewKeywordSearchTool(retriever))

    return registry
}

func (r *ToolRegistry) Register(t tool.Tool) {
    r.tools[t.Info().Name] = t
}

func (r *ToolRegistry) GetTools() []tool.Tool {
    tools := make([]tool.Tool, 0, len(r.tools))
    for _, t := range r.tools {
        tools = append(tools, t)
    }
    return tools
}
```

---

### 3. ReAct Prompt Template (`internal/core/application/usecases/agent/prompts/react_prompt.go`)

```go
const ReActPromptTemplate = `You are a helpful assistant with access to a knowledge base. You use a ReAct (Reasoning + Acting) approach to answer questions.

Available Tools:
{{- range .Tools }}
- {{.Name}}: {{.Description}}
{{- end}}

Instructions:
1. Think step-by-step about what information you need
2. Use available tools to gather relevant context
3. Reason about the information retrieved
4. If you need more information, use tools again with refined queries
5. When you have sufficient information, provide a clear, helpful answer

Response Format:
- Start your reasoning with "Thought:"
- When using a tool, format as: "Action: tool_name\nAction Input: {json_input}"
- After tool execution, continue with "Observation:" followed by your analysis
- When ready to answer, use "Final Answer:" followed by your response

Example:
Thought: I need to find information about X
Action: semantic_search
Action Input: {"query": "X", "top_k": 5}
Observation: [tool results]
Thought: The results provide context about X, but I need more details on Y
Action: keyword_search
Action Input: {"query": "Y exact phrase"}
Observation: [tool results]
Thought: Now I have comprehensive information
Final Answer: Based on the retrieved context, here's what I found...

Remember:
- Always cite sources when referencing retrieved information
- If no relevant information is found, say so clearly
- Don't make up information beyond what's in the context
- Use multiple tools if needed to gather comprehensive information
`
```

---

### 4. Agent Use Case (`internal/core/application/usecases/agent/agent_usecase.go`)

```go
type AgentUseCase struct {
    config           *AgentConfig
    toolRegistry     *ToolRegistry
    historyManager   *HistoryManager
    executionRepo    repositories.AgentExecutionRepository
}

type AgentRequest struct {
    NotebookID        *string
    Input             interface{}
    PreviousResponseID *string
    SourceIDs         []string
    SourceTypes       []string
    MaxIterations     int  // Default: 5
    EnableStreaming   bool
}

type AgentResponse struct {
    ResponseID       string
    Content          string
    ToolInvocations  []ToolInvocation
    ReasoningSteps   []ReasoningStep
    Status           string
    Metadata         map[string]string
}

func (uc *AgentUseCase) ExecuteAgent(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
    // 1. Load conversation history
    history, _ := uc.loadConversationHistory(ctx, req.PreviousResponseID)

    // 2. Configure agent with tools
    tools := uc.toolRegistry.GetTools()

    // 3. Create ChatModelAgent with ReAct prompt
    agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        ChatModel: uc.config.ChatModel,
        Tools:     tools,
        Prompt:    prompt.FromTemplate(ReActPromptTemplate),
    })

    // 4. Create runner with event handling
    runner := adk.NewAgentRunner(ctx, agent, &adk.AgentRunnerConfig{
        MaxIterations: req.MaxIterations,
        OnEvent:      uc.config.EventHandler.HandleEvent,
    })

    // 5. Execute agent
    result, err := runner.Run(ctx, req.Input, history)

    // 6. Log execution
    uc.logExecution(ctx, req, result)

    return uc.buildResponse(result), nil
}

func (uc *AgentUseCase) ExecuteAgentStream(ctx context.Context, req *AgentRequest) (io.ReadCloser, error) {
    // Similar to ExecuteAgent but with SSE streaming
    pr, pw := io.Pipe()

    go func() {
        defer pw.Close()

        // Stream agent events via SSE
        runner := adk.NewAgentRunner(ctx, agent, &adk.AgentRunnerConfig{
            MaxIterations: req.MaxIterations,
            OnEvent: func(ctx context.Context, event *adk.AgentEvent) {
                uc.sendStreamingEvent(pw, event)
            },
        })

        runner.RunStream(ctx, req.Input, history)
    }()

    return pr, nil
}
```

---

### 5. Event System (`internal/core/application/usecases/agent/events/`)

#### 5.1 Event Types (`agent_events.go`)

```go
type AgentEventType string

const (
    EventAgentStarted      AgentEventType = "agent.started"
    EventToolInvoking      AgentEventType = "tool.invoking"
    EventToolInvoked       AgentEventType = "tool.invoked"
    EventReasoning         AgentEventType = "agent.reasoning"
    EventIterationComplete AgentEventType = "iteration.complete"
    EventAgentCompleted    AgentEventType = "agent.completed"
    EventAgentFailed       AgentEventType = "agent.failed"
)

type AgentEvent struct {
    Type        AgentEventType `json:"type"`
    Timestamp   int64          `json:"timestamp"`
    SequenceNum int            `json:"sequence_number"`
    Data        interface{}    `json:"data"`
}

type ToolInvokingEvent struct {
    ToolName string                 `json:"tool_name"`
    Input    map[string]interface{} `json:"input"`
}

type ToolInvokedEvent struct {
    ToolName string                 `json:"tool_name"`
    Input    map[string]interface{} `json:"input"`
    Output   interface{}            `json:"output"`
    Error    string                 `json:"error,omitempty"`
    Duration int64                  `json:"duration_ms"`
}
```

#### 5.2 Event Handler (`event_handler.go`)

```go
type AgentEventHandler interface {
    HandleEvent(ctx context.Context, event *adk.AgentEvent) error
}

type StreamingEventHandler struct {
    writer io.Writer
}

func (h *StreamingEventHandler) HandleEvent(ctx context.Context, event *adk.AgentEvent) error {
    // Convert to SSE format and send
    sseEvent := uc.convertToSSEEvent(event)
    data, _ := json.Marshal(sseEvent)
    fmt.Fprintf(h.writer, "data: %s\n\n", string(data))
    return nil
}
```

---

### 6. Domain Entities

#### 6.1 Agent Execution (`internal/core/domain/entities/agent_execution.go`)

```go
type AgentExecution struct {
    ID                string
    NotebookID        *string
    RequestID         string
    Status            string  // "running", "completed", "failed"
    Iterations        int
    ToolInvocations   []ToolInvocation
    ReasoningSteps    []ReasoningStep
    FinalResponse     string
    Error             string
    DurationMs        int64
    TokensUsed        int
    CreatedAt         int64
    CompletedAt       *int64
}

type ToolInvocation struct {
    ToolName    string
    Input       interface{}
    Output      interface{}
    Error       string
    DurationMs  int64
    Timestamp   int64
}

type ReasoningStep struct {
    StepNumber int
    Thought    string
    Action     string
    Observation string
    Timestamp  int64
}
```

---

### 7. Repository Layer

#### 7.1 Repository Interface (`internal/core/domain/repositories/agent_execution_repository.go`)

```go
type AgentExecutionRepository interface {
    Save(ctx context.Context, execution *entities.AgentExecution) error
    FindByID(ctx context.Context, id string) (*entities.AgentExecution, error)
    FindByNotebookID(ctx context.Context, notebookID string, limit int) ([]*entities.AgentExecution, error)
    FindByRequestID(ctx context.Context, requestID string) (*entities.AgentExecution, error)
}
```

#### 7.2 Implementation (`internal/infrastructure/persistence/agent_execution.go`)

```go
type agentExecutionRepository struct {
    pool *pgxpool.Pool
}

func (r *agentExecutionRepository) Save(ctx context.Context, execution *entities.AgentExecution) error {
    query := `
        INSERT INTO agent_executions (
            id, notebook_id, request_id, status, iterations,
            tool_invocations, reasoning_steps, final_response,
            error, duration_ms, tokens_used, created_at, completed_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
    `
    // Implementation...
}
```

---

## Integration Points

### 1. Response Use Case Integration

Modify `internal/core/application/usecases/response/response_usecase.go`:

```go
type responseUseCase struct {
    // Existing fields...
    agentUseCase chat.AgentResponseUseCase  // New field
}

type ResponseRequest struct {
    // Existing fields...
    UseAgent     bool   `json:"use_agent,omitempty"`     // Enable agent mode
    MaxIterations int    `json:"max_iterations,omitempty"` // Max reasoning iterations
}

func (uc *responseUseCase) CreateResponse(ctx context.Context, req *dtos.ResponseRequest) (*dtos.ResponseResource, error) {
    // If agent mode is enabled, delegate to agent
    if req.UseAgent {
        return uc.agentUseCase.ExecuteAgent(ctx, &agent.AgentRequest{
            NotebookID:        req.NotebookID,
            Input:             req.Input,
            PreviousResponseID: req.PreviousResponseID,
            SourceIDs:         req.SourceIDs,
            SourceTypes:       req.SourceTypes,
            MaxIterations:     req.MaxIterations,
            EnableStreaming:   false,
        })
    }

    // Otherwise, use existing simple chain
    // ... existing code ...
}
```

---

## Database Migration

Create migration file: `migrations/000010_add_agent_executions_table.up.sql`

```sql
CREATE TABLE IF NOT EXISTS agent_executions (
    id TEXT PRIMARY KEY,
    notebook_id TEXT,
    request_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('running', 'completed', 'failed')),
    iterations INTEGER NOT NULL DEFAULT 0,
    tool_invocations JSONB,
    reasoning_steps JSONB,
    final_response TEXT,
    error TEXT,
    duration_ms BIGINT,
    tokens_used INTEGER,
    created_at BIGINT NOT NULL,
    completed_at BIGINT,

    FOREIGN KEY (notebook_id) REFERENCES notebooks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_executions_notebook_id ON agent_executions(notebook_id);
CREATE INDEX IF NOT EXISTS idx_agent_executions_request_id ON agent_executions(request_id);
CREATE INDEX IF NOT EXISTS idx_agent_executions_created_at ON agent_executions(created_at DESC);
```

---

## Acceptance Criteria

### Phase 1: Core Agent Infrastructure
- [ ] Agent configuration and factory implemented
- [ ] Tool registry with retrieval, search, and context tools
- [ ] ReAct prompt template configured
- [ ] Agent use case with non-streaming execution
- [ ] Unit tests for tool implementations
- [ ] Integration tests for agent execution

### Phase 2: Event Streaming
- [ ] Event types defined and documented
- [ ] Streaming event handler implemented
- [ ] SSE integration for agent events
- [ ] Tool invocation events streamed
- [ ] Reasoning step events streamed
- [ ] Error handling and recovery

### Phase 3: Persistence & Monitoring
- [ ] Agent execution repository implemented
- [ ] Database migration created and run
- [ ] Execution tracking for all agent runs
- [ ] Tool invocation logging
- [ ] Performance metrics (duration, tokens)
- [ ] Error tracking

### Phase 4: Integration & Testing
- [ ] Response use case integration complete
- [ ] HTTP endpoints wired up
- [ ] End-to-end tests passing
- [ ] Performance benchmarks (agent vs. simple chain)
- [ ] Documentation complete
- [ ] Example queries demonstrating ReAct behavior

---

## Risks and Mitigations

### Risk 1: Agent Loop/Infinite Iteration
**Mitigation:**
- Enforce `MaxIterations` limit (default: 5)
- Implement timeout (default: 60 seconds)
- Monitor for repetitive tool calls

### Risk 2: Tool Execution Failures
**Mitigation:**
- Graceful error handling in tool implementations
- Fallback to simple chain on repeated failures
- Detailed error logging for debugging

### Risk 3: High Token Usage
**Mitigation:**
- Token estimation before execution
- Configurable iteration limits
- Cost monitoring and alerts

### Risk 4: Streaming Complexity
**Mitigation:**
- Clear event schema documentation
- Robust error handling in streams
- Client-side reconnection logic

### Risk 5: ReAct Prompt Misalignment
**Mitigation:**
- A/B test different prompt templates
- Monitor reasoning quality
- Allow custom prompt injection

---

## Testing Strategy

### Unit Tests
- Tool implementations (retrieval, search, context)
- Prompt template generation
- Event conversion and formatting
- Entity mappings and validations

### Integration Tests
- Agent execution with mock tools
- Tool invocation flow
- Event streaming end-to-end
- Repository operations

### E2E Tests
- Complete agent request flow
- Multi-turn conversations with agent
- Error recovery scenarios
- Performance benchmarks

---

## Performance Considerations

### Optimization Targets
1. **Tool Latency:** Keep retrieval under 500ms
2. **Agent Overhead:** Limit ReAct loop to < 5 seconds total
3. **Token Efficiency:** Reasoning should be concise
4. **Memory Usage:** Stream events, don't buffer

### Monitoring Metrics
- Average iterations per query
- Tool invocation success rate
- Token usage per agent run
- Latency percentiles (p50, p95, p99)

---

## Implementation Phases

### Phase 1: Foundation (Week 1)
1. Create file structure and base entities
2. Implement tool registry and basic tools
3. Create ReAct prompt templates
4. Set up agent configuration

### Phase 2: Core Logic (Week 2)
1. Implement agent use case (non-streaming)
2. Add event types and handlers
3. Integrate with response use case
4. Unit and integration tests

### Phase 3: Streaming (Week 3)
1. Implement SSE event streaming
2. Add tool invocation events
3. Add reasoning step events
4. Error handling and recovery

### Phase 4: Production Readiness (Week 4)
1. Add persistence layer
2. Implement monitoring and logging
3. Performance optimization
4. Documentation and examples

---

## Success Metrics

- [ ] Agent can successfully retrieve and reason about context
- [ ] Multi-step reasoning visible in event stream
- [ ] Tool invocations logged and trackable
- [ ] End-to-end latency < 5 seconds for typical queries
- [ ] Token usage within 2x of simple chain
- [ ] Zero data loss in streaming scenarios
- [ ] All tests passing with > 80% coverage

---

## Open Questions

1. **Prompt Customization:** Should users be able to customize the ReAct prompt template?
   - Decision: Defer to Phase 2 - allow prompt injection via config

2. **Tool Granularity:** Are 3 tools sufficient or should we add more specialized tools?
   - Decision: Start with 3, add based on usage patterns

3. **Caching:** Should tool results be cached to reduce duplicate calls?
   - Decision: Add in Phase 4 based on performance data

4. **Concurrent Tools:** Should the agent be able to call multiple tools in parallel?
   - Decision: Defer - requires more complex prompt engineering

5. **Cost Controls:** Should we add per-request token limits?
   - Decision: Add in Phase 4 as a safety measure

---

## References

- Eino ADK Documentation: https://github.com/cloudwego/eino/tree/main/adk
- ReAct Paper: "ReAct: Synergizing Reasoning and Acting in Language Models"
- Current RAG Implementation: `/internal/core/application/usecases/response/response_usecase.go`
- Retrieval Components: `/pkg/retriever/pgvector/`