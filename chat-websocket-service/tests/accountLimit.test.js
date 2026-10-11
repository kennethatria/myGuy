jest.mock('../src/utils/logger', () => ({ debug: jest.fn(), info: jest.fn(), warn: jest.fn(), error: jest.fn() }));

const logger = require('../src/utils/logger');
const { RateLimit } = require('../src/utils/accountLimit');

const MINUTE = 60 * 1000;

function clocked(rules) {
  let now = Date.parse('2026-10-11T12:00:00Z');
  const limit = new RateLimit('message', rules, () => now);
  return { limit, tick: ms => { now += ms; } };
}

describe('RateLimit', () => {
  beforeEach(() => logger.warn.mockClear());

  it('counts per account, and every rule must allow', () => {
    const { limit, tick } = clocked([{ limit: 2, perMs: MINUTE }, { limit: 3, perMs: 60 * MINUTE }]);
    expect(limit.take(1)).toBe(0);
    expect(limit.take(1)).toBe(0);
    expect(limit.take(1)).toBe(MINUTE);
    expect(limit.take(2)).toBe(0); // another account
    tick(MINUTE);
    expect(limit.take(1)).toBe(0);
    expect(limit.take(1)).toBe(59 * MINUTE); // the hourly rule now; refusals didn't count
  });

  it('refuses with a 429 people can read, and logs a run of refusals once', () => {
    const { limit } = clocked([{ limit: 1, perMs: 60 * MINUTE }]);
    limit.check(7, { username: 'jane', email: 'jane@example.com' });
    expect(() => limit.check(7, { username: 'jane', email: 'jane@example.com' })).toThrow(expect.objectContaining({
      status: 429, code: 'rate_limited', retryAfter: 3600, message: "You're doing that a lot. Try again in 60 minutes."
    }));
    expect(() => limit.check(7)).toThrow();
    expect(logger.warn).toHaveBeenCalledTimes(1);
    expect(logger.warn).toHaveBeenCalledWith('limit event=hit action=message user=7 username="jane" email="jane@example.com"');
  });

  it('forgets idle accounts', () => {
    const { limit, tick } = clocked([{ limit: 5, perMs: MINUTE }]);
    limit.take(1);
    tick(2 * MINUTE);
    for (let i = 0; i < 1000; i++) limit.take(2);
    expect(limit.hits.has(1)).toBe(false);
  });
});
