const { Pool } = require('pg');
const logger = require('../utils/logger');

const pool = new Pool({
  connectionString: process.env.DB_CONNECTION || 'postgresql://postgres:mysecretpassword@localhost:5433/my_guy',
  max: 20,
  idleTimeoutMillis: 30000,
  connectionTimeoutMillis: 2000,
});

pool.on('error', (err) => {
  logger.error('Unexpected database error:', err);
});

const query = async (text, params) => {
  const start = Date.now();
  try {
    const res = await pool.query(text, params);
    const duration = Date.now() - start;
    logger.debug('Executed query', { text, duration, rows: res.rowCount });
    return res;
  } catch (error) {
    logger.error('Database query error:', { text, error: error.message });
    throw error;
  }
};

const getClient = async () => {
  const client = await pool.connect();
  const query = client.query.bind(client);
  const release = () => {
    client.release();
  };

  const timeout = setTimeout(() => {
    logger.error('Client has been checked out for more than 5 seconds!');
  }, 5000);

  client.on('error', (err) => {
    logger.error('Client error:', err);
    client.release();
  });

  return {
    query,
    release: () => {
      clearTimeout(timeout);
      release();
    }
  };
};

module.exports = {
  query,
  getClient,
  pool
};