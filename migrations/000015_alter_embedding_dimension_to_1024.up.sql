-- Migration: 000015_alter_embedding_dimension_to_1024
-- Description: Alter embedding vector dimensions from 2048 to 1024 to match qwen3-embedding:0.6b model output.
-- qwen3-embedding:0.6b is 8x faster than nomic-embed-text-v2-moe and provides 1024-dimensional embeddings.

BEGIN;

-- 1. Drop existing indexes
DROP INDEX IF EXISTS idx_sentences_embedding CASCADE;
DROP INDEX IF EXISTS idx_images_embedding CASCADE;

-- 2. Clear existing embeddings since dimension is changing
-- Note: This will require re-ingestion of sources to populate new embeddings
TRUNCATE sentences, images RESTART IDENTITY CASCADE;

-- 3. Alter sentences table embedding column to halfvec(1024)
ALTER TABLE sentences
    ALTER COLUMN embedding TYPE halfvec(1024);

-- 4. Alter images table embedding column to halfvec(1024)
ALTER TABLE images
    ALTER COLUMN embedding TYPE halfvec(1024);

-- 5. Recreate HNSW indexes with new dimension
CREATE INDEX idx_sentences_embedding
    ON sentences
    USING hnsw (embedding halfvec_cosine_ops);

CREATE INDEX idx_images_embedding
    ON images
    USING hnsw (embedding halfvec_cosine_ops);

COMMIT;
