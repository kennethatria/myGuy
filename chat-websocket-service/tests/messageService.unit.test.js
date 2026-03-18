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
    it('returns array of updated message IDs', async () => {
      db.query.mockResolvedValue({ rows: [{ id: 1 }, { id: 2 }] });

      const result = await messageService.markConversationAsRead(5, 10);

      expect(result).toHaveLength(2);
      expect(db.query).toHaveBeenCalledWith(
        expect.stringContaining('task_id = $1'),
        [5, 10]
      );
    });

    it('returns empty array when nothing to mark', async () => {
      db.query.mockResolvedValue({ rows: [] });
      const result = await messageService.markConversationAsRead(5, 10);
      expect(result).toEqual([]);
    });
  });

  describe('getUserConversations', () => {
    it('returns conversation rows for the user', async () => {
      const mockRows = [{ id: 1, task_id: 5, content: 'hi', other_user_id: 3 }];
      db.query.mockResolvedValue({ rows: mockRows });

      const result = await messageService.getUserConversations(10);

      expect(result).toEqual(mockRows);
      expect(db.query).toHaveBeenCalledWith(expect.any(String), [10]);
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
