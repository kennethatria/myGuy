const axios = require('axios');
const db = require('../config/database');
const logger = require('../utils/logger');

// What the other person sees in each conversation with a blocked or
// suspended account, and when that's over
const NOTICES = {
  account_flagged: 'This account has been flagged for breaking the site rules.',
  account_restored: 'This account is active again.'
};

/** "Wed 14 Oct, 12:00 UTC"; the app shows it in the viewer's time. */
function formatUntil(until) {
  return new Date(until).toLocaleString('en-GB', {
    weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit', timeZone: 'UTC'
  }) + ' UTC';
}

function noticeText(notice, until) {
  return notice === 'account_suspended'
    ? `This account is suspended until ${formatUntil(until)} for breaking the site rules.`
    : NOTICES[notice];
}

/**
 * The accounts the backend blocked (by email) or suspended, read from its
 * GET /internal/v1/blocked-users every minute. Chat refuses blocked accounts
 * (sign-in on sockets and REST) and drops their sockets; suspended accounts
 * stay signed in but read-only. Either way their conversations close to
 * typing while it lasts, and the people they were talking to are told.
 */
class BlockService {
  constructor({ url = process.env.BACKEND_INTERNAL_URL, key = process.env.INTERNAL_API_KEY, http = axios, database = db } = {}) {
    this.url = url && key ? `${url}/internal/v1/blocked-users` : null;
    this.key = key;
    this.http = http;
    this.db = database;
    this.blocked = new Set();
    this.suspended = new Map(); // id -> Date it ends
    this.now = () => new Date();
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

  /** When an account's suspension ends, or null if it isn't suspended. */
  suspendedUntil(userId) {
    const until = this.suspended.get(parseInt(userId));
    return until && until > this.now() ? until : null;
  }

  /**
   * Read the list once and act on what changed. A failed read keeps the last
   * list (blocks stay in force) and is logged.
   */
  async refresh() {
    if (!this.url) return;
    let ids, suspended;
    try {
      const response = await this.http.get(this.url, { headers: { 'X-Internal-API-Key': this.key }, timeout: 10000 });
      ids = (response.data?.user_ids || []).map(id => parseInt(id)).filter(id => id > 0);
      suspended = new Map((response.data?.suspended || [])
        .map(s => [parseInt(s.user_id), new Date(s.until)])
        .filter(([id, until]) => id > 0 && !isNaN(until)));
    } catch (error) {
      logger.error('Blocked accounts: refresh failed', { error: error.message });
      return;
    }
    this.blocked = new Set(ids);
    this.suspended = suspended;
    for (const id of ids) {
      this.disconnect(id);
    }
    await this.tellPartners(this.states());
  }

  /**
   * What each restricted account's partners should have been told: a block
   * outranks a suspension, and a new end date is a new suspension.
   */
  states() {
    const states = new Map();
    for (const [id, until] of this.suspended) {
      if (until > this.now()) states.set(id, `suspended:${until.toISOString()}`);
    }
    for (const id of this.blocked) states.set(id, 'flagged');
    return states;
  }

  disconnect(userId) {
    this.io?.in(`user:${userId}`).disconnectSockets(true);
  }

  /**
   * Post a notice for each account whose state changed (flagged, suspended
   * until a date, active again). Each change is claimed in account_notices
   * first, so it's posted once whatever the number of chat instances or
   * restarts.
   */
  async tellPartners(states) {
    const ids = [...states.keys()];
    const changed = await this.db.query(
      `INSERT INTO account_notices (user_id, state, updated_at)
       SELECT id, state, NOW() FROM unnest($1::int[], $2::text[]) AS s(id, state)
       ON CONFLICT (user_id) DO UPDATE SET state = EXCLUDED.state, updated_at = NOW()
         WHERE account_notices.state <> EXCLUDED.state
       RETURNING user_id, state`,
      [ids, ids.map(id => states.get(id))]
    );
    const restored = await this.db.query(
      `UPDATE account_notices SET state = 'active', updated_at = NOW()
       WHERE state <> 'active' AND NOT (user_id = ANY($1::int[]))
       RETURNING user_id`,
      [ids]
    );
    for (const { user_id: id, state } of changed.rows) {
      if (state === 'flagged') await this.notify(id, 'account_flagged');
      else await this.notify(id, 'account_suspended', state.slice('suspended:'.length));
    }
    for (const { user_id: id } of restored.rows) await this.notify(id, 'account_restored');
  }

  /** One notice, from the account, in each of its conversations. */
  async notify(userId, notice, until) {
    if (!this.messageService) return;
    logger.info('Account notice', { userId, notice, until });
    const conversations = await this.messageService.getUserConversations(userId);
    for (const conv of conversations) {
      try {
        const message = await this.messageService.sendMessage({
          taskId: conv.task_id, applicationId: conv.application_id, storeItemId: conv.store_item_id,
          senderId: userId, recipientId: conv.other_user_id, content: noticeText(notice, until),
          messageType: 'system_alert', metadata: until ? { notice, until } : { notice }
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
module.exports.noticeText = noticeText;
