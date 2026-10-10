const axios = require('axios');
const db = require('../config/database');
const logger = require('../utils/logger');

// What the other person sees in each conversation with a blocked account,
// and when the block is lifted
const NOTICES = {
  account_flagged: 'This account has been flagged for breaking the site rules.',
  account_restored: 'This account is active again.'
};

/**
 * The accounts the backend blocked (by email), read from its
 * GET /internal/v1/blocked-users every minute. Chat refuses them (sign-in on
 * sockets and REST), drops their open sockets, closes their conversations to
 * typing, and tells the people they were talking to.
 */
class BlockService {
  constructor({ url = process.env.BACKEND_INTERNAL_URL, key = process.env.INTERNAL_API_KEY, http = axios, database = db } = {}) {
    this.url = url && key ? `${url}/internal/v1/blocked-users` : null;
    this.key = key;
    this.http = http;
    this.db = database;
    this.blocked = new Set();
    this.io = null;
    this.messageService = null;
    if (!this.url) logger.warn('BACKEND_INTERNAL_URL or INTERNAL_API_KEY not set; blocked accounts will not be refused');
  }

  /** Sockets to drop and the service that posts notices (set at startup). */
  attach({ io, messageService }) {
    this.io = io;
    this.messageService = messageService;
  }

  isBlocked(userId) {
    return this.blocked.has(parseInt(userId));
  }

  /**
   * Read the list once and act on what changed. A failed read keeps the last
   * list (blocks stay in force) and is logged.
   */
  async refresh() {
    if (!this.url) return;
    let ids;
    try {
      const response = await this.http.get(this.url, { headers: { 'X-Internal-API-Key': this.key }, timeout: 10000 });
      ids = (response.data?.user_ids || []).map(id => parseInt(id)).filter(id => id > 0);
    } catch (error) {
      logger.error('Blocked accounts: refresh failed', { error: error.message });
      return;
    }
    this.blocked = new Set(ids);
    for (const id of ids) {
      this.disconnect(id);
    }
    await this.tellPartners(ids);
  }

  disconnect(userId) {
    this.io?.in(`user:${userId}`).disconnectSockets(true);
  }

  /**
   * Post the flagged notice for newly blocked accounts and the restored one
   * for unblocked accounts. Each change is claimed in account_notices first,
   * so it's posted once whatever the number of chat instances or restarts.
   */
  async tellPartners(ids) {
    const flagged = await this.db.query(
      `INSERT INTO account_notices (user_id, flagged, updated_at)
       SELECT id, true, NOW() FROM unnest($1::int[]) AS id
       ON CONFLICT (user_id) DO UPDATE SET flagged = true, updated_at = NOW()
         WHERE account_notices.flagged = false
       RETURNING user_id`,
      [ids]
    );
    const restored = await this.db.query(
      `UPDATE account_notices SET flagged = false, updated_at = NOW()
       WHERE flagged AND NOT (user_id = ANY($1::int[]))
       RETURNING user_id`,
      [ids]
    );
    for (const { user_id: id } of flagged.rows) await this.notify(id, 'account_flagged');
    for (const { user_id: id } of restored.rows) await this.notify(id, 'account_restored');
  }

  /** One notice, from the account, in each of its conversations. */
  async notify(userId, notice) {
    if (!this.messageService) return;
    logger.info('Account notice', { userId, notice });
    const conversations = await this.messageService.getUserConversations(userId);
    for (const conv of conversations) {
      try {
        const message = await this.messageService.sendMessage({
          taskId: conv.task_id, applicationId: conv.application_id, storeItemId: conv.store_item_id,
          senderId: userId, recipientId: conv.other_user_id, content: NOTICES[notice],
          messageType: 'system_alert', metadata: { notice }
        });
        this.io?.to(`user:${conv.other_user_id}`).emit('message:new', {
          ...message, sender: { id: userId, username: 'User' }, recipient: { id: conv.other_user_id, username: 'User' }
        });
      } catch (error) {
        logger.error('Account notice failed', { userId, otherUserId: conv.other_user_id, error: error.message });
      }
    }
  }

  /** Keep the list current: now, then every interval. */
  start(interval = 60 * 1000) {
    this.refresh();
    this.timer = setInterval(() => this.refresh(), interval);
    this.timer.unref?.();
  }

  stop() {
    clearInterval(this.timer);
  }
}

module.exports = new BlockService();
module.exports.BlockService = BlockService;
module.exports.NOTICES = NOTICES;
