const db = require('../config/database');

/**
 * Create a booking request system message
 */
async function createBookingRequestMessage({
  bookingId,
  itemId,
  itemTitle,
  itemImage,
  buyerId,
  sellerId,
  message,
  io
}) {
  try {
    // Create system message in chat
    const result = await db.query(
      `INSERT INTO messages (
        sender_id,
        recipient_id,
        store_item_id,
        message_type,
        content,
        metadata,
        created_at
      ) VALUES ($1, $2, $3, $4, $5, $6, NOW())
      RETURNING *`,
      [
        buyerId,
        sellerId,
        itemId,
        'booking_request',
        // The text previews the conversation; the buyer's own note, if any,
        // is kept apart so the booking card quotes only what they wrote
        message || `Booking request for ${itemTitle}`,
        JSON.stringify({
          booking_id: bookingId,
          item_id: itemId,
          item_title: itemTitle,
          item_image: itemImage,
          note: message || '',
          status: 'pending'
        })
      ]
    );

    const createdMessage = result.rows[0];

    // Deliver to both: the seller answers it, and the buyer sees it in the
    // conversation they may already have open (e.g. booking from chat)
    if (io) {
      io.to(`user:${sellerId}`).to(`user:${buyerId}`).emit('message:new', createdMessage);
      console.log(`📋 Booking request message sent for item ${itemId}`);
    }

    return createdMessage;
  } catch (error) {
    console.error('Error creating booking request message:', error);
    throw error;
  }
}

/**
 * Update booking message status and create status update message
 */
// note, when given, replaces the usual status note: a system message saying
// why (e.g. the seller removed the listing).
// approverId is whoever acted (seller or buyer); the note goes to the other.
async function updateBookingMessageStatus(bookingId, status, approverId, io, bookingData = null, note = null) {
  try {
    // Find the original booking request message
    const findResult = await db.query(
      `SELECT * FROM messages
       WHERE message_type = 'booking_request'
       AND metadata->>'booking_id' = $1
       LIMIT 1`,
      [bookingId.toString()]
    );

    if (findResult.rows.length === 0) {
      throw new Error('Booking request message not found');
    }

    const requestMessage = findResult.rows[0];

    // The buyer sent the request to the seller; a note goes from whoever
    // acted to the other one (never to themselves, which would make a
    // conversation of its own)
    const buyerId = Number(requestMessage.sender_id);
    const sellerId = Number(requestMessage.recipient_id);
    const otherParty = Number(approverId) === buyerId ? sellerId : buyerId;

    // Update the original message metadata with status and ratings
    const updatedMetadata = {
      ...requestMessage.metadata,
      status: status
    };

    // If we have booking data with ratings, include them in metadata
    if (bookingData) {
      if (bookingData.buyer_rating !== undefined && bookingData.buyer_rating !== null) {
        updatedMetadata.buyer_rating = bookingData.buyer_rating;
      }
      if (bookingData.buyer_review !== undefined && bookingData.buyer_review !== null) {
        updatedMetadata.buyer_review = bookingData.buyer_review;
      }
      if (bookingData.seller_rating !== undefined && bookingData.seller_rating !== null) {
        updatedMetadata.seller_rating = bookingData.seller_rating;
      }
      if (bookingData.seller_review !== undefined && bookingData.seller_review !== null) {
        updatedMetadata.seller_review = bookingData.seller_review;
      }
    }

    await db.query(
      `UPDATE messages
       SET metadata = $1
       WHERE id = $2`,
      [JSON.stringify(updatedMetadata), requestMessage.id]
    );

    // Check if a status message for this booking and status already exists
    // This prevents duplicate messages if the function is called multiple times.
    // The request itself (just given this status above) doesn't count.
    const existingStatusMessage = await db.query(
      `SELECT * FROM messages
       WHERE store_item_id = $1
       AND metadata->>'booking_id' = $2
       AND metadata->>'status' = $3
       AND id <> $4
       ORDER BY created_at DESC
       LIMIT 1`,
      [requestMessage.store_item_id, bookingId.toString(), status, requestMessage.id]
    );

    let statusMessage;

    // If a status message already exists for this status, reuse it instead of creating duplicate
    if (existingStatusMessage.rows.length > 0) {
      statusMessage = existingStatusMessage.rows[0];
      console.log(`ℹ️ Status message already exists for booking ${bookingId} status ${status} - skipping duplicate creation`);
    } else {
      // Create a new system message for the status change
      let messageType;
      let content;

      if (note) {
        messageType = 'system_alert';
        content = note;
      } else if (status === 'approved') {
        messageType = 'booking_approved';
        content = '✅ Booking approved. Arrange the pickup here; the seller marks it picked up.';
      } else if (status === 'rejected') {
        messageType = 'booking_declined';
        content = 'Booking request was declined.';
      } else if (status === 'picked_up') {
        messageType = 'booking_picked_up';
        content = '📦 The seller marked it picked up. Confirm once you have it.';
      } else if (status === 'item_received') {
        messageType = 'booking_item_received';
        content = '📦 Buyer confirmed they received the item.';
      } else if (status === 'completed') {
        messageType = 'booking_completed';
        content = '🎉 Sale complete. Leave each other a review.';
      } else if (status === 'released') {
        // A note to both, not anyone's message to edit or delete
        messageType = 'system_alert';
        content = '↩️ The seller released the reservation. The item is back on the board.';
      } else {
        messageType = 'booking_status_update';
        content = `Booking status updated to: ${status}`;
      }

      const statusResult = await db.query(
        `INSERT INTO messages (
          sender_id,
          recipient_id,
          store_item_id,
          message_type,
          content,
          metadata,
          created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, NOW())
        RETURNING *`,
        [
          approverId,
          otherParty,
          requestMessage.store_item_id,
          messageType,
          content,
          JSON.stringify({
            booking_id: bookingId,
            item_id: requestMessage.metadata.item_id,
            status: status
          })
        ]
      );

      statusMessage = statusResult.rows[0];
      console.log(`✅ Created new status message for booking ${bookingId} status ${status}`);
    }

    // Deliver to both buyer and seller, whoever acted
    if (io) {
      io.to(`user:${buyerId}`).to(`user:${sellerId}`).emit('message:new', statusMessage);
      io.to(`user:${buyerId}`).to(`user:${sellerId}`).emit('message:updated', {
        ...requestMessage,
        metadata: updatedMetadata
      });

      console.log(`✅ Booking ${bookingId} ${status} - notifications sent to both parties`);
    }

    return statusMessage;
  } catch (error) {
    console.error('Error updating booking message status:', error);
    throw error;
  }
}

module.exports = {
  createBookingRequestMessage,
  updateBookingMessageStatus
};
