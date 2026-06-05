# Design: Conversation Message Refactor

## Context

**Current State:**
Conversation storage uses a checkpoint pattern where each conversation row contains the full message history in a single JSONB column (`conversations.messages`). This design was optimized for conversation resumption and checkpoint recovery, but creates problems for UI pagination requiring efficient retrieval of subsets of messages from conversations containing 50-500 messages.

**Constraints:**
- Existing conversations memory middleware (`internal/adk/middleware/conversation_memory.go`) currently loads full message history from JSONB
- Frontend expects paginated message display (latest 10-20 messages, load more on scroll)
- PostgreSQL database with JSONB support
- Go 1.22+, Clean Architecture with DDD layers

**Stakeholders:**
- Frontend: Chat interface requiring efficient pagination
- Backend: Response pipeline needing conversation history context
- Database: Performance at scale with large conversations

## Goals / Non-Goals

**Goals:**
- Enable efficient cursor-based pagination for conversation messages
- Support reverse pagination (latest messages first, load older on scroll up)
- Maintain conversation threading via `previous_response_id`
- Separate concerns: conversations = session metadata, messages = content + turn metadata
- Support 50-500 messages per conversation with sub-10ms query performance

**Non-Goals:**
- Migrating existing conversation data (fresh start acceptable)
- Backward compatibility with existing JSONB structure
- Real-time message streaming (handled separately by SSE)
- Cross-conversation queries (all queries scoped to single conversation)

## Decisions

### Database Schema: Normalized Messages Table

**Decision:** Store messages in separate table with `response_id` and turn metadata in messages row, not conversations.

**Rationale:** 
- **Separation of concerns**: Conversations table becomes pure session container, messages table holds all message data and turn-level metadata
- **Efficient pagination**: Indexed queries on `(conversation_id, sequence_num DESC)` enable cursor-based pagination
- **No data duplication**: Each message stored once, no repetition across conversation rows
- **Flexible turn tracking**: `response_id` groups messages by turn, `previous_response_id` enables threading

**Alternatives considered:**
- Keep messages JSONB and add pagination metadata → Rejected: Still requires fetching/entire JSONB for pagination
- Store messages in separate table but keep turn metadata in conversations → Rejected: More complex joins, violates single responsibility

### Message Storage: JSONB-Only Design

**Decision:** Use JSONB for message content in simplified 5-column design (id, conversation_id, sequence_num, message, created_at) with turn metadata columns (response_id, previous_response_id, model, tokens, etc.) stored as separate columns.

**Rationale:**
- **Simplicity**: 5 core columns vs 13 normalized columns reduces complexity
- **Flexibility**: JSONB accommodates evolving message structures (tool calls, multimodal, reasoning) without schema changes
- **Performance**: Indexed queries on frequently accessed fields (response_id, role) still efficient
- **Reusability**: Leverages existing `StoredMessage` structure without transformation

**Alternatives considered:**
- Fully normalize all message fields → Rejected: Over-engineering for current use case, complex schema changes

### Pagination: Reverse Cursor-Based

**Decision:** Use `sequence_num` as cursor with `beforeSequence` parameter for reverse pagination (latest first).

**Rationale:**
- **User expectation**: Chat interfaces show latest messages first, load older on scroll
- **Performance**: `WHERE sequence_num < cursor` uses index efficiently, no OFFSET scans
- **Consistency**: New messages arriving don't break pagination state
- **Simplicity**: Frontend tracks single integer cursor

**Alternatives considered:**
- Page-based OFFSET pagination → Rejected: Performance degrades with large OFFSET values
- Forward pagination with reverse in frontend → Rejected: More complex frontend logic

### Conversations Table: Session Metadata Only

**Decision:** Simplify conversations table to 4 columns (id, notebook_id, metadata, created_at).

**Rationale:**
- **Clean separation**: Conversations = session container only
- **No redundancy**: All turn-specific data moved to messages table
- **Simpler queries**: Session-level queries don't need to join message data

**Alternatives considered:**
- Keep some turn metadata in conversations → Rejected: Violates single responsibility, creates data duplication

## Risks / Trade-offs

### Risk: JSONB queries slower than normalized columns
**Impact:** Filtering by role or checking for tool_calls requires JSONB extraction → **Mitigation:** Use JSONB operations sparingly, optimize common queries, add indexes if needed

### Risk: Breaking change to conversation memory middleware
**Impact:** Existing middleware loads from `conversation.Messages` field → **Mitigation:** Update middleware to use new `GetMessages()` repository method with pagination

### Risk: API endpoint changes may break existing clients
**Impact:** Clients expecting old response format → **Mitigation:** Document API changes clearly, maintain response structure compatibility where possible

### Trade-off: No data migration
**Benefit:** Simpler migration, faster deployment  
**Cost:** Existing conversations inaccessible through new API (acceptable for early-stage project)

## Migration Plan

### Deployment Steps

1. **Apply database migration**
   ```bash
   migrate -path migrations -database $DATABASE_URL up
   ```

2. **Deploy code changes** (order matters):
   - Update entity (`Conversation` struct)
   - Update repository (`Save()`, `GetMessages()`)
   - Update middleware (`conversation_memory.go`)
   - Add new API handler (`GetMessages`)

3. **Verify functionality**
   - Test fresh chat creation
   - Test message pagination
   - Test conversation threading

### Rollback Strategy

```bash
migrate -path migrations -database $DATABASE_URL down 1
```

Restores `conversations.messages` JSONB column and dropped columns. Code rollback may be required if already deployed.

## Open Questions

1. **Question:** Should we add GIN index on `message` JSONB for role extraction?
   - **Impact:** Faster JSONB queries at cost of write performance and storage
   - **Decision:** Defer until performance testing shows need

2. **Question:** Default `limit` value for pagination?
   - **Options:** 10 (current discussion), 20, 50
   - **Decision:** Set to 10 based on chat UI requirements, make configurable

3. **Question:** How to handle conversations with no messages?
   - **Current design:** Return empty array with `has_more: false`
   - **Validation:** Consider if 404 is more appropriate for non-existent conversations