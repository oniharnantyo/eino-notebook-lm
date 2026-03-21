-- Add notebook_id column to conversations table
-- This establishes the relationship between conversations and notebooks

-- Add the column as nullable first (for existing data)
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS notebook_id UUID;

-- Add foreign key constraint to notebooks table
ALTER TABLE conversations
ADD CONSTRAINT fk_conversations_notebook
FOREIGN KEY (notebook_id) REFERENCES notebooks(id) ON DELETE SET NULL;

-- Create index on notebook_id for fast filtering
CREATE INDEX IF NOT EXISTS idx_conversations_notebook_id ON conversations(notebook_id);

-- Add comment
COMMENT ON COLUMN conversations.notebook_id IS 'Optional reference to the notebook this conversation is associated with. Used for filtering conversations by notebook.';
