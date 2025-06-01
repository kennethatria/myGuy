-- Add new columns to messages table
ALTER TABLE messages 
ADD COLUMN IF NOT EXISTS original_content TEXT,
ADD COLUMN IF NOT EXISTS is_read BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS read_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS is_edited BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS edited_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_messages_read_status ON messages(recipient_id, is_read);
CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(task_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_sender_recipient ON messages(sender_id, recipient_id);

-- User activity tracking table
CREATE TABLE IF NOT EXISTS user_activity (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),
    last_seen TIMESTAMP NOT NULL DEFAULT NOW(),
    last_conversation_id INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Message deletion warnings table
CREATE TABLE IF NOT EXISTS message_deletion_warnings (
    id SERIAL PRIMARY KEY,
    task_id INTEGER NOT NULL REFERENCES tasks(id),
    deletion_scheduled_at TIMESTAMP NOT NULL,
    warning_shown BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(task_id)
);

-- Add completed_at to tasks table if not exists
ALTER TABLE tasks 
ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP;

-- Update completed_at for already completed tasks
UPDATE tasks 
SET completed_at = updated_at 
WHERE status = 'completed' AND completed_at IS NULL;