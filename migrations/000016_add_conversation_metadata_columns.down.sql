-- Migration: 000016_add_conversation_metadata_columns (Rollback)
-- Description: Rollback: Remove finish_reason and token count columns from conversations table.

BEGIN;

-- 1. Drop indexes
DROP INDEX IF EXISTS idx_conversations_finish_reason CASCADE;
DROP INDEX IF EXISTS idx_conversations_token_usage CASCADE;

-- 2. Drop columns from conversations table
ALTER TABLE conversations
    DROP COLUMN IF EXISTS finish_reason;

ALTER TABLE conversations
    DROP COLUMN IF EXISTS prompt_tokens;

ALTER TABLE conversations
    DROP COLUMN IF EXISTS completion_tokens;

ALTER TABLE conversations
    DROP COLUMN IF EXISTS total_tokens;

COMMIT;
