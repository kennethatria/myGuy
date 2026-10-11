const logger = require('./logger');

/**
 * How often one account may do an action, counted in memory (one chat
 * instance; a restart forgetting the counts is fine for limits). The same
 * rules as the Go services' internal/ratelimit: every rule must allow, a
 * refused action doesn't count, and the first refusal of a run is logged as
 * `limit event=hit` for the daily summary and /caught.
 */
class RateLimit {
  constructor(action, rules, now = () => Date.now()) {
    this.action = action;
    this.rules = rules; // [{ limit, perMs }]
    this.longest = Math.max(...rules.map(r => r.perMs));
    this.now = now;
    this.hits = new Map(); // user id -> timestamps, oldest first
    this.refused = new Set();
    this.calls = 0;
  }

  /** Records an action; returns 0 if allowed, else ms until it would be. */
  take(userId) {
    const now = this.now();
    this.sweep(now);
    const hits = (this.hits.get(userId) || []).filter(t => now - t < this.longest);
    let wait = 0;
    for (const { limit, perMs } of this.rules) {
      const within = hits.filter(t => now - t < perMs);
      if (within.length >= limit) wait = Math.max(wait, within[within.length - limit] + perMs - now);
    }
    if (!wait) hits.push(now);
    this.hits.set(userId, hits);
    return wait;
  }

  /**
   * Throws a 429 (code rate_limited, retryAfter seconds) when the account is
   * over its limit. sender ({ username, email }) only names it in the log.
   */
  check(userId, sender = {}) {
    const wait = this.take(userId);
    if (!wait) {
      this.refused.delete(userId);
      return;
    }
    if (!this.refused.has(userId)) {
      this.refused.add(userId);
      logger.warn(`limit event=hit action=${this.action} user=${userId} username=${JSON.stringify(sender.username || '')} email=${JSON.stringify(sender.email || '')}`);
    }
    const seconds = Math.ceil(wait / 1000);
    const minutes = Math.ceil(seconds / 60);
    const error = new Error(`You're doing that a lot. Try again in ${seconds < 60 ? 'a minute' : minutes === 1 ? '1 minute' : `${minutes} minutes`}.`);
    error.status = 429;
    error.code = 'rate_limited';
    error.retryAfter = seconds;
    throw error;
  }

  /** Drop idle accounts now and then, so memory follows active ones. */
  sweep(now) {
    if (++this.calls % 1000) return;
    for (const [id, hits] of this.hits) {
      if (!hits.some(t => now - t < this.longest)) {
        this.hits.delete(id);
        this.refused.delete(id);
      }
    }
  }
}

const MINUTE = 60 * 1000;
// People's own messages: 20 a minute, 300 an hour
const messageLimit = new RateLimit('message', [{ limit: 20, perMs: MINUTE }, { limit: 300, perMs: 60 * MINUTE }]);

module.exports = { RateLimit, messageLimit };
