-- Drop the index on user_id first
DROP INDEX IF EXISTS idx_notebooks_user_id;

-- Drop the foreign key constraint
ALTER TABLE notebooks DROP CONSTRAINT IF EXISTS fk_notebooks_user;

-- Drop the user_id column
ALTER TABLE notebooks DROP COLUMN IF EXISTS user_id;