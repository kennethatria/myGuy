-- ============================================================================
-- Migration: Account notices
-- Date: 2026-10-10
-- Description: Which blocked accounts their conversation partners have been
-- told about. The backend owns blocks; chat reads the blocked accounts every
-- minute and, once per change, posts "flagged" (or "active again") into each
-- of that account's conversations. Claiming the row first means only one chat
-- instance posts, and a restart doesn't post again.
-- ============================================================================

CREATE TABLE IF NOT EXISTS account_notices (
    user_id    INTEGER   PRIMARY KEY,
    flagged    BOOLEAN   NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE account_notices IS 'Blocked accounts whose conversation partners were told (flagged), or told again when unblocked';
