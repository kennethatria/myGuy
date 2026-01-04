-- ============================================================================
-- Migration: Add Booking Message Support
-- Date: 2026-01-04
-- Description: Adds support for booking-related system messages
-- ============================================================================

-- Add metadata column for structured message data
ALTER TABLE messages
ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Add index for metadata queries
CREATE INDEX IF NOT EXISTS idx_messages_metadata_booking_id
    ON messages((metadata->>'booking_id'))
    WHERE metadata IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_messages_metadata_item_id
    ON messages((metadata->>'item_id'))
    WHERE metadata IS NOT NULL;

-- Add comments
COMMENT ON COLUMN messages.metadata IS 'Structured data for special message types (booking requests, system alerts, etc.)';

-- Update message_type comment to reflect new types
COMMENT ON COLUMN messages.message_type IS 'Type of message: text (default), booking_request, booking_approved, booking_declined, system_alert';

-- ============================================================================
-- VERIFICATION
-- ============================================================================

DO $$
BEGIN
    -- Check if metadata column exists
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'messages'
        AND column_name = 'metadata'
    ) THEN
        RAISE NOTICE '✓ metadata column added successfully';
    ELSE
        RAISE WARNING '⚠ metadata column was not created';
    END IF;

    -- Check if indexes exist
    IF EXISTS (
        SELECT 1
        FROM pg_indexes
        WHERE indexname = 'idx_messages_metadata_booking_id'
    ) THEN
        RAISE NOTICE '✓ booking_id index created successfully';
    ELSE
        RAISE WARNING '⚠ booking_id index was not created';
    END IF;
END $$;

-- ============================================================================
-- Example metadata structure for booking messages:
-- {
--   "booking_id": 123,
--   "item_id": 456,
--   "item_title": "Red Bicycle",
--   "item_image": "/uploads/store/bicycle.jpg",
--   "status": "pending"
-- }
-- ============================================================================

-- Migration complete
