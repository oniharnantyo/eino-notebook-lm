-- Migration: Normalize messages table with response_id in messages
-- Description: Create messages table with response_id and simplified conversations table
-- Version: 017_normalize_messages_table.up.sql

-- Step 1: Create the messages table with response_id
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL,
    sequence_num INTEGER NOT NULL,
    response_id TEXT NOT NULL,
    previous_response_id TEXT,
    message JSONB NOT NULL,
    model VARCHAR(100) NOT NULL,
    finish_reason VARCHAR(50),
    prompt_tokens INTEGER,
    completion_tokens INTEGER,
    total_tokens INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (conversation_id, sequence_num)
);

-- Step 2: Create indexes for messages table
CREATE INDEX idx_messages_conversation_seq ON messages(conversation_id, sequence_num DESC);
CREATE INDEX idx_messages_response_id ON messages(response_id);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);

-- Step 3: Drop columns from conversations table
ALTER TABLE conversations DROP COLUMN IF EXISTS messages;
ALTER TABLE conversations DROP COLUMN IF EXISTS response_text;
ALTER TABLE conversations DROP COLUMN IF EXISTS response_message;
ALTER TABLE conversations DROP COLUMN IF EXISTS response_id;
ALTER TABLE conversations DROP COLUMN IF EXISTS previous_response_id;
ALTER TABLE conversations DROP COLUMN IF EXISTS model;
ALTER TABLE conversations DROP COLUMN IF EXISTS finish_reason;
ALTER TABLE conversations DROP COLUMN IF EXISTS prompt_tokens;
ALTER TABLE conversations DROP COLUMN IF EXISTS completion_tokens;
ALTER TABLE conversations DROP COLUMN IF EXISTS total_tokens;

-- Step 4: Add foreign key constraint to messages table
ALTER TABLE messages
ADD CONSTRAINT fk_messages_conversation
FOREIGN KEY (conversation_id)
REFERENCES conversations(id)
ON DELETE CASCADE;

-- Step 5: Add comments
COMMENT ON TABLE messages IS 'Message storage with response-level metadata. Each row is a single message. response_id groups messages by turn, previous_response_id links turns.';
COMMENT ON COLUMN messages.conversation_id IS 'Foreign key to conversations.id';
COMMENT ON COLUMN messages.sequence_num IS 'Order within conversation (1, 2, 3, ...). DESC for latest-first pagination.';
COMMENT ON COLUMN messages.response_id IS 'Identifies which turn this message belongs to';
COMMENT ON COLUMN messages.previous_response_id IS 'Links to previous turn for threading';
COMMENT ON COLUMN messages.message IS 'Full message as JSONB: {role, content, extra, timestamp}';
COMMENT ON COLUMN messages.model IS 'AI model used for this turn';
COMMENT ON COLUMN messages.finish_reason IS 'Why the response ended (stop, length, tool_calls, etc)';

COMMENT ON TABLE conversations IS 'Chat session container. One conversation = one continuous chat session with multiple turns.';