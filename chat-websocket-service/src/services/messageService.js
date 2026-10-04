const db = require('../config/database');
const { filterContent } = require('../utils/contentFilter');
const logger = require('../utils/logger');

/**
 * SQL filter selecting one conversation: its context (task, application or
 * store item) AND its two participants. Including the other participant is
 * what keeps a seller's chats with different buyers of the same item apart.
 * Uses $1 (context id), $2 (userId) and, when given, $3 (otherUserId).
 */
function conversationFilter({ taskId, applicationId, itemId, userId, otherUserId }) {
  const column = taskId ? 'task_id' : applicationId ? 'application_id' : itemId ? 'store_item_id' : null;
  if (!column) return null;

  const params = [taskId || applicationId || itemId, userId];
  let sql = `m.${column} = $1 AND (m.sender_id = $2 OR m.recipient_id = $2)`;
  if (otherUserId) {
    params.push(otherUserId);
    sql += ' AND (m.sender_id = $3 OR m.recipient_id = $3)';
  }
  return { sql, params };
}

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
  async markConversationAsRead(userId, { taskId, applicationId, itemId, otherUserId }) {
    const filter = conversationFilter({ taskId, applicationId, itemId, userId, otherUserId });
    if (!filter) return [];

    const query = `
      UPDATE messages m
      SET is_read = true,
          read_at = NOW()
      WHERE ${filter.sql}
        AND m.recipient_id = $2
        AND m.is_read = false
      RETURNING m.id, m.sender_id
    `;

    const result = await db.query(query, filter.params);
    return result.rows;
  }

  /**
   * Get messages for a conversation with pagination
   */
  async getMessages({ taskId, applicationId, itemId, userId, otherUserId, limit = 5, offset = 0 }) {
    // Access control: only messages the user sent or received are returned.
    const filter = conversationFilter({ taskId, applicationId, itemId, userId, otherUserId });
    if (!filter) return [];

    const n = filter.params.length;
    const query = `
      SELECT m.*
      FROM messages m
      WHERE ${filter.sql}
      ORDER BY m.created_at DESC
      LIMIT $${n + 1} OFFSET $${n + 2}
    `;

    const result = await db.query(query, [...filter.params, limit, offset]);
    return result.rows.reverse(); // Reverse to show oldest first
  }

  /**
   * Get total message count for a conversation
   */
  async getTotalMessageCount({ taskId, applicationId, itemId, userId, otherUserId }) {
    const filter = conversationFilter({ taskId, applicationId, itemId, userId, otherUserId });
    if (!filter) return 0;

    const result = await db.query(
      `SELECT COUNT(*) as total FROM messages m WHERE ${filter.sql}`,
      filter.params
    );
    return parseInt(result.rows[0].total);
  }

  /**
   * Get user conversations list
   */
  async getUserConversations(userId) {
    // A conversation is (context type, context id, other participant).
    // Task, application and store item ids share a number space, and a store
    // item can have several buyers, so none of these may be collapsed.
    // No cross-database JOINs: the frontend fetches task/user/item details.
    const query = `
      WITH UserMessages AS (
        SELECT
          m.*,
          CASE
            WHEN m.task_id IS NOT NULL THEN 'task'
            WHEN m.application_id IS NOT NULL THEN 'application'
            ELSE 'store'
          END AS context_type,
          COALESCE(m.task_id, m.application_id, m.store_item_id) AS context_id,
          CASE WHEN m.sender_id = $1 THEN m.recipient_id ELSE m.sender_id END AS other_user_id
        FROM messages m
        WHERE (m.sender_id = $1 OR m.recipient_id = $1)
          AND COALESCE(m.task_id, m.application_id, m.store_item_id) IS NOT NULL
      ),
      LatestMessages AS (
        SELECT DISTINCT ON (context_type, context_id, other_user_id) *
        FROM UserMessages
        ORDER BY context_type, context_id, other_user_id, created_at DESC
      ),
      UnreadCounts AS (
        SELECT context_type, context_id, other_user_id, COUNT(*)::int AS unread_count
        FROM UserMessages
        WHERE recipient_id = $1 AND is_read = false
        GROUP BY context_type, context_id, other_user_id
      )
      SELECT lm.*, COALESCE(uc.unread_count, 0) AS unread_count
      FROM LatestMessages lm
      LEFT JOIN UnreadCounts uc
        ON uc.context_type = lm.context_type
       AND uc.context_id = lm.context_id
       AND uc.other_user_id = lm.other_user_id
      ORDER BY lm.created_at DESC
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