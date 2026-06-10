## Context

The `messages` table currently stores each individual message (user, assistant, tool) as a separate row with `sequence_num` ordering. A single agentic RAG turn produces ~5-15 rows sharing the same `response_id`. The conversation memory middleware (`buildConversation`) creates N `entities.Message` structs per turn, one per message, and inserts them in a single transaction.

The existing `StoredMessage` JSONB format and `MessageToStoredContent` / `ToEinoMessage` conversion functions work correctly — no changes needed to the message content format itself. Only the grouping of messages into rows changes.

Key constraint: existing data will be truncated manually before migration, so no data aggregation logic is needed.

## Goals / Non-Goals

**Goals:**
- Collapse all messages within a single agent turn into one database row
- Reduce `messages` table row count by ~5-15x
- Maintain backward-compatible `GetMessages` behavior for consumers (return flat history)
- Preserve per-turn metadata (model, finish_reason, token usage)

**Non-Goals:**
- Migrating existing data (manual truncation)
- Changing the `StoredMessage` JSONB content format
- Changing the conversation lookup/resolution logic
- Adding per-message token tracking within the array (future consideration)

## Decisions

### 1. `Message.Messages []*StoredMessage` instead of `Message.Message *StoredMessage`

**Rationale**: Direct field change on the entity. The array naturally represents all messages from one turn. Consumers already iterate over `[]*entities.Message` — adding an inner loop over `.Messages` is minimal code change.

**Alternative considered**: Keep `.Message` field and add a separate `.AllMessages` field. Rejected — two fields for the same conceptual data is confusing and error-prone.

### 2. Token metadata at row level = last model call's usage

**Rationale**: The final model call in a turn (the one with `finish_reason: "stop"`) represents the complete response. Earlier model calls are intermediate tool-calling steps. Storing the last call's tokens gives the most useful aggregate for billing/display.

**Alternative considered**: Sum all model call tokens across the turn. Rejected — would overcount prompt tokens since each call includes full context. The last call already contains the cumulative prompt.

### 3. `sequence_num` becomes turn number

**Rationale**: With 1 row per turn, `sequence_num` naturally becomes the turn counter. The existing `UNIQUE(conversation_id, sequence_num)` constraint and `ORDER BY sequence_num DESC` pagination still work unchanged.

### 4. Flatten at consumer level, not at repository level

**Rationale**: `GetMessages` returns turn-level `[]*entities.Message`. Consumers that need flat message lists (history loading, conversation memory) iterate over `.Messages` arrays. This keeps the repository interface clean and lets consumers choose whether to work with turn-grouped or flat data.

### 5. Migration: simple column rename

**Rationale**: Since data is truncated manually, the migration is just `ALTER TABLE messages RENAME COLUMN message TO messages` with a default change. No data transformation, no downtime risk.

## Risks / Trade-offs

- **[API Breaking Change]** `message` (object) → `messages` (array) in HTTP response → document in API changelog, clients must update
- **[JSONB array growth]** Long agentic turns with many tool calls produce large arrays → mitigated by the fact that tool content is already truncated at the search level; typical turns stay under 50KB
- **[Query pattern change]** Consumers must flatten arrays for history reconstruction → minimal overhead since array iteration is in-memory after the DB read
