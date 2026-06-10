## Why

The `messages` table stores each individual message (user, assistant, tool) as a separate row. A single agentic RAG turn creates ~5-15 rows with the same `response_id`, causing unnecessary row bloat and complicating queries. Grouping all messages from one agent turn into a single JSONB array row reduces storage by ~5-15x and aligns the data model with the natural turn-based structure of conversations.

## What Changes

- **BREAKING**: Rename `message` column to `messages` and change from single `StoredMessage` JSONB object to `[]*StoredMessage` JSONB array
- **BREAKING**: `sequence_num` changes from per-message ordering to per-turn ordering (1 row per agent turn)
- Entity `Message.Message *StoredMessage` → `Message.Messages []*StoredMessage`
- `buildConversation()` in conversation memory middleware creates 1 `entities.Message` per turn instead of N
- `GetMessages` consumers flatten turn arrays when reconstructing full history
- HTTP API response changes `message` (object) → `messages` (array) in message endpoints
- Token metadata at row level represents the final model call's usage per turn
- Database migration: `ALTER TABLE messages RENAME COLUMN message TO messages` (existing data truncated manually)

## Capabilities

### New Capabilities
- `turn-based-message-storage`: Consolidated per-turn message storage with JSONB array, replacing per-message row model

### Modified Capabilities

## Impact

**Core entities**: `internal/core/domain/entities/message.go` — field type change
**Persistence**: `internal/infrastructure/persistence/conversation.go` — marshal/unmarshal array
**Middleware**: `internal/adk/middleware/conversation_memory.go` — buildConversation grouping, BeforeModelRewriteState array iteration
**DTOs**: `internal/core/application/dtos/message.go` — response shape change (Message → Messages)
**History**: `internal/core/application/usecases/response/stages/history_stage.go` — flatten arrays
**API**: **BREAKING** — `/messages` endpoint response field changes from `message` (object) to `messages` (array)
**Database**: Migration 018 — column rename, data truncation required before applying
**Tests**: 3 test files need fixture updates
