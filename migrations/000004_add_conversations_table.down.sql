-- Drop conversations table
DROP INDEX IF EXISTS idx_conversations_metadata;
DROP INDEX IF EXISTS idx_conversations_created_at;
DROP INDEX IF EXISTS idx_conversations_previous_response_id;
DROP INDEX IF EXISTS idx_conversations_response_id;
DROP TABLE IF EXISTS conversations;