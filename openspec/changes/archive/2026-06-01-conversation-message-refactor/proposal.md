# Proposal: Conversation Message Refactor

## Why

Current conversation storage stores all messages in a single JSONB column within the conversations table, preventing efficient pagination. Chat interfaces need to display the latest 10-20 messages and load older messages on scroll up, but the current structure requires fetching entire conversation histories. This refactor normalizes message storage to enable efficient cursor-based pagination for conversations with 50-500 messages.

## What Changes

### Database Schema
- **Create** `messages` table with normalized message storage:
  - Columns: id, conversation_id, sequence_num, response_id, previous_response_id, message (JSONB), model, finish_reason, prompt_tokens, completion_tokens, total_tokens, created_at
  - Indexes on (conversation_id, sequence_num DESC) for pagination, response_id for turn lookup
- **Simplify** `conversations` table to session metadata only:
  - Columns: id, notebook_id, metadata, created_at
- **Drop** redundant columns from conversations: messages, response_id, previous_response_id, response_text, response_message, model, finish_reason, prompt_tokens, completion_tokens, total_tokens

### Application Changes
- **BREAKING**: Update `Conversation` entity (remove Messages field, remove response-related fields)
- **BREAKING**: Update `ConversationRepository.Save()` to write conversation + messages separately
- **Add** `ConversationRepository.GetMessages()` with cursor pagination (beforeSequence parameter)
- **Update** `conversation_memory` middleware to fetch messages from new repository method
- **Add** API endpoint: `GET /api/v1/notebooks/{id}/conversations/messages` with pagination support

## Capabilities

### New Capabilities
- `conversation-pagination`: Paginated message retrieval with reverse cursor (latest-first, load older on scroll)

### Modified Capabilities
- `conversations`: Core conversation storage mechanism changes from JSONB to normalized table structure

## Impact

### Database
- Migration creates `messages` table and simplifies `conversations` table
- Existing conversations incompatible (no data migration - fresh start acceptable)

### Code Changes
- **Entities**: `internal/core/domain/entities/conversation.go` - remove Messages, response_id, previous_response_id, model, tokens fields
- **Repository**: `internal/infrastructure/persistence/conversation.go` - rewrite Save(), add GetMessages()
- **Middleware**: `internal/adk/middleware/conversation_memory.go` - update to use GetMessages() with pagination
- **Handlers**: `internal/interfaces/http/handlers/conversation.go` - add GetMessages endpoint

### API
- **New**: `GET /api/v1/notebooks/{notebookId}/conversations/messages?conversation_id={optional}&limit={default:10}&before_sequence={optional}`
- Response includes messages array + pagination metadata (has_more, oldest_sequence)