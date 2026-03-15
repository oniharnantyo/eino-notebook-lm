-- Enable pgvector extension for vector similarity search
CREATE EXTENSION IF NOT EXISTS vector;

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Create index on email for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
-- Create index on status for filtering
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
-- Create index on deleted_at for soft deletes
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- Notebooks table
CREATE TABLE IF NOT EXISTS notebooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    content TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    tags TEXT[],
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_notebooks_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create index on title for search
CREATE INDEX IF NOT EXISTS idx_notebooks_title ON notebooks(title);
-- Create index on user_id for faster queries
CREATE INDEX IF NOT EXISTS idx_notebooks_user_id ON notebooks(user_id);
-- Create index on status for filtering
CREATE INDEX IF NOT EXISTS idx_notebooks_status ON notebooks(status);
-- Create index on tags for tag-based queries
CREATE INDEX IF NOT EXISTS idx_notebooks_tags ON notebooks USING GIN(tags);
-- Create index on metadata for JSONB queries
CREATE INDEX IF NOT EXISTS idx_notebooks_metadata ON notebooks USING GIN(metadata);
-- Create index on deleted_at for soft deletes
CREATE INDEX IF NOT EXISTS idx_notebooks_deleted_at ON notebooks(deleted_at);

-- Knowledges table for pgvector embeddings (supports documents, websites, text, API, etc.)
CREATE TABLE IF NOT EXISTS knowledges (
    knowledge_id TEXT PRIMARY KEY,
    notebook_id UUID NOT NULL,
    title TEXT,
    content TEXT NOT NULL,
    embedding vector(768),
    source_type VARCHAR(50) NOT NULL DEFAULT 'document',
    metadata JSONB,
    sub_indexes TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_knowledges_notebook FOREIGN KEY (notebook_id) REFERENCES notebooks(id) ON DELETE CASCADE,
    CONSTRAINT chk_knowledges_source_type CHECK (source_type IN ('document', 'website', 'text', 'api', 'other'))
);

-- Create HNSW index for vector similarity search
CREATE INDEX IF NOT EXISTS knowledges_embedding_idx
    ON knowledges
    USING hnsw (embedding vector_cosine_ops);

-- Create index on notebook_id for faster queries
CREATE INDEX IF NOT EXISTS idx_knowledges_notebook_id ON knowledges(notebook_id);

-- Create index on source_type for filtering
CREATE INDEX IF NOT EXISTS idx_knowledges_source_type ON knowledges(source_type);

-- Create index on sub_indexes for filtering
CREATE INDEX IF NOT EXISTS idx_knowledges_sub_indexes ON knowledges USING GIN(sub_indexes);
