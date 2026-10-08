jest.mock('../src/config/database', () => ({
  query: jest.fn(),
  getClient: jest.fn()
}));

jest.mock('../src/utils/logger', () => ({
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn()
}));

const db = require('../src/config/database');
const messageService = require('../src/services/messageService');

describe('MessageService', () => {
  let mockClient;

  beforeEach(() => {
    jest.clearAllMocks();
    mockClient = {
      query: jest.fn(),
      release: jest.fn()
    };
    db.getClient.mockResolvedValue(mockClient);
    db.query.mockResolvedValue({ rows: [] });
  });

  describe('getStoreMessages', () => {
    it('returns messages for a given item and user', async () => {
      const mockRows = [{ id: 1, content: 'hello', store_item_id: 5 }];
      db.query.mockResolvedValue({ rows: mockRows });

      const result = await messageService.getStoreMessages(5, 10);

      expect(result).toEqual(mockRows);
      expect(db.query).toHaveBeenCalledWith(
        expect.stringContaining('store_item_id'),
        [5, 10]
      );
    });

    it('returns empty array when no messages found', async () => {
      db.query.mockResolvedValue({ rows: [] });
      const result = await messageService.getStoreMessages(99, 1);
      expect(result).toEqual([]);
    });
  });

  describe('getUserStoreMessageCount', () => {
    it('returns parsed integer count', async () => {
      db.query.mockResolvedValue({ rows: [{ count: '7' }] });
      const result = await messageService.getUserStoreMessageCount(1, 2);
      expect(result).toBe(7);
    });

    it('returns 0 when count is zero', async () => {
      db.query.mockResolvedValue({ rows: [{ count: '0' }] });
      const result = await messageService.getUserStoreMessageCount(1, 2);
      expect(result).toBe(0);
    });
  });

  describe('getBookingStatus', () => {
    it('returns null (not yet implemented — separate DB)', async () => {
      const result = await messageService.getBookingStatus(1, 2);
      expect(result).toBeNull();
    });
  });

  describe('getMessageLimit', () => {
    it('returns 3 when booking status is null', async () => {
      const result = await messageService.getMessageLimit(1, 2);
      expect(result).toBe(3);
    });
  });

  describe('deleteMessage', () => {
    it('soft-deletes message and returns it', async () => {
      const mockMsg = { id: 1, content: '[Message deleted]', is_deleted: true };
      db.query.mockResolvedValue({ rows: [mockMsg] });

      const result = await messageService.deleteMessage(1, 10);

      expect(result).toEqual(mockMsg);
      expect(db.query).toHaveBeenCalledWith(
        expect.stringContaining('is_deleted = true'),
        [1, 10]
      );
    });

    it('throws when message not found or user is not the sender', async () => {
      db.query.mockResolvedValue({ rows: [] });
      await expect(messageService.deleteMessage(99, 10)).rejects.toThrow(
        'Message not found or unauthorized'
      );
    });

    it('never deletes a record of what happened (events, booking notes)', async () => {
      db.query.mockResolvedValue({ rows: [{ id: 1 }] });
      await messageService.deleteMessage(1, 10);
      const sql = db.query.mock.calls[0][0];
      for (const type of ['system_alert', 'booking_approved', 'booking_item_received', 'booking_completed']) {
        expect(sql).toContain(`'${type}'`);
      }
      expect(sql).toContain('NOT IN');
    });
  });

  describe('markAsRead', () => {
    it('returns the updated message', async () => {
      const mockMsg = { id: 1, is_read: true };
      db.query.mockResolvedValue({ rows: [mockMsg] });

      const result = await messageService.markAsRead(1, 2);

      expect(result).toEqual(mockMsg);
    });

    it('returns undefined when message was already read or not found', async () => {
      db.query.mockResolvedValue({ rows: [] });
      const result = await messageService.markAsRead(99, 2);
      expect(result).toBeUndefined();
    });
  });

  describe('markConversationAsRead', () => {
    it('marks a task conversation read for the recipient', async () => {
      db.query.mockResolvedValue({ rows: [{ id: 1, sender_id: 3 }, { id: 2, sender_id: 3 }] });

      const result = await messageService.markConversationAsRead(10, { taskId: 5 });

      expect(result).toHaveLength(2);
      expect(db.query).toHaveBeenCalledWith(expect.stringContaining('m.task_id = $1'), [5, 10]);
      expect(db.query.mock.calls[0][0]).toContain('m.recipient_id = $2');
    });

    it('supports application conversations', async () => {
      await messageService.markConversationAsRead(10, { applicationId: 7 });
      expect(db.query).toHaveBeenCalledWith(expect.stringContaining('m.application_id = $1'), [7, 10]);
    });

    it('scopes a store conversation to the other participant', async () => {
      await messageService.markConversationAsRead(10, { itemId: 4, otherUserId: 22 });
      const [sql, params] = db.query.mock.calls[0];
      expect(sql).toContain('m.store_item_id = $1');
      expect(sql).toContain('(m.sender_id = $3 OR m.recipient_id = $3)');
      expect(params).toEqual([4, 10, 22]);
    });

    it('returns empty array when nothing to mark', async () => {
      db.query.mockResolvedValue({ rows: [] });
      const result = await messageService.markConversationAsRead(10, { taskId: 5 });
      expect(result).toEqual([]);
    });

    it('does nothing without a conversation context', async () => {
      const result = await messageService.markConversationAsRead(10, {});
      expect(result).toEqual([]);
      expect(db.query).not.toHaveBeenCalled();
    });
  });

  describe('gig chats before a match', () => {
    // The client answers like the database: matched pairs per `unlocked`
    const answer = (unlocked) => mockClient.query.mockImplementation(async (sql) => {
      if (sql.includes('FROM contact_unlocks')) return { rows: unlocked ? [{}] : [] };
      if (sql.includes('INSERT INTO messages')) return { rows: [{ id: 7, task_id: 1 }] };
      return { rows: [] };
    });
    const send = (extra = {}) => messageService.sendMessage({
      taskId: 1, senderId: 2, recipientId: 9, content: 'When can you come?', ...extra
    });

    it('refuses a message until the poster accepts', async () => {
      answer(false);
      await expect(send()).rejects.toMatchObject({ code: 'chat_locked', status: 403 });
      expect(mockClient.query).toHaveBeenCalledWith('ROLLBACK');
      expect(mockClient.query).not.toHaveBeenCalledWith(expect.stringContaining('INSERT INTO messages'), expect.anything());
    });

    it('lets matched people write', async () => {
      answer(true);
      await expect(send()).resolves.toMatchObject({ id: 7 });
    });

    it('always posts gig events, with what they are about', async () => {
      answer(false);
      await send({ messageType: 'system_alert', metadata: { event: 'application', application_id: 3 } });
      const insert = mockClient.query.mock.calls.find(([sql]) => sql.includes('INSERT INTO messages'));
      expect(insert[1]).toContain(JSON.stringify({ event: 'application', application_id: 3 }));
    });

    it('refuses messages once the deal is done or closed, but still posts events', async () => {
      for (const [context, state] of [[{ taskId: 1 }, 'completed'], [{ taskId: 1 }, 'declined'], [{ taskId: undefined, storeItemId: 4 }, 'released']]) {
        mockClient.query.mockReset();
        mockClient.query.mockImplementation(async (sql) => {
          if (sql.includes('FROM contact_unlocks')) return { rows: [{}] };
          if (sql.includes('AS state')) return { rows: [{ state }] };
          if (sql.includes('INSERT INTO messages')) return { rows: [{ id: 7 }] };
          return { rows: [] };
        });
        await expect(send(context)).rejects.toMatchObject({ code: 'chat_ended', message: 'This conversation has ended.' });
        await expect(send({ ...context, messageType: 'system_alert' })).resolves.toMatchObject({ id: 7 });
      }
    });

    it('reports a chat as ended to the app', async () => {
      db.query.mockResolvedValueOnce({ rows: [{ state: 'completed' }] });
      expect(await messageService.isChatEnded({ taskId: 1, userId: 2, otherUserId: 9 })).toBe(true);
      db.query.mockResolvedValueOnce({ rows: [{ state: 'picked_up' }] });
      expect(await messageService.isChatEnded({ itemId: 4, userId: 2, otherUserId: 9 })).toBe(false);
      expect(await messageService.isChatEnded({ applicationId: 3, userId: 2, otherUserId: 9 })).toBe(false);
    });

    it('reports a gig chat as locked to the app', async () => {
      db.query.mockResolvedValueOnce({ rows: [] });
      expect(await messageService.isChatLocked({ taskId: 1, userId: 2, otherUserId: 9 })).toBe(true);
      db.query.mockResolvedValueOnce({ rows: [{}] });
      expect(await messageService.isChatLocked({ taskId: 1, userId: 2, otherUserId: 9 })).toBe(false);
      // Marketplace chats too, until the seller approves a booking
      db.query.mockResolvedValueOnce({ rows: [] });
      expect(await messageService.isChatLocked({ itemId: 4, userId: 2, otherUserId: 9 })).toBe(true);
      expect(await messageService.isChatLocked({ applicationId: 3, userId: 2, otherUserId: 9 })).toBe(false);
    });

    it('refuses a marketplace message until the seller approves', async () => {
      answer(false);
      await expect(send({ taskId: undefined, storeItemId: 4 })).rejects.toMatchObject({
        code: 'chat_locked', message: 'You can chat once the seller approves the booking.'
      });
    });
  });

  describe('getMessages', () => {
    it('scopes store messages to both participants and paginates after them', async () => {
      db.query.mockResolvedValue({ rows: [{ id: 2 }, { id: 1 }] });

      const result = await messageService.getMessages({ itemId: 4, userId: 10, otherUserId: 22, limit: 5, offset: 0 });

      const [sql, params] = db.query.mock.calls[0];
      expect(sql).toContain('(m.sender_id = $3 OR m.recipient_id = $3)');
      expect(sql).toContain('LIMIT $4 OFFSET $5');
      expect(params).toEqual([4, 10, 22, 5, 0]);
      expect(result.map(m => m.id)).toEqual([1, 2]); // oldest first
    });

    it('works without otherUserId for older clients', async () => {
      await messageService.getMessages({ taskId: 5, userId: 10, limit: 5, offset: 10 });
      const [sql, params] = db.query.mock.calls[0];
      expect(sql).toContain('LIMIT $3 OFFSET $4');
      expect(params).toEqual([5, 10, 5, 10]);
    });
  });

  describe('haveConversed', () => {
    it('is true when the two users exchanged a message in either direction', async () => {
      db.query.mockResolvedValue({ rows: [{ '?column?': 1 }] });
      expect(await messageService.haveConversed(1, 2)).toBe(true);
      const [sql, params] = db.query.mock.calls[0];
      expect(sql).toContain('(sender_id = $1 AND recipient_id = $2) OR (sender_id = $2 AND recipient_id = $1)');
      expect(params).toEqual([1, 2]);
    });

    it('is false for strangers', async () => {
      db.query.mockResolvedValue({ rows: [] });
      expect(await messageService.haveConversed(1, 99)).toBe(false);
    });
  });

  describe('getUserConversations', () => {
    it('returns conversation rows for the user', async () => {
      const mockRows = [
        { id: 1, task_id: 5, content: 'hi', other_user_id: 3, context_type: 'task', state: 'accepted' },
        { id: 2, store_item_id: 7, content: 'done', other_user_id: 4, context_type: 'store', state: 'completed' },
        { id: 3, task_id: 6, content: 'no', other_user_id: 5, context_type: 'task', state: 'declined' }
      ];
      db.query.mockResolvedValue({ rows: mockRows });

      const result = await messageService.getUserConversations(10);

      // Each says where it stands, and whether it has ended (done or closed)
      expect(result.map(r => [r.id, r.state, r.ended])).toEqual([[1, 'accepted', false], [2, 'completed', true], [3, 'declined', true]]);
      expect(db.query).toHaveBeenCalledWith(expect.any(String), [10]);
      // One conversation per context AND other participant, never per bare id
      expect(db.query.mock.calls[0][0]).toContain('DISTINCT ON (context_type, context_id, other_user_id)');
      // And when it got there (for "Completed 3 d ago")
      expect(db.query.mock.calls[0][0]).toContain('AS state_at');
    });

    it('lists each conversation with where it stands and its last message type', () => {
      const { formatConversation } = require('../src/services/messageService');
      expect(formatConversation({
        store_item_id: 7, content: 'Booking request for Bike', message_type: 'booking_request',
        created_at: 't', other_user_id: 4, unread_count: 1, state: 'completed', state_at: 'done-at', ended: true
      })).toMatchObject({
        item_id: 7, last_message_type: 'booking_request', state: 'completed', state_at: 'done-at', ended: true, conversation_type: 'store'
      });
    });

    it('returns empty array when user has no conversations', async () => {
      db.query.mockResolvedValue({ rows: [] });
      const result = await messageService.getUserConversations(999);
      expect(result).toEqual([]);
    });
  });

  describe('updateUserActivity', () => {
    it('executes upsert into user_activity', async () => {
      db.query.mockResolvedValue({ rows: [] });

      await messageService.updateUserActivity(1, 5);

      expect(db.query).toHaveBeenCalledWith(
        expect.stringContaining('INSERT INTO user_activity'),
        [1, 5]
      );
    });
  });

  describe('getUserLastSeen', () => {
    it('returns last_seen timestamp when user has activity', async () => {
      const mockDate = new Date('2024-01-01T10:00:00Z');
      db.query.mockResolvedValue({ rows: [{ last_seen: mockDate }] });

      const result = await messageService.getUserLastSeen(1);

      expect(result).toEqual(mockDate);
    });

    it('returns null when user has no recorded activity', async () => {
      db.query.mockResolvedValue({ rows: [] });
      const result = await messageService.getUserLastSeen(999);
      expect(result).toBeNull();
    });
  });

  describe('getMessagesForDeletion', () => {
    it('returns rows of old messages eligible for deletion', async () => {
      const mockRows = [{ task_id: 1, message_count: 5 }];
      db.query.mockResolvedValue({ rows: mockRows });

      const result = await messageService.getMessagesForDeletion();

      expect(result).toEqual(mockRows);
    });
  });

  describe('createDeletionWarning', () => {
    it('executes insert into message_deletion_warnings', async () => {
      db.query.mockResolvedValue({ rows: [] });

      await messageService.createDeletionWarning(1, new Date('2024-06-01'));

      expect(db.query).toHaveBeenCalledWith(
        expect.stringContaining('message_deletion_warnings'),
        expect.arrayContaining([1])
      );
    });
  });

  describe('getUserDeletionWarnings', () => {
    it('returns deletion warning rows for a user', async () => {
      const mockRows = [{ id: 1, task_id: 5, deletion_scheduled_at: new Date() }];
      db.query.mockResolvedValue({ rows: mockRows });

      const result = await messageService.getUserDeletionWarnings(1);

      expect(result).toEqual(mockRows);
    });
  });

  describe('markWarningAsShown', () => {
    it('executes update on message_deletion_warnings', async () => {
      db.query.mockResolvedValue({ rows: [] });

      await messageService.markWarningAsShown(5);

      expect(db.query).toHaveBeenCalledWith(
        expect.stringContaining('warning_shown = true'),
        [5]
      );
    });
  });

  describe('deleteOldMessages', () => {
    it('returns deleted count from query result', async () => {
      db.query.mockResolvedValue({ rows: [{ count: '15' }] });

      const result = await messageService.deleteOldMessages(1);

      expect(result).toBe('15');
    });
  });

  describe('getUserTaskMessageCount', () => {
    it('returns parsed integer count', async () => {
      db.query.mockResolvedValue({ rows: [{ count: '5' }] });
      const result = await messageService.getUserTaskMessageCount(1, 2);
      expect(result).toBe(5);
    });
  });

  describe('getTaskMessageLimit', () => {
    it('returns default limit of 3 (not yet implemented — separate DB)', async () => {
      const result = await messageService.getTaskMessageLimit(1, 2);
      expect(result).toBe(3);
    });
  });

  describe('getMessages', () => {
    it('returns messages for a store item (itemId branch)', async () => {
      const mockRows = [{ id: 1 }, { id: 2 }];
      db.query.mockResolvedValue({ rows: mockRows });

      const result = await messageService.getMessages({ itemId: 5, userId: 10, limit: 10, offset: 0 });

      expect(result).toHaveLength(2);
      expect(db.query).toHaveBeenCalledWith(
        expect.stringContaining('store_item_id = $1'),
        expect.any(Array)
      );
    });

    it('returns messages for a task (taskId branch)', async () => {
      const mockRows = [{ id: 3 }];
      db.query.mockResolvedValue({ rows: mockRows });

      const result = await messageService.getMessages({ taskId: 1, userId: 10, limit: 10, offset: 0 });

      expect(result).toHaveLength(1);
      expect(db.query).toHaveBeenCalledWith(
        expect.stringContaining('task_id = $1'),
        expect.any(Array)
      );
    });

    it('returns messages for an application (applicationId branch)', async () => {
      const mockRows = [{ id: 4 }];
      db.query.mockResolvedValue({ rows: mockRows });

      const result = await messageService.getMessages({ applicationId: 2, userId: 10, limit: 10, offset: 0 });

      expect(result).toHaveLength(1);
      expect(db.query).toHaveBeenCalledWith(
        expect.stringContaining('application_id = $1'),
        expect.any(Array)
      );
    });

    it('returns empty array for unknown conversation type (no id provided)', async () => {
      const result = await messageService.getMessages({ userId: 10 });
      expect(result).toEqual([]);
    });

    it('uses default limit and offset when not provided', async () => {
      db.query.mockResolvedValue({ rows: [] });
      await messageService.getMessages({ taskId: 1, userId: 5 });
      expect(db.query).toHaveBeenCalledWith(
        expect.any(String),
        [1, 5, 5, 0]
      );
    });
  });

  describe('getTotalMessageCount', () => {
    it('counts store item messages (itemId branch)', async () => {
      db.query.mockResolvedValue({ rows: [{ total: '7' }] });
      const result = await messageService.getTotalMessageCount({ itemId: 5, userId: 10 });
      expect(result).toBe(7);
    });

    it('counts task messages (taskId branch)', async () => {
      db.query.mockResolvedValue({ rows: [{ total: '3' }] });
      const result = await messageService.getTotalMessageCount({ taskId: 1, userId: 10 });
      expect(result).toBe(3);
    });

    it('counts application messages (applicationId branch)', async () => {
      db.query.mockResolvedValue({ rows: [{ total: '2' }] });
      const result = await messageService.getTotalMessageCount({ applicationId: 3, userId: 10 });
      expect(result).toBe(2);
    });

    it('returns 0 when no context identifier is provided', async () => {
      const result = await messageService.getTotalMessageCount({ userId: 10 });
      expect(result).toBe(0);
    });
  });

  describe('createStoreMessage', () => {
    it('creates message successfully within a transaction', async () => {
      const mockMsg = { id: 1, content: 'Hello there', store_item_id: 5, sender_id: 1, recipient_id: 2 };
      mockClient.query
        .mockResolvedValueOnce({})             // BEGIN
        .mockResolvedValueOnce({ rows: [mockMsg] }) // INSERT
        .mockResolvedValueOnce({});            // COMMIT

      const result = await messageService.createStoreMessage({
        store_item_id: 5,
        sender_id: 1,
        recipient_id: 2,
        content: 'Hello there'
      });

      expect(result.id).toBe(1);
      expect(mockClient.query).toHaveBeenCalledWith('BEGIN');
      expect(mockClient.query).toHaveBeenCalledWith('COMMIT');
      expect(mockClient.release).toHaveBeenCalled();
    });

    it('filters PII from message content before inserting', async () => {
      const mockMsg = { id: 2, content: 'Hi, email is [email removed]', store_item_id: 5 };
      mockClient.query
        .mockResolvedValueOnce({})
        .mockResolvedValueOnce({ rows: [mockMsg] })
        .mockResolvedValueOnce({});

      const result = await messageService.createStoreMessage({
        store_item_id: 5,
        sender_id: 1,
        recipient_id: 2,
        content: 'Hi, email is user@example.com'
      });

      expect(result.id).toBe(2);
      // Verify INSERT was called with filtered content (no raw email)
      const insertCall = mockClient.query.mock.calls.find(
        call => typeof call[0] === 'string' && call[0].includes('INSERT INTO messages')
      );
      expect(insertCall[1][3]).not.toContain('user@example.com');
    });

    it('rolls back transaction on database error', async () => {
      mockClient.query
        .mockResolvedValueOnce({})             // BEGIN
        .mockRejectedValueOnce(new Error('DB error')); // INSERT fails

      await expect(messageService.createStoreMessage({
        store_item_id: 5,
        sender_id: 1,
        recipient_id: 2,
        content: 'test'
      })).rejects.toThrow('DB error');

      expect(mockClient.query).toHaveBeenCalledWith('ROLLBACK');
      expect(mockClient.release).toHaveBeenCalled();
    });
  });

  describe('sendMessage', () => {
    it('sends a task message successfully', async () => {
      const mockMsg = { id: 1, content: 'Hello', task_id: 1, message_type: 'task' };
      mockClient.query
        .mockResolvedValueOnce({})             // BEGIN
        .mockResolvedValueOnce({ rows: [{}] }) // matched: the gig chat is open
        .mockResolvedValueOnce({ rows: [{ state: 'accepted' }] }) // still going
        .mockResolvedValueOnce({ rows: [{}] }) // contact unlock check
        .mockResolvedValueOnce({ rows: [mockMsg] }) // INSERT message
        .mockResolvedValueOnce({});            // COMMIT
      db.query.mockResolvedValue({ rows: [] }); // updateUserActivity

      const result = await messageService.sendMessage({
        taskId: 1,
        senderId: 1,
        recipientId: 2,
        content: 'Hello'
      });

      expect(result.id).toBe(1);
      expect(result.message_type).toBe('task');
    });

    it('sends a store message when storeItemId is provided', async () => {
      const mockMsg = { id: 2, content: 'Store msg', store_item_id: 5, message_type: 'store' };
      mockClient.query
        .mockResolvedValueOnce({})
        .mockResolvedValueOnce({ rows: [{}] }) // approved: the chat is open
        .mockResolvedValueOnce({ rows: [{ state: 'approved' }] }) // still going
        .mockResolvedValueOnce({ rows: [{}] }) // contact unlock check
        .mockResolvedValueOnce({ rows: [mockMsg] })
        .mockResolvedValueOnce({});
      db.query.mockResolvedValue({ rows: [] });

      const result = await messageService.sendMessage({
        storeItemId: 5,
        senderId: 1,
        recipientId: 2,
        content: 'Store msg'
      });

      expect(result.message_type).toBe('store');
    });

    it('rolls back on error', async () => {
      mockClient.query
        .mockResolvedValueOnce({})             // BEGIN
        .mockRejectedValueOnce(new Error('Insert failed'));

      await expect(messageService.sendMessage({
        taskId: 1,
        senderId: 1,
        recipientId: 2,
        content: 'Test'
      })).rejects.toThrow('Insert failed');

      expect(mockClient.query).toHaveBeenCalledWith('ROLLBACK');
      expect(mockClient.release).toHaveBeenCalled();
    });
  });

  describe('message length', () => {
    it('refuses messages over the hard limit before touching the database', async () => {
      const content = 'a'.repeat(messageService.MAX_MESSAGE_LENGTH + 1);
      await expect(messageService.sendMessage({ taskId: 1, senderId: 1, recipientId: 2, content }))
        .rejects.toMatchObject({ status: 400 });
      await expect(messageService.editMessage(1, 1, content)).rejects.toMatchObject({ status: 400 });
      expect(db.getClient).not.toHaveBeenCalled();
    });
  });

  describe('contact unlocks', () => {
    const insertedContent = () =>
      mockClient.query.mock.calls.find(([sql]) => /INSERT INTO messages/.test(sql))[1][5];

    const send = (unlocked, extra = {}) => {
      mockClient.query.mockImplementation(async (sql) => {
        if (/FROM contact_unlocks/.test(sql)) return { rows: unlocked ? [{ '?column?': 1 }] : [] };
        if (/INSERT INTO messages/.test(sql)) return { rows: [{ id: 1 }] };
        if (/AS state/.test(sql)) return { rows: [{ state: 'approved' }] };
        return {};
      });
      return messageService.sendMessage({ taskId: 4, senderId: 9, recipientId: 2, content: 'call 0772 123 456', ...extra });
    };

    it('masks contacts in chats that never unlock', async () => {
      // (gig and marketplace chats can't be written before a match at all)
      const result = await send(false, { taskId: undefined, applicationId: 3 });
      expect(insertedContent()).toBe('call [phone removed]');
      expect(result.hasRemovedContent).toBe(true);
    });

    it('keeps contacts once they are matched', async () => {
      const result = await send(true);
      expect(insertedContent()).toBe('call 0772 123 456');
      expect(result.hasRemovedContent).toBe(false);
    });

    it('looks the pair up in either direction', async () => {
      await send(true, { taskId: undefined, storeItemId: 4 });
      const lookup = mockClient.query.mock.calls.find(([sql]) => /FROM contact_unlocks/.test(sql));
      expect(lookup[1]).toEqual(['store', 4, 2, 9]);
    });

    it('never unlocks application chats', async () => {
      await send(true, { taskId: undefined, applicationId: 3 });
      expect(insertedContent()).toBe('call [phone removed]');
    });

    it('records an unlock with the pair ordered', async () => {
      await messageService.unlockContacts({ storeItemId: '7', userA: 12, userB: '5' });
      expect(db.query).toHaveBeenCalledWith(expect.stringContaining('INSERT INTO contact_unlocks'), ['store', 7, 5, 12]);
    });

    it('refuses an unlock without a context or with one person', async () => {
      await expect(messageService.unlockContacts({ userA: 1, userB: 2 })).rejects.toThrow();
      await expect(messageService.unlockContacts({ taskId: 1, userA: 2, userB: 2 })).rejects.toThrow();
    });
  });

  describe('editMessage', () => {
    it('edits a message when the user owns it', async () => {
      const updatedMsg = { id: 1, content: 'Updated content', is_edited: true };
      mockClient.query
        .mockResolvedValueOnce({})                           // BEGIN
        .mockResolvedValueOnce({ rows: [{ id: 1, sender_id: 10 }] }) // ownership check
        .mockResolvedValueOnce({ rows: [updatedMsg] })       // UPDATE
        .mockResolvedValueOnce({});                          // COMMIT

      const result = await messageService.editMessage(1, 10, 'Updated content');

      expect(result.content).toBe('Updated content');
      expect(mockClient.release).toHaveBeenCalled();
    });

    it('throws when user does not own the message', async () => {
      mockClient.query
        .mockResolvedValueOnce({})             // BEGIN
        .mockResolvedValueOnce({ rows: [] }); // ownership check returns nothing

      await expect(messageService.editMessage(99, 10, 'new content'))
        .rejects.toThrow('Message not found or unauthorized');

      expect(mockClient.query).toHaveBeenCalledWith('ROLLBACK');
      expect(mockClient.release).toHaveBeenCalled();
    });
  });
});
