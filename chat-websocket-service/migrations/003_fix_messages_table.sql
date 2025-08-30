-- Fix messages table to allow store messages
BEGIN;

-- Make task_id nullable
ALTER TABLE messages ALTER COLUMN task_id DROP NOT NULL;

-- Add store_item_id column
ALTER TABLE messages ADD COLUMN IF NOT EXISTS store_item_id INTEGER;

-- Create indexes for store messages
CREATE INDEX IF NOT EXISTS idx_messages_store_item ON messages(store_item_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_store_item_participants ON messages(store_item_id, sender_id, recipient_id);

-- Add foreign key for store_items
ALTER TABLE messages 
ADD CONSTRAINT fk_messages_store_item 
FOREIGN KEY (store_item_id) 
REFERENCES store_items(id)
ON DELETE CASCADE;

-- Migrate data from store_messages
INSERT INTO messages (
    store_item_id,
    sender_id,
    recipient_id,
    content,
    original_content,
    created_at,
    is_read,
    read_at
)
SELECT 
    store_item_id,
    sender_id,
    recipient_id,
    content,
    original_content,
    created_at,
    is_read,
    read_at
FROM store_messages
ON CONFLICT DO NOTHING;

COMMIT;
