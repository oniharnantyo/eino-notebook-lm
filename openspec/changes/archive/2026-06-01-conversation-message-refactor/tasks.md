# Tasks: Conversation Message Refactor

## 1. Database Migration

- [x] 1.1 Review and validate migration SQL (`migrations/017_normalize_messages_table.up.sql`)
- [x] 1.2 Test migration locally with `make test` to ensure no schema conflicts
- [x] 1.3 Run database migration in development environment
- [x] 1.4 Verify messages table created with correct indexes
- [x] 1.5 Verify conversations table columns dropped correctly

## 2. Entity Layer Updates

- [x] 2.1 Update `Conversation` entity struct (`internal/core/domain/entities/conversation.go`)
  - Remove `Messages` field
  - Remove `ResponseID` field
  - Remove `PreviousResponseID` field  
  - Remove `Model` field
  - Remove `FinishReason` field
  - Remove `PromptTokens` field
  - Remove `CompletionTokens` field
  - Remove `TotalTokens` field
- [x] 2.2 Update `NewConversation` constructor signature to remove deleted parameters
- [x] 2.3 Verify no remaining references to deleted fields in entity code

## 3. Repository Interface Changes

- [x] 3.1 Update `ConversationRepository` interface (`internal/core/domain/repositories/conversation.go`)
  - Add `GetMessages(ctx, conversationID, pagination) ([]*StoredMessage, error)` method signature
  - Add `GetLatestConversationID(ctx, notebookID) (string, error)` method signature
  - Update `Save(ctx, conversation, messages) error` signature to accept messages
- [x] 3.2 Update `ConversationFilter` struct if needed for pagination parameters

## 4. Repository Implementation

- [x] 4.1 Rewrite `Save()` method (`internal/infrastructure/persistence/conversation.go`)
  - Insert conversation row (id, notebook_id, metadata, created_at)
  - Loop through messages and insert each row with sequence_num
  - Use transaction for atomicity
- [x] 4.2 Implement `GetMessages()` method
  - Query messages table with pagination (conversation_id, limit, beforeSequence)
  - Handle default to latest conversation when conversationID is empty
  - Parse JSONB message content into StoredMessage structs
  - Return pagination metadata (has_more, oldest_sequence)
- [x] 4.3 Implement `GetLatestConversationID()` method
  - Query conversations table by notebook_id, order by created_at DESC, limit 1
- [x] 4.4 Update `FindByResponseID()` to not load messages (return nil Messages slice)

## 5. Message Entity

- [x] 5.1 Add `Message` entity struct (`internal/core/domain/entities/message.go`)
  - Fields: ID, ConversationID, SequenceNum, ResponseID, PreviousResponseID, Message (JSONB), CreatedAt
  - Fields: Model, FinishReason, PromptTokens, CompletionTokens, TotalTokens
- [x] 5.2 Add `ToStoredMessage()` method to convert Message to StoredMessage
- [x] 5.3 Add `MessageRepository` interface if needed for separate message queries

## 6. Middleware Updates

- [x] 6.1 Update `conversation_memory` middleware (`internal/adk/middleware/conversation_memory.go`)
  - Change from `conv.Messages` to `repo.GetMessages()` call
  - Pass pagination parameters (limit, beforeSequence)
  - Update history loading logic for reverse pagination
- [x] 6.2 Update middleware to handle `GetLatestConversationID()` when no conversation exists
- [x] 6.3 Verify token counting still works with new structure

## 7. API Handler

- [x] 7.1 Add `GetMessages()` handler to `ConversationHandler` (`internal/interfaces/http/handlers/conversation.go`)
  - Parse query parameters: conversation_id, limit, before_sequence
  - Call use case to fetch messages
  - Return JSON response with messages array + pagination metadata
- [x] 7.2 Add route registration (`internal/interfaces/http/routes/routes.go`)
  - `GET /api/v1/notebooks/{notebookId}/conversations/messages`
- [x] 7.3 Update response DTO to include pagination structure

## 8. Use Case Layer

- [x] 8.1 Update response use case to pass messages to repository Save() method
- [x] 8.2 Add pagination parameters to message loading use case if needed
- [x] 8.3 Ensure use case handles empty message list gracefully

## 9. Testing

- [x] 9.1 Update conversation repository tests (`internal/infrastructure/persistence/*_test.go`)
  - Test Save() with transaction
  - Test GetMessages() pagination
  - Test GetMessages() default to latest conversation
- [x] 9.2 Update middleware tests (`internal/adk/middleware/conversation_memory_integration_test.go`)
  - Test pagination with beforeSequence cursor
  - Test loading latest conversation when conversation_id is null
- [x] 9.3 Add integration test for GET /conversations/messages endpoint
- [x] 9.4 Run full test suite with `make test`

## 10. Documentation

- [x] 10.1 Update API documentation for new pagination endpoint
- [x] 10.2 Document migration steps in README if needed
- [x] 10.3 Update CLAUDE.md if conversation-related conventions changed

## 11. Verification

- [x] 11.1 Test fresh chat creation flow
- [x] 11.2 Test conversation continuation (follow-up messages)
- [x] 11.3 Test message pagination (load latest, load more)
- [x] 11.4 Verify conversation threading works with previous_response_id
- [x] 11.5 Test conversation memory middleware with real database