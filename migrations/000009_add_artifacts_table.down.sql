-- Drop indexes
DROP INDEX IF EXISTS idx_artifacts_source_ids;
DROP INDEX IF EXISTS idx_artifacts_notebook_id;
DROP INDEX IF EXISTS idx_artifacts_type;
DROP INDEX IF EXISTS idx_artifacts_status;
DROP INDEX IF EXISTS idx_artifacts_deleted_at;
DROP INDEX IF EXISTS idx_artifacts_metadata;

-- Drop artifacts table
DROP TABLE IF EXISTS artifacts;
