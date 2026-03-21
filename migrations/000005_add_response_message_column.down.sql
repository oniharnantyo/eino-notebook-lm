-- Drop the response_message column and its index
DROP INDEX IF EXISTS idx_conversations_response_message;
ALTER TABLE conversations DROP COLUMN IF EXISTS response_message;