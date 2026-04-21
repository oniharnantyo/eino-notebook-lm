BEGIN;

-- 1. Drop BM25 index on description
DROP INDEX IF EXISTS images_bm25_idx;

-- 2. Add ocr_text column
ALTER TABLE images ADD COLUMN ocr_text TEXT;

-- 3. Drop description column
ALTER TABLE images DROP COLUMN description;

-- 4. Re-create BM25 index on ocr_text
CREATE INDEX IF NOT EXISTS images_bm25_idx
    ON images
    USING bm25(ocr_text) WITH (text_config='english');

COMMIT;
