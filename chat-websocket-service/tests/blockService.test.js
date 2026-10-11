jest.mock('../src/config/database', () => ({ query: jest.fn(), getClient: jest.fn() }));
jest.mock('../src/utils/logger', () => ({ debug: jest.fn(), info: jest.fn(), warn: jest.fn(), error: jest.fn() }));

const jwt = require('jsonwebtoken');
const blockService = require('../src/services/blockService');
const { BlockService, NOTICES, noticeText } = blockService;
const { authenticateSocket, authenticateHTTP, ACCOUNT_UNAVAILABLE } = require('../src/middleware/auth');

// Fake account_notices: what each account's partners were last told
function fakeNotices(told = {}) {
  const state = new Map(Object.entries(told).map(([id, s]) => [parseInt(id), s]));
  return {
    state,
    query: jest.fn(async (sql, [ids, states]) => {
      if (sql.includes('INSERT INTO account_notices')) {
        const changed = ids.map((id, i) => [id, states[i]]).filter(([id, s]) => state.get(id) !== s);
        changed.forEach(([id, s]) => state.set(id, s));
        return { rows: changed.map(([user_id, s]) => ({ user_id, state: s })) };
      }
      const lifted = [...state].filter(([id, s]) => s !== 'active' && !ids.includes(id)).map(([id]) => id);
      lifted.forEach(id => state.set(id, 'active'));
      return { rows: lifted.map(user_id => ({ user_id })) };
    })
  };
}

const UNTIL = '2026-10-14T12:00:00.000Z';

// Each response: blocked ids, or { user_ids, suspended }, or an Error
function setUp(responses, told) {
  const http = { get: jest.fn() };
  for (const r of responses) {
    if (r instanceof Error) http.get.mockRejectedValueOnce(r);
    else http.get.mockResolvedValueOnce({ data: Array.isArray(r) ? { user_ids: r } : r });
  }
  const database = fakeNotices(told);
  const service = new BlockService({ url: 'http://api:8080', key: 'k', http, database });
  const sockets = { disconnectSockets: jest.fn() };
  const emit = jest.fn();
  const io = { in: jest.fn(() => sockets), to: jest.fn(() => ({ emit })) };
  const messageService = {
    getUserConversations: jest.fn(async () => [
      { task_id: 4, other_user_id: 2 },
      { store_item_id: 9, other_user_id: 3 }
    ]),
    sendMessage: jest.fn(async m => ({ id: 1, ...m }))
  };
  service.attach({ io, messageService });
  service.now = () => new Date('2026-10-11T12:00:00Z');
  return { service, http, database, io, sockets, emit, messageService };
}

describe('BlockService', () => {
  it('reads the blocked accounts with the internal key', async () => {
    const { service, http } = setUp([[7]]);
    await service.refresh();
    expect(http.get).toHaveBeenCalledWith('http://api:8080/internal/v1/blocked-users',
      expect.objectContaining({ headers: { 'X-Internal-API-Key': 'k' } }));
    expect(service.isBlocked(7)).toBe(true);
    expect(service.isBlocked('7')).toBe(true);
    expect(service.isBlocked(8)).toBe(false);
  });

  it('drops the blocked account\'s sockets and tells each partner once', async () => {
    const { service, io, sockets, messageService, emit } = setUp([[7], [7]]);
    await service.refresh();
    expect(io.in).toHaveBeenCalledWith('user:7');
    expect(sockets.disconnectSockets).toHaveBeenCalledWith(true);
    expect(messageService.sendMessage).toHaveBeenCalledTimes(2);
    expect(messageService.sendMessage).toHaveBeenCalledWith(expect.objectContaining({
      taskId: 4, senderId: 7, recipientId: 2, content: NOTICES.account_flagged,
      messageType: 'system_alert', metadata: { notice: 'account_flagged' }
    }));
    expect(messageService.sendMessage).toHaveBeenCalledWith(expect.objectContaining({ storeItemId: 9, recipientId: 3 }));
    expect(io.to).toHaveBeenCalledWith('user:2');
    expect(emit).toHaveBeenCalledWith('message:new', expect.objectContaining({ content: NOTICES.account_flagged }));

    // Still blocked a minute later: no second notice
    await service.refresh();
    expect(messageService.sendMessage).toHaveBeenCalledTimes(2);
  });

  it('a restart doesn\'t tell partners again', async () => {
    const { service, messageService } = setUp([[7]], { 7: 'flagged' });
    await service.refresh();
    expect(messageService.sendMessage).not.toHaveBeenCalled();
    expect(service.isBlocked(7)).toBe(true);
  });

  it('says so when the block is lifted', async () => {
    const { service, messageService } = setUp([[]], { 7: 'flagged' });
    await service.refresh();
    expect(service.isBlocked(7)).toBe(false);
    expect(messageService.sendMessage).toHaveBeenCalledWith(expect.objectContaining({
      senderId: 7, content: NOTICES.account_restored, metadata: { notice: 'account_restored' }
    }));
  });

  it('keeps a suspended account signed in, and tells its partners until when', async () => {
    const { service, sockets, messageService } = setUp([
      { user_ids: [], suspended: [{ user_id: 7, until: UNTIL }] },
      { user_ids: [], suspended: [{ user_id: 7, until: UNTIL }] }
    ]);
    await service.refresh();
    expect(service.isBlocked(7)).toBe(false);
    expect(service.suspendedUntil(7)).toEqual(new Date(UNTIL));
    expect(service.suspendedUntil(8)).toBeNull();
    expect(sockets.disconnectSockets).not.toHaveBeenCalled();
    expect(messageService.sendMessage).toHaveBeenCalledTimes(2);
    expect(messageService.sendMessage).toHaveBeenCalledWith(expect.objectContaining({
      senderId: 7, recipientId: 2, messageType: 'system_alert',
      content: 'This account is suspended until Wed 14 Oct, 12:00 UTC for breaking the site rules.',
      metadata: { notice: 'account_suspended', until: UNTIL }
    }));

    await service.refresh(); // the same suspension: told once
    expect(messageService.sendMessage).toHaveBeenCalledTimes(2);
  });

  it('a suspension that runs out stops counting before the next refresh', () => {
    const { service } = setUp([]);
    service.suspended = new Map([[7, new Date(UNTIL)]]);
    service.now = () => new Date(UNTIL);
    expect(service.suspendedUntil(7)).toBeNull();
    expect(service.states().size).toBe(0);
  });

  it('says so when a suspension is over, and a new one is a new notice', async () => {
    const later = '2026-10-20T00:00:00.000Z';
    const { service, messageService } = setUp([
      { user_ids: [], suspended: [] },
      { user_ids: [], suspended: [{ user_id: 7, until: later }] }
    ], { 7: `suspended:${UNTIL}` });
    await service.refresh();
    expect(messageService.sendMessage).toHaveBeenLastCalledWith(expect.objectContaining({
      content: NOTICES.account_restored, metadata: { notice: 'account_restored' }
    }));
    await service.refresh();
    expect(messageService.sendMessage).toHaveBeenLastCalledWith(expect.objectContaining({
      metadata: { notice: 'account_suspended', until: later }
    }));
  });

  it('a block outranks a suspension', async () => {
    const { service, messageService, sockets } = setUp([{ user_ids: [7], suspended: [{ user_id: 7, until: UNTIL }] }]);
    await service.refresh();
    expect(sockets.disconnectSockets).toHaveBeenCalled();
    expect(messageService.sendMessage).toHaveBeenCalledWith(expect.objectContaining({
      metadata: { notice: 'account_flagged' }
    }));
  });

  it('keeps the last list when the backend can\'t be reached', async () => {
    const { service, messageService } = setUp([[7], new Error('down')]);
    await service.refresh();
    await service.refresh();
    expect(service.isBlocked(7)).toBe(true);
    expect(messageService.sendMessage).toHaveBeenCalledTimes(2); // only the first notices
  });

  it('does nothing without the backend URL or key', async () => {
    const http = { get: jest.fn() };
    const service = new BlockService({ url: '', key: 'k', http });
    await service.refresh();
    expect(http.get).not.toHaveBeenCalled();
    expect(service.isBlocked(7)).toBe(false);
  });
});

describe('auth refuses blocked accounts', () => {
  const token = jwt.sign({ user_id: 7 }, 'your-secret-key');
  beforeEach(() => { blockService.blocked = new Set([7]); });
  afterEach(() => { blockService.blocked = new Set(); });

  it('on sockets', async () => {
    const next = jest.fn();
    await authenticateSocket({ handshake: { auth: { token }, headers: {} } }, next);
    expect(next).toHaveBeenCalledWith(expect.objectContaining({ message: ACCOUNT_UNAVAILABLE }));
  });

  it('on REST', () => {
    const res = { status: jest.fn().mockReturnThis(), json: jest.fn() };
    const next = jest.fn();
    authenticateHTTP({ headers: { authorization: `Bearer ${token}` } }, res, next);
    expect(res.status).toHaveBeenCalledWith(401);
    expect(res.json).toHaveBeenCalledWith(expect.objectContaining({ code: ACCOUNT_UNAVAILABLE }));
    expect(next).not.toHaveBeenCalled();
  });
});

describe('noticeText', () => {
  it('dates a suspension in UTC', () => {
    expect(noticeText('account_suspended', UNTIL)).toBe(
      'This account is suspended until Wed 14 Oct, 12:00 UTC for breaking the site rules.');
    expect(noticeText('account_flagged')).toBe(NOTICES.account_flagged);
  });
});
