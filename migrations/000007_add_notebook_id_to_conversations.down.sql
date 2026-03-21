-- Remove notebook_id column from conversations table

-- Drop the index
DROP INDEX IF EXISTS idx_conversations_notebook_id;

-- Drop the foreign key constraint
ALTER TABLE conversations DROP CONSTRAINT IF EXISTS fk_conversations_notebook;

-- Drop the column
ALTER TABLE conversations DROP COLUMN IF EXISTS notebook_id;
