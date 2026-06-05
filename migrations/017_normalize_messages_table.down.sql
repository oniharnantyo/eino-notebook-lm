-- Down Migration: Rollback normalize messages table
-- Description: Restore conversations table and drop messages table
-- Version: 017_normalize_messages_table.down.sql

-- Step 1: Drop the messages table
DROP TABLE IF EXISTS messages CASCADE;

-- Step 2: Restore columns to conversations table
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS messages JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS response_text TEXT;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS response_message JSONB;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS response_id TEXT UNIQUE NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS previous_response_id TEXT;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS model VARCHAR(100) NOT NULL DEFAULT 'gemini-pro';
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS finish_reason VARCHAR(50);
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS prompt_tokens INTEGER;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS completion_tokens INTEGER;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS total_tokens INTEGER;

-- Step 3: Add indexes
CREATE INDEX IF NOT EXISTS idx_conversations_response_id ON conversations(response_id);
CREATE INDEX IF NOT EXISTS idx_conversations_notebook_id ON conversations(notebook_id);
CREATE INDEX IF NOT EXISTS idx_conversations_previous_response_id ON conversations(previous_response_id);
CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversations_model ON conversations(model);

-- Step 4: Add comments
COMMENT ON TABLE conversations IS 'Stores conversation history with support for threading via previous_response_id. Each response contains the full message history up to that point.';
COMMENT ON COLUMN conversations.messages IS 'Complete message history up to and including this response. Stored as JSONB array of message objects.';