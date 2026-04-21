-- Migration: 000011_alter_embedding_dimension_to_2048
-- Description: Alter embedding vector dimensions from 768 to 2048 to match qwen3-vl-embed model output.
-- Uses halfvec type which supports up to 4000 dimensions (vs vector's 2000 limit)

BEGIN;

-- 1. Drop existing indexes
DROP INDEX IF EXISTS idx_sentences_embedding CASCADE;
DROP INDEX IF EXISTS idx_images_embedding CASCADE;

-- 2. Alter sentences table embedding column to halfvec(2048)
ALTER TABLE sentences
    ALTER COLUMN embedding TYPE halfvec(2048);

-- 3. Alter images table embedding column to halfvec(2048)
ALTER TABLE images
    ALTER COLUMN embedding TYPE halfvec(2048);

-- 4. Recreate HNSW indexes with new dimension
-- halfvec supports up to 4000 dimensions, so 2048 works with HNSW
CREATE INDEX idx_sentences_embedding
    ON sentences
    USING hnsw (embedding halfvec_cosine_ops);

CREATE INDEX idx_images_embedding
    ON images
    USING hnsw (embedding halfvec_cosine_ops);

COMMIT;
