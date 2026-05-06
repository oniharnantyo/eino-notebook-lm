## 1. Phase 1: Create Pipeline Module

- [x] 1.1 Create `internal/core/application/usecases/pipeline/` package directory
- [x] 1.2 Define `Stage` interface with `Name()` and `Execute()` methods
- [x] 1.3 Define `StageInput` and `StageOutput` structs with typed fields
- [x] 1.4 Define `Progress` struct with Stage, Status, Error, and Metadata fields
- [x] 1.5 Define `IngestionPipeline` struct with stages slice and parallelism config
- [x] 1.6 Implement `NewIngestionPipeline()` constructor with validation
- [x] 1.7 Implement `Ingest()` method returning read-only progress channel
- [x] 1.8 Add goroutine with `defer close()` for progress channel lifecycle
- [x] 1.9 Implement stage execution loop with context cancellation check
- [x] 1.10 Add progress updates after each stage (in_progress, completed, failed)
- [x] 1.11 Write unit test for pipeline with mock stages
- [x] 1.12 Write unit test for context cancellation
- [x] 1.13 Write unit test for progress channel lifecycle

## 2. Phase 2: Create Stage Adapters

- [x] 2.1 Create `ExtractionStage` adapter wrapping `extractor.ContentExtractor`
- [x] 2.2 Implement `ExtractionStage.Execute()` delegating to `ExtractContent()`
- [x] 2.3 Write unit test for `ExtractionStage` with mock extractor
- [x] 2.4 Create `ParsingStage` adapter wrapping `document.DocumentParser`
- [x] 2.5 Implement `ParsingStage.Execute()` delegating to parser
- [x] 2.6 Write unit test for `ParsingStage` with mock parser
- [x] 2.7 Create `ChunkingStage` adapter wrapping chunking logic
- [x] 2.8 Implement `ChunkingStage.Execute()` with token limit logic
- [x] 2.9 Write unit test for `ChunkingStage` with various input sizes
- [x] 2.10 Create `EmbeddingStage` adapter with batch processing
- [x] 2.11 Implement parallel embedding with configurable batch size
- [x] 2.12 Write unit test for `EmbeddingStage` batch processing
- [x] 2.13 Create `StorageStage` adapter with transaction handling
- [x] 2.14 Implement `StorageStage.Execute()` with tx.Begin/Commit/Rollback
- [x] 2.15 Write unit test for `StorageStage` transaction rollback on error
- [x] 2.16 Create `StatusUpdateStage` adapter for source status
- [x] 2.17 Implement `StatusUpdateStage.Execute()` to update source entity
- [x] 2.18 Write unit test for `StatusUpdateStage` success and failure paths

## 3. Phase 3: Integration Testing

- [x] 3.1 Create integration test for full pipeline with mock dependencies
- [x] 3.2 Test successful ingestion flow through all stages
- [x] 3.3 Test pipeline failure at each stage
- [x] 3.4 Test progress updates are sent in correct order
- [x] 3.5 Test parallelism configuration with concurrent stages
- [ ] 3.6 Benchmark pipeline performance vs. current implementation

## 4. Phase 4: Migrate HTTP Handlers

- [x] 4.1 Update `SourceHandler` to use `IngestionPipeline` instead of `processAsync`/`processSync`
- [x] 4.2 Replace status polling with progress channel consumption
- [x] 4.3 Update handler to handle progress updates for streaming response
- [x] 4.4 Add deprecation comments to `processAsync` and `processSync` methods
- [x] 4.5 Update `source/usecase.go` to instantiate `IngestionPipeline` with all stages
- [x] 4.6 Write integration test for HTTP handler with pipeline
- [x] 4.7 Verify existing tests still pass after migration

## 5. Phase 5: Cleanup and Documentation

- [x] 5.1 Add package documentation for `pipeline` package
- [x] 5.2 Add examples in comments for `IngestionPipeline` usage
- [x] 5.3 Update CLAUDE.md with pipeline pattern guidance
- [x] 5.4 Run `make lint` and fix any issues
- [x] 5.5 Run `make test` and ensure all tests pass
- [x] 5.6 Create git commit for Phase 1 (pipeline module)
- [x] 5.7 Create git commit for Phase 2 (stage adapters)
- [x] 5.8 Create git commit for Phase 4 (handler migration)
