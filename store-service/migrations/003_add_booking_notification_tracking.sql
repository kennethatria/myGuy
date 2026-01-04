-- ============================================================================
-- Migration: Add Booking Notification Tracking
-- Date: 2026-01-04
-- Description: Adds columns to track chat service notification status for bookings
-- ============================================================================

-- Add columns to booking_requests table
ALTER TABLE booking_requests
ADD COLUMN IF NOT EXISTS chat_notified BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS notification_attempts INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS last_notification_attempt TIMESTAMP;

-- Add index for querying failed notifications
CREATE INDEX IF NOT EXISTS idx_booking_requests_chat_notified
    ON booking_requests(chat_notified, status)
    WHERE chat_notified = false AND status = 'pending';

-- Add comments
COMMENT ON COLUMN booking_requests.chat_notified IS 'Whether chat service was successfully notified about this booking';
COMMENT ON COLUMN booking_requests.notification_attempts IS 'Number of attempts to notify chat service';
COMMENT ON COLUMN booking_requests.last_notification_attempt IS 'Timestamp of last notification attempt';

-- ============================================================================
-- VERIFICATION
-- ============================================================================

DO $$
DECLARE
    col_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO col_count
    FROM information_schema.columns
    WHERE table_name = 'booking_requests'
    AND column_name IN ('chat_notified', 'notification_attempts', 'last_notification_attempt');

    IF col_count = 3 THEN
        RAISE NOTICE '✓ All 3 notification tracking columns added successfully';
    ELSE
        RAISE WARNING '⚠ Expected 3 columns, found %', col_count;
    END IF;

    -- Check index
    IF EXISTS (
        SELECT 1
        FROM pg_indexes
        WHERE indexname = 'idx_booking_requests_chat_notified'
    ) THEN
        RAISE NOTICE '✓ Notification tracking index created successfully';
    ELSE
        RAISE WARNING '⚠ Notification tracking index was not created';
    END IF;
END $$;

-- Migration complete
