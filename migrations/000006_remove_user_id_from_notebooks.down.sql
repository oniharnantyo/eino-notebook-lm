-- Add the user_id column back
ALTER TABLE notebooks ADD COLUMN IF NOT EXISTS user_id UUID NOT NULL DEFAULT gen_random_uuid();

-- Recreate the foreign key constraint
ALTER TABLE notebooks ADD CONSTRAINT fk_notebooks_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Recreate the index on user_id
CREATE INDEX IF NOT EXISTS idx_notebooks_user_id ON notebooks(user_id);