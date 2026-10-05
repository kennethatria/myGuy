jest.mock('../src/config/database', () => ({ query: jest.fn(), getClient: jest.fn() }));
jest.mock('../src/utils/logger', () => ({ debug: jest.fn(), info: jest.fn(), warn: jest.fn(), error: jest.fn() }));
jest.mock('../src/services/bookingMessageService', () => ({}));
jest.mock('../src/services/messageService', () => ({ sendMessage: jest.fn() }));

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
});
