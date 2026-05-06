## 1. Repository Layer

- [ ] 1.1 Add `GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.Source, error)` to `SourceRepository` interface
- [ ] 1.2 Implement `GetByIDs` in `PostgresSourceRepository` using `SELECT * FROM sources WHERE id = ANY($1)`
- [ ] 1.3 Add test for `GetByIDs` with multiple IDs, empty result, and partial matches

## 2. Agent Package Changes

- [ ] 2.1 Rename `AgentInstruction` to `BaseAgentInstruction` in `internal/core/application/agent/instruction.go`
- [ ] 2.2 Update `NewRetrievalAgent` signature to accept `instruction string` parameter
- [ ] 2.3 Replace `Instruction: AgentInstruction` with `Instruction: instruction` in agent creation

## 3. list_sources Tool

- [ ] 3.1 Create `internal/core/application/agent/tools/list_sources.go`
- [ ] 3.2 Define `ListSourcesOutput` and `SourceDetail` structs matching spec
- [ ] 3.3 Implement tool using `utils.InferTool` with no input parameters
- [ ] 3.4 Add `IContextTracker` interface for `ToolFactory` dependency injection
- [ ] 3.5 Wire `sourceRepo` and `sourceIDs` into tool via factory or closure

## 4. ToolFactory Update

- [ ] 4.1 Add `sourceRepo repositories.SourceRepository` field to `ToolFactory` struct
- [ ] 4.2 Update `NewToolFactory` constructor to accept `sourceRepo` parameter
- [ ] 4.3 Add `SourceIDs []uuid.UUID` field to `ScopeConfig`
- [ ] 4.4 Create `NewListSourcesTool` method that accepts `sourceRepo` and `sourceIDs`
- [ ] 4.5 Add `list_sources` tool to `NewScopedTools` return slice

## 5. Response UseCase Changes

- [ ] 5.1 Add `sourceRepo repositories.SourceRepository` field to `responseUseCase` struct
- [ ] 5.2 Update `NewResponseUseCase` constructor to accept `sourceRepo` parameter
- [ ] 5.3 Implement `buildSourceCatalogInstruction(sources []*entities.Source) string` function
- [ ] 5.4 In `CreateResponseStream`, fetch sources via `sourceRepo.GetByIDs(ctx, sourceUUIDs)`
- [ ] 5.5 Build instruction using `buildSourceCatalogInstruction(sources)`
- [ ] 5.6 Pass instruction to `agent.NewRetrievalAgent(ctx, model, tools, instruction)`
- [ ] 5.7 Update `factory.NewScopedTools` call to include `SourceIDs: sourceUUIDs` in config

## 6. Dependency Injection

- [ ] 6.1 Update `cmd/serve.go` to pass `sourceRepo` to `NewResponseUseCase`
- [ ] 6.2 Update `cmd/serve.go` to pass `sourceRepo` to `NewToolFactory`

## 7. Testing

- [ ] 7.1 Add unit test for `buildSourceCatalogInstruction` with various source states
- [ ] 7.2 Add unit test for `list_sources` tool with mocked repository
- [ ] 7.3 Add integration test for agent creation with source catalog
- [ ] 7.4 Test agent behavior with processing/failed sources
- [ ] 7.5 Verify list_sources tool respects source scope

## 8. Documentation

- [ ] 8.1 Update CLAUDE.md with agent source awareness behavior
- [ ] 8.2 Add example list_sources tool output to documentation
