jest.mock('../src/config/database', () => ({ query: jest.fn(), getClient: jest.fn() }));
jest.mock('../src/utils/logger', () => ({ debug: jest.fn(), info: jest.fn(), warn: jest.fn(), error: jest.fn() }));

const jwt = require('jsonwebtoken');
const blockService = require('../src/services/blockService');
const { BlockService, NOTICES } = blockService;
const { authenticateSocket, authenticateHTTP, ACCOUNT_UNAVAILABLE } = require('../src/middleware/auth');

// Fake account_notices: which accounts' partners were told
function fakeNotices(flaggedAlready = []) {
  const flagged = new Set(flaggedAlready);
  return {
    flagged,
    query: jest.fn(async (sql, [ids]) => {
      if (sql.includes('INSERT INTO account_notices')) {
        const claimed = ids.filter(id => !flagged.has(id));
        claimed.forEach(id => flagged.add(id));
        return { rows: claimed.map(user_id => ({ user_id })) };
      }
      const lifted = [...flagged].filter(id => !ids.includes(id));
      lifted.forEach(id => flagged.delete(id));
      return { rows: lifted.map(user_id => ({ user_id })) };
    })
  };
}

function setUp(responses, flaggedAlready) {
  const http = { get: jest.fn() };
  for (const r of responses) {
    if (r instanceof Error) http.get.mockRejectedValueOnce(r);
    else http.get.mockResolvedValueOnce({ data: { user_ids: r } });
  }
  const database = fakeNotices(flaggedAlready);
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
    const { service, messageService } = setUp([[7]], [7]);
    await service.refresh();
    expect(messageService.sendMessage).not.toHaveBeenCalled();
    expect(service.isBlocked(7)).toBe(true);
  });

  it('says so when the block is lifted', async () => {
    const { service, messageService } = setUp([[]], [7]);
    await service.refresh();
    expect(service.isBlocked(7)).toBe(false);
    expect(messageService.sendMessage).toHaveBeenCalledWith(expect.objectContaining({
      senderId: 7, content: NOTICES.account_restored, metadata: { notice: 'account_restored' }
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
