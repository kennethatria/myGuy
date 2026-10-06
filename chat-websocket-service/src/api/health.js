/**
 * Public health check. It is reachable from the internet (nginx proxies
 * /chat/), so it says only whether the service works; the details (database
 * errors, Redis state, counts) go to the log.
 */
function healthHandler({ db, getRedisHealth, logger }) {
  return async (req, res) => {
    try {
      await db.query('SELECT 1');
    } catch (error) {
      logger.error('Health check: database unavailable:', error);
      return res.status(503).json({ status: 'degraded' });
    }

    try {
      const redis = await getRedisHealth();
      if (redis && redis.configured && !redis.connected) {
        logger.warn('Health check: Redis configured but not connected', redis);
      }
    } catch (error) {
      logger.warn('Health check: Redis check failed:', error);
    }

    res.json({ status: 'ok' });
  };
}

module.exports = { healthHandler };
