const db = require('../db');

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
        `Booking request for ${itemTitle}`,
        JSON.stringify({
          booking_id: bookingId,
          item_id: itemId,
          item_title: itemTitle,
          item_image: itemImage,
          status: 'pending'
        })
      ]
    );

    const message = result.rows[0];

    // Emit to seller via WebSocket
    if (io) {
      io.to(`user_${sellerId}`).emit('message:new', message);
      console.log(`📋 Booking request message sent to seller ${sellerId} for item ${itemId}`);
    }

    return message;
  } catch (error) {
    console.error('Error creating booking request message:', error);
    throw error;
  }
}

/**
 * Update booking message status and create status update message
 */
async function updateBookingMessageStatus(bookingId, status, approverId, io) {
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

    // Update the original message metadata
    const updatedMetadata = {
      ...requestMessage.metadata,
      status: status
    };

    await db.query(
      `UPDATE messages
       SET metadata = $1
       WHERE id = $2`,
      [JSON.stringify(updatedMetadata), requestMessage.id]
    );

    // Create a new system message for the status change
    const messageType = status === 'approved' ? 'booking_approved' : 'booking_declined';
    const content = status === 'approved'
      ? `Booking approved ✅. You can now discuss pickup details.`
      : `Booking request was declined.`;

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

    const statusMessage = statusResult.rows[0];

    // Emit to both users via WebSocket
    if (io) {
      io.to(`user_${requestMessage.sender_id}`).emit('message:new', statusMessage);
      io.to(`user_${approverId}`).emit('message:new', statusMessage);

      // Also emit update for the original message
      io.to(`user_${requestMessage.sender_id}`).emit('message:updated', {
        ...requestMessage,
        metadata: updatedMetadata
      });
      io.to(`user_${approverId}`).emit('message:updated', {
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
