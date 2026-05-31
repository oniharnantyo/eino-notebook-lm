-- Migration: 000016_add_conversation_metadata_columns
-- Description: Add finish_reason and token count columns to conversations table for better analytics and cost tracking.

BEGIN;

-- 1. Add finish_reason column (e.g., "stop", "length", "content_filter", etc.)
ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS finish_reason TEXT;

-- 2. Add token usage columns for cost tracking
ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS prompt_tokens INTEGER;

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS completion_tokens INTEGER;

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS total_tokens INTEGER;

-- 3. Create index on finish_reason for filtering by completion status
CREATE INDEX IF NOT EXISTS idx_conversations_finish_reason
    ON conversations(finish_reason);

-- 4. Create composite index on token columns for analytics queries
CREATE INDEX IF NOT EXISTS idx_conversations_token_usage
    ON conversations(prompt_tokens, completion_tokens, total_tokens);

-- 5. Add comments for documentation
COMMENT ON COLUMN conversations.finish_reason IS 'Reason why the model finished generating (e.g., stop, length, content_filter, recitation)';
COMMENT ON COLUMN conversations.prompt_tokens IS 'Number of tokens in the prompt/input messages';
COMMENT ON COLUMN conversations.completion_tokens IS 'Number of tokens in the model response';
COMMENT ON COLUMN conversations.total_tokens IS 'Total tokens used (prompt_tokens + completion_tokens)';

COMMIT;
