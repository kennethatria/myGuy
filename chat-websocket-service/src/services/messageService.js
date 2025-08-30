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
      
      // Store message
      const messageQuery = `
        INSERT INTO messages (task_id, application_id, store_item_id, sender_id, recipient_id, content, original_content, created_at)
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
        content // Store original for audit
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
            original_content = $2,
            is_edited = true, 
            edited_at = NOW()
        WHERE id = $3
        RETURNING *
      `;
      
      const result = await client.query(updateQuery, [filtered, newContent, messageId]);

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
      const itemQuery = `
        SELECT si.*, 
               (si.seller_id = $1) as is_seller
        FROM store_items si 
        WHERE si.id = $2
      `;
      
      const itemResult = await db.query(itemQuery, [userId, itemId]);
      
      if (itemResult.rows.length > 0) {
        const item = itemResult.rows[0];
        messageQuery = `
          SELECT 
            m.*,
            s.username as sender_name,
            r.username as recipient_name
          FROM messages m
          LEFT JOIN users s ON m.sender_id = s.id
          LEFT JOIN users r ON m.recipient_id = r.id
          WHERE m.store_item_id = $1
            AND (m.sender_id = $2 OR m.recipient_id = $2 OR EXISTS(
              SELECT 1 FROM store_items si
              WHERE si.id = m.store_item_id
                AND si.seller_id = $2
            ))
          ORDER BY m.created_at DESC
          LIMIT $3 OFFSET $4
        `;
        queryParams = [itemId, userId, limit, offset];
      } else {
        return [];
      }
    } else {
      // First check if this is a task conversation
      const taskQuery = `
        SELECT t.*, 
               (t.created_by = $2 OR t.assigned_to = $2) as is_task_participant
        FROM tasks t 
        WHERE t.id = $1
      `;
    
      const taskResult = await db.query(taskQuery, [conversationId, userId]);
    
      if (taskResult.rows.length > 0) {
        // This is a task conversation
        const task = taskResult.rows[0];
        
        // Check privacy permissions
        if (!task.is_messages_public && !task.is_task_participant) {
          return []; // Return empty if private and user not participant
        }
        
        if (task.is_messages_public) {
          // If messages are public, show all messages
          messageQuery = `
            SELECT 
              m.*,
              s.username as sender_name,
              r.username as recipient_name
            FROM messages m
            LEFT JOIN users s ON m.sender_id = s.id
            LEFT JOIN users r ON m.recipient_id = r.id
            WHERE m.task_id = $1
            ORDER BY m.created_at DESC
            LIMIT $2 OFFSET $3
          `;
          queryParams = [conversationId, limit, offset];
        } else {
          // If messages are private, only show messages where user is participant
          messageQuery = `
            SELECT 
              m.*,
              s.username as sender_name,
              r.username as recipient_name
            FROM messages m
            LEFT JOIN users s ON m.sender_id = s.id
            LEFT JOIN users r ON m.recipient_id = r.id
            WHERE m.task_id = $1
              AND (m.sender_id = $2 OR m.recipient_id = $2)
            ORDER BY m.created_at DESC
            LIMIT $3 OFFSET $4
          `;
          queryParams = [conversationId, userId, limit, offset];
        }
      } else {
        // Check if this is a store item conversation
        const itemQuery = `
          SELECT si.*, 
                 (si.seller_id = $2) as is_seller
          FROM store_items si 
          WHERE si.id = $1
        `;
        
        const itemResult = await db.query(itemQuery, [conversationId, userId]);
        
        if (itemResult.rows.length > 0) {
          const item = itemResult.rows[0];
          // This is a store item conversation
          messageQuery = `
            SELECT 
              m.*,
              s.username as sender_name,
              r.username as recipient_name
            FROM messages m
            LEFT JOIN users s ON m.sender_id = s.id
            LEFT JOIN users r ON m.recipient_id = r.id
            LEFT JOIN store_items si ON m.store_item_id = si.id
            WHERE m.store_item_id = $1
              AND (m.sender_id = $2 OR m.recipient_id = $2 OR si.seller_id = $2)
            ORDER BY m.created_at DESC
            LIMIT $3 OFFSET $4
          `;
          queryParams = [conversationId, userId, limit, offset];
        } else {
          throw new Error('Conversation not found');
        }
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
      const query = `
        SELECT COUNT(*) as total
        FROM messages m
        WHERE m.store_item_id = $1
          AND (m.sender_id = $2 OR m.recipient_id = $2 OR EXISTS(
            SELECT 1 FROM store_items si
            WHERE si.id = m.store_item_id
              AND si.seller_id = $2
          ))
      `;
      const result = await db.query(query, [itemId, userId]);
      return parseInt(result.rows[0].total);
    } else if (taskId) {
      // First, get task information to check privacy settings
      const taskQuery = `
        SELECT t.*, 
               (t.created_by = $2 OR t.assigned_to = $2) as is_task_participant
        FROM tasks t 
        WHERE t.id = $1
      `;
      
      const taskResult = await db.query(taskQuery, [taskId, userId]);
      
      if (taskResult.rows.length === 0) {
        return 0;
      }
      
      const task = taskResult.rows[0];
      
      // Check privacy permissions
      if (!task.is_messages_public && !task.is_task_participant) {
        return 0;
      }
      
      let countQuery;
      let queryParams;
      
      if (task.is_messages_public) {
        // If messages are public, count all messages
        countQuery = `
          SELECT COUNT(*) as total
          FROM messages m
          WHERE m.task_id = $1
        `;
        queryParams = [taskId];
      } else {
        // If messages are private, only count messages where user is participant
        countQuery = `
          SELECT COUNT(*) as total
          FROM messages m
          WHERE m.task_id = $1
            AND (m.sender_id = $2 OR m.recipient_id = $2)
        `;
        queryParams = [taskId, userId];
      }
      
      const result = await db.query(countQuery, queryParams);
      return parseInt(result.rows[0].total);
    }
    return 0;
  }

  /**
   * Get user conversations list
   */
  async getUserConversations(userId) {
    const query = `
      WITH ConversationMessages AS (
        SELECT DISTINCT ON (COALESCE(task_id, application_id, store_item_id))
          m.*,
          t.title as task_title,
          t.description as task_description,
          t.status as task_status,
          si.title as item_title,
          si.seller_id,
          CASE 
            WHEN m.store_item_id IS NOT NULL THEN 
              CASE
                WHEN si.seller_id = $1 AND m.sender_id <> $1 THEN s.username
                WHEN si.seller_id = $1 AND m.sender_id = $1 THEN r.username
                WHEN m.sender_id = $1 THEN r.username
                ELSE s.username
              END
            WHEN m.sender_id = $1 THEN r.username
            ELSE s.username
          END as other_user_name,
          CASE 
            WHEN m.store_item_id IS NOT NULL THEN
              CASE
                WHEN si.seller_id = $1 AND m.sender_id <> $1 THEN m.sender_id
                WHEN si.seller_id = $1 AND m.sender_id = $1 THEN m.recipient_id
                WHEN m.sender_id = $1 THEN m.recipient_id
                ELSE m.sender_id
              END
            WHEN m.sender_id = $1 THEN m.recipient_id
            ELSE m.sender_id
          END as other_user_id
        FROM messages m
        LEFT JOIN tasks t ON m.task_id = t.id
        LEFT JOIN store_items si ON m.store_item_id = si.id
        LEFT JOIN users s ON m.sender_id = s.id
        LEFT JOIN users r ON m.recipient_id = r.id
        WHERE m.sender_id = $1 
           OR m.recipient_id = $1 
           OR (si.seller_id = $1)
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
    const query = `
      SELECT DISTINCT
        t.id as task_id,
        t.assignee_id,
        t.creator_id,
        t.completed_at,
        MAX(m.created_at) as last_message_date,
        COUNT(m.id) as message_count
      FROM tasks t
      INNER JOIN messages m ON m.task_id = t.id
      WHERE 
        (t.status = 'completed' AND t.completed_at < NOW() - INTERVAL '5 months')
        OR (t.status IN ('cancelled', 'pending') AND m.created_at < NOW() - INTERVAL '1 month')
      GROUP BY t.id
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
    const query = `
      SELECT 
        mdw.*,
        t.title as task_title
      FROM message_deletion_warnings mdw
      INNER JOIN tasks t ON mdw.task_id = t.id
      WHERE (t.created_by = $1 OR t.assigned_to = $1)
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
   */
  async getStoreMessages(itemId, userId) {
    const query = `
      SELECT 
        m.*,
        s.username as sender_username,
        r.username as recipient_username
      FROM messages m
      LEFT JOIN users s ON m.sender_id = s.id
      LEFT JOIN users r ON m.recipient_id = r.id
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
        INSERT INTO messages (store_item_id, sender_id, recipient_id, content, original_content, created_at)
        VALUES ($1, $2, $3, $4, $5, NOW())
        RETURNING *
      `;
      
      const messageResult = await client.query(messageQuery, [
        store_item_id,
        sender_id,
        recipient_id,
        filtered,
        content // Store original for audit
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
   */
  async getBookingStatus(itemId, userId) {
    try {
      // Check if there's an approved booking request for this item involving this user
      const query = `
        SELECT status
        FROM booking_requests
        WHERE item_id = $1 
        AND (requester_id = $2 OR item_id IN (
          SELECT id FROM store_items WHERE seller_id = $2
        ))
        ORDER BY created_at DESC
        LIMIT 1
      `;
      
      const result = await db.query(query, [itemId, userId]);
      
      if (result.rows.length > 0) {
        return result.rows[0].status;
      }
      
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
   */
  async getTaskMessageLimit(taskId, userId) {
    try {
      // Get task information
      const taskQuery = `
        SELECT created_by, assigned_to
        FROM tasks
        WHERE id = $1
      `;
      
      const result = await db.query(taskQuery, [taskId]);
      
      if (result.rows.length === 0) {
        return 3; // Default limit for non-existent tasks
      }
      
      const task = result.rows[0];
      
      // Users who are assigned to the task get 15 messages
      if (task.assigned_to === userId) {
        return 15;
      }
      
      // Task owners get 15 messages only if the task is assigned
      if (task.created_by === userId && task.assigned_to !== null) {
        return 15;
      }
      
      // Everyone else (non-assigned users) gets 3 messages
      return 3;
    } catch (error) {
      logger.error('Error getting task message limit:', error);
      return 3; // Default to safe limit on error
    }
  }
}

module.exports = new MessageService();