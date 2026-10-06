const request = require('supertest');
const express = require('express');
const { healthHandler } = require('../src/api/health');

const logger = { error: jest.fn(), warn: jest.fn() };

const appWith = (db, getRedisHealth) => {
  const app = express();
  app.get('/health', healthHandler({ db, getRedisHealth, logger }));
  return app;
};

describe('GET /health', () => {
  beforeEach(() => jest.clearAllMocks());

  it('says only that the service is up', async () => {
    const db = { query: jest.fn().mockResolvedValue({ rows: [] }) };
    const res = await request(appWith(db, async () => ({ configured: true, connected: true }))).get('/health');

    expect(res.status).toBe(200);
    expect(res.body).toEqual({ status: 'ok' });
  });

  it('reports a database failure without its details', async () => {
    const db = { query: jest.fn().mockRejectedValue(new Error('password authentication failed for user "postgres"')) };
    const res = await request(appWith(db, async () => ({}))).get('/health');

    expect(res.status).toBe(503);
    expect(res.body).toEqual({ status: 'degraded' });
    expect(JSON.stringify(res.body)).not.toMatch(/postgres|password/);
    expect(logger.error).toHaveBeenCalled();
  });

  it('logs Redis trouble but stays up', async () => {
    const db = { query: jest.fn().mockResolvedValue({ rows: [] }) };
    let res = await request(appWith(db, async () => ({ configured: true, connected: false }))).get('/health');
    expect(res.body).toEqual({ status: 'ok' });
    res = await request(appWith(db, async () => { throw new Error('ECONNREFUSED'); })).get('/health');
    expect(res.body).toEqual({ status: 'ok' });
    expect(logger.warn).toHaveBeenCalledTimes(2);
  });
});
