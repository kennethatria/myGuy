const express = require('express');
const router = express.Router();
const bookingMessageService = require('../services/bookingMessageService');
const { authenticateHTTP } = require('../middleware/auth');

/**
 * Internal endpoint for store-service to notify chat about booking requests
 * Secured with internal API key
 */
router.post('/internal/booking-created', async (req, res) => {
  try {
    const { bookingId, itemId, itemTitle, itemImage, buyerId, sellerId } = req.body;

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

    const message = await bookingMessageService.createBookingRequestMessage({
      bookingId,
      itemId,
      itemTitle: itemTitle || `Item #${itemId}`,
      itemImage: itemImage || null,
      buyerId,
      sellerId,
      io
    });

    console.log(`✅ Booking notification created: booking_id=${bookingId}, message_id=${message.id}`);
    res.json({ success: true, messageId: message.id });
  } catch (error) {
    console.error('Error creating booking notification:', error);
    res.status(500).json({ error: 'Failed to create booking notification' });
  }
});

/**
 * Endpoint for handling booking actions from chat UI
 * User clicks approve/decline in the chat interface
 */
router.post('/booking-action', authenticateHTTP, async (req, res) => {
  try {
    const { bookingId, action } = req.body; // action: 'approve' or 'decline'
    const userId = req.user.id;

    // Validate action
    if (!['approve', 'decline'].includes(action)) {
      return res.status(400).json({ error: 'Invalid action. Must be "approve" or "decline"' });
    }

    if (!bookingId) {
      return res.status(400).json({ error: 'Missing bookingId' });
    }

    // Call store-service to update booking status
    const storeApiUrl = process.env.STORE_API_URL || 'http://localhost:8081/api/v1';
    const endpoint = action === 'approve' ? 'approve' : 'reject';

    console.log(`📞 Calling store service: ${storeApiUrl}/items/booking-requests/${bookingId}/${endpoint}`);

    const response = await fetch(
      `${storeApiUrl}/items/booking-requests/${bookingId}/${endpoint}`,
      {
        method: 'POST',
        headers: {
          'Authorization': req.headers.authorization,
          'Content-Type': 'application/json'
        }
      }
    );

    if (!response.ok) {
      const errorText = await response.text();
      console.error(`Store service error: ${response.status} - ${errorText}`);
      throw new Error(`Failed to update booking status: ${response.status}`);
    }

    const booking = await response.json();

    // Get io instance from app
    const io = req.app.get('io');

    // Update chat message status
    await bookingMessageService.updateBookingMessageStatus(
      bookingId,
      booking.status,
      userId,
      io
    );

    res.json({ success: true, booking });
  } catch (error) {
    console.error('Error handling booking action:', error);
    res.status(500).json({ error: error.message });
  }
});

module.exports = router;
