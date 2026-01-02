-- ============================================================================
-- Chat Service Complete Schema
-- Database: my_guy_chat (separate from main backend)
--
-- This migration creates all tables needed for the chat service.
-- NO foreign keys to other databases - uses API validation instead.
-- ============================================================================

-- ============================================================================
-- 1. MESSAGES TABLE
-- Core messaging functionality for tasks, applications, and store items
-- ============================================================================
CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,

    -- User references (IDs only, validated via API)
    sender_id INTEGER NOT NULL,
    recipient_id INTEGER,

    -- Context references (IDs only, validated via API)
    -- One of these will be set depending on message context
    task_id INTEGER,
    application_id INTEGER,
    store_item_id INTEGER,

    -- Message content
    content TEXT NOT NULL,
    message_type VARCHAR(50) DEFAULT 'task',

    -- Message state tracking
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP,
    is_edited BOOLEAN DEFAULT FALSE,
    edited_at TIMESTAMP,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP,
    deletion_scheduled_at TIMESTAMP,

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CHECK (
        -- At least one context must be set
        task_id IS NOT NULL OR
        application_id IS NOT NULL OR
        store_item_id IS NOT NULL
    )
);

-- ============================================================================
-- 2. USER ACTIVITY TABLE
-- Tracks user presence and last seen timestamps
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_activity (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_conversation_id INTEGER,
    is_online BOOLEAN DEFAULT FALSE,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- 3. MESSAGE DELETION WARNINGS TABLE
-- Tracks pending message deletions and user notifications
-- ============================================================================
CREATE TABLE IF NOT EXISTS message_deletion_warnings (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    message_id INTEGER,
    task_id INTEGER,
    application_id INTEGER,
    store_item_id INTEGER,
    warning_shown BOOLEAN DEFAULT FALSE,
    deletion_scheduled_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Foreign key to messages table (same database)
    CONSTRAINT fk_deletion_warnings_message
        FOREIGN KEY (message_id)
        REFERENCES messages(id)
        ON DELETE CASCADE
);

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- Message indexes (critical for query performance)
CREATE INDEX IF NOT EXISTS idx_messages_sender_id
    ON messages(sender_id);

CREATE INDEX IF NOT EXISTS idx_messages_recipient_id
    ON messages(recipient_id)
    WHERE recipient_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_messages_task_id
    ON messages(task_id)
    WHERE task_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_messages_application_id
    ON messages(application_id)
    WHERE application_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_messages_store_item_id
    ON messages(store_item_id)
    WHERE store_item_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_messages_created_at
    ON messages(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_message_type
    ON messages(message_type);

-- Composite index for common queries
CREATE INDEX IF NOT EXISTS idx_messages_sender_created
    ON messages(sender_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_recipient_created
    ON messages(recipient_id, created_at DESC)
    WHERE recipient_id IS NOT NULL;

-- Index for cleanup queries
CREATE INDEX IF NOT EXISTS idx_messages_deleted_scheduled
    ON messages(is_deleted, deletion_scheduled_at)
    WHERE deletion_scheduled_at IS NOT NULL;

-- User activity indexes
CREATE INDEX IF NOT EXISTS idx_user_activity_user_id
    ON user_activity(user_id);

CREATE INDEX IF NOT EXISTS idx_user_activity_online
    ON user_activity(is_online)
    WHERE is_online = TRUE;

-- Message deletion warnings indexes
CREATE INDEX IF NOT EXISTS idx_message_deletion_warnings_user_id
    ON message_deletion_warnings(user_id);

CREATE INDEX IF NOT EXISTS idx_message_deletion_warnings_message_id
    ON message_deletion_warnings(message_id);

CREATE INDEX IF NOT EXISTS idx_message_deletion_warnings_task_id
    ON message_deletion_warnings(task_id);

CREATE INDEX IF NOT EXISTS idx_message_deletion_warnings_scheduled
    ON message_deletion_warnings(deletion_scheduled_at);

-- ============================================================================
-- TRIGGERS FOR AUTO-UPDATE TIMESTAMPS
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to messages table
DROP TRIGGER IF EXISTS update_messages_updated_at ON messages;
CREATE TRIGGER update_messages_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to user_activity table
DROP TRIGGER IF EXISTS update_user_activity_updated_at ON user_activity;
CREATE TRIGGER update_user_activity_updated_at
    BEFORE UPDATE ON user_activity
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE messages IS 'Stores all chat messages for tasks, applications, and store items';
COMMENT ON COLUMN messages.sender_id IS 'User ID of message sender (validated via API)';
COMMENT ON COLUMN messages.recipient_id IS 'User ID of message recipient (validated via API)';
COMMENT ON COLUMN messages.task_id IS 'Task ID reference (validated via Main API)';
COMMENT ON COLUMN messages.application_id IS 'Application ID reference (validated via Main API)';
COMMENT ON COLUMN messages.store_item_id IS 'Store item ID reference (validated via Store API)';
COMMENT ON COLUMN messages.message_type IS 'Type of message: task, application, or store';
COMMENT ON COLUMN messages.deletion_scheduled_at IS 'When this message will be permanently deleted';

COMMENT ON TABLE user_activity IS 'Tracks user online status and last seen timestamps';
COMMENT ON COLUMN user_activity.user_id IS 'User ID (validated via Main API)';
COMMENT ON COLUMN user_activity.is_online IS 'Whether user is currently connected via WebSocket';

COMMENT ON TABLE message_deletion_warnings IS 'Tracks pending message deletions and warnings shown to users';

-- ============================================================================
-- VERIFICATION
-- ============================================================================

-- Verify tables were created
DO $$
DECLARE
    table_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO table_count
    FROM information_schema.tables
    WHERE table_schema = 'public'
    AND table_name IN ('messages', 'user_activity', 'message_deletion_warnings');

    IF table_count = 3 THEN
        RAISE NOTICE '✓ All 3 tables created successfully';
    ELSE
        RAISE WARNING '⚠ Expected 3 tables, found %', table_count;
    END IF;
END $$;

-- ============================================================================
-- MIGRATION COMPLETE
-- ============================================================================
