## Why

We need to add conversation memory middleware to Eino ADK agents for automatic conversation persistence and retrieval. Currently, conversation saving is tightly coupled to the application layer (HistorySavingReader in ResponseUseCase), making it non-reusable across different agents. We also need to extract response metadata (token usage, finish reason) into separate database columns for analytics and cost tracking without parsing JSONB.

## What Changes

- **Add Eino ADK conversation memory middleware**: Automatic conversation loading (before model) and async saving (after agent) with per-notebook scoping
- **Add response metadata columns to conversations table**: `finish_reason`, `prompt_tokens`, `completion_tokens`, `total_tokens` for efficient analytics queries
- **Simplify error handling**: Async save with logging only on failure (no DLQ complexity for now)

## Capabilities

### New Capabilities
- `conversation-memory-middleware`: ADK middleware for automatic conversation persistence and retrieval across any agent using Eino Runner

### Modified Capabilities
- None (implementation change, no spec-level requirement changes)

## Impact

**Affected code:**
- `internal/adk/middleware/` (new directory)
- `internal/core/domain/entities/conversation.go` (add 4 metadata fields)
- `internal/infrastructure/persistence/conversation.go` (update INSERT/UPDATE queries)
- `cmd/serve.go` (wire middleware into ADK Runner)
- `migrations/` (add migration for metadata columns)

**Dependencies:**
- Eino ADK (`github.com/cloudwego/eino/adk`) - for Handler interface and AgentEvent types
- Existing ConversationRepository interface - no changes needed
