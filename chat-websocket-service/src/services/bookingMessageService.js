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
        message || `Booking request for ${itemTitle}`,
        JSON.stringify({
          booking_id: bookingId,
          item_id: itemId,
          item_title: itemTitle,
          item_image: itemImage,
          status: 'pending'
        })
      ]
    );

    const createdMessage = result.rows[0];

    // Emit to seller via WebSocket
    if (io) {
      io.to(`user:${sellerId}`).emit('message:new', createdMessage);
      console.log(`📋 Booking request message sent to seller ${sellerId} for item ${itemId}`);
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
async function updateBookingMessageStatus(bookingId, status, approverId, io, bookingData = null) {
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
    // This prevents duplicate messages if the function is called multiple times
    const existingStatusMessage = await db.query(
      `SELECT * FROM messages
       WHERE store_item_id = $1
       AND metadata->>'booking_id' = $2
       AND metadata->>'status' = $3
       ORDER BY created_at DESC
       LIMIT 1`,
      [requestMessage.store_item_id, bookingId.toString(), status]
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

      if (status === 'approved') {
        messageType = 'booking_approved';
        content = 'Booking approved ✅. You can now discuss pickup details.';
      } else if (status === 'rejected') {
        messageType = 'booking_declined';
        content = 'Booking request was declined.';
      } else if (status === 'item_received') {
        messageType = 'booking_item_received';
        content = '📦 Buyer confirmed they received the item.';
      } else if (status === 'completed') {
        messageType = 'booking_completed';
        content = '✅ Transaction completed! Both parties have confirmed.';
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
          requestMessage.sender_id,
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

    // Emit to both users via WebSocket
    if (io) {
      io.to(`user:${requestMessage.sender_id}`).emit('message:new', statusMessage);
      io.to(`user:${approverId}`).emit('message:new', statusMessage);

      // Also emit update for the original message
      io.to(`user:${requestMessage.sender_id}`).emit('message:updated', {
        ...requestMessage,
        metadata: updatedMetadata
      });
      io.to(`user:${approverId}`).emit('message:updated', {
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
