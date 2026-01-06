const { Pool } = require('pg');

// Use environment variable for connection string, fallback to example if not set
// In a real dockerized environment, DB_CONNECTION should be set in docker-compose.yml or .env
const dbConnectionString = process.env.DB_CONNECTION || 'postgresql://postgres:password@localhost:5432/myguy';

const pool = new Pool({
  connectionString: dbConnectionString,
  // SSL configuration for production environments.
  // For local development, ssl: false is often used if not using SSL.
  // In production, you might need to configure SSL properly.
  ssl: process.env.NODE_ENV === 'production' ? { rejectUnauthorized: false } : false,
});

// Ping the database to ensure connection
pool.on('connect', () => {
  console.log('✅ Connected to the PostgreSQL database');
});

pool.on('error', (err, client) => {
  console.error('❌ Unexpected error on idle client', err);
  // It's critical to handle pool errors. Depending on the error, you might want to exit.
  // For critical errors, exiting might be necessary to allow orchestration systems to restart the container.
  process.exit(-1);
});

/**
 * Executes a SQL query using the connection pool.
 * @param {string} text - The SQL query string.
 * @param {Array} params - An array of query parameters.
 * @returns {Promise<Object>} - The query result.
 */
const query = async (text, params) => {
  const start = Date.now();
  try {
    const res = await pool.query(text, params);
    const duration = Date.now() - start;
    console.log(`🚀 Executed query: ${text.substring(0, 100)}... in ${duration}ms`); // Log query and duration
    return res;
  } catch (e) {
    console.error('❌ Error executing query', { text, params, error: e });
    throw e;
  }
};

module.exports = {
  query,
  // You can export other pool methods or the pool itself if needed
  // pool,
};