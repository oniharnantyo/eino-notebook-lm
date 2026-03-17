-- Drop indexes
DROP INDEX IF EXISTS idx_knowledges_source_id;
DROP INDEX IF EXISTS idx_sources_metadata;
DROP INDEX IF NOT EXISTS idx_sources_deleted_at;
DROP INDEX IF NOT EXISTS idx_sources_content_type;
DROP INDEX IF NOT EXISTS idx_sources_notebook_id;

-- Drop source_id column from knowledges table
ALTER TABLE knowledges DROP COLUMN IF EXISTS source_id;

-- Drop sources table
DROP TABLE IF EXISTS sources;