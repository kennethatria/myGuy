const express = require('express');
const router = express.Router();
const db = require('../config/database');
const bookingMessageService = require('../services/bookingMessageService');
const messageService = require('../services/messageService');
const { authenticateHTTP } = require('../middleware/auth');

/**
 * Internal endpoint for store-service to notify chat about booking requests
 * Secured with internal API key
 */
router.post('/internal/booking-created', async (req, res) => {
  try {
    const { bookingId, itemId, itemTitle, itemImage, buyerId, sellerId, message } = req.body;

    // Validate internal API key
    const internalApiKey = req.headers['x-internal-api-key'];
    if (!internalApiKey || internalApiKey !== process.env.INTERNAL_API_KEY) {
      console.warn('⚠️ Unauthorized booking notification attempt');
      return res.status(401).json({ error: 'Unauthorized' });
    }

    // Validate required fields
    if (!bookingId || !itemId || !buyerId || !sellerId) {
      return res.status(400).json({ error: 'Missing required fields' });
    }

    // Get io instance from app
    const io = req.app.get('io');

    const createdMessage = await bookingMessageService.createBookingRequestMessage({
      bookingId,
      itemId,
      itemTitle: itemTitle || `Item #${itemId}`,
      itemImage: itemImage || null,
      buyerId,
      sellerId,
      message: message || `Booking request for ${itemTitle || `Item #${itemId}`}`,
      io
    });

    console.log(`✅ Booking notification created: booking_id=${bookingId}, message_id=${createdMessage.id}`);
    res.json({ success: true, messageId: createdMessage.id });
  } catch (error) {
    console.error('Error creating booking notification:', error);
    res.status(500).json({ error: 'Failed to create booking notification' });
  }
});

const hasInternalKey = (req) => {
  const internalApiKey = req.headers['x-internal-api-key'];
  return !!internalApiKey && internalApiKey === process.env.INTERNAL_API_KEY;
};

// Stores a system_alert in a conversation and delivers it live to both
// participants (all their tabs). context is { taskId } or { storeItemId }.
async function postSystemMessage(req, context, senderId, recipientId, content, metadata) {
  const message = await messageService.sendMessage({
    ...context, senderId, recipientId, content, messageType: 'system_alert', metadata
  });
  const formatted = {
    ...message,
    sender: { id: senderId, username: 'User' },
    recipient: { id: recipientId, username: 'User' }
  };

  const io = req.app.get('io');
  if (io) {
    io.to(`user:${senderId}`).to(`user:${recipientId}`).emit('message:new', formatted);
  }
  return message;
}

// Gig events the app shows actions on (accept, mark as done, approve, review)
const TASK_EVENTS = new Set(['application', 'accepted', 'declined', 'cancelled', 'done', 'not_done', 'completed']);

// The event a task message is about, or undefined. Only known events and a
// numeric application id are kept.
function taskEventMetadata(raw) {
  if (!raw || typeof raw !== 'object' || !TASK_EVENTS.has(raw.event)) return undefined;
  const metadata = { event: raw.event };
  const applicationId = parseInt(raw.application_id);
  if (applicationId > 0) metadata.application_id = applicationId;
  return metadata;
}

/**
 * Internal endpoint for the main API to post task events (new application,
 * accepted, declined, cancelled) into the owner↔applicant task conversation.
 * Stored as 'system_alert' so they can't be edited or deleted.
 * Secured with the internal API key.
 */
router.post('/internal/task-message', async (req, res) => {
  try {
    if (!hasInternalKey(req)) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const taskId = parseInt(req.body.task_id);
    const senderId = parseInt(req.body.sender_id);
    const recipientId = parseInt(req.body.recipient_id);
    const content = typeof req.body.content === 'string' ? req.body.content.trim() : '';
    const unlock = req.body.unlock_contacts === true;
    // Content may be left out only to record a match (catching up old ones)
    if (!taskId || !senderId || !recipientId || (!content && !unlock)) {
      return res.status(400).json({ error: 'task_id, sender_id, recipient_id and content are required' });
    }

    // An accepted application: the pair agreed to work together, so they
    // may now chat and share contact details in this conversation.
    if (unlock) {
      await messageService.unlockContacts({ taskId, userA: senderId, userB: recipientId });
    }
    if (!content) {
      return res.status(204).end();
    }

    const message = await postSystemMessage(
      req, { taskId }, senderId, recipientId, content, taskEventMetadata(req.body.metadata)
    );
    res.status(201).json({ id: message.id });
  } catch (error) {
    console.error('Error posting task message:', error);
    res.status(500).json({ error: 'Failed to post task message' });
  }
});

/**
 * Internal endpoint for store-service to post item events (a listing made
 * for someone's request) into the seller↔buyer conversation about the item.
 * Stored as 'system_alert'. Secured with the internal API key.
 */
router.post('/internal/store-message', async (req, res) => {
  try {
    if (!hasInternalKey(req)) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const storeItemId = parseInt(req.body.store_item_id);
    const senderId = parseInt(req.body.sender_id);
    const recipientId = parseInt(req.body.recipient_id);
    const content = typeof req.body.content === 'string' ? req.body.content.trim() : '';
    if (!storeItemId || !senderId || !recipientId || !content) {
      return res.status(400).json({ error: 'store_item_id, sender_id, recipient_id and content are required' });
    }

    const message = await postSystemMessage(req, { storeItemId }, senderId, recipientId, content);
    res.status(201).json({ id: message.id });
  } catch (error) {
    console.error('Error posting store message:', error);
    res.status(500).json({ error: 'Failed to post store message' });
  }
});

/**
 * Endpoint for handling booking actions from chat UI
 * User clicks approve/decline in the chat interface
 */
// Chat action → store-service endpoint. Only these can be called.
const STORE_BOOKING_ENDPOINTS = {
  'approve': 'approve',
  'decline': 'reject',
  'confirm-received': 'confirm-received',
  'confirm-delivery': 'confirm-delivery',
  'rate-seller': 'rate-seller',
  'rate-buyer': 'rate-buyer'
};

router.post('/booking-action', authenticateHTTP, async (req, res) => {
  try {
    const { action, rating, review } = req.body;
    const userId = req.user.id;

    // Validate action
    const endpoint = Object.prototype.hasOwnProperty.call(STORE_BOOKING_ENDPOINTS, action)
      ? STORE_BOOKING_ENDPOINTS[action]
      : null;
    if (!endpoint) {
      return res.status(400).json({ error: 'Invalid action' });
    }

    // The id goes into the store-service URL: accept only a positive integer
    const bookingId = Number(req.body.bookingId);
    if (!Number.isSafeInteger(bookingId) || bookingId <= 0) {
      return res.status(400).json({ error: 'Missing or invalid bookingId' });
    }

    // Validate rating if it's a rating action
    if ((action === 'rate-seller' || action === 'rate-buyer') && (!rating || rating < 1 || rating > 5)) {
      return res.status(400).json({ error: 'Rating must be between 1 and 5' });
    }

    // Call store-service to update booking status
    const storeApiUrl = process.env.STORE_API_URL || 'http://localhost:8081/api/v1';

    console.log(`📞 Calling store service: ${storeApiUrl}/booking-requests/${bookingId}/${endpoint}`);

    // Build request body for rating actions
    const requestBody = (action === 'rate-seller' || action === 'rate-buyer')
      ? JSON.stringify({ rating, review: review || '' })
      : null;

    const response = await fetch(
      `${storeApiUrl}/booking-requests/${bookingId}/${endpoint}`,
      {
        method: 'POST',
        headers: {
          'Authorization': req.headers.authorization,
          'Content-Type': 'application/json'
        },
        ...(requestBody && { body: requestBody })
      }
    );

    if (!response.ok) {
      // store-service only puts messages meant for users in `error`
      // (anything else it answers generically), so pass that on as is
      const body = await response.json().catch(() => ({}));
      console.error(`Store service refused booking action ${action}: ${response.status}`);
      return res.status(response.status >= 500 ? 502 : response.status)
        .json({ error: body.error || 'Could not update the booking. Please try again.' });
    }

    const booking = await response.json();

    // An approved booking: store-service checked this user is the seller, so
    // seller and buyer may now share contact details about this item.
    if (action === 'approve') {
      await messageService.unlockContacts({ storeItemId: booking.item_id, userA: userId, userB: booking.requester_id });
    }

    // Get io instance from app
    const io = req.app.get('io');

    // For rating actions, only update metadata without creating duplicate status messages
    // Since ratings don't change the booking status (it stays 'completed'),
    // we don't need to create another "Transaction completed" message
    if (action === 'rate-seller' || action === 'rate-buyer') {
      // Find the original booking request message
      const findResult = await db.query(
        `SELECT * FROM messages
         WHERE message_type = 'booking_request'
         AND metadata->>'booking_id' = $1
         LIMIT 1`,
        [bookingId.toString()]
      );

      if (findResult.rows.length > 0) {
        const requestMessage = findResult.rows[0];

        // Update metadata with ratings
        const updatedMetadata = {
          ...requestMessage.metadata,
          status: booking.status
        };

        // Add rating data to metadata
        if (booking.buyer_rating !== undefined && booking.buyer_rating !== null) {
          updatedMetadata.buyer_rating = booking.buyer_rating;
        }
        if (booking.buyer_review !== undefined && booking.buyer_review !== null) {
          updatedMetadata.buyer_review = booking.buyer_review;
        }
        if (booking.seller_rating !== undefined && booking.seller_rating !== null) {
          updatedMetadata.seller_rating = booking.seller_rating;
        }
        if (booking.seller_review !== undefined && booking.seller_review !== null) {
          updatedMetadata.seller_review = booking.seller_review;
        }

        // Update the message metadata
        await db.query(
          `UPDATE messages SET metadata = $1 WHERE id = $2`,
          [JSON.stringify(updatedMetadata), requestMessage.id]
        );

        // Emit update to both users (buyer and seller)
        const updatedMessage = { ...requestMessage, metadata: updatedMetadata };
        io.to(`user:${requestMessage.sender_id}`).emit('message:updated', updatedMessage);

        // Get seller ID from the booking/item
        if (booking.item && booking.item.seller_id) {
          io.to(`user:${booking.item.seller_id}`).emit('message:updated', updatedMessage);
        }

        console.log(`✅ Rating submitted for booking ${bookingId} - metadata updated without duplicate status message`);
      }
    } else {
      // For non-rating actions (approve, decline, confirm-received, confirm-delivery),
      // create status message as normal
      await bookingMessageService.updateBookingMessageStatus(
        bookingId,
        booking.status,
        userId,
        io,
        booking
      );
    }

    res.json({ success: true, booking });
  } catch (error) {
    console.error('Error handling booking action:', error);
    res.status(500).json({ error: 'Could not update the booking. Please try again.' });
  }
});

module.exports = router;
