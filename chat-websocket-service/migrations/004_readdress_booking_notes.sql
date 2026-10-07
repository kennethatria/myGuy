-- ============================================================================
-- Migration: Re-address booking notes sent to their own author
-- Date: 2026-10-07
-- Description: When a buyer took a booking step, its note was stored from the
-- buyer to the buyer, which showed up as a separate conversation with
-- themselves about the item. Each such note now goes to the seller (the
-- recipient of that booking's request), so it sits in the real conversation.
-- ============================================================================

UPDATE messages AS note
SET recipient_id = request.recipient_id
FROM messages AS request
WHERE note.sender_id = note.recipient_id
  AND note.store_item_id IS NOT NULL
  AND note.metadata ? 'booking_id'
  AND request.message_type = 'booking_request'
  AND request.metadata->>'booking_id' = note.metadata->>'booking_id'
  AND request.recipient_id <> note.sender_id;
