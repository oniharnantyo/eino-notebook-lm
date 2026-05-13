-- Migration: 000014_hardcode_indexes
-- Description: Explicitly define all indexes for knowledges, sentences, and images tables.

BEGIN;

-- 1. Knowledges indexes
CREATE INDEX IF NOT EXISTS idx_knowledges_source_id ON knowledges(source_id);
CREATE INDEX IF NOT EXISTS idx_knowledges_chunk_index ON knowledges(chunk_index);
CREATE INDEX IF NOT EXISTS knowledges_bm25_idx
    ON knowledges
    USING bm25(content) WITH (text_config='english');

-- 2. Sentences indexes
-- NOTE: Uses halfvec_cosine_ops for 2048-dim embeddings
CREATE INDEX IF NOT EXISTS idx_sentences_embedding 
    ON sentences 
    USING hnsw (embedding halfvec_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_sentences_knowledge_id ON sentences(knowledge_id);
CREATE INDEX IF NOT EXISTS sentences_bm25_idx
    ON sentences
    USING bm25(content) WITH (text_config='english');

-- 3. Images indexes
-- NOTE: Uses halfvec_cosine_ops for 2048-dim embeddings
CREATE INDEX IF NOT EXISTS idx_images_embedding 
    ON images 
    USING hnsw (embedding halfvec_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_images_source_id ON images(source_id);
CREATE INDEX IF NOT EXISTS idx_images_page_number ON images(page_number);
CREATE INDEX IF NOT EXISTS images_bm25_idx
    ON images
    USING bm25(description) WITH (text_config='english');

COMMIT;
