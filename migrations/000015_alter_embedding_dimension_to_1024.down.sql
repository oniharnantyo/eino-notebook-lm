-- Migration: 000015_alter_embedding_dimension_to_1024 (Rollback)
-- Description: Rollback embedding vector dimensions from 1024 back to 2048.

BEGIN;

-- 1. Drop existing indexes
DROP INDEX IF EXISTS idx_sentences_embedding CASCADE;
DROP INDEX IF EXISTS idx_images_embedding CASCADE;

-- 2. Clear existing embeddings since dimension is changing
TRUNCATE sentences, images RESTART IDENTITY CASCADE;

-- 3. Alter sentences table embedding column back to halfvec(2048)
ALTER TABLE sentences
    ALTER COLUMN embedding TYPE halfvec(2048);

-- 4. Alter images table embedding column back to halfvec(2048)
ALTER TABLE images
    ALTER COLUMN embedding TYPE halfvec(2048);

-- 5. Recreate HNSW indexes with original dimension
CREATE INDEX idx_sentences_embedding
    ON sentences
    USING hnsw (embedding halfvec_cosine_ops);

CREATE INDEX idx_images_embedding
    ON images
    USING hnsw (embedding halfvec_cosine_ops);

COMMIT;
