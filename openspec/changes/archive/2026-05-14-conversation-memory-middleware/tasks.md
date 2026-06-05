## 1. Database Migration

- [ ] 1.1 Create migration file for metadata columns
- [ ] 1.2 Add `finish_reason`, `prompt_tokens`, `completion_tokens`, `total_tokens` columns
- [ ] 1.3 Add indexes for metadata columns
- [ ] 1.4 Test migration up/down

## 2. Domain Entity Updates

- [ ] 2.1 Add metadata fields to `entities.Conversation` struct
- [ ] 2.2 Update `NewConversation()` constructor with metadata parameters
- [ ] 2.3 Ensure JSON serialization handles new fields correctly

## 3. Repository Layer Updates

- [ ] 3.1 Update `Save()` INSERT query to include metadata columns
- [ ] 3.2 Update `Save()` ON CONFLICT UPDATE clause for metadata columns
- [ ] 3.3 Update `Save()` error handling for new columns
- [ ] 3.4 Update `FindByResponseID()` to scan new metadata columns
- [ ] 3.5 Update `scanConversation()` helper to include metadata fields

## 4. Middleware Implementation

- [ ] 4.1 Create `internal/adk/middleware/` directory
- [ ] 4.2 Implement `ConversationMemoryMiddleware` struct
- [ ] 4.3 Implement `Handle()` method with event type switching
- [ ] 4.4 Implement `handleBeforeModel()` for history loading
- [ ] 4.5 Implement `getPreviousResponseID()` helper
- [ ] 4.6 Implement `handleAfterAgent()` for async save trigger
- [ ] 4.7 Implement `saveAsync()` with timeout and error logging
- [ ] 4.8 Implement `buildConversation()` from agent event
- [ ] 4.9 Implement metadata extractors (`extractFinishReason()`, `extractPromptTokens()`, etc.)

## 5. Dependency Injection Wiring

- [ ] 5.1 Create `NewConversationMemory()` constructor function
- [ ] 5.2 Wire middleware in `cmd/serve.go` ADK Runner config
- [ ] 5.3 Pass `conversationRepo` and `logger` to middleware
- [ ] 5.4 Verify middleware is registered in ADK Runner

## 6. Testing

- [ ] 6.1 Write unit tests for middleware `Handle()` method
- [ ] 6.2 Write unit tests for `handleBeforeModel()` with/without history
- [ ] 6.3 Write unit tests for `handleAfterAgent()` async save
- [ ] 6.4 Write unit tests for metadata extraction functions
- [ ] 6.5 Write integration test for full middleware flow
- [ ] 6.6 Test conversation loading with threading
- [ ] 6.7 Test async save with timeout scenarios
- [ ] 6.8 Test error logging on save failure

## 7. Cleanup (After Verification)

- [ ] 7.1 Remove `HistorySavingReader` from `ResponseUseCase`
- [ ] 7.2 Simplify `HistoryStage` to read-only (remove Save method)
- [ ] 7.3 Delete `internal/core/application/usecases/response/history_saving_reader.go`
- [ ] 7.4 Update `ResponseUseCase` to remove history accumulation logic
- [ ] 7.5 Run full test suite to ensure no regressions
- [ ] 7.6 Manual test: verify conversation persistence works end-to-end
