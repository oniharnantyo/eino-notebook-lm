## 1. Phase 0: Baseline Test (Safety Net)

- [x] 1.1 Create integration test file for response flow
- [x] 1.2 Test basic response generation with streaming
- [x] 1.3 Test response with tool calls
- [x] 1.4 Test response with history loading/saving
- [x] 1.5 Test different retrieval modes (semantic, keyword, hybrid)
- [x] 1.6 Verify baseline test passes against current code

## 2. Phase 1: Extract Stage Interfaces

- [x] 2.1 Create `internal/core/application/usecases/response/stages/` package
- [x] 2.2 Define `Stage` interface with `Execute()` method
- [x] 2.3 Define `RetrievalInput` and `RetrievalOutput` structs
- [x] 2.4 Define `ToolPreparationInput` and `ToolPreparationOutput` structs
- [x] 2.5 Define `GenerationInput` and `GenerationOutput` structs
- [x] 2.6 Define `HistoryInput` and `HistoryOutput` structs
- [x] 2.7 Create empty stage structs (RetrievalStage, ToolPreparationStage, etc.)
- [x] 2.8 Write interface tests for stage contracts

## 3. Phase 2: Implement Retrieval Stage

- [x] 3.1 Move retrieval logic from `response_usecase.go` to `RetrievalStage`
- [x] 3.2 Implement semantic search in `RetrievalStage.Execute()`
- [x] 3.3 Implement keyword search in `RetrievalStage.Execute()`
- [x] 3.4 Implement hybrid search with RRF fusion
- [x] 3.5 Write unit test for semantic retrieval
- [x] 3.6 Write unit test for keyword retrieval
- [x] 3.7 Write unit test for hybrid retrieval
- [x] 3.8 Write unit test for retrieval errors

## 4. Phase 2: Implement Tool Preparation Stage

- [x] 4.1 Move tool factory logic to `ToolPreparationStage`
- [x] 4.2 Implement tool initialization in `ToolPreparationStage.Execute()`
- [x] 4.3 Implement tool dependency validation
- [x] 4.4 Write unit test for tool preparation success
- [x] 4.5 Write unit test for tool preparation failure

## 5. Phase 2: Implement Generation Stage

- [x] 5.1 Move chat model invocation to `GenerationStage`
- [x] 5.2 Implement streaming response in `GenerationStage.Execute()`
- [x] 5.3 Implement non-streaming response in `GenerationStage.Execute()`
- [x] 5.4 Handle tool calls during generation via Eino framework
- [x] 5.5 Write unit test for streaming generation
- [x] 5.6 Write unit test for non-streaming generation
- [x] 5.7 Write unit test for generation errors

## 6. Phase 2: Implement History Stage

- [x] 6.1 Move history loading logic to `HistoryStage.Execute()`
- [x] 6.2 Move history saving logic to `HistoryStage.Save()`
- [x] 6.3 Implement history loading with token limit
- [x] 6.4 Implement async history saving in goroutine
- [x] 6.5 Write unit test for history loading
- [x] 6.6 Write unit test for history saving

## 7. Phase 3: Create Pipeline Orchestrator

- [x] 7.1 Create `ResponsePipeline` struct with stage dependencies
- [x] 7.2 Implement `NewResponsePipeline()` constructor
- [x] 7.3 Implement `Execute()` method wiring stages together
- [x] 7.4 Implement error propagation across stages
- [x] 7.5 Write unit test for pipeline with all stages
- [x] 7.6 Write unit test for pipeline failure at each stage

## 8. Phase 3: Replace Orchestration in ResponseUsecase

- [x] 8.1 Update `response_usecase.go` to instantiate `ResponsePipeline`
- [x] 8.2 Replace old orchestration with `pipeline.Execute()` call
- [x] 8.3 Update `CreateResponseStream()` to use pipeline
- [x] 8.4 Verify baseline test still passes
- [x] 8.5 Run existing handler tests to verify compatibility

## 9. Phase 4: Cleanup

- [x] 9.1 Check if `agent` package is still used elsewhere
- [x] 9.2 Delete `agent` package if unused
- [x] 9.3 Remove old orchestration code from `response_usecase.go`
- [x] 9.4 Add package documentation for `stages` package
- [x] 9.5 Add stage usage examples in comments
- [x] 9.6 Run `make lint` and fix issues
- [x] 9.7 Run `make test` and ensure all tests pass

## 10. Commits

- [x] 10.1 Create commit for Phase 0 (baseline test)
- [x] 10.2 Create commit for Phase 1 (stage interfaces)
- [x] 10.3 Create commit for Phase 2 (stage implementations)
- [x] 10.4 Create commit for Phase 3 (pipeline orchestrator)
- [x] 10.5 Create commit for Phase 4 (cleanup)
