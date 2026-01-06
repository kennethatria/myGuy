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

/**
 * Endpoint for handling booking actions from chat UI
 * User clicks approve/decline in the chat interface
 */
router.post('/booking-action', authenticateHTTP, async (req, res) => {
  try {
    const { bookingId, action, rating, review } = req.body;
    const userId = req.user.id;

    // Validate action
    if (!['approve', 'decline', 'confirm-received', 'confirm-delivery', 'rate-seller', 'rate-buyer'].includes(action)) {
      return res.status(400).json({ error: 'Invalid action' });
    }

    if (!bookingId) {
      return res.status(400).json({ error: 'Missing bookingId' });
    }

    // Validate rating if it's a rating action
    if ((action === 'rate-seller' || action === 'rate-buyer') && (!rating || rating < 1 || rating > 5)) {
      return res.status(400).json({ error: 'Rating must be between 1 and 5' });
    }

    // Call store-service to update booking status
    const storeApiUrl = process.env.STORE_API_URL || 'http://localhost:8081/api/v1';
    let endpoint;
    if (action === 'approve') {
      endpoint = 'approve';
    } else if (action === 'decline') {
      endpoint = 'reject';
    } else if (action === 'confirm-received') {
      endpoint = 'confirm-received';
    } else if (action === 'confirm-delivery') {
      endpoint = 'confirm-delivery';
    } else if (action === 'rate-seller') {
      endpoint = 'rate-seller';
    } else if (action === 'rate-buyer') {
      endpoint = 'rate-buyer';
    }

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
      const errorText = await response.text();
      console.error(`Store service error: ${response.status} - ${errorText}`);
      throw new Error(`Failed to update booking status: ${response.status}`);
    }

    const booking = await response.json();

    // Get io instance from app
    const io = req.app.get('io');

    // Update chat message status (pass full booking object for ratings)
    await bookingMessageService.updateBookingMessageStatus(
      bookingId,
      booking.status,
      userId,
      io,
      booking
    );

    res.json({ success: true, booking });
  } catch (error) {
    console.error('Error handling booking action:', error);
    res.status(500).json({ error: error.message });
  }
});

module.exports = router;
