jest.mock('../src/utils/logger', () => ({ debug: jest.fn(), info: jest.fn(), warn: jest.fn(), error: jest.fn() }));

const jwt = require('jsonwebtoken');
const express = require('express');
const request = require('supertest');
const { apiRateLimit, requestKey } = require('../src/middleware/rateLimit');

const JWT_SECRET = 'your-secret-key'; // matches the default in auth.js

describe('apiRateLimit', () => {
  const token = (userId) => jwt.sign({ user_id: userId }, JWT_SECRET);

  it('counts signed-in requests per user, others per IP', () => {
    expect(requestKey({ headers: { authorization: `Bearer ${token(7)}` }, ip: '10.0.0.1' })).toBe('user:7');
    expect(requestKey({ headers: { authorization: 'Bearer forged' }, ip: '10.0.0.1' })).toBe('ip:10.0.0.1');
    expect(requestKey({ headers: {}, ip: '10.0.0.2' })).toBe('ip:10.0.0.2');
  });

  it('answers 429 once a user exceeds the limit, without affecting others', async () => {
    const app = express();
    app.use(apiRateLimit);
    app.get('/ping', (req, res) => res.json({ ok: true }));
    app.post('/internal/task-message', (req, res) => res.status(201).end());

    const busy = `Bearer ${token(1)}`;
    for (let i = 0; i < 300; i++) {
      await request(app).get('/ping').set('Authorization', busy).expect(200);
    }
    const blocked = await request(app).get('/ping').set('Authorization', busy);
    expect(blocked.status).toBe(429);
    expect(blocked.body.error).toMatch(/too many requests/i);

    await request(app).get('/ping').set('Authorization', `Bearer ${token(2)}`).expect(200);
    // service-to-service calls are never limited
    await request(app).post('/internal/task-message').set('Authorization', busy).expect(201);
  });
});
