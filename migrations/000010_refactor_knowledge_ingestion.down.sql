-- Migration: 000010_refactor_knowledge_ingestion
-- Description: Revert refactoring of knowledge ingestion (drops new tables and recreates old knowledges table)

BEGIN;

-- DROP new tables
DROP TABLE IF EXISTS images CASCADE;
DROP TABLE IF EXISTS sentences CASCADE;
DROP TABLE IF EXISTS knowledges CASCADE;

-- RECREATE old knowledges table structure (as it was after migration 000009)
-- This restores the table schema but not the data.
CREATE TABLE knowledges (
    knowledge_id TEXT PRIMARY KEY,
    source_id TEXT, -- Note: This was TEXT in the previous version
    title TEXT,
    content TEXT NOT NULL,
    embedding vector(768),
    source_type VARCHAR(50) NOT NULL DEFAULT 'document',
    metadata JSONB DEFAULT '{}'::jsonb,
    sub_indexes TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_knowledges_source_type CHECK (source_type IN ('document', 'website', 'text', 'api', 'other'))
);

-- Recreate old indexes for the knowledges table
CREATE INDEX IF NOT EXISTS knowledges_embedding_idx ON knowledges USING hnsw (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_knowledges_source_type ON knowledges(source_type);
CREATE INDEX IF NOT EXISTS idx_knowledges_sub_indexes ON knowledges USING GIN(sub_indexes);
CREATE INDEX IF NOT EXISTS idx_knowledges_source_id ON knowledges(source_id);

COMMIT;
