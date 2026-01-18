/**
 * Redis Configuration for Socket.IO Adapter
 *
 * Enables horizontal scaling of the chat service by sharing
 * Socket.IO state across multiple instances via Redis pub/sub.
 *
 * When Redis is not configured, the service falls back to
 * in-memory adapter (single instance only).
 */

const { createClient } = require('redis');
const { createAdapter } = require('@socket.io/redis-adapter');
const logger = require('../utils/logger');

/**
 * Check if Redis is configured via environment variables
 */
function isRedisConfigured() {
  return !!(process.env.REDIS_URL || process.env.REDIS_HOST);
}

/**
 * Get Redis connection URL from environment
 */
function getRedisUrl() {
  if (process.env.REDIS_URL) {
    return process.env.REDIS_URL;
  }

  const host = process.env.REDIS_HOST || 'localhost';
  const port = process.env.REDIS_PORT || 6379;
  const password = process.env.REDIS_PASSWORD;
  const db = process.env.REDIS_DB || 0;

  if (password) {
    return `redis://:${password}@${host}:${port}/${db}`;
  }

  return `redis://${host}:${port}/${db}`;
}

/**
 * Create and configure Redis clients for Socket.IO adapter
 *
 * @returns {Promise<{pubClient: RedisClient, subClient: RedisClient} | null>}
 */
async function createRedisClients() {
  if (!isRedisConfigured()) {
    logger.info('Redis not configured - using in-memory Socket.IO adapter (single instance mode)');
    return null;
  }

  const redisUrl = getRedisUrl();
  logger.info('Connecting to Redis for Socket.IO adapter...');

  try {
    // Create pub/sub clients
    const pubClient = createClient({ url: redisUrl });
    const subClient = pubClient.duplicate();

    // Error handlers
    pubClient.on('error', (err) => {
      logger.error('Redis pub client error:', err);
    });

    subClient.on('error', (err) => {
      logger.error('Redis sub client error:', err);
    });

    // Connection handlers
    pubClient.on('connect', () => {
      logger.info('Redis pub client connected');
    });

    subClient.on('connect', () => {
      logger.info('Redis sub client connected');
    });

    pubClient.on('reconnecting', () => {
      logger.warn('Redis pub client reconnecting...');
    });

    subClient.on('reconnecting', () => {
      logger.warn('Redis sub client reconnecting...');
    });

    // Connect both clients
    await Promise.all([
      pubClient.connect(),
      subClient.connect()
    ]);

    logger.info('Redis clients connected successfully - multi-instance mode enabled');

    return { pubClient, subClient };
  } catch (error) {
    logger.error('Failed to connect to Redis:', error);
    logger.warn('Falling back to in-memory Socket.IO adapter (single instance mode)');
    return null;
  }
}

/**
 * Configure Socket.IO with Redis adapter for horizontal scaling
 *
 * @param {Server} io - Socket.IO server instance
 * @returns {Promise<boolean>} - True if Redis adapter was configured
 */
async function configureRedisAdapter(io) {
  const clients = await createRedisClients();

  if (!clients) {
    return false;
  }

  const { pubClient, subClient } = clients;

  // Configure Socket.IO to use Redis adapter
  io.adapter(createAdapter(pubClient, subClient));

  logger.info('Socket.IO Redis adapter configured successfully');

  return true;
}

/**
 * Get Redis health status for health check endpoint
 *
 * @returns {Promise<{configured: boolean, connected: boolean, mode: string}>}
 */
async function getRedisHealth() {
  if (!isRedisConfigured()) {
    return {
      configured: false,
      connected: false,
      mode: 'in-memory (single instance)'
    };
  }

  try {
    const testClient = createClient({ url: getRedisUrl() });
    await testClient.connect();
    await testClient.ping();
    await testClient.disconnect();

    return {
      configured: true,
      connected: true,
      mode: 'redis (multi-instance)'
    };
  } catch (error) {
    return {
      configured: true,
      connected: false,
      mode: 'redis (disconnected)',
      error: error.message
    };
  }
}

module.exports = {
  isRedisConfigured,
  createRedisClients,
  configureRedisAdapter,
  getRedisHealth
};
