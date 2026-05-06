## 1. Dependencies and Factory

- [x] 1.1 Add `github.com/cloudwego/eino/adk` to `go.mod`
- [x] 1.2 Run `go mod tidy` to sync dependencies
- [x] 1.3 Add `CreateToolCallingChatModel()` method to `pkg/model/chat_factory.go`
- [x] 1.4 Update `cmd/serve.go` to use `CreateToolCallingChatModel()` for response usecase model creation

## 2. Agent Package

- [x] 2.1 Create `internal/core/application/agent/agent.go` file
- [x] 2.2 Implement `NewRetrievalAgent()` function that creates ChatModelAgent with instruction, model, and tools
- [x] 2.3 Create `internal/core/application/agent/instruction.go` file
- [x] 2.4 Define `BaseAgentInstruction` constant with retrieval agent system prompt template
- [x] 2.5 Write unit tests for `NewRetrievalAgent()` with mocked model and tools

## 3. Agent Stage

- [x] 3.1 Create `internal/core/application/usecases/response/stages/agent_stage.go` file
- [x] 3.2 Define `AgentStage` struct with chatModel, sourceRepo, and baseInstruction
- [x] 3.3 Implement `NewAgentStage()` constructor
- [x] 3.4 Implement `Execute()` method that creates ChatModelAgent per-request and runs via ADK Runner
- [x] 3.5 Implement AsyncIterator consumption and conversion to GenerationOutput format
- [x] 3.6 Write unit tests for AgentStage with mocked agent and runner

## 4. Source Catalog Building

- [x] 4.1 Add `sourceRepo repositories.SourceRepository` field to `responseUseCase` struct
- [x] 4.2 Update `NewResponseUseCase()` constructor to accept `sourceRepo` parameter
- [x] 4.3 Implement `buildSourceCatalog(sources []*entities.Source) string` function in pipeline or usecase
- [x] 4.4 Update `ResponsePipeline.Execute()` to fetch sources via `sourceRepo.ListSourceSummariesByID()`
- [x] 4.5 Pass source catalog to AgentStage for instruction building
- [x] 4.6 Write unit tests for `buildSourceCatalog()` with various source states

## 5. Pipeline Integration

- [x] 5.1 Remove `RetrievalStage` from pipeline initialization in `NewResponsePipeline()`
- [x] 5.2 Replace `GenerationStage` with `AgentStage` in pipeline struct
- [x] 5.3 Update `ResponsePipeline.Execute()` to pass history, tools, and catalog to AgentStage
- [x] 5.4 Remove `RetrievalInput`/`RetrievalOutput` types from `stages/types.go`
- [x] 5.5 Delete `internal/core/application/usecases/response/stages/retrieval_stage.go` file
- [x] 5.6 Update pipeline to pass `GenerationInput` with enriched instruction to AgentStage

## 6. Streaming Conversion

- [x] 6.1 Implement `mapAgentEventToSSE()` function to convert ADK events to Responses API format
- [x] 6.2 Update `AgentStage.Execute()` to return streaming output for streaming requests
- [x] 6.3 Update `CreateResponseStream()` to consume AgentStage streaming output
- [x] 6.4 Ensure `response.output_text.delta` events are emitted correctly for streaming chunks
- [x] 6.5 Ensure `response.completed` event is emitted with final accumulated text
- [x] 6.6 Write integration test for streaming flow with mock agent

## 7. Dependency Injection

- [x] 7.1 Update `cmd/serve.go` to pass `sourceRepo` to `NewResponseUseCase()`
- [x] 7.2 Update tool factory initialization to ensure sourceRepo is passed
- [x] 7.3 Verify all dependencies are wired correctly in serve.go

## 8. Testing

- [x] 8.1 Add integration test for full agent flow (catalog → tools → response)
- [x] 8.2 Test agent with empty source selection
- [x] 8.3 Test agent with single source
- [x] 8.4 Test agent with multiple sources
- [x] 8.5 Test agent with processing/failed source status indicators
- [x] 8.6 Verify `list_sources` tool respects source scope
- [x] 8.7 Test streaming response with tool calls
- [ ] 8.8 Verify Langfuse tracing captures tool calls
- [ ] 8.9 Load test streaming endpoint

## 9. Documentation

- [x] 9.1 Update CLAUDE.md with agent source awareness behavior
- [x] 9.2 Add example list_sources tool output to documentation
- [x] 9.3 Document supported models for agent functionality
- [x] 9.4 Add architecture diagram showing agent flow

## 10. Verification

- [x] 10.1 Run `make test` and ensure all tests pass
- [x] 10.2 Run `make lint` and fix any issues
- [x] 10.3 Run `make build` and verify binary compiles
- [ ] 10.4 Start server and test agent with real query
- [ ] 10.5 Verify streaming responses work correctly
- [ ] 10.6 Check Langfuse dashboard for tool call traces
