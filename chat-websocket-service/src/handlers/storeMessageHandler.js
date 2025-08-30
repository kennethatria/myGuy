const db = require('../config/database');
const logger = require('../utils/logger');

/**
 * Handles store-specific message notifications and routing
 */
class StoreMessageHandler {
  constructor(io) {
    this.io = io;
  }

  /**
   * Get item owner information
   */
  async getItemOwner(itemId) {
    try {
      const query = `
        SELECT 
          si.seller_id,
          u.username as seller_username
        FROM store_items si
        JOIN users u ON si.seller_id = u.id
        WHERE si.id = $1
      `;
      const result = await db.query(query, [itemId]);
      return result.rows[0];
    } catch (error) {
      logger.error('Error getting item owner:', error);
      return null;
    }
  }

  /**
   * Handle new store message
   */
  async handleNewMessage(message, socket) {
    try {
      // Get item owner information
      const itemOwner = await this.getItemOwner(message.store_item_id);
      
      if (!itemOwner) {
        logger.error('Item owner not found for store message', {
          itemId: message.store_item_id,
          messageId: message.id
        });
        return;
      }

      // Join sender to the item room if not already joined
      const itemRoom = `item:${message.store_item_id}`;
      if (!socket.rooms.has(itemRoom)) {
        socket.join(itemRoom);
      }

      // Emit to the conversation room
      this.io.to(itemRoom).emit('message:new', {
        ...message,
        item_owner: {
          id: itemOwner.seller_id,
          username: itemOwner.seller_username
        }
      });

      // Notify item owner if they're not the sender
      if (itemOwner.seller_id !== message.sender_id) {
        this.io.to(`user:${itemOwner.seller_id}`).emit('store:message:notification', {
          messageId: message.id,
          itemId: message.store_item_id,
          senderId: message.sender_id,
          content: message.content,
          createdAt: message.created_at
        });
      }

      // Update conversation lists for both parties
      this.io.to(`user:${message.sender_id}`).emit('conversations:refresh');
      this.io.to(`user:${itemOwner.seller_id}`).emit('conversations:refresh');
    } catch (error) {
      logger.error('Error handling store message:', error);
    }
  }

  /**
   * Join store item room
   */
  async joinItemRoom(socket, itemId) {
    try {
      const itemOwner = await this.getItemOwner(itemId);
      const roomName = `item:${itemId}`;

      // Allow if user is either the item owner or has an existing conversation
      const canJoin = itemOwner.seller_id === socket.userId || 
                     await this.hasExistingConversation(itemId, socket.userId);

      if (canJoin) {
        socket.join(roomName);
        return true;
      }

      return false;
    } catch (error) {
      logger.error('Error joining item room:', error);
      return false;
    }
  }

  /**
   * Check if user has existing conversation for item
   */
  async hasExistingConversation(itemId, userId) {
    try {
      const query = `
        SELECT EXISTS(
          SELECT 1 FROM messages 
          WHERE store_item_id = $1 
          AND (sender_id = $2 OR recipient_id = $2)
        ) as has_conversation
      `;
      const result = await db.query(query, [itemId, userId]);
      return result.rows[0]?.has_conversation || false;
    } catch (error) {
      logger.error('Error checking existing conversation:', error);
      return false;
    }
  }

  /**
   * Get item details
   */
  async getItemDetails(itemId) {
    try {
      const query = `
        SELECT 
          si.*,
          u.username as seller_username,
          COUNT(m.id) as message_count
        FROM store_items si
        JOIN users u ON si.seller_id = u.id
        LEFT JOIN messages m ON m.store_item_id = si.id
        WHERE si.id = $1
        GROUP BY si.id, u.username
      `;
      const result = await db.query(query, [itemId]);
      return result.rows[0];
    } catch (error) {
      logger.error('Error getting item details:', error);
      return null;
    }
  }

  /**
   * Handle typing indicators for store messages
   */
  async handleTyping(socket, { itemId, isTyping }) {
    try {
      const roomName = `item:${itemId}`;
      
      // Only emit if user is in the room
      if (socket.rooms.has(roomName)) {
        const eventName = isTyping ? 'user:typing' : 'user:stopped-typing';
        socket.to(roomName).emit(eventName, {
          userId: socket.userId,
          itemId: itemId
        });
      }
    } catch (error) {
      logger.error('Error handling typing indicator:', error);
    }
  }
}

module.exports = StoreMessageHandler;
