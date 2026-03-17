-- Create sources table for knowledge ingestion
CREATE TABLE IF NOT EXISTS sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notebook_id UUID NOT NULL,
    title VARCHAR(500) NOT NULL,
    uri TEXT,
    content_type VARCHAR(100),
    content TEXT,
    chunk_count INTEGER NOT NULL DEFAULT 0,
    total_size INTEGER,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_sources_notebook FOREIGN KEY (notebook_id) REFERENCES notebooks(id) ON DELETE CASCADE
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_sources_notebook_id ON sources(notebook_id);
CREATE INDEX IF NOT EXISTS idx_sources_content_type ON sources(content_type);
CREATE INDEX IF NOT EXISTS idx_sources_deleted_at ON sources(deleted_at);
CREATE INDEX IF NOT EXISTS idx_sources_metadata ON sources USING GIN(metadata);

-- Add source_id column to knowledges table as TEXT
-- Note: No foreign key constraint since knowledges.source_id is TEXT and sources.id is UUID
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS source_id TEXT;

-- Create index on source_id for query performance
CREATE INDEX IF NOT EXISTS idx_knowledges_source_id ON knowledges(source_id);

-- Drop the old foreign key constraint to notebook
ALTER TABLE knowledges DROP CONSTRAINT IF EXISTS fk_knowledges_notebook;

-- Drop the old notebook_id column (data migration should be done first)
ALTER TABLE knowledges DROP COLUMN IF EXISTS notebook_id;