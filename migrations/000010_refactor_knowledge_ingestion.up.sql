-- Migration: 000010_refactor_knowledge_ingestion
-- Description: Refactor knowledges table for chunk-based storage, add sentences and images tables with pgvector support.

BEGIN;

-- 1. DROP existing knowledges table and its indexes
-- This is a breaking change and will remove all existing data in the knowledges table.
DROP TABLE IF EXISTS knowledges CASCADE;

-- 2. CREATE new knowledges table
-- Redefined for chunk-based storage with context and page information.
CREATE TABLE knowledges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL,
    content TEXT NOT NULL,
    chunk_index INT NOT NULL,
    heading_context JSONB,
    first_page INT,
    last_page INT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_knowledges_source FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
);

-- 3. CREATE sentences table
-- For sentence-level embeddings to support more granular search.
CREATE TABLE sentences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_id UUID NOT NULL,
    content TEXT NOT NULL,
    embedding vector(768),
    position INT NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_sentences_knowledge FOREIGN KEY (knowledge_id) REFERENCES knowledges(id) ON DELETE CASCADE
);

-- 4. CREATE images table
-- For storing visual information extracted from documents and their embeddings.
CREATE TABLE images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL,
    s3_key TEXT NOT NULL,
    format TEXT,
    width INT,
    height INT,
    ocr_text TEXT,
    page_number INT,
    embedding vector(768),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_images_source FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
);

-- 5. Create HNSW indexes for vector similarity search
-- Use vector_cosine_ops for cosine similarity.
CREATE INDEX idx_sentences_embedding 
    ON sentences 
    USING hnsw (embedding vector_cosine_ops);

CREATE INDEX idx_images_embedding 
    ON images 
    USING hnsw (embedding vector_cosine_ops);

-- 6. Create relevant B-tree indexes for foreign keys and common search fields
CREATE INDEX idx_knowledges_source_id ON knowledges(source_id);
CREATE INDEX idx_knowledges_chunk_index ON knowledges(chunk_index);
CREATE INDEX idx_sentences_knowledge_id ON sentences(knowledge_id);
CREATE INDEX idx_images_source_id ON images(source_id);
CREATE INDEX idx_images_page_number ON images(page_number);

COMMIT;
