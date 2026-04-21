-- Migration: 000012_add_bm25_fulltext_search (down)
-- Description: Remove BM25 indexes and pg_textsearch extension.

BEGIN;

-- Drop BM25 indexes (drop both old and new names to handle rollback)
DROP INDEX IF EXISTS idx_sentences_content_bm25;
DROP INDEX IF EXISTS idx_images_ocr_text_bm25;
DROP INDEX IF EXISTS idx_knowledges_content_bm25;
DROP INDEX IF EXISTS sentences_bm25_idx;
DROP INDEX IF EXISTS images_bm25_idx;
DROP INDEX IF EXISTS knowledges_bm25_idx;

-- Drop pg_textsearch extension
-- Note: This will fail if other objects depend on it
DROP EXTENSION IF EXISTS pg_textsearch;

COMMIT;
