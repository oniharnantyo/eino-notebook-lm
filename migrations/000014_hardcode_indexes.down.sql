-- Migration: 000014_hardcode_indexes
-- Description: Drop explicitly defined indexes.

BEGIN;

DROP INDEX IF EXISTS idx_knowledges_source_id;
DROP INDEX IF EXISTS idx_knowledges_chunk_index;
DROP INDEX IF EXISTS knowledges_bm25_idx;

DROP INDEX IF EXISTS idx_sentences_embedding;
DROP INDEX IF EXISTS idx_sentences_knowledge_id;
DROP INDEX IF EXISTS sentences_bm25_idx;

DROP INDEX IF EXISTS idx_images_embedding;
DROP INDEX IF EXISTS idx_images_source_id;
DROP INDEX IF EXISTS idx_images_page_number;
DROP INDEX IF EXISTS images_bm25_idx;

COMMIT;
