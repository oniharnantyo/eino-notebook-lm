## 1. Agent Package Changes

- [x] 1.1 Restructure `BaseAgentInstruction` into 6 sections: Identity & Values, Safety & Ethics, Knowledge & Facts (with `{catalog}`), Tools & Products, Behavioral Guidance, Style & Tone
- [x] 1.2 Add `BuildCatalog` function to `internal/core/application/agent/agent.go`
- [x] 1.3 Add imports for `repositories`, `strings`, `fmt`, and `uuid` packages in agent.go
- [x] 1.4 Remove `PrepareInstruction` function from agent.go
- [x] 1.5 Create `internal/core/application/agent/catalog_test.go` with BuildCatalog tests
- [x] 1.6 Run tests for agent package to verify BuildCatalog behavior

## 2. Agent Stage Updates

- [x] 2.1 Update `AgentStage.Execute` signature to accept `sourceIDs []uuid.UUID` instead of `catalog string`
- [x] 2.2 Replace `agent.PrepareInstruction(catalog)` call with `agent.BuildCatalog(ctx, s.sourceRepo, sourceIDs)`
- [x] 2.3 Update `runner.Query` to pass `catalog` via session values instead of `instruction`
- [x] 2.4 Update `AgentStage` constructor if needed (remove baseInstruction field if unused)
- [x] 2.5 Update `internal/core/application/usecases/response/stages/agent_stage_test.go`

## 3. Pipeline Updates

- [x] 3.1 Update `AgentStage` interface in `internal/core/application/usecases/response/pipeline.go`
- [x] 3.2 Change `ResponsePipeline.Execute` signature to accept `sourceIDs []uuid.UUID` instead of `catalog string`
- [x] 3.3 Update agent stage call from `p.agentStage.Execute(ctx, msg, catalog, tools)` to `p.agentStage.Execute(ctx, msg, req.SourceIDs, tools)`
- [x] 3.4 Update `internal/core/application/usecases/response/pipeline_test.go`

## 4. Response UseCase Updates

- [x] 4.1 Remove `buildSourceCatalog` call from `GenerateResponse` method
- [x] 4.2 Update pipeline call to pass `req.SourceIDs` instead of `catalog`
- [x] 4.3 Remove unused imports if any (sources fetching, catalog building)
- [x] 4.4 Update `internal/core/application/usecases/response/response_usecase_test.go`

## 5. Cleanup

- [x] 5.1 Delete `internal/core/application/usecases/response/catalog_builder.go`
- [x] 5.2 Delete `internal/core/application/usecases/response/catalog_builder_test.go`
- [x] 5.3 Verify no remaining references to `buildSourceCatalog` in codebase
- [x] 5.4 Verify no remaining references to `PrepareInstruction` in codebase

## 6. Verification

- [x] 6.1 Run full test suite: `make test`
- [x] 6.2 Run linter: `make lint`
- [x] 6.3 Build application: `make build`
- [x] 6.4 Manual test: start server and verify agent queries work with dynamic source selection
