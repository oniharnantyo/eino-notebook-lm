ALTER TABLE sources ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'pending';
ALTER TABLE sources ADD COLUMN IF NOT EXISTS error TEXT;
CREATE INDEX IF NOT EXISTS idx_sources_status ON sources(status) WHERE deleted_at IS NULL;
UPDATE sources SET status = 'completed' WHERE content IS NOT NULL AND chunk_count > 0;