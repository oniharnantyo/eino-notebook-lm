-- Conversations table for storing chat history
-- Supports conversation threading via previous_response_id
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    response_id TEXT UNIQUE NOT NULL,
    previous_response_id TEXT,
    messages JSONB NOT NULL DEFAULT '[]'::jsonb,
    request_input JSONB,
    response_text TEXT,
    model VARCHAR(100) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create unique index on response_id for fast lookups
CREATE INDEX IF NOT EXISTS idx_conversations_response_id ON conversations(response_id);

-- Create index on previous_response_id for conversation tree traversal
CREATE INDEX IF NOT EXISTS idx_conversations_previous_response_id ON conversations(previous_response_id);

-- Create index on created_at for sorting by time
CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at DESC);

-- Create index on metadata for JSONB queries
CREATE INDEX IF NOT EXISTS idx_conversations_metadata ON conversations USING GIN(metadata);

-- Add comment explaining the threading model
COMMENT ON TABLE conversations IS 'Stores conversation history with support for threading via previous_response_id. Each response contains the full message history up to that point.';
COMMENT ON COLUMN conversations.previous_response_id IS 'Points to the parent response in the conversation tree. NULL for root conversations.';
COMMENT ON COLUMN conversations.messages IS 'Complete message history up to and including this response. Stored as JSONB array of message objects.';