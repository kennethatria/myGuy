jest.mock('../src/config/database', () => ({
  query: jest.fn()
}));

jest.mock('../src/utils/logger', () => ({
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn()
}));

jest.mock('../src/services/bookingMessageService', () => ({
  createBookingRequestMessage: jest.fn(),
  updateBookingMessageStatus: jest.fn()
}));

const request = require('supertest');
const express = require('express');
const jwt = require('jsonwebtoken');

const db = require('../src/config/database');
const bookingMessageService = require('../src/services/bookingMessageService');

const JWT_SECRET = 'your-secret-key'; // matches default in auth.js
const INTERNAL_API_KEY = 'test-internal-api-key';

// Build a minimal express app for testing the router
function buildApp() {
  const app = express();
  app.use(express.json());

  const emitFn = jest.fn();
  const mockIo = { to: jest.fn().mockReturnValue({ emit: emitFn }) };
  app.set('io', mockIo);

  const router = require('../src/api/bookingNotifications');
  app.use('/', router);
  return app;
}

const createUserToken = (userId) =>
  jwt.sign({ user_id: userId, email: `u${userId}@test.com`, name: `User${userId}` }, JWT_SECRET);

describe('bookingNotifications router', () => {
  let app;

  beforeAll(() => {
    process.env.INTERNAL_API_KEY = INTERNAL_API_KEY;
    app = buildApp();
  });

  beforeEach(() => {
    jest.clearAllMocks();
    db.query.mockResolvedValue({ rows: [] });
  });

  describe('POST /internal/booking-created', () => {
    it('returns 401 when x-internal-api-key header is missing', async () => {
      await request(app)
        .post('/internal/booking-created')
        .send({ bookingId: 1, itemId: 1, buyerId: 2, sellerId: 3 })
        .expect(401);
    });

    it('returns 401 when x-internal-api-key is incorrect', async () => {
      await request(app)
        .post('/internal/booking-created')
        .set('x-internal-api-key', 'wrong-key')
        .send({ bookingId: 1, itemId: 1, buyerId: 2, sellerId: 3 })
        .expect(401);
    });

    it('returns 400 when required fields are missing', async () => {
      await request(app)
        .post('/internal/booking-created')
        .set('x-internal-api-key', INTERNAL_API_KEY)
        .send({ bookingId: 1 }) // missing itemId, buyerId, sellerId
        .expect(400);
    });

    it('creates booking notification and returns success with messageId', async () => {
      bookingMessageService.createBookingRequestMessage.mockResolvedValue({ id: 99 });

      const res = await request(app)
        .post('/internal/booking-created')
        .set('x-internal-api-key', INTERNAL_API_KEY)
        .send({ bookingId: 1, itemId: 1, itemTitle: 'Test Item', buyerId: 2, sellerId: 3 })
        .expect(200);

      expect(res.body.success).toBe(true);
      expect(res.body.messageId).toBe(99);
    });

    it('uses default itemTitle when not provided', async () => {
      bookingMessageService.createBookingRequestMessage.mockResolvedValue({ id: 100 });

      const res = await request(app)
        .post('/internal/booking-created')
        .set('x-internal-api-key', INTERNAL_API_KEY)
        .send({ bookingId: 2, itemId: 5, buyerId: 2, sellerId: 3 })
        .expect(200);

      expect(bookingMessageService.createBookingRequestMessage).toHaveBeenCalledWith(
        expect.objectContaining({ itemTitle: 'Item #5' })
      );
    });

    it('returns 500 when service throws', async () => {
      bookingMessageService.createBookingRequestMessage.mockRejectedValue(new Error('DB error'));

      await request(app)
        .post('/internal/booking-created')
        .set('x-internal-api-key', INTERNAL_API_KEY)
        .send({ bookingId: 1, itemId: 1, buyerId: 2, sellerId: 3 })
        .expect(500);
    });
  });

  describe('POST /booking-action', () => {
    it('returns 401 without authorization header', async () => {
      await request(app)
        .post('/booking-action')
        .send({ bookingId: 1, action: 'approve' })
        .expect(401);
    });

    it('returns 400 for an invalid action value', async () => {
      const token = createUserToken(1);
      await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ bookingId: 1, action: 'invalid-action' })
        .expect(400);
    });

    it('returns 400 when bookingId is missing', async () => {
      const token = createUserToken(1);
      await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ action: 'approve' })
        .expect(400);
    });

    it('returns 400 for rate-seller with rating below 1', async () => {
      const token = createUserToken(1);
      await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ bookingId: 1, action: 'rate-seller', rating: 0 })
        .expect(400);
    });

    it('returns 400 for rate-buyer with rating above 5', async () => {
      const token = createUserToken(1);
      await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ bookingId: 1, action: 'rate-buyer', rating: 6 })
        .expect(400);
    });

    it('returns 400 for rate-seller with missing rating', async () => {
      const token = createUserToken(1);
      await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ bookingId: 1, action: 'rate-seller' })
        .expect(400);
    });

    it('handles approve action and returns success', async () => {
      const token = createUserToken(1);
      const mockBooking = { status: 'approved' };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: jest.fn().mockResolvedValue(mockBooking)
      });
      bookingMessageService.updateBookingMessageStatus.mockResolvedValue({ id: 1 });

      const res = await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ bookingId: 1, action: 'approve' })
        .expect(200);

      expect(res.body.success).toBe(true);
    });

    it('handles decline action and returns success', async () => {
      const token = createUserToken(1);
      const mockBooking = { status: 'rejected' };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: jest.fn().mockResolvedValue(mockBooking)
      });
      bookingMessageService.updateBookingMessageStatus.mockResolvedValue({ id: 2 });

      await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ bookingId: 1, action: 'decline' })
        .expect(200);
    });

    it('handles confirm-received action', async () => {
      const token = createUserToken(1);
      const mockBooking = { status: 'item_received' };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: jest.fn().mockResolvedValue(mockBooking)
      });
      bookingMessageService.updateBookingMessageStatus.mockResolvedValue({ id: 3 });

      await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ bookingId: 1, action: 'confirm-received' })
        .expect(200);
    });

    it('handles confirm-delivery action', async () => {
      const token = createUserToken(1);
      const mockBooking = { status: 'completed' };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: jest.fn().mockResolvedValue(mockBooking)
      });
      bookingMessageService.updateBookingMessageStatus.mockResolvedValue({ id: 4 });

      await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ bookingId: 1, action: 'confirm-delivery' })
        .expect(200);
    });

    it('handles rate-seller action and updates message metadata', async () => {
      const token = createUserToken(1);
      const mockBooking = {
        status: 'completed',
        seller_rating: 5,
        seller_review: 'Great',
        item: { seller_id: 2 }
      };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: jest.fn().mockResolvedValue(mockBooking)
      });

      db.query
        .mockResolvedValueOnce({ rows: [{ id: 10, sender_id: 2, metadata: { booking_id: 1 } }] })
        .mockResolvedValueOnce({ rows: [] }); // UPDATE metadata

      await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ bookingId: 1, action: 'rate-seller', rating: 5, review: 'Great seller' })
        .expect(200);
    });

    it('handles rate-buyer action', async () => {
      const token = createUserToken(1);
      const mockBooking = { status: 'completed', buyer_rating: 4 };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: jest.fn().mockResolvedValue(mockBooking)
      });

      db.query
        .mockResolvedValueOnce({ rows: [{ id: 11, sender_id: 3, metadata: { booking_id: 1 } }] })
        .mockResolvedValueOnce({ rows: [] });

      await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ bookingId: 1, action: 'rate-buyer', rating: 4 })
        .expect(200);
    });

    it('handles rate-seller when booking request message not found (skips update)', async () => {
      const token = createUserToken(1);
      const mockBooking = { status: 'completed', item: null };

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: jest.fn().mockResolvedValue(mockBooking)
      });

      db.query.mockResolvedValue({ rows: [] }); // no booking request message found

      await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ bookingId: 1, action: 'rate-seller', rating: 3 })
        .expect(200);
    });

    it('returns 500 when store service returns an error response', async () => {
      const token = createUserToken(1);

      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 503,
        text: jest.fn().mockResolvedValue('Service Unavailable')
      });

      await request(app)
        .post('/booking-action')
        .set('Authorization', `Bearer ${token}`)
        .send({ bookingId: 1, action: 'approve' })
        .expect(500);
    });
  });
});
