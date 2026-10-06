const { parseUtcTimestamp } = require('../src/config/timestamps');

describe('parseUtcTimestamp', () => {
  it('reads a stored time as UTC, whatever the machine time zone', () => {
    expect(parseUtcTimestamp('2026-10-06 19:55:27.944563').toISOString()).toBe('2026-10-06T19:55:27.944Z');
    expect(parseUtcTimestamp('2026-10-06 19:55:27').toISOString()).toBe('2026-10-06T19:55:27.000Z');
  });

  it('leaves anything else to Date', () => {
    expect(isNaN(parseUtcTimestamp('infinity').getTime())).toBe(true);
  });
});
