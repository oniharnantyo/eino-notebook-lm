-- Migration: 000012_add_bm25_fulltext_search
-- Description: Add pg_textsearch extension and BM25 indexes for hybrid search on sentences and images tables.

BEGIN;

-- 1. Create pg_textsearch extension for BM25 indexing
-- This extension provides efficient BM25 (Best Matching 25) full-text search
CREATE EXTENSION IF NOT EXISTS pg_textsearch;

-- 2. Create BM25 index on sentences.content
-- Enables keyword-based full-text search alongside vector similarity search
-- Using 'english' text config for English language text processing
-- Index name follows convention: {table}_bm25_idx for automatic retriever discovery
CREATE INDEX IF NOT EXISTS sentences_bm25_idx
    ON sentences
    USING bm25(content) WITH (text_config='english');

-- 3. Create BM25 index on images.ocr_text
-- Enables full-text search on OCR-extracted text from images
CREATE INDEX IF NOT EXISTS images_bm25_idx
    ON images
    USING bm25(ocr_text) WITH (text_config='english');

-- 4. Create BM25 index on knowledges.content
-- Note: knowledges table stores document chunks, this enables hybrid search at chunk level
CREATE INDEX IF NOT EXISTS knowledges_bm25_idx
    ON knowledges
    USING bm25(content) WITH (text_config='english');

COMMIT;
