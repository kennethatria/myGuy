jest.mock('../src/config/database', () => ({ query: jest.fn(), getClient: jest.fn() }));
jest.mock('../src/utils/logger', () => ({ debug: jest.fn(), info: jest.fn(), warn: jest.fn(), error: jest.fn() }));
jest.mock('../src/services/bookingMessageService', () => ({}));
jest.mock('../src/services/messageService', () => ({ sendMessage: jest.fn(), unlockContacts: jest.fn() }));

const request = require('supertest');
const express = require('express');
const messageService = require('../src/services/messageService');

const INTERNAL_API_KEY = 'test-internal-api-key';

describe('POST /internal/task-message', () => {
  let app, rooms, emit;

  beforeAll(() => {
    process.env.INTERNAL_API_KEY = INTERNAL_API_KEY;
    app = express();
    app.use(express.json());
    // io.to(a).to(b).emit(...) — record the rooms targeted
    emit = jest.fn();
    const chain = { to: jest.fn(room => { rooms.push(room); return chain; }), emit };
    app.set('io', chain);
    app.use('/', require('../src/api/bookingNotifications'));
  });

  beforeEach(() => {
    jest.clearAllMocks();
    rooms = [];
  });

  const post = (body, key = INTERNAL_API_KEY) =>
    request(app).post('/internal/task-message').set('X-Internal-API-Key', key).send(body);

  it('rejects calls without the internal key', async () => {
    const res = await post({ task_id: 1, sender_id: 2, recipient_id: 3, content: 'hi' }, 'wrong');
    expect(res.status).toBe(401);
    expect(messageService.sendMessage).not.toHaveBeenCalled();
  });

  it('requires all fields', async () => {
    const res = await post({ task_id: 1, sender_id: 2, content: '  ' });
    expect(res.status).toBe(400);
  });

  it('stores a system message and delivers it to both participants', async () => {
    messageService.sendMessage.mockResolvedValue({ id: 42, task_id: 1, sender_id: 2, recipient_id: 3, content: 'You got it', message_type: 'system_alert' });

    const res = await post({ task_id: '1', sender_id: 2, recipient_id: 3, content: ' You got it ' });

    expect(res.status).toBe(201);
    expect(messageService.sendMessage).toHaveBeenCalledWith({
      taskId: 1, senderId: 2, recipientId: 3, content: 'You got it', messageType: 'system_alert'
    });
    expect(rooms).toEqual(['user:2', 'user:3']);
    expect(emit).toHaveBeenCalledWith('message:new', expect.objectContaining({ id: 42, message_type: 'system_alert' }));
  });

  it('unlocks contact sharing for an accepted application', async () => {
    messageService.sendMessage.mockResolvedValue({ id: 43 });

    const res = await post({ task_id: 1, sender_id: 9, recipient_id: 2, content: 'Accepted', unlock_contacts: true });

    expect(res.status).toBe(201);
    expect(messageService.unlockContacts).toHaveBeenCalledWith({ taskId: 1, userA: 9, userB: 2 });
    // unlocked before the message is stored
    expect(messageService.unlockContacts.mock.invocationCallOrder[0])
      .toBeLessThan(messageService.sendMessage.mock.invocationCallOrder[0]);
  });

  it('leaves contacts locked for other task events', async () => {
    messageService.sendMessage.mockResolvedValue({ id: 44 });

    await post({ task_id: 1, sender_id: 2, recipient_id: 9, content: 'New application' });

    expect(messageService.unlockContacts).not.toHaveBeenCalled();
  });
});

describe('POST /internal/store-message', () => {
  let app, rooms, emit;

  beforeAll(() => {
    process.env.INTERNAL_API_KEY = INTERNAL_API_KEY;
    app = express();
    app.use(express.json());
    emit = jest.fn();
    const chain = { to: jest.fn(room => { rooms.push(room); return chain; }), emit };
    app.set('io', chain);
    app.use('/', require('../src/api/bookingNotifications'));
  });

  beforeEach(() => {
    jest.clearAllMocks();
    rooms = [];
  });

  const post = (body, key = INTERNAL_API_KEY) =>
    request(app).post('/internal/store-message').set('X-Internal-API-Key', key).send(body);

  it('rejects calls without the internal key', async () => {
    const res = await post({ store_item_id: 1, sender_id: 2, recipient_id: 3, content: 'hi' }, 'wrong');
    expect(res.status).toBe(401);
    expect(messageService.sendMessage).not.toHaveBeenCalled();
  });

  it('requires all fields', async () => {
    const res = await post({ store_item_id: 1, sender_id: 2, content: 'hi' });
    expect(res.status).toBe(400);
  });

  it('stores a system message about the item and delivers it to both people', async () => {
    messageService.sendMessage.mockResolvedValue({ id: 50, store_item_id: 7, message_type: 'system_alert' });

    const res = await post({ store_item_id: '7', sender_id: 2, recipient_id: 3, content: ' Listed for your request ' });

    expect(res.status).toBe(201);
    expect(messageService.sendMessage).toHaveBeenCalledWith({
      storeItemId: 7, senderId: 2, recipientId: 3, content: 'Listed for your request', messageType: 'system_alert'
    });
    expect(messageService.unlockContacts).not.toHaveBeenCalled();
    expect(rooms).toEqual(['user:2', 'user:3']);
    expect(emit).toHaveBeenCalledWith('message:new', expect.objectContaining({ id: 50 }));
  });

  it('reports a storage failure', async () => {
    messageService.sendMessage.mockRejectedValue(new Error('db down'));
    const res = await post({ store_item_id: 7, sender_id: 2, recipient_id: 3, content: 'x' });
    expect(res.status).toBe(500);
  });
});
