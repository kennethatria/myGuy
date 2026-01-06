const db = require('../config/database');
const { filterContent } = require('../utils/contentFilter');
const logger = require('../utils/logger');

class MessageService {
  constructor() {
    // Initialization if needed
  }

  /**
   * Send a new message
   */
  async sendMessage({ taskId, applicationId, storeItemId, senderId, recipientId, content }) {
    const client = await db.getClient();

    try {
      await client.query('BEGIN');

      // Filter content
      const { filtered, hasRemovedContent } = filterContent(content);

      // Determine message type
      const messageType = taskId ? 'task' : (applicationId ? 'application' : 'store');

      // Store message
      const messageQuery = `
        INSERT INTO messages (task_id, application_id, store_item_id, sender_id, recipient_id, content, message_type, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
        RETURNING *
      `;

      const messageResult = await client.query(messageQuery, [
        taskId || null,
        applicationId || null,
        storeItemId || null,
        senderId,
        recipientId,
        filtered,
        messageType
      ]);

      // Update user activity
      await this.updateUserActivity(senderId, taskId || applicationId || storeItemId);

      await client.query('COMMIT');

      const message = messageResult.rows[0];
      message.hasRemovedContent = hasRemovedContent;

      return message;
    } catch (error) {
      await client.query('ROLLBACK');
      logger.error('Error sending message:', error);
      throw error;
    } finally {
      client.release();
    }
  }

  /**
   * Edit a message
   */
  async editMessage(messageId, userId, newContent) {
    const client = await db.getClient();

    try {
      await client.query('BEGIN');

      // Check if user owns the message
      const checkQuery = 'SELECT * FROM messages WHERE id = $1 AND sender_id = $2';
      const checkResult = await client.query(checkQuery, [messageId, userId]);

      if (checkResult.rows.length === 0) {
        throw new Error('Message not found or unauthorized');
      }

      // Filter new content
      const { filtered, hasRemovedContent } = filterContent(newContent);

      // Update message
      const updateQuery = `
        UPDATE messages
        SET content = $1,
            is_edited = true,
            edited_at = NOW()
        WHERE id = $2
        RETURNING *
      `;

      const result = await client.query(updateQuery, [filtered, messageId]);

      await client.query('COMMIT');

      const message = result.rows[0];
      message.hasRemovedContent = hasRemovedContent;

      return message;
    } catch (error) {
      await client.query('ROLLBACK');
      logger.error('Error editing message:', error);
      throw error;
    } finally {
      client.release();
    }
  }

  /**
   * Soft delete a message
   */
  async deleteMessage(messageId, userId) {
    const query = `
      UPDATE messages 
      SET is_deleted = true, 
          deleted_at = NOW(),
          content = '[Message deleted]'
      WHERE id = $1 AND sender_id = $2
      RETURNING *
    `;
    
    const result = await db.query(query, [messageId, userId]);
    
    if (result.rows.length === 0) {
      throw new Error('Message not found or unauthorized');
    }
    
    return result.rows[0];
  }

  /**
   * Mark message as read
   */
  async markAsRead(messageId, userId) {
    const query = `
      UPDATE messages 
      SET is_read = true, 
          read_at = NOW()
      WHERE id = $1 AND recipient_id = $2 AND is_read = false
      RETURNING *
    `;
    
    const result = await db.query(query, [messageId, userId]);
    return result.rows[0];
  }

  /**
   * Mark all messages in a conversation as read
   */
  async markConversationAsRead(taskId, userId) {
    const query = `
      UPDATE messages 
      SET is_read = true, 
          read_at = NOW()
      WHERE task_id = $1 
        AND recipient_id = $2 
        AND is_read = false
      RETURNING id
    `;
    
    const result = await db.query(query, [taskId, userId]);
    return result.rows;
  }

  /**
   * Get messages for a conversation with pagination
   */
  async getMessages({ taskId, applicationId, itemId, userId, limit = 5, offset = 0 }) {
    let messageQuery;
    let queryParams;
    let conversationId = taskId || applicationId || itemId;

    if (itemId) {
      // This is a store item conversation
      // Note: Removed store_items query - store_items table is in my_guy_store database
      // Access control is enforced by filtering messages where user is sender or recipient
      messageQuery = `
        SELECT m.*
        FROM messages m
        WHERE m.store_item_id = $1
          AND (m.sender_id = $2 OR m.recipient_id = $2)
        ORDER BY m.created_at DESC
        LIMIT $3 OFFSET $4
      `;
      queryParams = [itemId, userId, limit, offset];
    } else {
      // Task or application conversation
      // Note: Removed tasks/store_items queries - tables are in different databases
      // Access control is enforced by filtering messages where user is sender or recipient
      if (taskId) {
        messageQuery = `
          SELECT m.*
          FROM messages m
          WHERE m.task_id = $1
            AND (m.sender_id = $2 OR m.recipient_id = $2)
          ORDER BY m.created_at DESC
          LIMIT $3 OFFSET $4
        `;
        queryParams = [taskId, userId, limit, offset];
      } else if (applicationId) {
        messageQuery = `
          SELECT m.*
          FROM messages m
          WHERE m.application_id = $1
            AND (m.sender_id = $2 OR m.recipient_id = $2)
          ORDER BY m.created_at DESC
          LIMIT $3 OFFSET $4
        `;
        queryParams = [applicationId, userId, limit, offset];
      } else {
        return []; // Unknown conversation type
      }
    }
    
    const result = await db.query(messageQuery, queryParams);
    return result.rows.reverse(); // Reverse to show oldest first
  }

  /**
   * Get total message count for a conversation
   */
  async getTotalMessageCount({ taskId, applicationId, itemId, userId }) {
    if (itemId) {
      // Count store messages
      // Note: Removed store_items query - store_items table is in my_guy_store database
      // Access control is enforced by filtering messages where user is sender or recipient
      const query = `
        SELECT COUNT(*) as total
        FROM messages m
        WHERE m.store_item_id = $1
          AND (m.sender_id = $2 OR m.recipient_id = $2)
      `;
      const result = await db.query(query, [itemId, userId]);
      return parseInt(result.rows[0].total);
    } else if (taskId) {
      // Count task messages
      // Note: Removed tasks query - tasks table is in my_guy database
      // Access control is enforced by filtering messages where user is sender or recipient
      const query = `
        SELECT COUNT(*) as total
        FROM messages m
        WHERE m.task_id = $1
          AND (m.sender_id = $2 OR m.recipient_id = $2)
      `;
      const result = await db.query(query, [taskId, userId]);
      return parseInt(result.rows[0].total);
    } else if (applicationId) {
      // Count application messages
      const query = `
        SELECT COUNT(*) as total
        FROM messages m
        WHERE m.application_id = $1
          AND (m.sender_id = $2 OR m.recipient_id = $2)
      `;
      const result = await db.query(query, [applicationId, userId]);
      return parseInt(result.rows[0].total);
    }
    return 0;
  }

  /**
   * Get user conversations list
   */
  async getUserConversations(userId) {
    // Simplified query - no cross-database JOINs
    // Frontend should fetch task/user/item details via their respective APIs
    const query = `
      WITH ConversationMessages AS (
        SELECT DISTINCT ON (COALESCE(task_id, application_id, store_item_id))
          m.id,
          m.task_id,
          m.application_id,
          m.store_item_id,
          m.sender_id,
          m.recipient_id,
          m.content,
          m.message_type,
          m.metadata,
          m.is_read,
          m.created_at,
          m.updated_at,
          -- Determine other_user_id (person you're chatting with)
          CASE
            WHEN m.sender_id = $1 THEN m.recipient_id
            ELSE m.sender_id
          END as other_user_id
        FROM messages m
        WHERE m.sender_id = $1 OR m.recipient_id = $1
        ORDER BY COALESCE(task_id, application_id, store_item_id), created_at DESC
      ),
      UnreadCounts AS (
        SELECT
          COALESCE(task_id, application_id, store_item_id) as conversation_id,
          COUNT(*) as unread_count
        FROM messages
        WHERE recipient_id = $1 AND is_read = false
        GROUP BY COALESCE(task_id, application_id, store_item_id)
      )
      SELECT
        cm.*,
        COALESCE(uc.unread_count, 0) as unread_count
      FROM ConversationMessages cm
      LEFT JOIN UnreadCounts uc ON COALESCE(cm.task_id, cm.application_id, cm.store_item_id) = uc.conversation_id
      ORDER BY cm.created_at DESC
    `;

    const result = await db.query(query, [userId]);
    return result.rows;
  }

  /**
   * Update user's last activity
   */
  async updateUserActivity(userId, conversationId) {
    const query = `
      INSERT INTO user_activity (user_id, last_seen, last_conversation_id)
      VALUES ($1, NOW(), $2)
      ON CONFLICT (user_id) 
      DO UPDATE SET 
        last_seen = NOW(),
        last_conversation_id = $2
    `;
    
    await db.query(query, [userId, conversationId]);
  }

  /**
   * Get user's last seen
   */
  async getUserLastSeen(userId) {
    const query = 'SELECT last_seen FROM user_activity WHERE user_id = $1';
    const result = await db.query(query, [userId]);
    
    if (result.rows.length === 0) {
      return null;
    }
    
    return result.rows[0].last_seen;
  }

  /**
   * Check for messages to be deleted
   */
  async getMessagesForDeletion() {
    // Simplified query - no cross-database JOINs
    // Find old messages (> 6 months) that should be considered for deletion
    // Task status checking should be done via API in the scheduler
    const query = `
      SELECT DISTINCT
        m.task_id,
        m.application_id,
        m.store_item_id,
        MAX(m.created_at) as last_message_date,
        COUNT(m.id) as message_count,
        MIN(m.sender_id) as first_user_id,
        MIN(m.recipient_id) as second_user_id
      FROM messages m
      WHERE m.created_at < NOW() - INTERVAL '6 months'
        AND m.is_deleted = false
      GROUP BY m.task_id, m.application_id, m.store_item_id
      HAVING COUNT(m.id) > 0
    `;

    const result = await db.query(query);
    return result.rows;
  }

  /**
   * Schedule message deletion warning
   */
  async createDeletionWarning(taskId, deletionDate) {
    const query = `
      INSERT INTO message_deletion_warnings (task_id, deletion_scheduled_at, warning_shown)
      VALUES ($1, $2, false)
      ON CONFLICT (task_id) DO NOTHING
    `;
    
    await db.query(query, [taskId, deletionDate]);
  }

  /**
   * Get deletion warnings for user
   */
  async getUserDeletionWarnings(userId) {
    // Simplified query - no cross-database JOINs
    // Returns warnings for messages where user is sender or recipient
    const query = `
      SELECT DISTINCT
        mdw.*
      FROM message_deletion_warnings mdw
      WHERE mdw.user_id = $1
        AND mdw.deletion_scheduled_at > NOW()
        AND mdw.deletion_scheduled_at < NOW() + INTERVAL '1 month'
        AND mdw.warning_shown = false
    `;
    
    const result = await db.query(query, [userId]);
    return result.rows;
  }

  /**
   * Mark warning as shown
   */
  async markWarningAsShown(warningId) {
    const query = `
      UPDATE message_deletion_warnings 
      SET warning_shown = true 
      WHERE id = $1
    `;
    
    await db.query(query, [warningId]);
  }

  /**
   * Delete old messages
   */
  async deleteOldMessages(taskId) {
    const query = `
      DELETE FROM messages 
      WHERE task_id = $1
      RETURNING COUNT(*)
    `;
    
    const result = await db.query(query, [taskId]);
    logger.info(`Deleted ${result.rows[0].count} messages for task ${taskId}`);
    return result.rows[0].count;
  }

  /**
   * Store-specific message methods
   */

  /**
   * Get store messages for a specific item (only between involved parties)
   * Returns message data with user IDs only - frontend should fetch user details via Main API
   */
  async getStoreMessages(itemId, userId) {
    const query = `
      SELECT m.*
      FROM messages m
      WHERE m.store_item_id = $1
        AND (m.sender_id = $2 OR m.recipient_id = $2)
      ORDER BY m.created_at ASC
    `;

    const result = await db.query(query, [itemId, userId]);
    return result.rows;
  }

  /**
   * Create a new store message
   */
  async createStoreMessage({ store_item_id, sender_id, recipient_id, content }) {
    const client = await db.getClient();

    try {
      await client.query('BEGIN');

      // Filter content
      const { filtered, hasRemovedContent } = filterContent(content);

      // Store message in unified messages table
      const messageQuery = `
        INSERT INTO messages (store_item_id, sender_id, recipient_id, content, message_type, created_at)
        VALUES ($1, $2, $3, $4, 'store', NOW())
        RETURNING *
      `;

      const messageResult = await client.query(messageQuery, [
        store_item_id,
        sender_id,
        recipient_id,
        filtered
      ]);

      await client.query('COMMIT');

      const message = messageResult.rows[0];
      message.hasRemovedContent = hasRemovedContent;

      return message;
    } catch (error) {
      await client.query('ROLLBACK');
      logger.error('Error creating store message:', error);
      throw error;
    } finally {
      client.release();
    }
  }

  /**
   * Get user's message count for a specific store item
   */
  async getUserStoreMessageCount(itemId, userId) {
    const query = `
      SELECT COUNT(*) as count
      FROM messages
      WHERE store_item_id = $1 AND sender_id = $2
    `;
    
    const result = await db.query(query, [itemId, userId]);
    return parseInt(result.rows[0].count);
  }

  /**
   * Check booking status for dynamic message limits
   * TODO: Integrate with ValidationService to check via Store API
   */
  async getBookingStatus(itemId, userId) {
    try {
      // booking_requests table exists in my_guy_store database (not accessible from my_guy_chat)
      // For now, return null to default to 3 message limit
      // Future: Use ValidationService to check via Store API endpoint
      logger.warn(`Booking status check not implemented for item ${itemId} (separate database)`);
      return null;
    } catch (error) {
      logger.error('Error checking booking status:', error);
      return null;
    }
  }

  /**
   * Get dynamic message limit based on booking status
   */
  async getMessageLimit(itemId, userId) {
    const bookingStatus = await this.getBookingStatus(itemId, userId);
    
    // 3 messages before booking approval, 10 messages after approval
    return bookingStatus === 'approved' ? 10 : 3;
  }

  /**
   * Task message limit methods
   */

  /**
   * Get user's message count for a specific task
   */
  async getUserTaskMessageCount(taskId, userId) {
    const query = `
      SELECT COUNT(*) as count
      FROM messages
      WHERE task_id = $1 AND sender_id = $2
    `;
    
    const result = await db.query(query, [taskId, userId]);
    return parseInt(result.rows[0].count);
  }

  /**
   * Get message limit for a task based on user role and assignment status
   * TODO: Integrate with ValidationService to check task ownership via Main API
   */
  async getTaskMessageLimit(taskId, userId) {
    try {
      // tasks table exists in my_guy database (not accessible from my_guy_chat)
      // For now, return default limit of 3 messages
      // Future: Use ValidationService to check task ownership/assignment via Main API endpoint
      logger.warn(`Task message limit check not implemented for task ${taskId} (separate database)`);
      return 3; // Default to safe limit
    } catch (error) {
      logger.error('Error getting task message limit:', error);
      return 3; // Default to safe limit on error
    }
  }
}

module.exports = new MessageService();