-- Migration: 000011_alter_embedding_dimension_to_2048
-- Description: Rollback embedding vector dimensions from 2048 back to 768.

BEGIN;

-- 1. Drop existing HNSW indexes
DROP INDEX IF EXISTS idx_sentences_embedding CASCADE;
DROP INDEX IF EXISTS idx_images_embedding CASCADE;

-- 2. Alter sentences table embedding column back to vector(768)
ALTER TABLE sentences
    ALTER COLUMN embedding TYPE vector(768);

-- 3. Alter images table embedding column back to vector(768)
ALTER TABLE images
    ALTER COLUMN embedding TYPE vector(768);

-- 4. Recreate HNSW indexes with original dimension
CREATE INDEX idx_sentences_embedding
    ON sentences
    USING hnsw (embedding vector_cosine_ops);

CREATE INDEX idx_images_embedding
    ON images
    USING hnsw (embedding vector_cosine_ops);

COMMIT;
