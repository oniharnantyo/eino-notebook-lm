ALTER TABLE messages RENAME COLUMN message TO messages;
ALTER TABLE messages ALTER COLUMN messages SET DEFAULT '[]'::jsonb;
COMMENT ON COLUMN messages.messages IS 'List of messages in this turn as JSONB array of StoredMessage';
