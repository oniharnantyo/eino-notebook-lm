## 1. Database Migration

- [ ] 1.1 Create `migrations/018_messages_per_turn.up.sql` — rename `message` column to `messages`, set default to `'[]'::jsonb`
- [ ] 1.2 Create `migrations/018_messages_per_turn.down.sql` — reverse rename `messages` to `message`, restore default

## 2. Entity Layer

- [ ] 2.1 Update `internal/core/domain/entities/message.go` — change `Message *StoredMessage` to `Messages []*StoredMessage`, update `NewMessage()` constructor signature

## 3. Persistence Layer

- [ ] 3.1 Update `internal/infrastructure/persistence/conversation.go` `Save()` — marshal `msg.Messages` (array) instead of `msg.Message`
- [ ] 3.2 Update `internal/infrastructure/persistence/conversation.go` `GetMessages()` — unmarshal into `msg.Messages` (array)

## 4. Middleware Layer

- [ ] 4.1 Update `internal/adk/middleware/conversation_memory.go` `buildConversation()` — group all pending messages into single `entities.Message` with `Messages []*StoredMessage` array
- [ ] 4.2 Update `internal/adk/middleware/conversation_memory.go` `BeforeModelRewriteState()` — iterate over `.Messages` arrays instead of single `.Message`
- [ ] 4.3 Update token aggregation in `buildConversation()` — use last output message's token usage for row-level metadata

## 5. DTO & API Layer

- [ ] 5.1 Update `internal/core/application/dtos/message.go` — change `MessageResponse.Message` to `Messages []map[string]any`, update `ToMessageResponse()` mapping

## 6. History Pipeline

- [ ] 6.1 Update `internal/core/application/usecases/response/stages/history_stage.go` — flatten turn arrays when building `fullHistory`

## 7. Mocks & Tests

- [ ] 7.1 Update `internal/mocks/repositories/conversation_repository.go` — reflect field access changes
- [ ] 7.2 Update test fixtures in `internal/adk/middleware/conversation_memory_test.go`
- [ ] 7.3 Update test fixtures in `internal/adk/middleware/conversation_memory_integration_test.go`
- [ ] 7.4 Update test fixtures in `internal/interfaces/http/handlers/conversation_test.go`

## 8. Verification

- [ ] 8.1 Run `make build` — confirm zero compilation errors
- [ ] 8.2 Run `make test` — confirm all tests pass
- [ ] 8.3 Run migration and verify schema with `SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'messages'`
- [ ] 8.4 End-to-end test — send chat request, verify 1 row per turn in `messages` table, verify history loads correctly
