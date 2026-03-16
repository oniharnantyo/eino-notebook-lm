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

-- Add source_id column to knowledges table
ALTER TABLE knowledges ADD COLUMN IF NOT EXISTS source_id UUID;

-- Drop the old foreign key constraint to notebook
ALTER TABLE knowledges DROP CONSTRAINT IF EXISTS fk_knowledges_notebook;

-- Create foreign key from knowledges to sources
ALTER TABLE knowledges ADD CONSTRAINT fk_knowledges_source
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE;

-- Create index on source_id
CREATE INDEX IF NOT EXISTS idx_knowledges_source_id ON knowledges(source_id);

-- Drop the old notebook_id column (data migration should be done first)
ALTER TABLE knowledges DROP COLUMN IF EXISTS notebook_id;
