-- Drop indexes
DROP INDEX IF EXISTS idx_knowledges_sub_indexes;
DROP INDEX IF EXISTS idx_knowledges_source_type;
DROP INDEX IF EXISTS idx_knowledges_notebook_id;
DROP INDEX IF EXISTS knowledges_embedding_idx;
DROP INDEX IF EXISTS idx_notebooks_deleted_at;
DROP INDEX IF EXISTS idx_notebooks_metadata;
DROP INDEX IF EXISTS idx_notebooks_tags;
DROP INDEX IF EXISTS idx_notebooks_status;
DROP INDEX IF EXISTS idx_notebooks_user_id;
DROP INDEX IF EXISTS idx_notebooks_title;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_email;

-- Drop tables
DROP TABLE IF EXISTS knowledges;
DROP TABLE IF EXISTS notebooks;
DROP TABLE IF EXISTS users;

-- Drop pgvector extension
DROP EXTENSION IF EXISTS vector;
