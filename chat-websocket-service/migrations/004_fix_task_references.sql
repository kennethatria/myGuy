-- Fix task references in messages table
BEGIN;

-- Ensure tasks table has correct column names
ALTER TABLE tasks 
RENAME COLUMN creator_id TO created_by;

-- Update foreign key references
ALTER TABLE messages DROP CONSTRAINT IF EXISTS fk_messages_task;
ALTER TABLE messages 
ADD CONSTRAINT fk_messages_task 
FOREIGN KEY (task_id) 
REFERENCES tasks(id)
ON DELETE CASCADE;

COMMIT;
