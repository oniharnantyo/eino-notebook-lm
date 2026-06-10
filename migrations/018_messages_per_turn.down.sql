ALTER TABLE messages ALTER COLUMN messages DROP DEFAULT;
ALTER TABLE messages RENAME COLUMN messages TO message;
COMMENT ON COLUMN messages.message IS 'Full message as JSONB: {role, content, extra, timestamp}';
