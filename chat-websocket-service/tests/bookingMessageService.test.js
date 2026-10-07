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
      // The text previews the conversation; no note of the buyer's own
      const [, params] = db.query.mock.calls[0];
      expect(params[4]).toBe('Booking request for Another Item');
      expect(JSON.parse(params[5]).note).toBe('');
    });

    it('delivers message:new to both seller and buyer when io is provided', async () => {
      const mockMsg = { id: 3 };
      db.query.mockResolvedValue({ rows: [mockMsg] });

      const emitFn = jest.fn();
      const chain = { emit: emitFn };
      const mockIo = { to: jest.fn(() => chain) };
      chain.to = mockIo.to;

      await createBookingRequestMessage({
        bookingId: 102,
        itemId: 1,
        itemTitle: 'Item',
        buyerId: 2,
        sellerId: 5,
        io: mockIo
      });

      expect(mockIo.to).toHaveBeenCalledWith('user:5');
      expect(mockIo.to).toHaveBeenCalledWith('user:2');
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
    // The buyer (2) asked the seller (3) to book
    const mockRequestMessage = {
      id: 10,
      store_item_id: 5,
      sender_id: 2,
      recipient_id: 3,
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

    it('tells both people as a system note when the seller releases the reservation', async () => {
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [{ id: 24, message_type: 'system_alert' }] });

      await updateBookingMessageStatus(100, 'released', 3, null);

      const insert = db.query.mock.calls[3];
      expect(insert[1][3]).toBe('system_alert');
      expect(insert[1][4]).toContain('released the reservation');
    });

    it('posts the given note as a system message instead of the usual note', async () => {
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [{ id: 28 }] });

      await updateBookingMessageStatus(100, 'rejected', 3, null, null, 'The seller removed "Bike".');

      const insert = db.query.mock.calls[3];
      expect(insert[1][3]).toBe('system_alert');
      expect(insert[1][4]).toBe('The seller removed "Bike".');
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

    it('does not mistake the request itself for an existing status message', async () => {
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [{ id: 27 }] });

      await updateBookingMessageStatus(100, 'approved', 3, null);

      // the lookup leaves out the request message, which now has the status too
      expect(db.query.mock.calls[2][1]).toContain(mockRequestMessage.id);
      expect(db.query.mock.calls[2][0]).toMatch(/id <> \$4/);
    });

    it('emits to both users via WebSocket when io is provided', async () => {
      const mockStatusMsg = { id: 26 };
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [mockStatusMsg] });

      const emitFn = jest.fn();
      const chain = { emit: emitFn };
      const mockIo = { to: jest.fn(() => chain) };
      chain.to = mockIo.to;

      await updateBookingMessageStatus(100, 'approved', 3, mockIo);

      expect(mockIo.to).toHaveBeenCalledWith('user:2');
      expect(mockIo.to).toHaveBeenCalledWith('user:3');
      expect(emitFn).toHaveBeenCalledWith('message:new', mockStatusMsg);
    });

    it('addresses a note to the other person, whoever acted', async () => {
      for (const [actor, other] of [[3, 2], [2, 3]]) {
        db.query.mockReset();
        db.query
          .mockResolvedValueOnce({ rows: [mockRequestMessage] })
          .mockResolvedValueOnce({ rows: [] })
          .mockResolvedValueOnce({ rows: [] })
          .mockResolvedValueOnce({ rows: [{ id: 30 }] });

        await updateBookingMessageStatus(100, actor === 3 ? 'picked_up' : 'completed', actor, null);

        const insert = db.query.mock.calls[3][1];
        // sender, recipient: never the actor to themselves
        expect([insert[0], insert[1]]).toEqual([actor, other]);
      }
    });

    it('asks the buyer to confirm once the seller marks it picked up', async () => {
      db.query
        .mockResolvedValueOnce({ rows: [mockRequestMessage] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [{ id: 31 }] });

      await updateBookingMessageStatus(100, 'picked_up', 3, null);

      const insert = db.query.mock.calls[3][1];
      expect(insert[3]).toBe('booking_picked_up');
      expect(insert[4]).toContain('Confirm once you have it');
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
