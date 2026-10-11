-- ============================================================================
-- Migration: Account notice states
-- Date: 2026-10-11
-- Description: Accounts can now be suspended (read-only until a date) as well
-- as blocked. account_notices records what the partners of each account were
-- last told: 'flagged' (blocked), 'suspended:<ISO end>' (a new end is a new
-- notice) or 'active' (told it's over), replacing the flagged flag.
-- ============================================================================

ALTER TABLE account_notices ADD COLUMN IF NOT EXISTS state TEXT;
UPDATE account_notices SET state = CASE WHEN flagged THEN 'flagged' ELSE 'active' END WHERE state IS NULL;
ALTER TABLE account_notices ALTER COLUMN state SET NOT NULL;
ALTER TABLE account_notices DROP COLUMN IF EXISTS flagged;

COMMENT ON TABLE account_notices IS 'What the conversation partners of each blocked or suspended account were last told: flagged, suspended:<end> or active';
