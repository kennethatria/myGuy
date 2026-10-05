-- ============================================================================
-- Migration: Contact unlocks
-- Date: 2026-10-05
-- Description: Records pairs who agreed to work together (an accepted gig
-- application or an approved store booking). Messages between them in that
-- conversation are not contact-filtered.
-- ============================================================================

CREATE TABLE IF NOT EXISTS contact_unlocks (
    context_type VARCHAR(10) NOT NULL CHECK (context_type IN ('task', 'store')),
    context_id   INTEGER     NOT NULL,
    -- the two participants, smaller id first, so (a, b) and (b, a) match
    user_low     INTEGER     NOT NULL,
    user_high    INTEGER     NOT NULL,
    created_at   TIMESTAMP   NOT NULL DEFAULT NOW(),
    PRIMARY KEY (context_type, context_id, user_low, user_high),
    CHECK (user_low < user_high)
);

COMMENT ON TABLE contact_unlocks IS 'Pairs allowed to share contact details in one conversation, after agreeing to work together';
