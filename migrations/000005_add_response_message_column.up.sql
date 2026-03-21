-- Add response_message column to store the full chatModel response
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS response_message JSONB;

-- Create index on response_message for querying response metadata
CREATE INDEX IF NOT EXISTS idx_conversations_response_message ON conversations USING GIN(response_message);

-- Add comment explaining the purpose
COMMENT ON COLUMN conversations.response_message IS 'Complete response message from chatModel including role, content, tool_calls, response_meta, etc.';