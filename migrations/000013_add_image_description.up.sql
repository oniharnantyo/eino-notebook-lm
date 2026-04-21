BEGIN;

-- 1. Drop old BM25 index on ocr_text if it exists
DROP INDEX IF EXISTS images_bm25_idx;
DROP INDEX IF EXISTS idx_images_ocr_text_bm25;

-- 2. Add description column
ALTER TABLE images ADD COLUMN description TEXT;

-- 3. Drop ocr_text column
ALTER TABLE images DROP COLUMN ocr_text;

-- 4. Create new BM25 index on description
CREATE INDEX IF NOT EXISTS images_bm25_idx
    ON images
    USING bm25(description) WITH (text_config='english');

COMMIT;
