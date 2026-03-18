jest.mock('../src/config/database', () => ({
  query: jest.fn()
}));

const db = require('../src/config/database');
const { createBookingRequestMessage, updateBookingMessageStatus } = require('../src/services/bookingMessageService');

describe('bookingMessageService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('createBookingRequestMessage', () => {
    it('creates a booking request message and returns it', async () => {
      const mockMsg = { id: 1, message_type: 'booking_request', store_item_id: 1 };
      db.query.mockResolvedValue({ rows: [mockMsg] });

      const result = await createBookingRequestMessage({
        bookingId: 100,
        itemId: 1,
        itemTitle: 'Test Item',
        buyerId: 2,
        sellerId: 3,
        message: 'Booking request for Test Item'
      });

      expect(result.id).toBe(1);
      expect(db.query).toHaveBeenCalledWith(
        expect.stringContaining('message_type'),
        expect.arrayContaining(['booking_request'])
      );
    });

    it('uses default message when message param is omitted', async () => {
      const mockMsg = { id: 2, message_type: 'booking_request' };
      db.query.mockResolvedValue({ rows: [mockMsg] });

      const result = await createBookingRequestMessage({
        bookingId: 101,
        itemId: 2,
        itemTitle: 'Another Item',
        buyerId: 2,
        sellerId: 3
      });

      expect(result.id).toBe(2);
    });

    it('emits message:new to seller socket when io is provided', async () => {
      const mockMsg = { id: 3 };
      db.query.mockResolvedValue({ rows: [mockMsg] });

      const emitFn = jest.fn();
      const mockIo = { to: jest.fn().mockReturnValue({ emit: emitFn }) };

      await createBookingRequestMessage({
        bookingId: 102,
        itemId: 1,
        itemTitle: 'Item',
        buyerId: 2,
        sellerId: 5,
        io: mockIo
      });

      expect(mockIo.to).toHaveBeenCalledWith('user:5');
      expect(emitFn).toHaveBeenCalledWith('message:new', mockMsg);
    });

    it('does not throw when io is not provided', async () => {
      const mockMsg = { id: 4 };
      db.query.mockResolvedValue({ rows: [mockMsg] });

      await expect(createBookingRequestMessage({
        bookingId: 103,
        itemId: 1,
        buyerId: 2,
        sellerId: 3
      })).resolves.toEqual(mockMsg);
    });

    it('propagates database errors', async () => {
      db.query.mockRejectedValue(new Error('DB connection failed'));

      await expect(createBookingRequestMessage({
        bookingId: 1,
        itemId: 1,
        buyerId: 2,
        sellerId: 3
      })).rejects.toThrow('DB connection failed');
    });
  });

  describe('updateBookingMessageStatus', () => {
    const mockRequestMessage = {
      id: 10,
      store_item_id: 5,
      sender_id: 2,
      metadata: { booking_id: 100, item_id: 1 }
    };

    it('throws when the original booking request message is not found', async () => {
      db.query.mockResolvedValue({ rows: [] });

      await expect(updateBookingMessageStatus(100, 'approved', 3, null))
        .rejects.toThrow('Booking request message not found');
    });

    it('creates an "approved" booking_approved status message', async () => {
      const mockStatusMsg = { id: 20, message_type: 'booking_approved' };
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] }) // find original
        .mockResolvedValueOnce({ rows: [] })                   // update metadata
        .mockResolvedValueOnce({ rows: [] })                   // check existing status
        .mockResolvedValueOnce({ rows: [mockStatusMsg] });     // insert status msg

      const result = await updateBookingMessageStatus(100, 'approved', 3, null);

      expect(result.id).toBe(20);
      expect(result.message_type).toBe('booking_approved');
    });

    it('creates a "declined" booking_declined status message', async () => {
      const mockStatusMsg = { id: 21, message_type: 'booking_declined' };
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [mockStatusMsg] });

      const result = await updateBookingMessageStatus(100, 'rejected', 3, null);

      expect(result.message_type).toBe('booking_declined');
    });

    it('creates a "item_received" booking_item_received status message', async () => {
      const mockStatusMsg = { id: 22, message_type: 'booking_item_received' };
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [mockStatusMsg] });

      const result = await updateBookingMessageStatus(100, 'item_received', 3, null);

      expect(result.message_type).toBe('booking_item_received');
    });

    it('creates a "completed" booking_completed status message', async () => {
      const mockStatusMsg = { id: 23, message_type: 'booking_completed' };
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [mockStatusMsg] });

      const result = await updateBookingMessageStatus(100, 'completed', 3, null);

      expect(result.message_type).toBe('booking_completed');
    });

    it('creates a generic "booking_status_update" for an unknown status', async () => {
      const mockStatusMsg = { id: 24, message_type: 'booking_status_update' };
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [mockStatusMsg] });

      const result = await updateBookingMessageStatus(100, 'pending', 3, null);

      expect(result.message_type).toBe('booking_status_update');
    });

    it('reuses an existing status message to prevent duplicates', async () => {
      const existingMsg = { id: 25, message_type: 'booking_approved' };
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] }) // find original
        .mockResolvedValueOnce({ rows: [] })                   // update metadata
        .mockResolvedValueOnce({ rows: [existingMsg] });       // existing status msg found

      const result = await updateBookingMessageStatus(100, 'approved', 3, null);

      expect(result.id).toBe(25);
      expect(db.query).toHaveBeenCalledTimes(3); // no INSERT for new status msg
    });

    it('emits to both users via WebSocket when io is provided', async () => {
      const mockStatusMsg = { id: 26 };
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [mockStatusMsg] });

      const emitFn = jest.fn();
      const mockIo = { to: jest.fn().mockReturnValue({ emit: emitFn }) };

      await updateBookingMessageStatus(100, 'approved', 3, mockIo);

      expect(mockIo.to).toHaveBeenCalled();
      expect(emitFn).toHaveBeenCalledWith('message:new', mockStatusMsg);
    });

    it('includes rating data in metadata when bookingData has ratings', async () => {
      const mockStatusMsg = { id: 27, message_type: 'booking_approved' };
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [mockStatusMsg] });

      const bookingData = {
        buyer_rating: 5,
        buyer_review: 'Excellent',
        seller_rating: 4,
        seller_review: 'Good'
      };

      const result = await updateBookingMessageStatus(100, 'approved', 3, null, bookingData);

      expect(result.id).toBe(27);
      // Check that the metadata update was called with rating data
      const updateCall = db.query.mock.calls.find(
        call => typeof call[0] === 'string' && call[0].includes('UPDATE messages')
      );
      const updatedMetadata = JSON.parse(updateCall[1][0]);
      expect(updatedMetadata.buyer_rating).toBe(5);
      expect(updatedMetadata.seller_rating).toBe(4);
    });

    it('handles null rating values gracefully in bookingData', async () => {
      const mockStatusMsg = { id: 28, message_type: 'booking_approved' };
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [mockStatusMsg] });

      const bookingData = {
        buyer_rating: null,
        seller_rating: null
      };

      const result = await updateBookingMessageStatus(100, 'approved', 3, null, bookingData);

      expect(result.id).toBe(28);
    });

    it('propagates database errors', async () => {
      db.query.mockRejectedValue(new Error('DB error'));

      await expect(updateBookingMessageStatus(100, 'approved', 3, null))
        .rejects.toThrow('DB error');
    });
  });
});
