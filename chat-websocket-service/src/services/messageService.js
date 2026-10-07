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

/**
 * The conversation a contact unlock applies to, or null for contexts that
 * never unlock (application chats). Participants are ordered so the pair
 * matches whichever of them is sending.
 */
function unlockKey({ taskId, storeItemId, userA, userB }) {
  const [type, id] = taskId ? ['task', taskId] : storeItemId ? ['store', storeItemId] : [null, null];
  const a = parseInt(userA), b = parseInt(userB);
  if (!type || !a || !b || a === b) return null;
  return [type, parseInt(id), Math.min(a, b), Math.max(a, b)];
}

// Messages that record what happened (gig events, booking status notes):
// nobody may edit or delete them.
const RECORD_TYPES = "('system_alert', 'booking_approved', 'booking_declined', 'booking_picked_up', 'booking_item_received', 'booking_completed', 'booking_status_update')";

// The app's message box allows 1000 characters; this is the hard ceiling for
// anything sent to the API directly, which also bounds filtering work.
const MAX_MESSAGE_LENGTH = 2000;

function assertMessageLength(content) {
  if (typeof content !== 'string' || content.length > MAX_MESSAGE_LENGTH) {
    const error = new Error(`Messages can be at most ${MAX_MESSAGE_LENGTH} characters`);
    error.status = 400;
    throw error;
  }
}

// Gig chats open once the poster accepts the application, marketplace chats
// once the seller approves a booking; until then a conversation holds only
// events (the application, the booking request).
function chatLockedError(storeItemId) {
  const error = new Error(storeItemId
    ? 'You can chat once the seller approves the booking.'
    : 'You can chat once the poster accepts the application.');
  error.status = 403;
  error.code = 'chat_locked';
  return error;
}

// Where a conversation stands, from its latest gig event or booking request.
// A conversation that ended (deal done, or closed without one) is read-only.
const ENDED_STATES = {
  task: new Set(['completed', 'declined', 'cancelled']),
  store: new Set(['completed', 'rejected', 'released'])
};

function hasEnded(contextType, state) {
  return !!state && !!ENDED_STATES[contextType]?.has(state);
}

function chatEndedError() {
  const error = new Error('This conversation has ended.');
  error.status = 403;
  error.code = 'chat_ended';
  return error;
}

class MessageService {
  constructor() {
    // Initialization if needed
  }

  /**
   * Let two people share contact details in their conversation about a task
   * or store item: they agreed to work together. Idempotent.
   */
  async unlockContacts({ taskId, storeItemId, userA, userB }) {
    const key = unlockKey({ taskId, storeItemId, userA, userB });
    if (!key) throw new Error('unlockContacts needs a task or store item and two different users');
    await db.query(
      `INSERT INTO contact_unlocks (context_type, context_id, user_low, user_high)
       VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
      key
    );
  }

  /**
   * Whether the two people in this conversation may share contact details.
   */
  async contactsUnlocked(client, { taskId, storeItemId, senderId, recipientId }) {
    const key = unlockKey({ taskId, storeItemId, userA: senderId, userB: recipientId });
    if (!key) return false;
    const result = await client.query(
      `SELECT 1 FROM contact_unlocks
       WHERE context_type = $1 AND context_id = $2 AND user_low = $3 AND user_high = $4`,
      key
    );
    return result.rows.length > 0;
  }

  /**
   * Where the conversation between two people about a gig or an item stands
   * (its latest gig event or booking status), and whether it has ended.
   */
  async conversationState(client, { taskId, storeItemId, userA, userB }) {
    if ((!taskId && !storeItemId) || !userA || !userB) return { state: null, ended: false };
    const [contextType, column, condition] = taskId
      ? ['task', 'task_id', "metadata ? 'event'"]
      : ['store', 'store_item_id', "message_type = 'booking_request'"];
    const result = await client.query(
      `SELECT CASE WHEN message_type = 'booking_request' THEN metadata->>'status' ELSE metadata->>'event' END AS state
       FROM messages
       WHERE ${column} = $1 AND ${condition}
         AND ((sender_id = $2 AND recipient_id = $3) OR (sender_id = $3 AND recipient_id = $2))
       ORDER BY created_at DESC, id DESC
       LIMIT 1`,
      [taskId || storeItemId, userA, userB]
    );
    const state = result.rows?.[0]?.state ?? null;
    return { state, ended: hasEnded(contextType, state) };
  }

  /**
   * Whether the conversation between userId and otherUserId about the gig
   * taskId or item itemId has ended (read-only from now on).
   */
  async isChatEnded({ taskId, itemId, userId, otherUserId }) {
    return (await this.conversationState(db, { taskId, storeItemId: itemId, userA: userId, userB: otherUserId })).ended;
  }

  /**
   * Whether userId may not yet write to otherUserId about the gig taskId or
   * the item itemId: closed until the poster accepts or the seller approves.
   */
  async isChatLocked({ taskId, itemId, userId, otherUserId }) {
    if ((!taskId && !itemId) || !otherUserId) return false;
    return !(await this.contactsUnlocked(db, { taskId, storeItemId: itemId, senderId: userId, recipientId: otherUserId }));
  }

  /**
   * Contact details are masked until the two people are matched.
   */
  async filterFor(client, conversation, content) {
    if (await this.contactsUnlocked(client, conversation)) {
      return { filtered: content.trim(), hasRemovedContent: false };
    }
    return filterContent(content);
  }

  /**
   * Send a new message
   */
  async sendMessage({ taskId, applicationId, storeItemId, senderId, recipientId, content, messageType, metadata }) {
    assertMessageLength(content);
    const client = await db.getClient();

    try {
      await client.query('BEGIN');

      // People write about a gig or an item only once they're matched (the
      // poster accepted, the seller approved a booking); events (messageType
      // set by the services) are always posted.
      if ((taskId || storeItemId) && !messageType &&
          !(await this.contactsUnlocked(client, { taskId, storeItemId, senderId, recipientId }))) {
        throw chatLockedError(storeItemId);
      }
      // ...and no longer once the deal is done or closed (read-only history)
      if ((taskId || storeItemId) && !messageType &&
          (await this.conversationState(client, { taskId, storeItemId, userA: senderId, userB: recipientId })).ended) {
        throw chatEndedError();
      }

      const { filtered, hasRemovedContent } = await this.filterFor(
        client, { taskId, storeItemId, senderId, recipientId }, content
      );

      // Determine message type (callers may set e.g. 'system_alert')
      const type = messageType || (taskId ? 'task' : (applicationId ? 'application' : 'store'));

      // Store message
      const messageQuery = `
        INSERT INTO messages (task_id, application_id, store_item_id, sender_id, recipient_id, content, message_type, metadata, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
        RETURNING *
      `;

      const messageResult = await client.query(messageQuery, [
        taskId || null,
        applicationId || null,
        storeItemId || null,
        senderId,
        recipientId,
        filtered,
        type,
        metadata ? JSON.stringify(metadata) : null
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
    assertMessageLength(newContent);
    const client = await db.getClient();

    try {
      await client.query('BEGIN');

      // Check if user owns the message
      // Records of what happened (events, booking notes) are not editable
      const checkQuery = `SELECT * FROM messages WHERE id = $1 AND sender_id = $2 AND message_type NOT IN ${RECORD_TYPES}`;
      const checkResult = await client.query(checkQuery, [messageId, userId]);

      if (checkResult.rows.length === 0) {
        throw new Error('Message not found or unauthorized');
      }

      const original = checkResult.rows[0];
      const { filtered, hasRemovedContent } = await this.filterFor(client, {
        taskId: original.task_id,
        storeItemId: original.store_item_id,
        senderId: original.sender_id,
        recipientId: original.recipient_id
      }, newContent);

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
      WHERE id = $1 AND sender_id = $2 AND message_type NOT IN ${RECORD_TYPES}
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
      ),
      -- Where each conversation stands: its latest booking request's status
      -- (kept current as the booking moves on) or its latest gig event
      LatestState AS (
        SELECT DISTINCT ON (context_type, context_id, other_user_id)
          context_type, context_id, other_user_id,
          CASE WHEN message_type = 'booking_request' THEN metadata->>'status' ELSE metadata->>'event' END AS state
        FROM UserMessages
        WHERE message_type = 'booking_request' OR (task_id IS NOT NULL AND metadata ? 'event')
        ORDER BY context_type, context_id, other_user_id, created_at DESC, id DESC
      )
      SELECT lm.*, COALESCE(uc.unread_count, 0) AS unread_count, ls.state
      FROM LatestMessages lm
      LEFT JOIN UnreadCounts uc
        ON uc.context_type = lm.context_type
       AND uc.context_id = lm.context_id
       AND uc.other_user_id = lm.other_user_id
      LEFT JOIN LatestState ls
        ON ls.context_type = lm.context_type
       AND ls.context_id = lm.context_id
       AND ls.other_user_id = lm.other_user_id
      ORDER BY lm.created_at DESC
    `;

    const result = await db.query(query, [userId]);
    return result.rows.map(row => ({ ...row, ended: hasEnded(row.context_type, row.state) }));
  }

  /**
   * Whether two users have exchanged at least one message. Presence (last
   * seen) is only shared between people who are actually talking.
   */
  async haveConversed(userId, otherUserId) {
    const result = await db.query(
      `SELECT 1 FROM messages
       WHERE (sender_id = $1 AND recipient_id = $2) OR (sender_id = $2 AND recipient_id = $1)
       LIMIT 1`,
      [userId, otherUserId]
    );
    return result.rows.length > 0;
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

/**
 * A conversation as the app lists it: where it is, its last message, unread
 * count, and where it stands (state, ended).
 */
function formatConversation(conv) {
  return {
    task_id: conv.task_id,
    application_id: conv.application_id,
    item_id: conv.store_item_id,
    task_title: conv.task_title,
    task_description: conv.task_description,
    task_status: conv.task_status,
    item_title: conv.item_title,
    last_message: conv.content || '',
    last_message_type: conv.message_type,
    last_message_time: conv.created_at,
    other_user_id: conv.other_user_id,
    other_user_name: conv.other_user_name,
    unread_count: conv.unread_count || 0,
    state: conv.state ?? null,
    ended: !!conv.ended,
    conversation_type: conv.task_id ? 'task'
      : conv.application_id ? 'application'
        : conv.store_item_id ? 'store' : 'unknown'
  };
}

module.exports = new MessageService();
module.exports.formatConversation = formatConversation;
module.exports.MAX_MESSAGE_LENGTH = MAX_MESSAGE_LENGTH;